package runtime

type Event struct {
	Name    string
	Value   string
	Payload map[string]any
	Form    map[string]string
}

type Updates interface{ isUpdates() }

type EventHandler func(e Event) Updates

type HandlerID string

type HandlerRegistry interface {
	Ensure(fn EventHandler) HandlerID
	Get(id HandlerID) (EventHandler, bool)
	AddRef(id HandlerID)
	DelRef(id HandlerID)
}
