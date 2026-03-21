package chat

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

// channelsHandler returns the channels the authenticated user is subscribed to.
func channelsHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _ := resolve(r)

		subs := store.Subscriptions(username)
		if subs == nil {
			subs = []*Channel{}
		}

		serverutil.WriteJSON(w, http.StatusOK, subs)
	}
}

// joinHandler subscribes the authenticated user to a channel.
func joinHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _ := resolve(r)
		if username == "" {
			httputil.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var body struct {
			Code string `json:"code"`
			Name string `json:"name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httputil.Error(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		body.Code = strings.TrimSpace(body.Code)
		if body.Code == "" {
			httputil.Error(w, http.StatusBadRequest, `"code" is required`)
			return
		}

		ch := store.JoinChannel(username, body.Code, strings.TrimSpace(body.Name))
		serverutil.WriteJSON(w, http.StatusOK, ch)
	}
}

// leaveHandler unsubscribes the authenticated user from a channel.
func leaveHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _ := resolve(r)
		if username == "" {
			httputil.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var body struct {
			Code string `json:"code"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httputil.Error(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		body.Code = strings.TrimSpace(body.Code)
		if body.Code == "" {
			httputil.Error(w, http.StatusBadRequest, `"code" is required`)
			return
		}

		store.LeaveChannel(username, body.Code)
		httputil.NoContent(w)
	}
}
