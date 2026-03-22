package chat

import (
	"bytes"
	"encoding/json"
	"fmt"
	"mime"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/ricochhet/fileserver/pkg/errutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

// UserResolver extracts the authenticated username, display name, and admin flag from a request.
type UserResolver func(r *http.Request) (username, displayName string, isAdmin bool)

// Handler returns an http.Handler mounting the chat page and REST/SSE endpoints at /chat.
func Handler(store *Store, resolve UserResolver, html []byte) http.Handler {
	r := chi.NewRouter()

	if len(html) > 0 {
		r.Get("/", serveBytes("chat.html", "text/html; charset=utf-8", html))
	}

	r.Get("/api/me", meHandler(resolve))
	r.Get("/api/channels", channelsHandler(store, resolve))
	r.Get("/api/channels/available", availableChannelsHandler(store, resolve))
	r.Post("/api/channels/join", joinHandler(store, resolve))
	r.Post("/api/channels/leave", leaveHandler(store, resolve))
	r.Get("/api/messages", messagesHandler(store, resolve))
	r.Post("/api/messages", postMessageHandler(store, resolve))
	r.Get("/api/events", eventsHandler(store, resolve))

	return r
}

// serveBytes returns a HandlerFunc that serves a static in-memory byte slice.
func serveBytes(name, contentType string, data []byte) http.HandlerFunc {
	if contentType == "" {
		contentType = mime.TypeByExtension(name)
		if contentType == "" {
			contentType = http.DetectContentType(data)
		}
	}

	mod := time.Now()

	return func(w http.ResponseWriter, r *http.Request) {
		httputil.SetHeader(w, httputil.HeaderContentType, contentType)
		http.ServeContent(w, r, name, mod, bytes.NewReader(data))
	}
}

// eventsHandler streams new messages to the client over SSE with 30s keepalives.
func eventsHandler(store *Store, resolve UserResolver) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		username, _, _ := resolve(r)
		if username == "" {
			errutil.HTTPUnauthorized(w)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			errutil.HTTPInternalServerErrorf(w, "streaming not supported")
			return
		}

		httputil.SSEHeaders(w)

		msgs, unsub := store.Subscribe(username)
		defer unsub()

		fmt.Fprintf(w, ": connected\n\n")
		flusher.Flush()

		keepalive := time.NewTicker(30 * time.Second)
		defer keepalive.Stop()

		for {
			select {
			case <-r.Context().Done():
				return
			case <-keepalive.C:
				fmt.Fprintf(w, ": keepalive\n\n")
				flusher.Flush()
			case msg, ok := <-msgs:
				if !ok {
					return
				}

				if !store.IsSubscribed(username, msg.ChannelCode) {
					continue
				}

				data, err := json.Marshal(msg)
				if err != nil {
					continue
				}

				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
			}
		}
	}
}
