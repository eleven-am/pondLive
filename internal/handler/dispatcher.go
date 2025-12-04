package handler

import (
	"net/http"
	"net/url"
	"path"
	"strings"

	"github.com/eleven-am/pondlive/internal/session"
)

const PathPrefix = "/_handlers/"

type SessionRegistry interface {
	Lookup(id session.SessionID) (*session.LiveSession, bool)
}

type Dispatcher struct {
	registry SessionRegistry
}

func NewDispatcher(reg SessionRegistry) *Dispatcher {
	return &Dispatcher{registry: reg}
}

func (d *Dispatcher) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store, private")

	if d == nil || d.registry == nil {
		http.Error(w, "handler: dispatcher not available", http.StatusServiceUnavailable)
		return
	}

	sessionID := extractSessionID(r.URL.Path)
	if sessionID == "" {
		http.Error(w, "handler: invalid path", http.StatusBadRequest)
		return
	}

	handlerID := extractHandlerID(r.URL.Path)
	if handlerID == "" {
		http.Error(w, "handler: missing handler ID", http.StatusBadRequest)
		return
	}

	sess, ok := d.registry.Lookup(session.SessionID(sessionID))
	if !ok || sess == nil {
		http.Error(w, "handler: session not found", http.StatusNotFound)
		return
	}

	sess.ServeHTTP(w, r)
}

func normalizePath(rawPath string) string {
	decoded, err := url.PathUnescape(rawPath)
	if err != nil {
		return path.Clean(rawPath)
	}
	return path.Clean(decoded)
}

func extractSessionID(rawPath string) string {
	cleanPath := normalizePath(rawPath)
	trimmed := strings.TrimPrefix(cleanPath, PathPrefix)
	if trimmed == cleanPath {
		return ""
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 1 || parts[0] == "" {
		return ""
	}
	if strings.Contains(parts[0], "/") {
		return ""
	}
	return parts[0]
}

func extractHandlerID(rawPath string) string {
	cleanPath := normalizePath(rawPath)
	trimmed := strings.TrimPrefix(cleanPath, PathPrefix)
	if trimmed == cleanPath {
		return ""
	}
	parts := strings.SplitN(trimmed, "/", 2)
	if len(parts) < 2 || parts[1] == "" {
		return ""
	}
	handlerParts := strings.SplitN(parts[1], "/", 2)
	if handlerParts[0] == "" {
		return ""
	}
	return handlerParts[0]
}
