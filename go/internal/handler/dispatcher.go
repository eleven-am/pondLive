package handler

import (
	"net/http"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
	"github.com/eleven-am/pondlive/go/internal/session"
)

const PathPrefix = "/_handlers/"

// Dispatcher routes HTTP requests to registered component handlers.
type Dispatcher struct {
	registry *server.SessionRegistry
}

// NewDispatcher creates a handler dispatcher bound to the session registry.
func NewDispatcher(reg *server.SessionRegistry) *Dispatcher {
	return &Dispatcher{registry: reg}
}

// ServeHTTP handles requests to /_handlers/{sessionId}/{handlerId}.
func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if d == nil || d.registry == nil {
		http.Error(w, "handler: dispatcher not available", http.StatusServiceUnavailable)
		return
	}

	sessionID, handlerID := extractHandlerTarget(r.URL.Path)
	if sessionID == "" || handlerID == "" {
		http.Error(w, "handler: invalid target", http.StatusBadRequest)
		return
	}

	sess, ok := d.registry.Lookup(session.SessionID(sessionID))
	if !ok || sess == nil {
		http.Error(w, "handler: session not found", http.StatusNotFound)
		return
	}

	component := sess.ComponentSession()
	if component == nil {
		http.Error(w, "handler: session unavailable", http.StatusGone)
		return
	}

	entry := component.FindHandler(handlerID)
	if entry == nil {
		http.Error(w, "handler: handler not found", http.StatusNotFound)
		return
	}

	if !strings.EqualFold(r.Method, entry.Method) {
		w.Header().Set("Allow", entry.Method)
		http.Error(w, "handler: method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Cache-Control", "no-store, private")
	if err := executeChain(w, r, entry.Chain); err != nil {
		if !responseWritten(w) {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}

func extractHandlerTarget(path string) (string, string) {
	trimmed := strings.TrimPrefix(path, PathPrefix)
	if trimmed == path {
		return "", ""
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 {
		return "", ""
	}
	sessionID := strings.TrimSpace(parts[0])
	handlerID := strings.TrimSpace(parts[1])
	return sessionID, handlerID
}

func executeChain(w http.ResponseWriter, r *http.Request, chain []runtime.HandlerFunc) error {
	for _, fn := range chain {
		if fn == nil {
			continue
		}
		if err := fn(w, r); err != nil {
			return err
		}
	}
	return nil
}

// responseWritten attempts to detect if the ResponseWriter has written a response.
// This is a best-effort check using type assertion to common response writer types.
func responseWritten(w http.ResponseWriter) bool {
	type statusWriter interface {
		Status() int
	}
	if sw, ok := w.(statusWriter); ok {
		return sw.Status() != 0
	}

	return false
}
