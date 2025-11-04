package testh

// RegistryFactory produces registries backed by deterministic fakes.
type RegistryFactory interface {
	// NewRegistry constructs a handler registry for a single in-memory session.
	NewRegistry() HandlerRegistry
}

// HandlerRegistry exposes lookup capabilities for fake event handlers.
type HandlerRegistry interface {
	// Lookup returns the handler payload associated with a handlerID, or nil if missing.
	Lookup(handlerID string) any
}

// ProtocolChannel simulates the runtime transport layer used by the engine.
type ProtocolChannel interface {
	// Enqueue schedules an outbound message to be observed by the recorder.
	Enqueue(message any)

	// Drain returns all queued outbound messages in FIFO order.
	Drain() []any
}
