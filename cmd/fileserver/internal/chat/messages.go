package chat

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

// messagesHandler returns message history for a channel.
func messagesHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _ := resolve(r)

		code := strings.TrimSpace(r.URL.Query().Get("channel"))
		if code == "" {
			httputil.Error(w, http.StatusBadRequest, `query param "channel" is required`)
			return
		}

		msgs, err := store.Messages(r.Context(), username, code)
		if err != nil {
			httputil.Error(w, http.StatusForbidden, err.Error())
			return
		}

		if msgs == nil {
			msgs = []*Message{}
		}

		serverutil.WriteJSON(w, http.StatusOK, msgs)
	}
}

// postMessageHandler posts a message to a channel on behalf of the authenticated user.
func postMessageHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, displayName := resolve(r)
		if username == "" {
			httputil.Error(w, http.StatusUnauthorized, "unauthorized")
			return
		}

		var body struct {
			Channel string `json:"channel"`
			Body    string `json:"body"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			httputil.Error(w, http.StatusBadRequest, "invalid JSON body")
			return
		}

		body.Channel = strings.TrimSpace(body.Channel)
		body.Body = strings.TrimSpace(body.Body)

		if body.Channel == "" {
			httputil.Error(w, http.StatusBadRequest, `"channel" is required`)
			return
		}

		if body.Body == "" {
			httputil.Error(w, http.StatusBadRequest, `"body" must not be empty`)
			return
		}

		msg, err := store.Post(r.Context(), username, displayName, body.Channel, body.Body)
		if err != nil {
			httputil.Error(w, http.StatusForbidden, err.Error())
			return
		}

		serverutil.WriteJSON(w, http.StatusCreated, msg)
	}
}
