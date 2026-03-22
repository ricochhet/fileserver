package admin

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	chatdb "github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/pkg/errutil"
)

const DefaultAdminRoute = "/admin"

// UserResolver extracts the authenticated username and admin flag from a request.
type UserResolver func(r *http.Request) (username string, isAdmin bool)

// UserResponse is the public shape of a user returned by the admin API.
// The password field is intentionally omitted.
type UserResponse struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	IsAdmin     bool   `json:"isAdmin"`
}

// ChannelResponse is the public shape of a channel returned by the admin API.
type ChannelResponse struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// Handler returns an http.Handler for all admin endpoints mounted under /admin.
func Handler(database *chatdb.DB, uploadDir string, resolve UserResolver) http.Handler {
	r := chi.NewRouter()
	r.Use(requireAdmin(resolve))

	r.Get("/users", listUsersHandler(database))
	r.Post("/users", createUserHandler(database, resolve))
	r.Delete("/users/{username}", deleteUserHandler(database, resolve))

	r.Get("/channels", listChannelsHandler(database))
	r.Post("/channels", createChannelHandler(database))
	r.Delete("/channels/{code}", deleteChannelHandler(database))

	r.Post("/upload", uploadHandler(uploadDir))

	return r
}

// requireAdmin is middleware that rejects requests from non-admin users with 403.
func requireAdmin(resolve UserResolver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, isAdmin := resolve(r)
			if !isAdmin {
				errutil.HTTPForbiddenf(w, "forbidden: admin required")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
