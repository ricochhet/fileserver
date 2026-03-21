package errutil

import (
	"fmt"
	"net/http"
)

func HTTPUnauthorizedf(w http.ResponseWriter, format string, a ...any) {
	httpError(w, http.StatusUnauthorized, fmt.Sprintf(format, a...))
}

func HTTPUnauthorized(w http.ResponseWriter) {
	httpError(w, http.StatusUnauthorized, "")
}

func HTTPBadRequestf(w http.ResponseWriter, format string, a ...any) {
	httpError(w, http.StatusBadRequest, fmt.Sprintf(format, a...))
}

func HTTPBadRequest(w http.ResponseWriter) {
	httpError(w, http.StatusBadRequest, "")
}

func HTTPInternalServerErrorf(w http.ResponseWriter, format string, a ...any) {
	httpError(w, http.StatusInternalServerError, fmt.Sprintf(format, a...))
}

func HTTPInternalServerError(w http.ResponseWriter) {
	httpError(w, http.StatusInternalServerError, "")
}

func HTTPForbiddenf(w http.ResponseWriter, format string, a ...any) {
	httpError(w, http.StatusForbidden, fmt.Sprintf(format, a...))
}

func HTTPForbidden(w http.ResponseWriter) {
	httpError(w, http.StatusForbidden, "")
}

func httpError(w http.ResponseWriter, code int, msg string) {
	if msg == "" {
		msg = http.StatusText(code)
	}

	http.Error(w, msg, code)
}
