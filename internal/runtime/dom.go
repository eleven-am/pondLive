package runtime

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
)

var (
	ErrNilRef       = errors.New("runtime: ref is nil")
	ErrNilSession   = errors.New("runtime: session is nil")
	ErrQueryTimeout = errors.New("runtime: query timeout")
)

const defaultDOMTimeout = 5 * time.Second

type domRequestManager struct {
	pending   map[string]chan protocol.DOMResponsePayload
	mu        sync.Mutex
	nextID    atomic.Uint64
	timeout   atomic.Int64
	closedSub *protocol.Subscription
}

func newDOMRequestManager(bus *protocol.Bus, timeout time.Duration) *domRequestManager {
	if timeout == 0 {
		timeout = defaultDOMTimeout
	}

	mgr := &domRequestManager{
		pending: make(map[string]chan protocol.DOMResponsePayload),
	}
	mgr.setTimeout(timeout)

	mgr.closedSub = bus.SubscribeToDOMResponses(func(resp protocol.DOMResponsePayload) {
		mgr.handleResponse(resp)
	})

	return mgr
}

func (m *domRequestManager) allocateRequest() (string, chan protocol.DOMResponsePayload) {
	id := fmt.Sprintf("dom-%d", m.nextID.Add(1))
	ch := make(chan protocol.DOMResponsePayload, 1)

	m.mu.Lock()
	m.pending[id] = ch
	m.mu.Unlock()

	return id, ch
}

func (m *domRequestManager) releaseRequest(id string) {
	m.mu.Lock()
	delete(m.pending, id)
	m.mu.Unlock()
}

func (m *domRequestManager) handleResponse(resp protocol.DOMResponsePayload) {
	log.Printf("[handleResponse] received requestId=%s values=%v error=%s", resp.RequestID, resp.Values, resp.Error)
	m.mu.Lock()
	ch, ok := m.pending[resp.RequestID]
	pendingCount := len(m.pending)
	m.mu.Unlock()

	log.Printf("[handleResponse] pending=%d found=%v requestId=%s", pendingCount, ok, resp.RequestID)
	if ok && ch != nil {
		select {
		case ch <- resp:
			log.Printf("[handleResponse] sent to channel requestId=%s", resp.RequestID)
		default:
			log.Printf("[handleResponse] channel blocked requestId=%s", resp.RequestID)
		}
	}
}

func (m *domRequestManager) wait(ch chan protocol.DOMResponsePayload) (protocol.DOMResponsePayload, error) {
	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(time.Duration(m.timeout.Load())):
		return protocol.DOMResponsePayload{}, ErrQueryTimeout
	}
}

func (m *domRequestManager) setTimeout(timeout time.Duration) {
	if timeout == 0 {
		timeout = defaultDOMTimeout
	}
	m.timeout.Store(int64(timeout))
}

func (m *domRequestManager) close() {
	if m.closedSub != nil {
		m.closedSub.Unsubscribe()
	}
	m.mu.Lock()
	m.pending = nil
	m.mu.Unlock()
}

func (c *Ctx) Call(ref *ElementRef, method string, args ...any) {
	if ref == nil || ref.RefID() == "" {
		return
	}
	if c.session == nil || c.session.Bus == nil {
		return
	}

	c.session.Bus.PublishDOMCall(protocol.DOMCallPayload{
		Ref:    ref.RefID(),
		Method: method,
		Args:   args,
	})
}

func (c *Ctx) Set(ref *ElementRef, prop string, value any) {
	if ref == nil || ref.RefID() == "" {
		return
	}
	if c.session == nil || c.session.Bus == nil {
		return
	}

	c.session.Bus.PublishDOMSet(protocol.DOMSetPayload{
		Ref:   ref.RefID(),
		Prop:  prop,
		Value: value,
	})
}

func (c *Ctx) Query(ref *ElementRef, selectors ...string) (map[string]any, error) {
	if ref == nil || ref.RefID() == "" {
		return nil, ErrNilRef
	}
	if c.session == nil || c.session.Bus == nil {
		return nil, ErrNilSession
	}
	if len(selectors) == 0 {
		return map[string]any{}, nil
	}

	mgr := c.session.getDOMRequestManager()
	if mgr == nil {
		return nil, ErrNilSession
	}

	requestID, ch := mgr.allocateRequest()
	defer mgr.releaseRequest(requestID)

	log.Printf("[Query] sending query requestId=%s ref=%s selectors=%v", requestID, ref.RefID(), selectors)

	c.session.Bus.PublishDOMQuery(protocol.DOMQueryPayload{
		RequestID: requestID,
		Ref:       ref.RefID(),
		Selectors: selectors,
	})

	log.Printf("[Query] waiting for response requestId=%s", requestID)
	resp, err := mgr.wait(ch)
	if err != nil {
		log.Printf("[Query] error requestId=%s err=%v", requestID, err)
		return nil, err
	}
	log.Printf("[Query] got response requestId=%s values=%v", requestID, resp.Values)

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	return resp.Values, nil
}

func (c *Ctx) AsyncCall(ref *ElementRef, method string, args ...any) (any, error) {
	if ref == nil || ref.RefID() == "" {
		return nil, ErrNilRef
	}
	if c.session == nil || c.session.Bus == nil {
		return nil, ErrNilSession
	}

	mgr := c.session.getDOMRequestManager()
	if mgr == nil {
		return nil, ErrNilSession
	}

	requestID, ch := mgr.allocateRequest()
	defer mgr.releaseRequest(requestID)

	c.session.Bus.PublishDOMAsync(protocol.DOMAsyncPayload{
		RequestID: requestID,
		Ref:       ref.RefID(),
		Method:    method,
		Args:      args,
	})

	resp, err := mgr.wait(ch)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	return resp.Result, nil
}
