package runtime

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/go/internal/work"
)

// DOM action topic constants
const (
	TopicDOM         = "dom"
	EventDOMCall     = "call"
	EventDOMSet      = "set"
	EventDOMQuery    = "query"
	EventDOMAsync    = "async"
	EventDOMResponse = "response"
)

// DOMCallPayload represents a fire-and-forget method call on an element.
type DOMCallPayload struct {
	Ref    string `json:"ref"`
	Method string `json:"method"`
	Args   []any  `json:"args,omitempty"`
}

// DOMSetPayload represents a property assignment on an element.
type DOMSetPayload struct {
	Ref   string `json:"ref"`
	Prop  string `json:"prop"`
	Value any    `json:"value"`
}

// DOMQueryPayload represents a query request for element properties.
type DOMQueryPayload struct {
	RequestID string   `json:"requestId"`
	Ref       string   `json:"ref"`
	Selectors []string `json:"selectors"`
}

// DOMAsyncPayload represents an async method call request.
type DOMAsyncPayload struct {
	RequestID string `json:"requestId"`
	Ref       string `json:"ref"`
	Method    string `json:"method"`
	Args      []any  `json:"args,omitempty"`
}

// DOMResponsePayload represents a response from the client.
type DOMResponsePayload struct {
	RequestID string         `json:"requestId"`
	Values    map[string]any `json:"values,omitempty"`
	Result    any            `json:"result,omitempty"`
	Error     string         `json:"error,omitempty"`
}

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
	pending   map[string]chan DOMResponsePayload
	mu        sync.Mutex
	nextID    atomic.Uint64
	timeout   atomic.Int64
	closedSub *Subscription
}

// newDOMRequestManager creates a new request manager and wires it to the bus.
func newDOMRequestManager(bus *Bus, timeout time.Duration) *domRequestManager {
	if timeout == 0 {
		timeout = defaultDOMTimeout
	}

	mgr := &domRequestManager{
		pending: make(map[string]chan DOMResponsePayload),
	}
	mgr.setTimeout(timeout)

	mgr.closedSub = bus.Subscribe(TopicDOM, func(event string, data interface{}) {
		if event != EventDOMResponse {
			return
		}
		resp, ok := data.(DOMResponsePayload)
		if !ok {
			return
		}
		mgr.handleResponse(resp)
	})

	return mgr
}

// allocateRequest creates a new request ID and registers the response channel.
func (m *domRequestManager) allocateRequest() (string, chan DOMResponsePayload) {
	id := fmt.Sprintf("dom-%d", m.nextID.Add(1))
	ch := make(chan DOMResponsePayload, 1)

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
func (m *domRequestManager) handleResponse(resp DOMResponsePayload) {
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
func (m *domRequestManager) wait(ch chan DOMResponsePayload) (DOMResponsePayload, error) {
	select {
	case resp := <-ch:
		return resp, nil
	case <-time.After(time.Duration(m.timeout.Load())):
		return DOMResponsePayload{}, ErrQueryTimeout
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

	c.session.Bus.Publish(TopicDOM, EventDOMCall, DOMCallPayload{
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

	c.session.Bus.Publish(TopicDOM, EventDOMSet, DOMSetPayload{
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

	c.session.Bus.Publish(TopicDOM, EventDOMQuery, DOMQueryPayload{
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

	c.session.Bus.Publish(TopicDOM, EventDOMAsync, DOMAsyncPayload{
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
