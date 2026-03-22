package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	SessionCookieName = "fs_session"
	SessionTTL        = 24 * time.Hour
)

// NewSessionSecret generates a cryptographically random 32-byte secret.
func NewSessionSecret() []byte {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		panic("auth: cannot generate session secret: " + err.Error())
	}

	return b
}

// SignSession returns a tamper-evident token: base64url(username).unix_ts.hex(HMAC-SHA256).
func SignSession(secret []byte, username string, ts int64) string {
	encoded := base64.RawURLEncoding.EncodeToString([]byte(username))
	tsStr := strconv.FormatInt(ts, 10)
	payload := encoded + "." + tsStr

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payload))

	return payload + "." + hex.EncodeToString(mac.Sum(nil))
}

// VerifySession validates a session token and returns the embedded username on success.
func VerifySession(secret []byte, value string) (string, bool) {
	parts := strings.SplitN(value, ".", 3)
	if len(parts) != 3 {
		return "", false
	}

	encoded, tsStr, sig := parts[0], parts[1], parts[2]

	ts, err := strconv.ParseInt(tsStr, 10, 64)
	if err != nil {
		return "", false
	}

	if time.Since(time.Unix(ts, 0)) > SessionTTL {
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

// SetSessionCookie writes a signed session cookie for username.
func SetSessionCookie(w http.ResponseWriter, secret []byte, username string) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    SignSession(secret, username, time.Now().Unix()),
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Now().Add(SessionTTL),
	})
}

// ClearSessionCookie immediately expires the session cookie.
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     SessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1,
	})
}

// SessionUser reads and verifies the session cookie, returning the username on success.
func SessionUser(r *http.Request, secret []byte) (string, bool) {
	c, err := r.Cookie(SessionCookieName)
	if err != nil {
		return "", false
	}

	return VerifySession(secret, c.Value)
}
