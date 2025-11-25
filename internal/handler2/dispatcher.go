package handler2

import (
	"net/http"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/session2"
)

// PathPrefix is the URL prefix for all handler endpoints.
const PathPrefix = "/_handlers/"

// SessionRegistry provides session lookup by ID.
type SessionRegistry interface {
	Lookup(id session2.SessionID) (*session2.LiveSession, bool)
}

// Dispatcher routes HTTP requests to registered component handlers.
// It extracts the session ID from the URL and delegates to the session's ServeHTTP.
type Dispatcher struct {
	registry SessionRegistry
}

// NewDispatcher creates a handler dispatcher bound to the session registry.
func NewDispatcher(reg SessionRegistry) *Dispatcher {
	return &Dispatcher{registry: reg}
}

// ServeHTTP handles requests to /_handlers/{sessionId}/{handlerId}.
// The session's ServeHTTP validates the handler ID and executes the chain.
func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d == nil || d.registry == nil {
		http.Error(w, "handler: dispatcher not available", http.StatusServiceUnavailable)
		return
	}

	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		http.Error(w, "handler: invalid path", http.StatusBadRequest)
		return
	}

	sess, ok := d.registry.Lookup(session2.SessionID(sessionID))
	if !ok || sess == nil {
		http.Error(w, "handler: session not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Cache-Control", "no-store, private")

	sess.ServeHTTP(w, r)
}

// extractSessionID extracts the session ID from /_handlers/{sessionID}/{handlerID}
func extractSessionID(path string) string {
	trimmed := strings.TrimPrefix(path, PathPrefix)
	if trimmed == path {
		return ""
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		return ""
	}
	return strings.TrimSpace(parts[0])
}
