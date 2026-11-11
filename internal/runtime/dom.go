package runtime

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
)

var (
	ErrNilRef         = errors.New("runtime: ref is nil")
	ErrRefNotAttached = errors.New("runtime: ref is not attached to an element")
	ErrNotConnected   = errors.New("runtime: the browser is not connected")
	ErrForbiddenCall  = errors.New("runtime: the ref does not allow this call")
	ErrNilSession     = errors.New("runtime: session is nil")
	ErrQueryTimeout   = errors.New("runtime: query timeout")
	ErrTooManyQueries = errors.New("runtime: too many pending DOM queries")
)

const (
	defaultDOMTimeout     = 5 * time.Second
	maxPendingDOMRequests = 256
)

type domPending struct {
	o     *reactive.Owner
	fn    func(protocol.DOMResponsePayload, error)
	timer *Timer
}

type domRequestManager struct {
	mu      sync.Mutex
	pending map[string]*domPending
	closed  bool
	nextID  atomic.Uint64
	timeout atomic.Int64
}

func newDOMRequestManager(timeout time.Duration) *domRequestManager {
	mgr := &domRequestManager{pending: make(map[string]*domPending)}
	mgr.setTimeout(timeout)
	return mgr
}

func (m *domRequestManager) add(p *domPending) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return "", ErrSessionClosed
	}
	if len(m.pending) >= maxPendingDOMRequests {
		return "", ErrTooManyQueries
	}
	id := fmt.Sprintf("dom-%d", m.nextID.Add(1))
	m.pending[id] = p
	return id, nil
}

func (m *domRequestManager) take(id string) *domPending {
	m.mu.Lock()
	defer m.mu.Unlock()
	p := m.pending[id]
	delete(m.pending, id)
	return p
}

func (m *domRequestManager) setTimeout(timeout time.Duration) {
	if timeout == 0 {
		timeout = defaultDOMTimeout
	}
	m.timeout.Store(int64(timeout))
}

func (m *domRequestManager) close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.closed = true
	m.pending = nil
}

func (s *Session) domAsk(o *reactive.Owner, publish func(requestID string), fn func(protocol.DOMResponsePayload, error)) {
	fail := func(err error) {
		s.RunOnActor(func() { s.invokeHandler(o, func() { fn(protocol.DOMResponsePayload{}, err) }) })
	}
	mgr := s.getDOMRequestManager()
	if mgr == nil {
		if s.domIsClosed() {
			fail(ErrSessionClosed)
			return
		}
		fail(ErrNilSession)
		return
	}
	p := &domPending{o: o, fn: fn}
	id, err := mgr.add(p)
	if err != nil {
		fail(err)
		return
	}
	p.timer = s.After(time.Duration(mgr.timeout.Load()), func() {
		s.finishDOM(id, protocol.DOMResponsePayload{}, ErrQueryTimeout)
	})
	publish(id)
}

func (s *Session) finishDOM(id string, resp protocol.DOMResponsePayload, err error) {
	mgr := s.domReqMgr.Load()
	if mgr == nil {
		return
	}
	p := mgr.take(id)
	if p == nil {
		return
	}
	if p.timer != nil {
		p.timer.Stop()
	}
	if err == nil && resp.Error != "" {
		err = errors.New(resp.Error)
	}
	s.invokeHandler(p.o, func() { p.fn(resp, err) })
}
