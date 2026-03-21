package serverutil

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/contextutil"
)

// HTTPServer holds the router and server configuration for a single listener.
type HTTPServer struct {
	Router   chi.Router
	TLS      *configutil.TLS
	Timeouts *configutil.Timeouts
}

// Context wraps a generic contextutil.Context typed to HTTPServer.
type Context struct {
	*contextutil.Context[HTTPServer]
}

// NewContext returns an initialized serverutil Context.
func NewContext() *Context {
	return &Context{&contextutil.Context[HTTPServer]{}}
}

// Handle registers a handler for the given pattern on the router.
func (h *Context) Handle(pattern string, handler http.Handler) {
	h.Mutex.Lock()
	defer h.Mutex.Unlock()

	h.Get().Router.Handle(pattern, handler)
}
