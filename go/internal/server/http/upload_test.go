package http

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
)

func TestUploadHandlerUnavailable(t *testing.T) {
	req := httptest.NewRequest(http.MethodPost, "http://example.com"+UploadPathPrefix, nil)

	var nilHandler *UploadHandler
	rec := httptest.NewRecorder()
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

func TestUploadHandlerRejectsInvalidRequests(t *testing.T) {
	handler := NewUploadHandler(server.NewSessionRegistry())

	t.Run("method", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com"+UploadPathPrefix+"sid/upload", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		res := rec.Result()
		if res.StatusCode != http.StatusMethodNotAllowed {
			t.Fatalf("expected 405, got %d", res.StatusCode)
		}
		if allow := res.Header.Get("Allow"); allow != http.MethodPost {
			t.Fatalf("expected Allow header to advertise POST, got %q", allow)
		}
	})

	t.Run("target", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com"+UploadPathPrefix, nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if res := rec.Result(); res.StatusCode != http.StatusBadRequest {
			t.Fatalf("expected 400 for malformed upload path, got %d", res.StatusCode)
		}
	})

	t.Run("session", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "http://example.com"+UploadPathPrefix+"missing/upload", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		if res := rec.Result(); res.StatusCode != http.StatusNotFound {
			t.Fatalf("expected 404 for unknown session, got %d", res.StatusCode)
		}
	})
}

func TestExtractUploadTarget(t *testing.T) {
	cases := []struct {
		path         string
		wantSID      string
		wantUploadID string
	}{
		{path: "/something-else", wantSID: "", wantUploadID: ""},
		{path: UploadPathPrefix + "only", wantSID: "", wantUploadID: ""},
		{path: UploadPathPrefix + "sid/upload", wantSID: "sid", wantUploadID: "upload"},
		{path: UploadPathPrefix + " sid / upload ", wantSID: "sid", wantUploadID: "upload"},
		{path: UploadPathPrefix + "sid/upload/extra", wantSID: "sid", wantUploadID: "upload"},
	}

	for _, tc := range cases {
		t.Run(tc.path, func(t *testing.T) {
			sid, uploadID := extractUploadTarget(tc.path)
			if sid != tc.wantSID || uploadID != tc.wantUploadID {
				t.Fatalf("extractUploadTarget(%q) = (%q, %q), want (%q, %q)", tc.path, sid, uploadID, tc.wantSID, tc.wantUploadID)
			}
		})
	}
}

func TestCleanupUploadedFile(t *testing.T) {
	dir := t.TempDir()
	tempFile, err := os.CreateTemp(dir, "upload-*")
	if err != nil {
		t.Fatalf("create temp file: %v", err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		t.Fatalf("close temp file: %v", err)
	}

	reader := &trackingReadSeekCloser{}
	cleanupUploadedFile(runtime.UploadedFile{TempPath: tempPath, Reader: reader})

	if !reader.closed {
		t.Fatal("expected reader to be closed")
	}
	if _, err := os.Stat(tempPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected temp file to be removed, got %v", err)
	}

	cleanupUploadedFile(runtime.UploadedFile{})
}

type trackingReadSeekCloser struct {
	closed bool
}

func (t *trackingReadSeekCloser) Read(p []byte) (int, error)                   { return 0, io.EOF }
func (t *trackingReadSeekCloser) Seek(offset int64, whence int) (int64, error) { return 0, nil }
func (t *trackingReadSeekCloser) Close() error {
	t.closed = true
	return nil
}
