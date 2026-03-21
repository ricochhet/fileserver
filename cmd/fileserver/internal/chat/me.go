package chat

import (
	"net/http"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
)

type meResponse struct {
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
}

// meHandler returns the authenticated user's identity.
func meHandler(resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, displayName := resolve(r)
		serverutil.WriteJSON(
			w,
			http.StatusOK,
			meResponse{Username: username, DisplayName: displayName},
		)
	}
}
