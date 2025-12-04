package work

type Item interface {
	ApplyTo(*Element)
}

type Attachment interface {
	AttachTo(*Element)
}

type HandlerProvider interface {
	Events() []string

	ProxyHandler(event string) Handler
}

type HandlerAdder interface {
	AddHandler(event string, handler Handler)
}

type ElementDescriptor interface {
	TagName() string
}
