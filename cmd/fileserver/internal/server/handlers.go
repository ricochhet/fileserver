package server

import (
	"bytes"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/ricochhet/fileserver/pkg/embedutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
)

// NotFoundHandler is a middleware for 404 not found.
func (c *Context) NotFoundHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	httputil.ContentType(w, httputil.ContentTypeHTML)
	_, _ = w.Write(embedutil.MaybeRead(c.FS, "404.html"))
}

// SPANotFound returns a SPA-style fallback HandlerFunc.
func SPANotFound(name string, data []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(filepath.Base(r.URL.Path), ".") {
			http.NotFound(w, r)
			return
		}

		http.ServeContent(
			w,
			r,
			name,
			time.Now(),
			bytes.NewReader(data),
		)
	}
}
