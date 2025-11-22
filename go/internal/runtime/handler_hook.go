package runtime

import (
	"fmt"
	"net/http"
)

// HandlerFunc is a function that handles HTTP requests.
// It receives the standard http.ResponseWriter and *http.Request, and returns an error.
// If an error is returned and no response has been written, a 500 status is sent.
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

// UseHandler registers an HTTP handler for the current component.
func UseHandler(ctx Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	if ctx.frame == nil {
		panic("runtime2: UseHandler called outside render")
	}

	idx := ctx.frame.idx
	ctx.frame.idx++

	if idx >= len(ctx.frame.cells) {
		cell := &handlerCell{}
		ctx.frame.cells = append(ctx.frame.cells, cell)
	}

	raw := ctx.frame.cells[idx]
	cell, ok := raw.(*handlerCell)
	if !ok {
		panicHookMismatch(ctx.comp, idx, "UseHandler", raw)
	}

	if ctx.sess != nil {
		if cell.entry == nil || cell.entry.sess == nil {
			cell.entry = ctx.sess.registerHandler(ctx.comp, idx, method, chain)
		}
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
	sess      *ComponentSession
	comp      *component
	method    string
	chain     []HandlerFunc
}

func (s *ComponentSession) registerHandler(comp *component, index int, method string, chain []HandlerFunc) *handlerEntry {
	if s == nil || comp == nil {
		return nil
	}

	s.httpHandlerMu.Lock()
	defer s.httpHandlerMu.Unlock()

	id := fmt.Sprintf("%s:h%d", comp.id, index)

	entry := &handlerEntry{
		id:        id,
		sessionID: s.sessionID,
		sess:      s,
		comp:      comp,
		method:    method,
		chain:     append([]HandlerFunc(nil), chain...),
	}

	s.httpHandlers[id] = entry
	return entry
}

// HandlerEntry represents a handler entry retrieved from the component session.
type HandlerEntry struct {
	ID     string
	Method string
	Chain  []HandlerFunc
}

// FindHandler looks up a handler entry by ID.
func (s *ComponentSession) FindHandler(id string) *HandlerEntry {
	if s == nil || id == "" {
		return nil
	}

	s.httpHandlerMu.Lock()
	entry := s.httpHandlers[id]
	s.httpHandlerMu.Unlock()

	if entry == nil {
		return nil
	}

	return &HandlerEntry{
		ID:     entry.id,
		Method: entry.method,
		Chain:  entry.chain,
	}
}

func (s *ComponentSession) removeHandler(id string) {
	if s == nil || id == "" {
		return
	}

	s.httpHandlerMu.Lock()
	delete(s.httpHandlers, id)
	s.httpHandlerMu.Unlock()
}
