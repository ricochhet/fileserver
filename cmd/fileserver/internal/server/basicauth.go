package server

import (
	"net/http"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

// withBasicAuth returns middleware enforcing HTTP Basic Authentication.
func withBasicAuth(user, password string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			u, p, ok := r.BasicAuth()
			if !ok || u != user || p != password {
				httputil.BasicAuthChallenge(w, "fileserver")
				errutil.HTTPUnauthorized(w)

				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// wrapBasicAuth wraps a handler with Basic Auth when credentials are non-empty.
func wrapBasicAuth(auth configutil.BasicAuth, h http.Handler) http.Handler {
	if auth.Username == "" || auth.Password == "" {
		return h
	}

	return withBasicAuth(auth.Username, auth.Password)(h)
}
