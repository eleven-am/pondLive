package runtime

import (
	"context"
	"errors"
	"fmt"
	"log"
	"runtime/debug"
	"sync"
	"sync/atomic"
	"time"

	"github.com/petermattis/goid"

	"github.com/eleven-am/pondlive/internal/effectguard"
	"github.com/eleven-am/pondlive/internal/protocol"
)

var (
	ErrSessionClosed = errors.New("runtime: session closed")
	ErrSessionBusy   = errors.New("runtime: session busy")
	ErrInboxOverflow = errors.New("runtime: session inbox overflow")
	ErrClientBacklog = errors.New("runtime: client event backlog")
	ErrNotSettled    = errors.New("runtime: flush did not settle")
	ErrNotStarted    = errors.New("runtime: session not started")
)

const (
	defaultMailboxCap = 1 << 16
	maxClientEvents   = 1024
	flushEverySteps   = 64
	flushAskBound     = 10 * time.Second
	closeWaitBound    = 10 * time.Second
)

type msgKind uint8

const (
	msgStep msgKind = iota
	msgSetter
	msgClientEvent
)

type message struct {
	kind  msgKind
	run   func(*Session)
	thunk func()
}

func (m message) exec(s *Session) {
	if m.thunk != nil {
		m.thunk()
		return
	}
	m.run(s)
}

type mailbox struct {
	mu     sync.Mutex
	q      []message
	head   int
	final  *message
	closed bool
	wake   chan struct{}
	max    int
}

func (m *mailbox) wakeChan() chan struct{} {
	if m.wake == nil {
		m.wake = make(chan struct{}, 1)
	}
	return m.wake
}

func (m *mailbox) signal(wake chan struct{}) {
	select {
	case wake <- struct{}{}:
	default:
	}
}

func (m *mailbox) post(msg message) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrSessionClosed
	}
	limit := m.max
	if limit <= 0 {
		limit = defaultMailboxCap
	}
	if len(m.q)-m.head >= limit {
		m.mu.Unlock()
		return ErrInboxOverflow
	}
	if m.head > 0 && len(m.q) == cap(m.q) {
		n := copy(m.q, m.q[m.head:])
		clear(m.q[n:])
		m.q = m.q[:n]
		m.head = 0
	}
	m.q = append(m.q, msg)
	wake := m.wakeChan()
	m.mu.Unlock()
	m.signal(wake)
	return nil
}

func (m *mailbox) requestFinal(msg message) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrSessionClosed
	}
	if m.final == nil {
		m.final = &msg
	}
	wake := m.wakeChan()
	m.mu.Unlock()
	m.signal(wake)
	return nil
}

func (m *mailbox) pop() (message, bool) {
	for {
		m.mu.Lock()
		if m.final != nil {
			msg := *m.final
			m.final = nil
			m.q = nil
			m.head = 0
			m.mu.Unlock()
			return msg, true
		}
		if m.head < len(m.q) {
			msg := m.dequeue()
			m.mu.Unlock()
			return msg, true
		}
		if m.closed {
			m.mu.Unlock()
			return message{}, false
		}
		wake := m.wakeChan()
		m.mu.Unlock()
		<-wake
	}
}

func (m *mailbox) dequeue() message {
	msg := m.q[m.head]
	m.q[m.head] = message{}
	m.head++
	if m.head == len(m.q) {
		m.q = m.q[:0]
		m.head = 0
	}
	return msg
}

func (m *mailbox) popSetter() (message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.final != nil || m.head >= len(m.q) || m.q[m.head].kind != msgSetter {
		return message{}, false
	}
	return m.dequeue(), true
}

func (m *mailbox) empty() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.head >= len(m.q) && m.final == nil
}

func (m *mailbox) close() {
	m.mu.Lock()
	m.closed = true
	m.q = nil
	m.head = 0
	m.final = nil
	m.mu.Unlock()
}

func (s *Session) Start() {
	if s == nil || !s.starting.CompareAndSwap(false, true) {
		return
	}
	s.exited = make(chan struct{})
	s.ctx, s.cancelCtx = context.WithCancel(context.Background())
	s.running.Store(true)
	_ = s.post(message{kind: msgSetter, thunk: s.syncExternals})
	go s.loop()
}

func (s *Session) Done() <-chan struct{} {
	if s == nil || s.exited == nil {
		return nil
	}
	return s.exited
}

func (s *Session) loop() {
	s.actorGID.Store(goid.Get())
	s.graph.BindActor()
	defer close(s.exited)

	flush := message{kind: msgStep, run: func(s *Session) { s.flush() }}
	steps := 0
	for {
		msg, ok := s.mb.pop()
		if !ok {
			return
		}
		s.runStep(msg)
		if s.closed {
			return
		}
		steps++
		if s.hasWork() && (steps >= flushEverySteps || s.mb.empty()) {
			s.runStep(flush)
			steps = 0
			if s.closed {
				return
			}
		}
	}
}

func (s *Session) runStep(msg message) {
	s.stepKind.Store(uint32(msg.kind))
	s.stepSeq.Add(1)
	s.stepStart.Store(time.Now().UnixNano())
	defer func() {
		s.stepStart.Store(0)
		if r := recover(); r != nil {
			s.invariantFailure(r)
		}
	}()
	msg.exec(s)
}

func (s *Session) invariantFailure(r any) {
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "actor:panic",
		Message: fmt.Sprintf("panic: %v", r),
		Stack:   string(debug.Stack()),
		Meta: map[string]any{
			"code": string(ErrCodeSession),
		},
	})
	s.closeStep()
}

const PostOp = "Session.Post/RunOnActor"

func (s *Session) post(msg message) error {
	if s == nil {
		return ErrSessionClosed
	}
	if effectguard.Refuse(PostOp) {
		return nil
	}
	if !s.running.Load() {
		return ErrNotStarted
	}
	err := s.mb.post(msg)
	if errors.Is(err, ErrInboxOverflow) {
		s.reportOverflow()
	}
	return err
}

func (s *Session) reportOverflow() {
	if s.overflows.Add(1) != 1 {
		return
	}
	_ = s.mb.requestFinal(message{kind: msgStep, run: func(s *Session) {
		s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "actor:inbox-overflow",
			Message: "session inbox overflow, closing session",
			Meta: map[string]any{
				"code":    string(ErrCodeSession),
				"dropped": s.overflows.Load(),
			},
		})
		s.closeStep()
	}})
}

func (s *Session) report(d protocol.Diagnostic) {
	if s.Bus == nil {
		return
	}
	defer func() {
		if r := recover(); r != nil {
			log.Printf("runtime: diagnostics subscriber panicked while reporting %q: %v", d.Phase, r)
		}
	}()
	s.Bus.ReportDiagnostic(d)
}

func (s *Session) Post(fn func()) error {
	if fn == nil {
		return nil
	}
	return s.post(message{kind: msgStep, run: func(*Session) { fn() }})
}

func (s *Session) RunOnActor(fn func()) {
	if s == nil || fn == nil {
		return
	}
	_ = s.post(message{kind: msgStep, thunk: fn})
}

func (s *Session) PostClientEvent(fn func()) error {
	if s == nil || fn == nil {
		return nil
	}
	if s.clientEvents.Add(1) > maxClientEvents {
		s.clientEvents.Add(-1)
		return ErrClientBacklog
	}
	err := s.post(message{kind: msgClientEvent, run: func(*Session) {
		s.clientEvents.Add(-1)
		fn()
	}})
	if err != nil {
		s.clientEvents.Add(-1)
	}
	return err
}

func (s *Session) onActor() bool {
	return s.running.Load() && s.actorGID.Load() == goid.Get()
}

type callState int32

const (
	callWaiting callState = iota
	callClaimed
	callAbandoned
)

type call[T any] struct {
	state atomic.Int32
	done  chan result[T]
}

type result[T any] struct {
	val T
	err error
}

func (c *call[T]) claim() bool {
	return c.state.CompareAndSwap(int32(callWaiting), int32(callClaimed))
}

func ask[T any](s *Session, bound time.Duration, fn func(*Session) (T, error)) (T, error) {
	return askWait(s, nil, bound, fn)
}

func askWait[T any](s *Session, ctx context.Context, bound time.Duration, fn func(*Session) (T, error)) (T, error) {
	var zero T
	if s == nil {
		return zero, ErrSessionClosed
	}
	if !s.running.Load() {
		return zero, ErrNotStarted
	}
	if s.devMode && s.onActor() {
		panic("runtime: session actor awaited itself")
	}

	c := &call[T]{done: make(chan result[T], 1)}
	err := s.post(message{kind: msgStep, run: func(s *Session) {
		if !c.claim() {
			return
		}
		r := result[T]{err: ErrSessionClosed}
		defer func() { c.done <- r }()
		r.val, r.err = fn(s)
	}})
	if err != nil {
		return zero, err
	}

	var ctxDone <-chan struct{}
	if ctx != nil {
		ctxDone = ctx.Done()
	}
	var timerC <-chan time.Time
	if bound > 0 {
		timer := time.NewTimer(bound)
		defer timer.Stop()
		timerC = timer.C
	}
	select {
	case r := <-c.done:
		return r.val, r.err
	case <-s.exited:
		if c.state.Load() == int32(callClaimed) {
			r := <-c.done
			return r.val, r.err
		}
		return zero, ErrSessionClosed
	case <-timerC:
		c.state.CompareAndSwap(int32(callWaiting), int32(callAbandoned))
		select {
		case r := <-c.done:
			return r.val, r.err
		default:
			return zero, ErrSessionBusy
		}
	case <-ctxDone:
		c.state.CompareAndSwap(int32(callWaiting), int32(callAbandoned))
		return zero, ctx.Err()
	}
}

func (s *Session) AskCtx(ctx context.Context, bound time.Duration, fn func() error) error {
	_, err := askWait(s, ctx, bound, func(*Session) (struct{}, error) { return struct{}{}, fn() })
	return err
}

func (s *Session) Ask(bound time.Duration, fn func() error) error {
	_, err := ask(s, bound, func(*Session) (struct{}, error) { return struct{}{}, fn() })
	return err
}

func (s *Session) RequestClose() error {
	if s == nil {
		return nil
	}
	if !s.running.Load() {
		s.mb.close()
		return nil
	}
	err := s.mb.requestFinal(message{kind: msgStep, run: func(s *Session) { s.closeStep() }})
	if errors.Is(err, ErrSessionClosed) {
		return nil
	}
	return err
}

func (s *Session) Close() {
	if err := s.CloseWithin(closeWaitBound); err != nil {
		s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "actor:close-timeout",
			Message: err.Error(),
			Meta: map[string]any{
				"code": string(ErrCodeSession),
			},
		})
	}
}

func (s *Session) CloseWithin(bound time.Duration) error {
	if s == nil {
		return nil
	}
	_ = s.RequestClose()
	if s.devMode && s.running.Load() && s.onActor() {
		panic("runtime: session actor awaited its own close")
	}
	if s.running.Load() {
		select {
		case <-s.exited:
		case <-time.After(bound):
			return ErrSessionBusy
		}
	}
	return nil
}

type Timer struct {
	t    Stopper
	dead bool
	d    time.Duration
	fn   func()
}

func (t *Timer) Stop() {
	if t == nil {
		return
	}
	t.dead = true
	t.fn = nil
	if t.t != nil {
		t.t.Stop()
	}
}

func (s *Session) After(d time.Duration, fn func()) *Timer {
	tm := &Timer{}
	if s == nil || fn == nil {
		tm.dead = true
		return tm
	}
	if s.deferred.Load() {
		_ = s.post(message{kind: msgStep, run: func(s *Session) { s.armOrDefer(tm, d, fn) }})
		return tm
	}
	s.arm(tm, d, fn)
	return tm
}

func (s *Session) armOrDefer(tm *Timer, d time.Duration, fn func()) {
	if tm.dead {
		return
	}
	if s.deferred.Load() {
		tm.d = d
		tm.fn = fn
		s.deferredTimers = append(s.deferredTimers, tm)
		return
	}
	s.arm(tm, d, fn)
}

func (s *Session) arm(tm *Timer, d time.Duration, fn func()) {
	tm.d = 0
	tm.fn = nil
	tm.t = s.clock.AfterFunc(d, func() {
		_ = s.post(message{kind: msgStep, run: func(*Session) {
			if tm.dead {
				return
			}
			tm.dead = true
			fn()
		}})
	})
}

func (s *Session) drainSetters() {
	for {
		msg, ok := s.mb.popSetter()
		if !ok {
			return
		}
		msg.exec(s)
	}
}
