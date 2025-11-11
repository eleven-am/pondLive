package dom

// ActionExecutor provides the interface for executing DOM actions on the client.
// This interface is implemented by runtime.Ctx to allow action mixins to enqueue
// DOM operations without creating import cycles.
type ActionExecutor interface {
	// EnqueueDOMAction enqueues a DOM action effect to be sent to the client.
	EnqueueDOMAction(effect DOMActionEffect)
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
