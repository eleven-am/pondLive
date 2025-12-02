package runtime

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"runtime/debug"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/protocol"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

type HandlerHandle struct {
	entry *handlerEntry
}

func (h HandlerHandle) URL() string {
	if h.entry == nil {
		return ""
	}
	return fmt.Sprintf("/_handlers/%s/%s", h.entry.sessionID, h.entry.id)
}

func (h HandlerHandle) GenerateToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf[:])
}

func (h HandlerHandle) Destroy() {
	if h.entry == nil || h.entry.sess == nil {
		return
	}
	h.entry.sess.removeHTTPHandler(h.entry.id)
	h.entry = nil
}

func UseHandler(ctx *Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	if ctx == nil || ctx.instance == nil {
		panic("runtime: UseHandler called outside component render")
	}

	idx := ctx.hookIndex
	ctx.hookIndex++

	isMount := idx >= len(ctx.instance.HookFrame)

	if isMount {
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

		if isMount {
			sess := ctx.session
			entry := cell.entry
			ctx.instance.RegisterCleanup(func() {
				sess.removeHTTPHandler(entry.id)
			})
		}
	}

	return HandlerHandle{entry: cell.entry}
}

type handlerCell struct {
	entry *handlerEntry
}

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

func (s *Session) removeHTTPHandler(id string) {
	if s == nil || id == "" {
		return
	}
	s.httpHandlerMu.Lock()
	delete(s.httpHandlers, id)
	s.httpHandlerMu.Unlock()
}

func (s *Session) findHTTPHandler(id string) *handlerEntry {
	if s == nil || id == "" {
		return nil
	}
	s.httpHandlerMu.RLock()
	entry := s.httpHandlers[id]
	s.httpHandlerMu.RUnlock()
	return entry
}

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
				if s.Bus != nil {
					s.Bus.ReportDiagnostic(protocol.Diagnostic{
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
