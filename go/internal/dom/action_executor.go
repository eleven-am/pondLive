package dom

// Dispatcher provides the interface for dispatching DOM operations to the client.
// This interface is implemented by runtime.Ctx to allow DOM operations without creating import cycles.
type Dispatcher interface {
	// EnqueueDOMAction enqueues a DOM action effect to be sent to the client.
	EnqueueDOMAction(effect DOMActionEffect)

	// DOMGet requests property values from the client for the given element ref.
	DOMGet(ref string, selectors ...string) (map[string]any, error)

	// DOMAsyncCall calls a method on the given element ref and returns the result.
	DOMAsyncCall(ref string, method string, args ...any) (any, error)
}

// DOMActionEffect represents a DOM manipulation to be performed on the client.
type DOMActionEffect struct {
	Type     string `json:"type"`               // "dom"
	Kind     string `json:"kind"`               // "dom.call", "dom.set", etc.
	Ref      string `json:"ref"`                // ref ID like "ref:7"
	Method   string `json:"method,omitempty"`   // method name for dom.call
	Args     []any  `json:"args,omitempty"`     // arguments for dom.call
	Prop     string `json:"prop,omitempty"`     // property name for dom.set/dom.toggle
	Value    any    `json:"value,omitempty"`    // value for dom.set/dom.toggle
	Class    string `json:"class,omitempty"`    // class name for dom.class
	On       *bool  `json:"on,omitempty"`       // toggle state for dom.class
	Behavior string `json:"behavior,omitempty"` // scroll behavior
	Block    string `json:"block,omitempty"`    // scroll block alignment
	Inline   string `json:"inline,omitempty"`   // scroll inline alignment
}

// ScrollOptions configure how the browser should scroll an element into view.
type ScrollOptions struct {
	Behavior string // "auto", "smooth", "instant"
	Block    string // "start", "center", "end", "nearest"
	Inline   string // "start", "center", "end", "nearest"
}
