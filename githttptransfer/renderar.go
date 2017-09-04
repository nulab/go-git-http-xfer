package githttptransfer

import (
	"net/http"
)

func RenderMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	if r.Proto == "HTTP/1.1" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Method Not Allowed"))
	} else {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte("Bad Request"))
	}
	w.Header().Set("Content-Type", "text/plain")
}

func RenderNotFound(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNotFound)
	w.Write([]byte("Not Found"))
	w.Header().Set("Content-Type", "text/plain")
}

func RenderNoAccess(w http.ResponseWriter) {
	w.WriteHeader(http.StatusForbidden)
	w.Write([]byte("Forbidden"))
	w.Header().Set("Content-Type", "text/plain")
}

func RenderInternalServerError(w http.ResponseWriter) {
	w.WriteHeader(http.StatusInternalServerError)
	w.Write([]byte("Internal Server Error"))
	w.Header().Set("Content-Type", "text/plain")
}
