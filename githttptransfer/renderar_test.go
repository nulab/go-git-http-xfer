package githttptransfer

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func Test_MethodNotAllowed_should_render_MethodNotAllowed(t *testing.T) {

	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "http://localhost/base/foo", nil)
	RenderMethodNotAllowed(w, r)

	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("StatusCode is not %d . result: %d", http.StatusMethodNotAllowed, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type is not 'text/plain' . result: %s", contentType)
	}
}

func Test_NotFound_should_render_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	RenderNotFound(w)

	if w.Code != http.StatusNotFound {
		t.Errorf("StatusCode is not %d . result: %d", http.StatusNotFound, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type is not 'text/plain' . result: %s", contentType)
	}
}

func Test_NoAccess_should_render_Forbidden(t *testing.T) {
	w := httptest.NewRecorder()
	RenderNoAccess(w)

	if w.Code != http.StatusForbidden {
		t.Errorf("StatusCode is not %d . result: %d", http.StatusForbidden, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type is not 'text/plain' . result: %s", contentType)
	}
}

func Test_InternalServerError_should_render_InternalServerError(t *testing.T) {
	w := httptest.NewRecorder()
	RenderInternalServerError(w)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("StatusCode is not %d . result: %d", http.StatusInternalServerError, w.Code)
	}

	contentType := w.Header().Get("Content-Type")
	if w.Header().Get("Content-Type") != "text/plain" {
		t.Errorf("Content-Type is not 'text/plain' . result: %s", contentType)
	}
}
