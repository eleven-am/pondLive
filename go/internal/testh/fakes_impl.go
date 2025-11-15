package testh

import (
	"github.com/eleven-am/pondlive/go/internal/render"
	"github.com/eleven-am/pondlive/go/pkg/live/html"
)

type mockComponent struct {
	id       string
	handlers []html.EventHandler
}

func (m *mockComponent) RegisterHandler(handler html.EventHandler) int {
	if m == nil {
		return -1
	}
	slotIdx := len(m.handlers)
	m.handlers = append(m.handlers, handler)
	return slotIdx
}

func (m *mockComponent) ComponentID() string {
	if m == nil {
		return ""
	}
	return m.id
}

type mockComponentLookup struct {
	components map[string]*mockComponent
}

func NewMockComponentLookup() render.ComponentLookup {
	return &mockComponentLookup{
		components: map[string]*mockComponent{
			"": &mockComponent{id: ""},
		},
	}
}

func (m *mockComponentLookup) LookupComponent(id string) render.ComponentHandlerTarget {
	if m == nil {
		return nil
	}
	if comp, ok := m.components[id]; ok {
		return comp
	}
	comp := &mockComponent{id: id}
	m.components[id] = comp
	return comp
}

type registryFactory struct{}

// NewRegistryFactory returns the default deterministic registry factory.
func NewRegistryFactory() RegistryFactory {
	return registryFactory{}
}

func (registryFactory) NewRegistry() HandlerRegistry {
	return &handlerRegistry{handlers: make(map[string]any)}
}

// handlerRegistry implements HandlerRegistry with a simple map.
type handlerRegistry struct {
	handlers map[string]any
}

func (r *handlerRegistry) Lookup(handlerID string) any {
	if r == nil || handlerID == "" {
		return nil
	}
	return r.handlers[handlerID]
}

func (r *handlerRegistry) Register(handlerID string, handler any) {
	if r == nil || handlerID == "" {
		return
	}
	r.handlers[handlerID] = handler
}

type channel struct {
	queue []any
}

// NewProtocolChannel constructs an in-memory protocol channel.
func NewProtocolChannel() ProtocolChannel {
	return &channel{}
}

func (c *channel) Enqueue(message any) {
	if c == nil {
		return
	}
	c.queue = append(c.queue, message)
}

func (c *channel) Drain() []any {
	if c == nil {
		return nil
	}
	if len(c.queue) == 0 {
		return nil
	}
	out := make([]any, len(c.queue))
	copy(out, c.queue)
	c.queue = nil
	return out
}
