package httputil

import (
	"fmt"
	"net/http"
)

type (
	ContentTypeValue string
	HeaderKey        string
)

const (
	ContentTypeZip         ContentTypeValue = "application/zip"
	ContentTypeJSON        ContentTypeValue = "application/json; charset=utf-8"
	ContentTypeHTML        ContentTypeValue = "text/html; charset=utf-8"
	ContentTypeText        ContentTypeValue = "text/plain; charset=utf-8"
	ContentTypeEventStream ContentTypeValue = "text/event-stream"
	ContentTypeBinary      ContentTypeValue = "application/octet-stream"
)

const (
	HeaderContentType         HeaderKey = "Content-Type"
	HeaderContentDisposition  HeaderKey = "Content-Disposition"
	HeaderWWWAuthenticate     HeaderKey = "WWW-Authenticate"
	HeaderCacheControl        HeaderKey = "Cache-Control"
	HeaderXContentTypeOptions HeaderKey = "X-Content-Type-Options"
	HeaderXFrameOptions       HeaderKey = "X-Frame-Options"
	HeaderConnection          HeaderKey = "Connection"
	HeaderXAccelBuffering     HeaderKey = "X-Accel-Buffering"
)

// ContentType sets the Content-Type response header.
func ContentType(w http.ResponseWriter, ct ContentTypeValue) {
	w.Header().Set(string(HeaderContentType), string(ct))
}

// SetHeader sets an arbitrary response header by typed key.
func SetHeader(w http.ResponseWriter, key HeaderKey, value string) {
	w.Header().Set(string(key), value)
}

// ContentDispositionAttachment sets Content-Disposition to attachment with the given filename.
func ContentDispositionAttachment(w http.ResponseWriter, filename string) {
	w.Header().
		Set(string(HeaderContentDisposition), fmt.Sprintf(`attachment; filename=%q`, filename))
}

// ContentDispositionInline sets Content-Disposition to inline with the given filename.
func ContentDispositionInline(w http.ResponseWriter, filename string) {
	w.Header().Set(string(HeaderContentDisposition), fmt.Sprintf(`inline; filename=%q`, filename))
}

// BasicAuthChallenge sets the WWW-Authenticate header for HTTP Basic auth with the given realm.
func BasicAuthChallenge(w http.ResponseWriter, realm string) {
	w.Header().Set(string(HeaderWWWAuthenticate), fmt.Sprintf(`Basic realm=%q`, realm))
}

// NoCache sets headers to prevent the response from being cached.
func NoCache(w http.ResponseWriter) {
	w.Header().Set(string(HeaderCacheControl), "no-store, no-cache, must-revalidate")
}

// NoSniff sets X-Content-Type-Options to nosniff to prevent MIME-type sniffing.
func NoSniff(w http.ResponseWriter) {
	w.Header().Set(string(HeaderXContentTypeOptions), "nosniff")
}

// DenyFrame sets X-Frame-Options to DENY to prevent clickjacking.
func DenyFrame(w http.ResponseWriter) {
	w.Header().Set(string(HeaderXFrameOptions), "DENY")
}

// Error writes a plain-text HTTP error response with the given status code and message.
func Error(w http.ResponseWriter, status int, msg string) {
	http.Error(w, msg, status)
}

// Errorf writes a formatted plain-text HTTP error response with the given status code.
func Errorf(w http.ResponseWriter, status int, format string, args ...any) {
	http.Error(w, fmt.Sprintf(format, args...), status)
}

// NoContent writes a 204 No Content response.
func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

// SSEHeaders sets the standard headers required for a Server-Sent Events stream.
// X-Accel-Buffering is set to "no" to disable proxy buffering for nginx and similar.
func SSEHeaders(w http.ResponseWriter) {
	ContentType(w, ContentTypeEventStream)
	SetHeader(w, HeaderCacheControl, "no-cache")
	SetHeader(w, HeaderConnection, "keep-alive")
	SetHeader(w, HeaderXAccelBuffering, "no")
}
