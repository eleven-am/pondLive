package protocol

// Effect represents a side effect to be executed on the client.
// Effects are sent in Frame messages and executed after DOM patches are applied.
//
// Note: Metadata (title, meta tags, link tags, script tags) are NOT effects.
// They are handled as regular StructuredNodes in the document head and diffed
// like any other DOM nodes. See structured-node-protocol.md section 23.
type Effect interface {
	effectMarker()
}

// DOMActionEffect executes DOM API methods on client-side elements.
// Used for imperative DOM operations like focus(), scrollIntoView(), etc.
type DOMActionEffect struct {
	Type     string `json:"type"` // Always "dom"
	Kind     string `json:"kind"`
	Ref      string `json:"ref"`
	Method   string `json:"method,omitempty"`
	Args     []any  `json:"args,omitempty"`
	Prop     string `json:"prop,omitempty"`
	Value    any    `json:"value,omitempty"`
	Class    string `json:"class,omitempty"`
	On       bool   `json:"on,omitempty"`
	Behavior string `json:"behavior,omitempty"`
	Block    string `json:"block,omitempty"`
	Inline   string `json:"inline,omitempty"`
}

func (DOMActionEffect) effectMarker() {}

// CookieEffect triggers client to sync cookies via HTTP endpoint.
// The client performs a fetch to the endpoint with the session ID and token
// to retrieve cookie mutations from the server.
type CookieEffect struct {
	Type     string `json:"type"` // Always "cookies"
	Endpoint string `json:"endpoint"`
	SID      string `json:"sid"`
	Token    string `json:"token"`
	Method   string `json:"method,omitempty"`
}

func (CookieEffect) effectMarker() {}
