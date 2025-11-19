package http

import (
	"bytes"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/server"
)

func TestUploadHandlerUnavailable(t *testing.T) {
	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "http://example.com"+UploadPathPrefix+"sid/upload1", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	rec := httptest.NewRecorder()
	var nilHandler *UploadHandler
	nilHandler.ServeHTTP(rec, req)
	if res := rec.Result(); res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for nil handler, got %d", res.StatusCode)
	}

	handler := &UploadHandler{}
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if res := rec.Result(); res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 for handler without registry, got %d", res.StatusCode)
	}
}

func TestUploadHandlerRejectsNonPost(t *testing.T) {
	handler := NewUploadHandler(server.NewSessionRegistry())
	req := httptest.NewRequest(http.MethodGet, "http://example.com"+UploadPathPrefix+"sid/upload1", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	res := rec.Result()
	if res.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", res.StatusCode)
	}
	if allow := res.Header.Get("Allow"); allow != http.MethodPost {
		t.Fatalf("expected Allow header to advertise POST, got %q", allow)
	}
}

func TestUploadHandlerInvalidTarget(t *testing.T) {
	handler := NewUploadHandler(server.NewSessionRegistry())

	cases := []string{
		"/pondlive/upload/",
		"/pondlive/upload/sid",
		"/pondlive/upload/sid/",
		"/other/path",
	}

	for _, path := range cases {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "http://example.com"+path, nil)
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)
			if res := rec.Result(); res.StatusCode != http.StatusBadRequest {
				t.Fatalf("expected 400 for invalid path %q, got %d", path, res.StatusCode)
			}
		})
	}
}

func TestUploadHandlerSessionNotFound(t *testing.T) {
	registry := server.NewSessionRegistry()
	handler := NewUploadHandler(registry)

	var buf bytes.Buffer
	w := multipart.NewWriter(&buf)
	_ = w.Close()

	req := httptest.NewRequest(http.MethodPost, "http://example.com"+UploadPathPrefix+"missing/upload1", &buf)
	req.Header.Set("Content-Type", w.FormDataContentType())

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if res := rec.Result(); res.StatusCode != http.StatusNotFound {
		t.Fatalf("expected 404 for missing session, got %d", res.StatusCode)
	}
}

func TestUploadHandlerSuccess(t *testing.T) {

	t.Skip("Upload tests require full session rendering lifecycle")
}

func TestUploadHandlerSizeLimit(t *testing.T) {
	t.Skip("Upload tests require full session rendering lifecycle")
}

func TestUploadHandlerMissingFile(t *testing.T) {
	t.Skip("Upload tests require full session rendering lifecycle")
}

func TestUploadHandlerInvalidMultipart(t *testing.T) {
	t.Skip("Upload tests require full session rendering lifecycle")
}
