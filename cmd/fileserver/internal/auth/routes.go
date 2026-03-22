package auth

import (
	"errors"
	"html/template"
	"net/http"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

const (
	loginRoute  = "/auth/login"
	logoutRoute = "/auth/logout"

	nextQuery       = "next"
	usernameFormKey = "username"
	passwordFormKey = "password"

	loginTmpl     = "login"
	loginTmplHTML = "login.html"
)

type loginPageData struct {
	Error string
	Next  string
}

// RegisterAuthRoutes mounts the login and logout handlers on the provided
// handle function. The database is checked first; the config user list is the
// fallback so that users defined only in config can still log in.
func RegisterAuthRoutes(
	handle func(pattern string, h http.Handler),
	users []configutil.FormAuthUser,
	secret []byte,
	database *db.DB,
	fs *embedutil.EmbeddedFileSystem,
) {
	b := embedutil.MaybeRead(fs, loginTmplHTML)
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
			if _, ok := SessionUser(r, secret); ok {
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

			SetSessionCookie(w, secret, matched)
			http.Redirect(w, r, next, http.StatusFound)

		default:
			http.NotFound(w, r)
		}
	}))

	handle(logoutRoute, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ClearSessionCookie(w)
		http.Redirect(w, r, loginRoute, http.StatusFound)
	}))
}
