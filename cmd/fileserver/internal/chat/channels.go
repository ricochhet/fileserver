package chat

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/serverutil"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

type joinBody struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

type leaveBody struct {
	Code string `json:"code"`
}

// availableChannelsHandler returns all channels in the store so the client can
// present a pick-list in the join modal. Channels the user is already subscribed
// to are included.
func availableChannelsHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _, _ := resolve(r)
		if username == "" {
			errutil.HTTPUnauthorized(w)
			return
		}

		all := store.AllChannels()
		if all == nil {
			all = []*Channel{}
		}

		serverutil.WriteJSON(w, http.StatusOK, all)
	}
}

// channelsHandler returns the channels the authenticated user is subscribed to.
func channelsHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _, _ := resolve(r)

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
		username, _, _ := resolve(r)
		if username == "" {
			errutil.HTTPUnauthorized(w)
			return
		}

		body := joinBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errutil.HTTPBadRequestf(w, "invalid JSON body")
			return
		}

		body.Code = strings.TrimSpace(body.Code)
		if body.Code == "" {
			errutil.HTTPBadRequestf(w, `"code" is required`)
			return
		}

		ch := store.JoinChannel(username, body.Code, strings.TrimSpace(body.Name))
		serverutil.WriteJSON(w, http.StatusOK, ch)
	}
}

// leaveHandler unsubscribes the authenticated user from a channel.
func leaveHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _, _ := resolve(r)
		if username == "" {
			errutil.HTTPUnauthorized(w)
			return
		}

		body := leaveBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errutil.HTTPBadRequestf(w, "invalid JSON body")
			return
		}

		body.Code = strings.TrimSpace(body.Code)
		if body.Code == "" {
			errutil.HTTPBadRequestf(w, `"code" is required`)
			return
		}

		store.LeaveChannel(username, body.Code)
		httputil.NoContent(w)
	}
}
