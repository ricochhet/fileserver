package chat

import (
	"net/http"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
)

type meResponse struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	IsAdmin     bool   `json:"isAdmin"`
}

// meHandler returns the authenticated user's identity, including their admin flag.
func meHandler(resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, displayName, isAdmin := resolve(r)
		serverutil.WriteJSON(
			w,
			http.StatusOK,
			meResponse{
				Username:    username,
				DisplayName: displayName,
				IsAdmin:     isAdmin,
			},
		)
	}
}
