package work

// Item represents something that can be applied to an element
// during construction. Items can be props (attrs, styles, events)
// or children (other nodes).
type Item interface {
	ApplyTo(*Element)
}

// Attachment is anything that provides a ref ID for attachment.
// This is implemented by runtime's ElementRef type.
type Attachment interface {
	RefID() string
}

// HandlerProvider is an optional interface for attachments that provide event handlers.
// When an attachment implements this, its handlers are applied to the element.
type HandlerProvider interface {
	// Events returns all event names that have handlers.
	Events() []string
	// ProxyHandler returns a single handler that dispatches to all handlers for the event.
	ProxyHandler(event string) Handler
}

// HandlerAdder is an optional interface for attachments that can receive event handlers.
// This allows action structs to register event handlers on refs.
type HandlerAdder interface {
	// AddHandler registers an event handler on the attachment.
	// Multiple handlers for the same event are accumulated.
	AddHandler(event string, handler Handler)
}

// ElementDescriptor provides type-safe ref attachment.
type ElementDescriptor interface {
	TagName() string
}
