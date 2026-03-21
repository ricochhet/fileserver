package serverutil

import (
	"bytes"
	"encoding/json"
	"mime"
	"net/http"
	"path/filepath"
	"time"

	"github.com/ricochhet/fileserver/cmd/fileserver/internal/configutil"
	"github.com/ricochhet/fileserver/pkg/httputil"
	"github.com/ricochhet/fileserver/pkg/logutil"
)

type headerWriter struct {
	http.ResponseWriter

	statusCode int
	allowed    map[string]struct{}
}

// WriteHeader strips disallowed headers then delegates to the underlying ResponseWriter.
func (h *headerWriter) WriteHeader(code int) {
	hdr := h.Header()
	for key := range hdr {
		if _, ok := h.allowed[http.CanonicalHeaderKey(key)]; !ok {
			hdr.Del(key)
		}
	}

	if h.statusCode != 0 {
		code = h.statusCode
	}

	h.ResponseWriter.WriteHeader(code)
}

// newHeaderWriter applies config headers, resolves content-type, and returns a headerWriter.
func newHeaderWriter(
	w http.ResponseWriter,
	name string,
	data []byte,
	info configutil.Info,
) *headerWriter {
	allowed := make(map[string]struct{})
	hasCT := false

	for key, value := range info.Headers {
		canon := http.CanonicalHeaderKey(key)
		httputil.SetHeader(w, httputil.HeaderKey(canon), value)
		allowed[canon] = struct{}{}

		if canon == string(httputil.HeaderContentType) {
			hasCT = true
		}
	}

	if !hasCT {
		ct := mime.TypeByExtension(filepath.Ext(name))
		if ct == "" && len(data) != 0 {
			ct = http.DetectContentType(data)
		}

		if ct != "" {
			httputil.SetHeader(w, httputil.HeaderContentType, ct)
		}

		allowed[string(httputil.HeaderContentType)] = struct{}{}
	}

	return &headerWriter{
		ResponseWriter: w,
		statusCode:     info.StatusCode,
		allowed:        allowed,
	}
}

// WithLogging is middleware that logs each request method and path.
func WithLogging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		logutil.Infof(logutil.Get(), "%s %s\n", r.Method, r.URL.Path)
		next.ServeHTTP(w, r)
	})
}

// ServeFileHandler returns a handler that serves a file from disk.
func ServeFileHandler(info configutil.Info, name string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeFile(newHeaderWriter(w, name, nil, info), r, name)
	})
}

// ServeContentHandler returns a handler that serves an in-memory byte slice.
func ServeContentHandler(info configutil.Info, name string, data []byte) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.ServeContent(
			newHeaderWriter(w, name, data, info),
			r,
			name,
			time.Now(),
			bytes.NewReader(data),
		)
	})
}

// WriteJSON encodes v as indented JSON and writes it to w with the given status code.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	httputil.ContentType(w, httputil.ContentTypeJSON)
	w.WriteHeader(status)
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")

	if err := enc.Encode(v); err != nil {
		logutil.Errorf(logutil.Get(), "WriteJSON encode: %v\n", err)
	}
}
