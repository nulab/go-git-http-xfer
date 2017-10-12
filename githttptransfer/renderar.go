package githttptransfer

import (
	"net/http"
)

func RenderMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Proto == "HTTP/1.1" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method Not Allowed")) // could use http.StatusText(http.StatusMethodNotAllowed)
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request")) // could use http.StatusText(http.StatusBadRequest)
	}
	w.Header().Set("Content-Type", "text/plain")
}

func RenderNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not Found")) // could use http.StatusText(http.StatusNotFound)
	w.Header().Set("Content-Type", "text/plain")
}

func RenderNoAccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Forbidden")) // could use http.StatusText(http.StatusForbidden)
	w.Header().Set("Content-Type", "text/plain")
}

func RenderInternalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error")) // could use http.StatusText(http.StatusInternalServerError)
	w.Header().Set("Content-Type", "text/plain")
}
