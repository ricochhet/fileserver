package auth

import (
	"encoding/hex"
	"net/http"
	"net/url"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

// WithFormAuth returns middleware that enforces form-based session authentication
// and injects the resolved user identity into the request context.
func WithFormAuth(
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

			username, ok := SessionUser(r, secret)
			if !ok {
				dest := loginRoute + "?" + nextQuery + "=" + url.QueryEscape(r.URL.RequestURI())
				http.Redirect(w, r, dest, http.StatusFound)

				return
			}

			displayName := ResolveDisplayName(r.Context(), users, username, database)
			isAdmin := ResolveIsAdmin(r.Context(), users, username, database)

			ctx := WithIdentity(r.Context(), username, displayName, isAdmin)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// ResolveFormAuthSecret decodes the hex secret from config, generating a
// random one if the field is absent or contains invalid hex.
func ResolveFormAuthSecret(secret string) []byte {
	if secret != "" {
		b, err := hex.DecodeString(secret)
		if err == nil && len(b) >= 16 {
			return b
		}

		logutil.Errorf(logutil.Get(), "formAuth.secret is invalid hex; generating random secret\n")
	}

	return NewSessionSecret()
}
