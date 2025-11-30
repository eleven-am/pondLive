package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/session"
)

type mockRegistry struct {
	sessions map[session.SessionID]*session.LiveSession
}

func (r *mockRegistry) Lookup(id session.SessionID) (*session.LiveSession, bool) {
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
			name:     "empty session id after double slash cleaned",
			path:     "/_handlers//handler",
			expected: "handler",
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

func TestExtractSessionID_Normalization(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "encoded space in session id",
			path:     "/_handlers/abc%20123/h0",
			expected: "abc 123",
		},
		{
			name:     "path traversal cleaned",
			path:     "/_handlers/../_handlers/abc123/h0",
			expected: "abc123",
		},
		{
			name:     "double slashes cleaned",
			path:     "/_handlers//abc123/h0",
			expected: "abc123",
		},
		{
			name:     "encoded slash decoded and split",
			path:     "/_handlers/abc%2F123/h0",
			expected: "abc",
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

func TestExtractHandlerID(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "valid path",
			path:     "/_handlers/abc123/root:h0",
			expected: "root:h0",
		},
		{
			name:     "missing handler id",
			path:     "/_handlers/abc123",
			expected: "",
		},
		{
			name:     "empty handler id",
			path:     "/_handlers/abc123/",
			expected: "",
		},
		{
			name:     "handler with extra path",
			path:     "/_handlers/abc123/h0/extra",
			expected: "h0",
		},
		{
			name:     "wrong prefix",
			path:     "/api/handlers/abc123/h0",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractHandlerID(tt.path)
			if result != tt.expected {
				t.Errorf("extractHandlerID(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestNormalizePath(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "clean path unchanged",
			path:     "/_handlers/abc/h0",
			expected: "/_handlers/abc/h0",
		},
		{
			name:     "double slashes cleaned",
			path:     "/_handlers//abc//h0",
			expected: "/_handlers/abc/h0",
		},
		{
			name:     "dot segments removed",
			path:     "/_handlers/./abc/../abc/h0",
			expected: "/_handlers/abc/h0",
		},
		{
			name:     "encoded chars decoded",
			path:     "/_handlers/abc%20def/h0",
			expected: "/_handlers/abc def/h0",
		},
		{
			name:     "trailing slash removed",
			path:     "/_handlers/abc/h0/",
			expected: "/_handlers/abc/h0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := normalizePath(tt.path)
			if result != tt.expected {
				t.Errorf("normalizePath(%q) = %q, want %q", tt.path, result, tt.expected)
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

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-store, private" {
		t.Errorf("expected Cache-Control on error, got %q", cacheControl)
	}
}

func TestDispatcherInvalidPath(t *testing.T) {
	reg := &mockRegistry{sessions: make(map[session.SessionID]*session.LiveSession)}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/invalid/path", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-store, private" {
		t.Errorf("expected Cache-Control on error, got %q", cacheControl)
	}
}

func TestDispatcherMissingHandlerID(t *testing.T) {
	reg := &mockRegistry{sessions: make(map[session.SessionID]*session.LiveSession)}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/_handlers/abc123", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-store, private" {
		t.Errorf("expected Cache-Control on error, got %q", cacheControl)
	}
}

func TestDispatcherSessionNotFound(t *testing.T) {
	reg := &mockRegistry{sessions: make(map[session.SessionID]*session.LiveSession)}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/_handlers/nonexistent/h0", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-store, private" {
		t.Errorf("expected Cache-Control on error, got %q", cacheControl)
	}
}

func TestDispatcherCacheControlHeader(t *testing.T) {
	sess := session.NewLiveSession("test-session", 1, nil, nil)
	reg := &mockRegistry{
		sessions: map[session.SessionID]*session.LiveSession{
			"test-session": sess,
		},
	}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/_handlers/test-session/h0", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-store, private" {
		t.Errorf("expected Cache-Control 'no-store, private', got %q", cacheControl)
	}
}

func TestDispatcherHappyPath(t *testing.T) {
	sess := session.NewLiveSession("test-session", 1, nil, nil)
	reg := &mockRegistry{
		sessions: map[session.SessionID]*session.LiveSession{
			"test-session": sess,
		},
	}
	d := NewDispatcher(reg)

	req := httptest.NewRequest(http.MethodGet, "/_handlers/test-session/root:h0", nil)
	w := httptest.NewRecorder()

	d.ServeHTTP(w, req)

	cacheControl := w.Header().Get("Cache-Control")
	if cacheControl != "no-store, private" {
		t.Errorf("expected Cache-Control header, got %q", cacheControl)
	}
}
