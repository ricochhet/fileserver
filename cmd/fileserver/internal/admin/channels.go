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

type createChannelBody struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// listChannelsHandler returns all channels known to the database.
func listChannelsHandler(database *chatdb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			errutil.HTTPNotImplementedf(w, "database not available")
			return
		}

		channels, err := database.ListChannels(r.Context())
		if err != nil {
			logutil.Errorf(logutil.Get(), "admin: ListChannels: %v\n", err)
			errutil.HTTPInternalServerErrorf(w, "could not list channels")

			return
		}

		out := make([]ChannelResponse, len(channels))
		for i, ch := range channels {
			out[i] = ChannelResponse{Code: ch.Code, Name: ch.Name}
		}

		serverutil.WriteJSON(w, http.StatusOK, out)
	}
}

// createChannelHandler creates or updates a channel.
func createChannelHandler(database *chatdb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			errutil.HTTPNotImplementedf(w, "database not available")
			return
		}

		body := createChannelBody{}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			errutil.HTTPBadRequestf(w, "invalid JSON body")
			return
		}

		body.Code = strings.TrimSpace(body.Code)
		body.Name = strings.TrimSpace(body.Name)

		if body.Code == "" {
			errutil.HTTPBadRequestf(w, `"code" is required`)
			return
		}

		if body.Name == "" {
			body.Name = "#" + body.Code
		}

		if err := database.UpsertChannel(r.Context(), body.Code, body.Name); err != nil {
			logutil.Errorf(logutil.Get(), "admin: UpsertChannel %q: %v\n", body.Code, err)
			errutil.HTTPInternalServerErrorf(w, "could not create channel")

			return
		}

		logutil.Infof(
			logutil.Get(),
			"admin: channel %q (%s) created/updated\n",
			body.Code,
			body.Name,
		)

		serverutil.WriteJSON(w, http.StatusCreated, ChannelResponse(body))
	}
}

// deleteChannelHandler removes a channel by code.
func deleteChannelHandler(database *chatdb.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if database == nil {
			errutil.HTTPNotImplementedf(w, "database not available")
			return
		}

		code := chi.URLParam(r, "code")
		if code == "" {
			errutil.HTTPBadRequestf(w, "code is required")
			return
		}

		if err := database.DeleteChannel(r.Context(), code); err != nil {
			if errors.Is(err, chatdb.ErrChannelNotFound) {
				errutil.HTTPNotFoundf(w, "channel not found")
			} else {
				logutil.Errorf(logutil.Get(), "admin: DeleteChannel %q: %v\n", code, err)
				errutil.HTTPInternalServerErrorf(w, "could not delete channel")
			}

			return
		}

		logutil.Infof(logutil.Get(), "admin: channel %q deleted\n", code)
		httputil.NoContent(w)
	}
}
