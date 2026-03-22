package admin

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	chatdb "github.com/ricochhet/fileserver/cmd/fileserver/internal/db"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

type createUserBody struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"displayName"`
	IsAdmin     bool   `json:"isAdmin"`
}

// listUsersHandler returns all users known to the database.
func listUsersHandler(database *chatdb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			errutil.HTTPNotImplementedf(w, "database not available")
			return
		}

		users, err := database.ListUsers(r.Context())
		if err != nil {
			logutil.Errorf(logutil.Get(), "admin: ListUsers: %v\n", err)
			errutil.HTTPInternalServerErrorf(w, "could not list users")

			return
		}

		out := make([]UserResponse, len(users))
		for i, u := range users {
			out[i] = UserResponse{
				Username:    u.Username,
				DisplayName: u.DisplayName,
				IsAdmin:     u.IsAdmin,
			}
		}

		serverutil.WriteJSON(w, http.StatusOK, out)
	}
}

// createUserHandler creates or updates a user.
func createUserHandler(database *chatdb.DB, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			errutil.HTTPNotImplementedf(w, "database not available")
			return
		}

		body := createUserBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errutil.HTTPBadRequestf(w, "invalid JSON body")
			return
		}

		body.Username = strings.TrimSpace(body.Username)
		body.Password = strings.TrimSpace(body.Password)
		body.DisplayName = strings.TrimSpace(body.DisplayName)

		if body.Username == "" {
			errutil.HTTPBadRequestf(w, `"username" is required`)
			return
		}

		if body.Password == "" {
			errutil.HTTPBadRequestf(w, `"password" is required`)
			return
		}

		if err := database.UpsertUser(
			r.Context(),
			body.Username,
			body.Password,
			body.DisplayName,
			body.IsAdmin,
		); err != nil {
			logutil.Errorf(logutil.Get(), "admin: UpsertUser %q: %v\n", body.Username, err)
			errutil.HTTPInternalServerErrorf(w, "could not create user")

			return
		}

		username, _ := resolve(r)
		logutil.Infof(
			logutil.Get(),
			"admin: user %q created/updated by %q (isAdmin=%v)\n",
			body.Username,
			username,
			body.IsAdmin,
		)

		serverutil.WriteJSON(w, http.StatusCreated, UserResponse{
			Username:    body.Username,
			DisplayName: body.DisplayName,
			IsAdmin:     body.IsAdmin,
		})
	}
}

// deleteUserHandler removes a user by username.
// Admins cannot delete their own account to prevent accidental lockout.
func deleteUserHandler(database *chatdb.DB, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			errutil.HTTPNotImplementedf(w, "database not available")
			return
		}

		target := chi.URLParam(r, "username")
		if target == "" {
			errutil.HTTPBadRequestf(w, "username is required")
			return
		}

		self, _ := resolve(r)
		if target == self {
			errutil.HTTPBadRequestf(w, "cannot delete your own account")
			return
		}

		if err := database.DeleteUser(r.Context(), target); err != nil {
			if errors.Is(err, chatdb.ErrUserNotFound) {
				errutil.HTTPNotFoundf(w, "user not found")
			} else {
				logutil.Errorf(logutil.Get(), "admin: DeleteUser %q: %v\n", target, err)
				errutil.HTTPInternalServerErrorf(w, "could not delete user")
			}

			return
		}

		logutil.Infof(logutil.Get(), "admin: user %q deleted by %q\n", target, self)
		httputil.NoContent(w)
	}
}
