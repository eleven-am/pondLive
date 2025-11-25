package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"
)

// HandlerFunc handles an HTTP request and returns an error.
// If an error is returned and no response has been written, a 500 is sent.
type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

// HandlerHandle exposes the URL for a registered HTTP handler.
type HandlerHandle struct {
	entry *handlerEntry
}

// URL returns the HTTP endpoint path for this handler.
func (h HandlerHandle) URL() string {
	if h.entry == nil {
		return ""
	}
	return fmt.Sprintf("/_handlers/%s/%s", h.entry.sessionID, h.entry.id)
}

// GenerateToken returns a random capability token (hex).
// Caller can append it to the URL or use it as needed.
func (h HandlerHandle) GenerateToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf[:])
}

// Destroy explicitly removes the handler, if still registered.
// Safe to call multiple times.
func (h HandlerHandle) Destroy() {
	if h.entry == nil || h.entry.sess == nil {
		return
	}
	h.entry.sess.removeHTTPHandler(h.entry.id)
	h.entry = nil
}

// UseHandler registers or updates an HTTP handler for the current component.
// The handler uses stable ID/component hook index; rerenders update the callback.
func UseHandler(ctx *Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	if ctx == nil || ctx.instance == nil {
		panic("runtime: UseHandler called outside component render")
	}

	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypeHandler,
			Value: &handlerCell{},
		})
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*handlerCell)
	if !ok {
		panic("runtime: UseHandler hook mismatch")
	}

	if ctx.session != nil {
		if cell.entry == nil || cell.entry.sess == nil {
			cell.entry = ctx.session.registerHTTPHandler(ctx.instance, idx, method, chain)
		} else {
			ctx.session.updateHTTPHandler(cell.entry, method, chain)
		}

		ctx.instance.RegisterCleanup(func() {
			ctx.session.removeHTTPHandler(cell.entry.id)
		})
	}

	return HandlerHandle{entry: cell.entry}
}

type handlerCell struct {
	entry *handlerEntry
}

// handlerEntry tracks a single HTTP handler registration.
type handlerEntry struct {
	id        string
	sessionID string
	sess      *Session
	inst      *Instance
	method    string
	chain     []HandlerFunc
}

func (s *Session) registerHTTPHandler(inst *Instance, index int, method string, chain []HandlerFunc) *handlerEntry {
	if s == nil || inst == nil {
		return nil
	}

	s.httpHandlerMu.Lock()
	defer s.httpHandlerMu.Unlock()

	id := fmt.Sprintf("%s:h%d", inst.ID, index)

	entry := &handlerEntry{
		id:        id,
		sessionID: s.SessionID,
		sess:      s,
		inst:      inst,
		method:    method,
		chain:     append([]HandlerFunc(nil), chain...),
	}

	if s.httpHandlers == nil {
		s.httpHandlers = make(map[string]*handlerEntry)
	}
	s.httpHandlers[id] = entry
	return entry
}

func (s *Session) updateHTTPHandler(entry *handlerEntry, method string, chain []HandlerFunc) {
	if s == nil || entry == nil {
		return
	}
	s.httpHandlerMu.Lock()
	entry.method = method
	entry.chain = append([]HandlerFunc(nil), chain...)
	s.httpHandlerMu.Unlock()
}

// removeHTTPHandler removes a handler by ID.
func (s *Session) removeHTTPHandler(id string) {
	if s == nil || id == "" {
		return
	}
	s.httpHandlerMu.Lock()
	delete(s.httpHandlers, id)
	s.httpHandlerMu.Unlock()
}

// findHTTPHandler looks up a handler by ID.
func (s *Session) findHTTPHandler(id string) *handlerEntry {
	if s == nil || id == "" {
		return nil
	}
	s.httpHandlerMu.RLock()
	entry := s.httpHandlers[id]
	s.httpHandlerMu.RUnlock()
	return entry
}

// ServeHTTP dispatches to a registered handler by ID and session.
// Callers should wire this into their HTTP mux: e.g., mux.Handle("/_handlers/", session).
func (s *Session) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil {
		http.NotFound(w, r)
		return
	}

	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/"), "/")
	if len(parts) < 3 || parts[0] != "_handlers" {
		http.NotFound(w, r)
		return
	}
	reqSessionID, handlerID := parts[1], parts[2]
	if reqSessionID != s.SessionID {
		http.NotFound(w, r)
		return
	}

	entry := s.findHTTPHandler(handlerID)
	if entry == nil {
		http.NotFound(w, r)
		return
	}

	if entry.method != "" && !strings.EqualFold(r.Method, entry.method) {
		w.Header().Set("Allow", entry.method)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	exec := func() {
		wrote := false
		ww := &responseWriterTracker{ResponseWriter: w, wrote: &wrote}

		defer func() {
			if rec := recover(); rec != nil {
				if s.reporter != nil {
					s.reporter.ReportDiagnostic(Diagnostic{
						Phase:      "http_handler",
						Message:    fmt.Sprintf("panic: %v", rec),
						StackTrace: string(debug.Stack()),
						Metadata: map[string]any{
							"session_id": s.SessionID,
							"handler_id": entry.id,
							"method":     r.Method,
							"path":       r.URL.Path,
						},
					})
				}
				if !*ww.wrote {
					http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}
		}()

		for _, h := range entry.chain {
			if h == nil {
				continue
			}
			if err := h(ww, r); err != nil && !*ww.wrote {
				http.Error(ww, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				return
			}
		}
	}

	exec()
}

// responseWriterTracker tracks if a response was written.
type responseWriterTracker struct {
	http.ResponseWriter
	wrote *bool
}

func (w *responseWriterTracker) WriteHeader(statusCode int) {
	*w.wrote = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *responseWriterTracker) Write(b []byte) (int, error) {
	*w.wrote = true
	return w.ResponseWriter.Write(b)
}
