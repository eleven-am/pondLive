package server

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
)

const (
	livePath = "/live"

	DefaultReadLimit    = 64 << 10
	DefaultJoinDeadline = 10 * time.Second
)

var ErrNilRegistry = errors.New("server: session registry is nil")

type TransportConfig struct {
	AllowedOrigins  []string
	AttachSecret    []byte
	TokenTTL        time.Duration
	ReadLimit       int64
	JoinDeadline    time.Duration
	RetainedHeaders []string
	session.WebSocketOptions
}

func (c TransportConfig) withDefaults() TransportConfig {
	if c.ReadLimit <= 0 {
		c.ReadLimit = DefaultReadLimit
	}
	if c.JoinDeadline <= 0 {
		c.JoinDeadline = DefaultJoinDeadline
	}
	c.WebSocketOptions = c.WebSocketOptions.WithDefaults()
	return c
}

type Endpoint struct {
	ctx      context.Context
	cancel   context.CancelFunc
	registry *SessionRegistry
	tokens   *attachTokens
	cfg      TransportConfig
	publish  func(sess *session.LiveSession, gen uint64, evt protocol.ClientEvt) (bool, error)
	conns    connTracker

	refreshEvery  time.Duration
	disconnectTTL time.Duration
}

type connTracker struct {
	mu     sync.Mutex
	active int
	closed bool
	idle   chan struct{}
}

func (c *connTracker) enter() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.closed {
		return false
	}
	c.active++
	return true
}

func (c *connTracker) leave() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.active--
	if c.active == 0 && c.idle != nil {
		close(c.idle)
		c.idle = nil
	}
}

func (c *connTracker) drain() <-chan struct{} {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.closed = true
	done := make(chan struct{})
	if c.active == 0 {
		close(done)
		return done
	}
	if c.idle == nil {
		c.idle = make(chan struct{})
	}
	return c.idle
}

type hijackCapture struct {
	http.ResponseWriter
	conn net.Conn
}

func (h *hijackCapture) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hj, ok := h.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("server: response writer cannot be hijacked")
	}
	conn, rw, err := hj.Hijack()
	if err == nil {
		h.conn = conn
	}
	return conn, rw, err
}

type inboundFrame struct {
	protocol.ClientEvt
	Seq    uint64 `json:"seq"`
	Reason string `json:"reason"`
}

func NewEndpoint(ctx context.Context, registry *SessionRegistry, cfg TransportConfig) (*Endpoint, error) {
	if registry == nil {
		return nil, ErrNilRegistry
	}
	cfg = cfg.withDefaults()
	tokens, err := newAttachTokens(cfg.AttachSecret, cfg.TokenTTL)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithCancel(ctx)
	disconnectTTL := tokens.ttl
	refreshEvery := disconnectTTL / 2
	tokens.ttl = disconnectTTL + refreshEvery
	return &Endpoint{ctx: ctx, cancel: cancel, registry: registry, tokens: tokens, cfg: cfg, publish: publishToBus, refreshEvery: refreshEvery, disconnectTTL: disconnectTTL}, nil
}

func (e *Endpoint) Shutdown(ctx context.Context) error {
	idle := e.conns.drain()
	var err error
	select {
	case <-idle:
	case <-ctx.Done():
		err = ctx.Err()
	}
	e.cancel()
	return err
}

func (e *Endpoint) originAllowed(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return false
	}
	for _, allowed := range e.cfg.AllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" {
		return false
	}
	return strings.EqualFold(u.Host, r.Host)
}

func (e *Endpoint) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !e.originAllowed(r) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}
	if !e.conns.enter() {
		http.Error(w, "server shutting down", http.StatusServiceUnavailable)
		return
	}
	defer e.conns.leave()

	capture := &hijackCapture{ResponseWriter: w}
	ws, err := websocket.Accept(capture, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		return
	}
	ws.SetReadLimit(e.cfg.ReadLimit)

	upgrade := headers.NewRequestInfoFromHeaders(r.Header, e.cfg.RetainedHeaders...).Headers
	deadline := time.Now().Add(e.cfg.JoinDeadline)

	tr := session.NewWebSocketTransport(e.ctx, ws, e.cfg.WebSocketOptions)
	tr.BindConn(capture.conn)
	joinTimer := time.AfterFunc(e.cfg.JoinDeadline, func() {
		tr.Terminate(session.CloseJoinTimeout, "join timeout")
	})

	var sess *session.LiveSession
	defer func() {
		joinTimer.Stop()
		tr.Abort()
		if sess != nil {
			sess.Detach(tr)
		}
		tr.Wait()
	}()

	data, err := tr.Receive()
	if err != nil {
		return
	}
	var (
		sid session.SessionID
		res session.AttachResult
		ok  bool
	)
	sess, sid, res, ok = e.join(tr, data, upgrade, deadline)
	if !ok {
		return
	}
	joinTimer.Stop()

	first, _ := json.Marshal(protocol.JoinedFrame{T: protocol.FrameJoined, Gen: res.Gen, Tok: e.tokens.issue(string(sid)), CS: res.CS, Mode: res.Mode})
	tr.Start(first, session.TokenRefresh{Every: e.refreshEvery, Frame: func() []byte {
		frame, _ := json.Marshal(protocol.TokFrame{T: protocol.FrameTok, Tok: e.tokens.issue(string(sid))})
		return frame
	}})

	for {
		data, err := tr.Receive()
		if err != nil {
			return
		}
		if !e.dispatch(sess, tr, sid, res.Gen, data) {
			return
		}
	}
}

func (e *Endpoint) join(tr *session.WebSocketTransport, data []byte, upgrade http.Header, deadline time.Time) (*session.LiveSession, session.SessionID, session.AttachResult, bool) {
	var frame protocol.JoinFrame
	if err := json.Unmarshal(data, &frame); err != nil || frame.T != protocol.FrameJoin || frame.SID == "" {
		e.decline(tr, session.CloseBadRequest, "malformed join")
		return nil, "", session.AttachResult{}, false
	}

	if err := e.tokens.verify(frame.SID, frame.Tok); err != nil {
		e.decline(tr, session.CloseUnauthorized, "invalid or expired token")
		return nil, "", session.AttachResult{}, false
	}

	sid := session.SessionID(frame.SID)
	live, found := e.registry.Lookup(sid)
	if !found {
		e.decline(tr, session.CloseNotFound, "session not found")
		return nil, "", session.AttachResult{}, false
	}

	if frame.Ver != live.Version() {
		e.decline(tr, session.CloseBadRequest, "unsupported version")
		return nil, "", session.AttachResult{}, false
	}

	ctx, cancel := context.WithDeadline(e.ctx, deadline)
	defer cancel()
	res, err := live.Attach(ctx, tr, upgrade, frame.BS, frame.Resync)
	switch {
	case err == nil:
		return live, sid, res, true
	case errors.Is(err, context.DeadlineExceeded), errors.Is(err, runtime.ErrSessionBusy):
		e.decline(tr, session.CloseJoinTimeout, "join timeout")
	case errors.Is(err, session.ErrResetTooLarge):
		live.RequestClose(session.ReasonResetTooLarge)
		e.decline(tr, session.CloseNotFound, "session reset too large")
	default:
		e.decline(tr, session.CloseNotFound, "session not found")
	}
	return live, "", session.AttachResult{}, false
}

func (e *Endpoint) decline(tr *session.WebSocketTransport, code websocket.StatusCode, reason string) {
	frame, _ := json.Marshal(protocol.DeclinedFrame{T: protocol.FrameDeclined, Code: int(code), Reason: reason})
	tr.Decline(code, reason, frame)
}

func (e *Endpoint) dispatch(sess *session.LiveSession, tr *session.WebSocketTransport, sid session.SessionID, gen uint64, data []byte) bool {
	var in inboundFrame
	if err := json.Unmarshal(data, &in); err != nil {
		tr.Terminate(websocket.StatusInvalidFramePayloadData, "malformed message")
		return false
	}
	if in.SID != string(sid) {
		tr.Terminate(websocket.StatusPolicyViolation, "session mismatch")
		return false
	}

	switch string(in.Type) {
	case protocol.FrameAck:
		if err := sess.Ack(gen, in.Seq); err != nil {
			tr.Terminate(websocket.StatusPolicyViolation, "ack beyond head")
			return false
		}
		return true
	case protocol.FrameResync:
		sess.RequestResync(gen, in.Seq, in.Reason)
		return true
	}

	if in.CS == 0 {
		tr.Terminate(websocket.StatusPolicyViolation, "missing client seq")
		return false
	}

	accepted, err := e.publish(sess, gen, in.ClientEvt)
	switch {
	case errors.Is(err, session.ErrStaleConnection):
		return false
	case errors.Is(err, runtime.ErrClientBacklog):
		tr.Terminate(websocket.StatusTryAgainLater, "too many pending events")
		return false
	case err != nil:
		tr.Terminate(websocket.StatusGoingAway, "session closed")
		return false
	}
	if accepted {
		tr.NoteClientSeq(in.CS)
	}
	return true
}

func publishToBus(sess *session.LiveSession, gen uint64, evt protocol.ClientEvt) (bool, error) {
	return sess.Deliver(gen, evt)
}
