package handler2

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/session2"
)

type mockRegistry struct {
	sessions map[session2.SessionID]*session2.LiveSession
}

func (r *mockRegistry) Lookup(id session2.SessionID) (*session2.LiveSession, bool) {
	if r.sessions == nil {
		return nil, false
	}
	sess, ok := r.sessions[id]
	return sess, ok
}

func TestExtractSessionID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "valid path",
			path:     "/_handlers/abc123/root:h0",
			expected: "abc123",
		},
		{
			name:     "valid path with trailing content",
			path:     "/_handlers/session-id-here/handler-id/extra",
			expected: "session-id-here",
		},
		{
			name:     "only session id",
			path:     "/_handlers/abc123",
			expected: "abc123",
		},
		{
			name:     "empty session id",
			path:     "/_handlers//handler",
			expected: "",
		},
		{
			name:     "wrong prefix",
			path:     "/api/handlers/abc123/h0",
			expected: "",
		},
		{
			name:     "no prefix",
			path:     "/abc123/h0",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractSessionID(tt.path)
			if result != tt.expected {
				t.Errorf("extractSessionID(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestDispatcherNilRegistry(t *testing.T) {
	d := NewDispatcher(nil)

	req := httptest.NewRequest(http.MethodGet, "/_handlers/abc123/h0", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestDispatcherInvalidPath(t *testing.T) {
	reg := &mockRegistry{sessions: make(map[session2.SessionID]*session2.LiveSession)}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/invalid/path", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}
}

func TestDispatcherSessionNotFound(t *testing.T) {
	reg := &mockRegistry{sessions: make(map[session2.SessionID]*session2.LiveSession)}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/_handlers/nonexistent/h0", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}
}

func TestDispatcherCacheControlHeader(t *testing.T) {

	sess := session2.NewLiveSession("test-session", 1, nil, nil)
	reg := &mockRegistry{
		sessions: map[session2.SessionID]*session2.LiveSession{
			"test-session": sess,
		},
	}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/_handlers/test-session/nonexistent", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-store, private" {
		t.Errorf("expected Cache-Control 'no-store, private', got %q", cacheControl)
	}
}
