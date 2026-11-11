package runtime

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/internal/reactive"
)

type HandlerFunc func(w http.ResponseWriter, r *http.Request) error

const (
	HandlerBodyLimit     = 1 << 20
	HandlerResponseLimit = 8 << 20
	handlerPhase         = "http_handler"
)

const DefaultHandlerQueueCap = 32

var handlerQueueTimeout = 10 * time.Second

var ErrHandlerResponseTooLarge = errors.New("runtime: handler response exceeds the buffer limit")

type handlerCell struct {
	id        string
	sessionID string
	sess      *Session
	destroyed atomic.Bool
}

type handlerEntryHTTP struct {
	cell   *handlerCell
	o      *reactive.Owner
	method string
	chain  []HandlerFunc
}

type HandlerHandle struct {
	cell *handlerCell
}

func (h HandlerHandle) URL() string {
	if h.cell == nil || h.cell.destroyed.Load() {
		return ""
	}
	return fmt.Sprintf("/_handlers/%s/%s", h.cell.sessionID, h.cell.id)
}

func (h HandlerHandle) GenerateToken() string {
	var buf [16]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return ""
	}
	return hex.EncodeToString(buf[:])
}

func (h HandlerHandle) Destroy() {
	if h.cell == nil {
		return
	}
	h.cell.destroy()
}

func (c *handlerCell) destroy() {
	if !c.destroyed.CompareAndSwap(false, true) {
		return
	}
	if c.sess == nil {
		return
	}
	cell := c
	c.sess.RunOnActor(func() { cell.sess.removeHTTPHandler(cell) })
}

func UseHandler(ctx *Ctx, method string, chain ...HandlerFunc) HandlerHandle {
	cell := reactive.Cell(ctx.o, func() *handlerCell {
		s := ctx.s
		s.httpSeq++
		c := &handlerCell{sess: s, sessionID: s.SessionID, id: fmt.Sprintf("h%d", s.httpSeq)}
		if s.httpHandlers == nil {
			s.httpHandlers = make(map[string]*handlerEntryHTTP)
		}
		s.httpHandlers[c.id] = &handlerEntryHTTP{
			cell:   c,
			o:      ctx.o,
			method: method,
			chain:  append([]HandlerFunc(nil), chain...),
		}
		ctx.o.OnCleanup(func() {
			c.destroyed.Store(true)
			s.removeHTTPHandler(c)
		})
		return c
	})
	return HandlerHandle{cell: cell}
}

func (s *Session) removeHTTPHandler(cell *handlerCell) {
	if s == nil || cell == nil {
		return
	}
	if entry := s.httpHandlers[cell.id]; entry != nil && entry.cell == cell {
		delete(s.httpHandlers, cell.id)
	}
}

func (s *Session) findHTTPHandler(id string) *handlerEntryHTTP {
	if s == nil || id == "" {
		return nil
	}
	entry := s.httpHandlers[id]
	if entry == nil || entry.cell.destroyed.Load() {
		return nil
	}
	return entry
}

type bufferedResponse struct {
	header   http.Header
	status   int
	body     bytes.Buffer
	limit    int
	exceeded bool
}

func newBufferedResponse(limit int) *bufferedResponse {
	return &bufferedResponse{header: make(http.Header), limit: limit}
}

func (b *bufferedResponse) Header() http.Header {
	return b.header
}

func (b *bufferedResponse) WriteHeader(status int) {
	if b.status == 0 {
		b.status = status
	}
}

func (b *bufferedResponse) Write(p []byte) (int, error) {
	if b.status == 0 {
		b.status = http.StatusOK
	}
	if b.exceeded || b.body.Len()+len(p) > b.limit {
		b.exceeded = true
		return 0, ErrHandlerResponseTooLarge
	}
	return b.body.Write(p)
}

func (b *bufferedResponse) wrote() bool {
	return b.status != 0
}

func (b *bufferedResponse) reset(status int) {
	b.header = make(http.Header)
	b.body.Reset()
	b.exceeded = false
	b.status = 0
	b.header.Set("Content-Type", "text/plain; charset=utf-8")
	b.header.Set("X-Content-Type-Options", "nosniff")
	b.WriteHeader(status)
	_, _ = b.body.WriteString(http.StatusText(status) + "\n")
}

func (b *bufferedResponse) copyTo(w http.ResponseWriter) {
	dst := w.Header()
	for k, v := range b.header {
		dst[k] = append([]string(nil), v...)
	}
	status := b.status
	if status == 0 {
		status = http.StatusOK
	}
	w.WriteHeader(status)
	_, _ = w.Write(b.body.Bytes())
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

	limit := s.handlerQueueCap.Load()
	if limit <= 0 {
		limit = DefaultHandlerQueueCap
	}
	if s.handlerSlots.Add(1) > limit {
		s.handlerSlots.Add(-1)
		w.Header().Set("Retry-After", "1")
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	defer s.handlerSlots.Add(-1)

	body, status := readHandlerBody(w, r)
	if status != 0 {
		http.Error(w, http.StatusText(status), status)
		return
	}

	req := r.Clone(r.Context())
	req.Body = io.NopCloser(bytes.NewReader(body))
	req.ContentLength = int64(len(body))

	resp := newBufferedResponse(HandlerResponseLimit)
	err := s.AskCtx(r.Context(), handlerQueueTimeout, func() error {
		s.serveHandler(resp, req, handlerID)
		return nil
	})
	if r.Context().Err() != nil {
		return
	}
	if err != nil {
		w.Header().Set("Retry-After", "1")
		http.Error(w, http.StatusText(http.StatusServiceUnavailable), http.StatusServiceUnavailable)
		return
	}
	resp.copyTo(w)
}

func readHandlerBody(w http.ResponseWriter, r *http.Request) ([]byte, int) {
	if r.Body == nil || r.Body == http.NoBody {
		return nil, 0
	}
	if r.ContentLength > HandlerBodyLimit {
		return nil, http.StatusRequestEntityTooLarge
	}
	defer r.Body.Close()
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, HandlerBodyLimit))
	if err != nil {
		var tooLarge *http.MaxBytesError
		if errors.As(err, &tooLarge) {
			return nil, http.StatusRequestEntityTooLarge
		}
		return nil, http.StatusBadRequest
	}
	return body, 0
}

func (s *Session) serveHandler(w *bufferedResponse, r *http.Request, handlerID string) {
	entry := s.findHTTPHandler(handlerID)
	if entry == nil {
		w.reset(http.StatusNotFound)
		return
	}

	if entry.method != "" && !strings.EqualFold(r.Method, entry.method) {
		w.reset(http.StatusMethodNotAllowed)
		w.header.Set("Allow", entry.method)
		return
	}

	var chainErr error
	panicErr := s.recoverUser(ErrCodeHandler, handlerPhase, entry.o, func() {
		for _, h := range entry.chain {
			if h == nil {
				continue
			}
			if err := h(w, r); err != nil {
				chainErr = err
				return
			}
		}
	})

	switch {
	case panicErr != nil:
		w.reset(http.StatusInternalServerError)
		s.deliver(entry.o, panicErr)
	case w.exceeded:
		w.reset(http.StatusInternalServerError)
		s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "http_handler:response-too-large",
			Message: fmt.Sprintf("handler %s wrote more than %d bytes", handlerID, w.limit),
			Meta: map[string]any{
				"handler_id": handlerID,
			},
		})
	case chainErr != nil && !w.wrote():
		w.reset(http.StatusInternalServerError)
	}
}
