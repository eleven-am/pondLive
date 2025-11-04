package testh

import (
	handlers "github.com/eleven-am/go/pondlive/internal/handlers"
	"github.com/eleven-am/go/pondlive/pkg/live/html"
)

type registryFactory struct{}

// NewRegistryFactory returns the default deterministic registry factory.
func NewRegistryFactory() RegistryFactory {
	return registryFactory{}
}

func (registryFactory) NewRegistry() HandlerRegistry {
	return &handlerRegistry{Registry: handlers.NewRegistry()}
}

// handlerRegistry adapts handlers.Registry to the HandlerRegistry contract.
type handlerRegistry struct {
	handlers.Registry
}

func (r *handlerRegistry) Lookup(handlerID string) any {
	if r == nil || handlerID == "" {
		return nil
	}
	fn, ok := r.Registry.Get(handlers.ID(handlerID))
	if !ok {
		return nil
	}
	return html.EventHandler(fn)
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
