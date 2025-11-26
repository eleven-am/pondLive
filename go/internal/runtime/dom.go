package runtime

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Errors
var (
	ErrNilRef       = errors.New("runtime: ref is nil")
	ErrNilSession   = errors.New("runtime: session is nil")
	ErrQueryTimeout = errors.New("runtime: query timeout")
)

// Default timeout for blocking DOM operations
const defaultDOMTimeout = 5 * time.Second

// domRequestManager manages pending DOM requests that expect responses.
type domRequestManager struct {
	pending   map[string]chan protocol.DOMResponsePayload
	mu        sync.Mutex
	nextID    atomic.Uint64
	timeout   atomic.Int64
	closedSub *protocol.Subscription
}

// newDOMRequestManager creates a new request manager and wires it to the bus.
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

// allocateRequest creates a new request ID and registers the response channel.
func (m *domRequestManager) allocateRequest() (string, chan protocol.DOMResponsePayload) {
	id := fmt.Sprintf("dom-%d", m.nextID.Add(1))
	ch := make(chan protocol.DOMResponsePayload, 1)

	m.mu.Lock()
	m.pending[id] = ch
	m.mu.Unlock()

	return id, ch
}

// releaseRequest removes the pending request.
func (m *domRequestManager) releaseRequest(id string) {
	m.mu.Lock()
	delete(m.pending, id)
	m.mu.Unlock()
}

// handleResponse routes a response to the waiting caller.
func (m *domRequestManager) handleResponse(resp protocol.DOMResponsePayload) {
	m.mu.Lock()
	ch, ok := m.pending[resp.RequestID]
	m.mu.Unlock()

	if ok && ch != nil {
		select {
		case ch <- resp:
		default:
		}
	}
}

// wait blocks until response or timeout.
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

// close cleans up the manager.
func (m *domRequestManager) close() {
	if m.closedSub != nil {
		m.closedSub.Unsubscribe()
	}
	m.mu.Lock()
	m.pending = nil
	m.mu.Unlock()
}

// Call invokes a method on the referenced element.
// This is fire-and-forget - it publishes to the bus and returns immediately.
func (c *Ctx) Call(ref work.Attachment, method string, args ...any) {
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

// Set assigns a value to a property on the referenced element.
// This is fire-and-forget - it publishes to the bus and returns immediately.
func (c *Ctx) Set(ref work.Attachment, prop string, value any) {
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

// Query retrieves property values from the referenced element.
// This blocks until the client responds or timeout occurs.
// Returns a map of selector -> value.
func (c *Ctx) Query(ref work.Attachment, selectors ...string) (map[string]any, error) {
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

	c.session.Bus.PublishDOMQuery(protocol.DOMQueryPayload{
		RequestID: requestID,
		Ref:       ref.RefID(),
		Selectors: selectors,
	})

	resp, err := mgr.wait(ch)
	if err != nil {
		return nil, err
	}

	if resp.Error != "" {
		return nil, errors.New(resp.Error)
	}

	return resp.Values, nil
}

// AsyncCall invokes a method on the referenced element and waits for the result.
// This blocks until the client responds or timeout occurs.
func (c *Ctx) AsyncCall(ref work.Attachment, method string, args ...any) (any, error) {
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
