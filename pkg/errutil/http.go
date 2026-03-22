package errutil

import (
	"fmt"
	"net/http"
)

func HTTPNotFoundf(w http.ResponseWriter, format string, a ...any) {
	if len(a) != 0 {
		err(w, http.StatusNotFound, fmt.Sprintf(format, a...))
		return
	}

	err(w, http.StatusNotFound, format)
}

func HTTPNotFound(w http.ResponseWriter) {
	err(w, http.StatusNotFound, "")
}

func HTTPNotImplementedf(w http.ResponseWriter, format string, a ...any) {
	if len(a) != 0 {
		err(w, http.StatusNotImplemented, fmt.Sprintf(format, a...))
		return
	}

	err(w, http.StatusNotImplemented, format)
}

func HTTPNotImplemented(w http.ResponseWriter) {
	err(w, http.StatusNotImplemented, "")
}

func HTTPUnauthorizedf(w http.ResponseWriter, format string, a ...any) {
	if len(a) != 0 {
		err(w, http.StatusUnauthorized, fmt.Sprintf(format, a...))
		return
	}

	err(w, http.StatusUnauthorized, format)
}

func HTTPUnauthorized(w http.ResponseWriter) {
	err(w, http.StatusUnauthorized, "")
}

func HTTPBadRequestf(w http.ResponseWriter, format string, a ...any) {
	if len(a) != 0 {
		err(w, http.StatusBadRequest, fmt.Sprintf(format, a...))
		return
	}

	err(w, http.StatusBadRequest, format)
}

func HTTPBadRequest(w http.ResponseWriter) {
	err(w, http.StatusBadRequest, "")
}

func HTTPInternalServerErrorf(w http.ResponseWriter, format string, a ...any) {
	if len(a) != 0 {
		err(w, http.StatusInternalServerError, fmt.Sprintf(format, a...))
		return
	}

	err(w, http.StatusInternalServerError, format)
}

func HTTPInternalServerError(w http.ResponseWriter) {
	err(w, http.StatusInternalServerError, "")
}

func HTTPForbiddenf(w http.ResponseWriter, format string, a ...any) {
	if len(a) != 0 {
		err(w, http.StatusForbidden, fmt.Sprintf(format, a...))
		return
	}

	err(w, http.StatusForbidden, format)
}

func HTTPForbidden(w http.ResponseWriter) {
	err(w, http.StatusForbidden, "")
}

func err(w http.ResponseWriter, code int, msg string) {
	if msg == "" {
		msg = http.StatusText(code)
	}

	http.Error(w, msg, code)
}
