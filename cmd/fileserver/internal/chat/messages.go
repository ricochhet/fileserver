package chat

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
)

type messageBody struct {
	Channel string `json:"channel"`
	Body    string `json:"body"`
}

// messagesHandler returns message history for a channel.
func messagesHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _, _ := resolve(r)

		code := strings.TrimSpace(r.URL.Query().Get("channel"))
		if code == "" {
			errutil.HTTPBadRequestf(w, `query param "channel" is required`)
			return
		}

		msgs, err := store.Messages(r.Context(), username, code)
		if err != nil {
			errutil.HTTPForbiddenf(w, "%s", err.Error())
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
		username, displayName, _ := resolve(r)
		if username == "" {
			errutil.HTTPUnauthorized(w)
			return
		}

		body := messageBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errutil.HTTPBadRequestf(w, "invalid JSON body")
			return
		}

		body.Channel = strings.TrimSpace(body.Channel)
		body.Body = strings.TrimSpace(body.Body)

		if body.Channel == "" {
			errutil.HTTPBadRequestf(w, `"channel" is required`)
			return
		}

		if body.Body == "" {
			errutil.HTTPBadRequestf(w, `"body" must not be empty`)
			return
		}

		msg, err := store.Post(r.Context(), username, displayName, body.Channel, body.Body)
		if err != nil {
			errutil.HTTPForbiddenf(w, "%s", err.Error())
			return
		}

		serverutil.WriteJSON(w, http.StatusCreated, msg)
	}
}
