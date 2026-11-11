package session

import (
	"context"
	"errors"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/coder/websocket"
	"github.com/eleven-am/pondlive/internal/protocol"
)

const (
	CloseBadRequest   websocket.StatusCode = 4400
	CloseUnauthorized websocket.StatusCode = 4401
	CloseNotFound     websocket.StatusCode = 4404
	CloseJoinTimeout  websocket.StatusCode = 4408
	CloseReplaced     websocket.StatusCode = 4409
)

const (
	DefaultOutboundBudget = 4 << 20
	DefaultPingInterval   = 20 * time.Second
	DefaultPongTimeout    = 10 * time.Second
	DefaultCloseTimeout   = time.Second
)

var ErrOutboundBudgetExceeded = errors.New("session: outbound budget exceeded")

type WebSocketOptions struct {
	OutboundBudget int
	PingInterval   time.Duration
	PongTimeout    time.Duration
	CloseTimeout   time.Duration
}

func (o WebSocketOptions) WithDefaults() WebSocketOptions {
	if o.OutboundBudget <= 0 {
		o.OutboundBudget = DefaultOutboundBudget
	}
	if o.PingInterval <= 0 {
		o.PingInterval = DefaultPingInterval
	}
	if o.PongTimeout <= 0 {
		o.PongTimeout = DefaultPongTimeout
	}
	if o.CloseTimeout <= 0 {
		o.CloseTimeout = DefaultCloseTimeout
	}
	return o
}

type TokenRefresh struct {
	Every time.Duration
	Frame func() []byte
}

type WebSocketTransport struct {
	ws      *websocket.Conn
	ctx     context.Context
	cancel  context.CancelFunc
	opts    WebSocketOptions
	closing chan struct{}
	wake    chan struct{}
	wg      sync.WaitGroup

	raw atomic.Pointer[net.Conn]

	evackWant atomic.Uint64
	evackSent uint64

	mu     sync.Mutex
	closed bool
	queue  [][]byte
	queued int
}

func NewWebSocketTransport(parent context.Context, ws *websocket.Conn, opts WebSocketOptions) *WebSocketTransport {
	ctx, cancel := context.WithCancel(parent)
	return &WebSocketTransport{
		ws:      ws,
		ctx:     ctx,
		cancel:  cancel,
		opts:    opts.WithDefaults(),
		closing: make(chan struct{}),
		wake:    make(chan struct{}, 1),
	}
}

func (t *WebSocketTransport) SendFrame(frame []byte) error {
	if t == nil {
		return nil
	}
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	if t.queued+len(frame) > t.opts.OutboundBudget {
		t.mu.Unlock()
		t.Terminate(websocket.StatusTryAgainLater, "outbound budget exceeded")
		return ErrOutboundBudgetExceeded
	}
	t.queue = append(t.queue, frame)
	t.queued += len(frame)
	t.mu.Unlock()

	t.signal()
	return nil
}

func (t *WebSocketTransport) NoteClientSeq(cs uint64) {
	if t == nil {
		return
	}
	for {
		cur := t.evackWant.Load()
		if cs <= cur {
			return
		}
		if t.evackWant.CompareAndSwap(cur, cs) {
			break
		}
	}
	t.signal()
}

func (t *WebSocketTransport) flushEvack() bool {
	want := t.evackWant.Load()
	if want <= t.evackSent {
		return true
	}
	if !t.write(evackFrame(want)) {
		return false
	}
	t.evackSent = want
	return true
}

func evackFrame(cs uint64) []byte {
	frame := strconv.AppendUint([]byte(`{"t":"`+protocol.FrameEvack+`","cs":`), cs, 10)
	return append(frame, '}')
}

func (t *WebSocketTransport) signal() {
	select {
	case t.wake <- struct{}{}:
	default:
	}
}

func (t *WebSocketTransport) Receive() ([]byte, error) {
	typ, data, err := t.ws.Read(t.ctx)
	if err != nil {
		return nil, err
	}
	if typ != websocket.MessageText {
		t.Terminate(websocket.StatusUnsupportedData, "text frames only")
		return nil, errors.New("session: binary frame received")
	}
	return data, nil
}

func (t *WebSocketTransport) Start(first []byte, refresh TokenRefresh) {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return
	}
	t.queue = append([][]byte{first}, t.queue...)
	t.queued += len(first)
	t.wg.Add(1)
	t.mu.Unlock()

	go t.writeLoop(refresh)
}

func (t *WebSocketTransport) writeLoop(refresh TokenRefresh) {
	defer t.wg.Done()

	ping := time.NewTicker(t.opts.PingInterval)
	defer ping.Stop()

	var refreshC <-chan time.Time
	if refresh.Every > 0 && refresh.Frame != nil {
		ticker := time.NewTicker(refresh.Every)
		defer ticker.Stop()
		refreshC = ticker.C
	}

	if !t.drain() || !t.flushEvack() {
		return
	}
	for {
		select {
		case <-t.closing:
			return
		case <-t.wake:
			if !t.drain() || !t.flushEvack() {
				return
			}
		case <-ping.C:
			if !t.ping() {
				return
			}
		case <-refreshC:
			if !t.write(refresh.Frame()) {
				return
			}
		}
	}
}

func (t *WebSocketTransport) drain() bool {
	for {
		t.mu.Lock()
		if t.closed {
			t.mu.Unlock()
			return false
		}
		if len(t.queue) == 0 {
			t.queue = nil
			t.mu.Unlock()
			return true
		}
		frame := t.queue[0]
		t.queue[0] = nil
		t.queue = t.queue[1:]
		t.mu.Unlock()

		if !t.write(frame) {
			return false
		}

		t.mu.Lock()
		t.queued -= len(frame)
		t.mu.Unlock()
	}
}

func (t *WebSocketTransport) write(frame []byte) bool {
	wctx, cancel := context.WithTimeout(t.ctx, t.opts.PongTimeout)
	defer cancel()
	if err := t.ws.Write(wctx, websocket.MessageText, frame); err != nil {
		t.Abort()
		return false
	}
	return true
}

func (t *WebSocketTransport) ping() bool {
	pctx, cancel := context.WithTimeout(t.ctx, t.opts.PongTimeout)
	defer cancel()
	if err := t.ws.Ping(pctx); err != nil {
		t.Abort()
		return false
	}
	return true
}

func (t *WebSocketTransport) begin(withCloser bool) bool {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.closed {
		return false
	}
	t.closed = true
	t.queue = nil
	t.queued = 0
	close(t.closing)
	if withCloser {
		t.wg.Add(1)
	}
	return true
}

func (t *WebSocketTransport) Terminate(code websocket.StatusCode, reason string) {
	if t == nil || !t.begin(true) {
		return
	}
	go func() {
		defer t.wg.Done()
		t.closeBounded(code, reason)
	}()
}

func (t *WebSocketTransport) Abort() {
	if t == nil || !t.begin(false) {
		return
	}
	_ = t.ws.CloseNow()
}

func (t *WebSocketTransport) Decline(code websocket.StatusCode, reason string, frame []byte) {
	if t == nil || !t.begin(false) {
		return
	}
	wctx, cancel := context.WithTimeout(t.ctx, t.opts.PongTimeout)
	_ = t.ws.Write(wctx, websocket.MessageText, frame)
	cancel()
	t.closeBounded(code, reason)
}

func (t *WebSocketTransport) BindConn(conn net.Conn) {
	if t == nil || conn == nil {
		return
	}
	t.raw.Store(&conn)
}

func (t *WebSocketTransport) closeBounded(code websocket.StatusCode, reason string) {
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = t.ws.Close(code, reason)
	}()
	timer := time.NewTimer(t.opts.CloseTimeout)
	defer timer.Stop()
	select {
	case <-done:
	case <-timer.C:
		if raw := t.raw.Load(); raw != nil {
			_ = (*raw).Close()
		}
		<-done
	}
}

func (t *WebSocketTransport) Wait() {
	if t == nil {
		return
	}
	t.wg.Wait()
	t.cancel()
}

func (t *WebSocketTransport) Closed() bool {
	if t == nil {
		return true
	}
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.closed
}

func (t *WebSocketTransport) Close() error {
	t.Terminate(websocket.StatusGoingAway, "session closed")
	return nil
}
