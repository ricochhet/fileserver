package server

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"hash/fnv"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

const (
	sessionCookieName = "fs_session"
	sessionTTL        = 24 * time.Hour
	loginRoute        = "/auth/login"
	logoutRoute       = "/auth/logout"

	nextQuery       = "next"
	usernameFormKey = "username"
	passwordFormKey = "password"

	loginTmpl     = "login"
	loginTmplHTML = "login.html"
)

type ctxKey string

const (
	ctxKeyUsername    ctxKey = "fs_username"
	ctxKeyDisplayName ctxKey = "fs_displayName"
)

type loginPageData struct {
	Error string
	Next  string
}

var nameAdjectives = []string{
	"Amber", "Azure", "Brass", "Bright", "Calm", "Clever", "Cobalt",
	"Coral", "Crisp", "Dark", "Dawn", "Deep", "Dusty", "Early",
	"Ember", "Faint", "Firm", "Fleet", "Frost", "Gentle", "Gilt",
	"Gold", "Grand", "Gray", "Green", "Hazy", "High", "Iron",
	"Jade", "Keen", "Kind", "Late", "Light", "Lone", "Lucky",
	"Mild", "Misty", "Muted", "Navy", "Noble", "Pale", "Pine",
	"Prime", "Quiet", "Rapid", "Red", "Rich", "Rosy", "Royal",
	"Rusty", "Sand", "Sharp", "Shy", "Silver", "Slim", "Soft",
	"Stern", "Still", "Stone", "Storm", "Strong", "Sunny", "Swift",
	"Tall", "Teal", "Thin", "True", "Warm", "Wild", "Wise",
}

var nameAnimals = []string{
	"Albatross", "Axolotl", "Badger", "Bear", "Beaver", "Bison",
	"Bobcat", "Buffalo", "Camel", "Capybara", "Caribou", "Cheetah",
	"Chinchilla", "Condor", "Coyote", "Crane", "Crow", "Dingo",
	"Dolphin", "Donkey", "Dove", "Eagle", "Egret", "Elk", "Falcon",
	"Ferret", "Finch", "Fox", "Gecko", "Gopher", "Grouse", "Hare",
	"Hawk", "Heron", "Ibis", "Jackal", "Jaguar", "Jay", "Kestrel",
	"Kite", "Lemur", "Leopard", "Loon", "Lynx", "Marten", "Mink",
	"Mole", "Moose", "Narwhal", "Newt", "Ocelot", "Osprey", "Otter",
	"Owl", "Panther", "Pelican", "Pika", "Porcupine", "Puffin",
	"Quail", "Raven", "Robin", "Salamander", "Seal", "Shrew",
	"Skunk", "Sloth", "Sparrow", "Stoat", "Swift", "Tapir",
	"Thrush", "Tiger", "Vole", "Vulture", "Walrus", "Weasel",
	"Wolf", "Wolverine", "Wombat", "Wren", "Yak",
}

// generateDisplayName derives a stable adjective-animal display name from username via FNV-32a.
func generateDisplayName(username string) string {
	h := fnv.New32a()
	h.Write([]byte(username))
	n := h.Sum32()

	return nameAdjectives[n%uint32(len(nameAdjectives))] +
		nameAnimals[(n>>8)%uint32(len(nameAnimals))]
}

// resolveDisplayName returns the best display name for username: DB > config > auto-generated.
func resolveDisplayName(
	ctx context.Context,
	users []configutil.FormAuthUser,
	username string,
	database *db.DB,
) string {
	if database != nil {
		u, err := database.GetUser(ctx, username)
		if err != nil && !errors.Is(err, db.ErrUserNotFound) {
			logutil.Errorf(logutil.Get(), "resolveDisplayName: db.GetUser %q: %v\n", username, err)
		}

		if u != nil && u.DisplayName != "" {
			return u.DisplayName
		}
	}

	for _, u := range users {
		if u.Username == username && u.DisplayName != "" {
			return u.DisplayName
		}
	}

	return generateDisplayName(username)
}

// usernameFromCtx returns the authenticated username from the request context.
func usernameFromCtx(r *http.Request) string {
	v, _ := r.Context().Value(ctxKeyUsername).(string)
	return v
}

// displayNameFromCtx returns the display name from the request context.
func displayNameFromCtx(r *http.Request) string {
	v, _ := r.Context().Value(ctxKeyDisplayName).(string)
	return v
}

// newSessionSecret generates a cryptographically random 32-byte secret.
func newSessionSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: cannot generate session secret: " + err.Error())
	}

	return b
}

// signSession returns a tamper-evident token: base64url(username).unix_ts.hex(HMAC-SHA256).
func signSession(secret []byte, username string, ts int64) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(username))
	tsStr := strconv.FormatInt(ts, 10)
	payload := encoded + "." + tsStr

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))

	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

// verifySession validates a session token and returns the embedded username on success.
func verifySession(secret []byte, value string) (string, bool) {
	parts := strings.SplitN(value, ".", 3)
	if len(parts) != 3 {
		return "", false
	}

	encoded, tsStr, sig := parts[0], parts[1], parts[2]

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", false
	}

	if time.Since(time.Unix(ts, 0)) > sessionTTL {
		return "", false
	}

	payload := encoded + "." + tsStr

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))

	if !hmac.Equal([]byte(sig), []byte(hex.EncodeToString(mac.Sum(nil)))) {
		return "", false
	}

	b, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", false
	}

	return string(b), true
}

// setSessionCookie writes a signed session cookie for username.
func setSessionCookie(w http.ResponseWriter, secret []byte, username string) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    signSession(secret, username, time.Now().Unix()),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(sessionTTL),
	})
}

// clearSessionCookie immediately expires the session cookie.
func clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// sessionUser reads and verifies the session cookie, returning the username on success.
func sessionUser(r *http.Request, secret []byte) (string, bool) {
	c, err := r.Cookie(sessionCookieName)
	if err != nil {
		return "", false
	}

	return verifySession(secret, c.Value)
}

// withFormAuth returns middleware that enforces form auth and injects user identity into the context.
func withFormAuth(
	users []configutil.FormAuthUser,
	secret []byte,
	prefixes []string,
	database *db.DB,
) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			p := r.URL.Path

			if p == loginRoute || p == logoutRoute {
				next.ServeHTTP(w, r)
				return
			}

			for _, prefix := range prefixes {
				if strings.HasPrefix(p, prefix) {
					next.ServeHTTP(w, r)
					return
				}
			}

			username, ok := sessionUser(r, secret)
			if !ok {
				dest := loginRoute + "?" + nextQuery + "=" + url.QueryEscape(r.URL.RequestURI())
				http.Redirect(w, r, dest, http.StatusFound)

				return
			}

			displayName := resolveDisplayName(r.Context(), users, username, database)
			ctx := context.WithValue(r.Context(), ctxKeyUsername, username)
			ctx = context.WithValue(ctx, ctxKeyDisplayName, displayName)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// resolveFormAuthSecret decodes the hex secret from config, generating a random one if absent or invalid.
func resolveFormAuthSecret(hexSecret string) []byte {
	if hexSecret != "" {
		b, err := hex.DecodeString(hexSecret)
		if err == nil && len(b) >= 16 {
			return b
		}

		logutil.Errorf(logutil.Get(), "formAuth.secret is invalid hex; generating random secret\n")
	}

	return newSessionSecret()
}

// registerAuthRoutes mounts the login and logout handlers; DB is checked first, config list is the fallback.
func (c *Context) registerAuthRoutes(
	handle func(pattern string, h http.Handler),
	users []configutil.FormAuthUser,
	secret []byte,
	database *db.DB,
) {
	b := embedutil.MaybeRead(c.FS, loginTmplHTML)
	tmpl := template.Must(template.New(loginTmpl).Parse(string(b)))

	serveLogin := func(w http.ResponseWriter, errMsg, next string) {
		httputil.ContentType(w, httputil.ContentTypeHTML)

		if errMsg != "" {
			w.WriteHeader(http.StatusUnauthorized)
		}

		if err := tmpl.Execute(w, loginPageData{Error: errMsg, Next: next}); err != nil {
			logutil.Errorf(logutil.Get(), "login tmpl: %v\n", err)
		}
	}

	handle(loginRoute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if _, ok := sessionUser(r, secret); ok {
				http.Redirect(w, r, "/", http.StatusFound)
				return
			}

			next := r.URL.Query().Get(nextQuery)
			if next == "" {
				next = "/"
			}

			serveLogin(w, "", next)

		case http.MethodPost:
			if err := r.ParseForm(); err != nil {
				errutil.HTTPBadRequest(w)
				return
			}

			u := r.FormValue(usernameFormKey)
			p := r.FormValue(passwordFormKey)

			next := r.FormValue(nextQuery)
			if next == "" || !strings.HasPrefix(next, "/") {
				next = "/"
			}

			matched := ""

			if database != nil {
				dbUser, err := database.AuthUser(r.Context(), u, p)

				switch {
				case err == nil:
					matched = dbUser.Username
				case errors.Is(err, db.ErrUserNotFound), errors.Is(err, db.ErrInvalidCredentials):
				default:
					logutil.Errorf(logutil.Get(), "login: db.AuthUser: %v\n", err)
				}
			}

			if matched == "" {
				for _, cu := range users {
					if cu.Username == u && cu.Password == p {
						matched = cu.Username
						break
					}
				}
			}

			if matched == "" {
				serveLogin(w, "invalid username or password.", next)
				return
			}

			setSessionCookie(w, secret, matched)
			http.Redirect(w, r, next, http.StatusFound)

		default:
			http.NotFound(w, r)
		}
	}))

	handle(logoutRoute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clearSessionCookie(w)
		http.Redirect(w, r, loginRoute, http.StatusFound)
	}))
}
