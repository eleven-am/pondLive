package runtime

type Effect interface{ isEffect() }

type DOMActionEffect struct {
	Type     string `json:"type"`
	Kind     string `json:"kind"`
	Ref      string `json:"ref"`
	Method   string `json:"method,omitempty"`
	Args     []any  `json:"args,omitempty"`
	Prop     string `json:"prop,omitempty"`
	Value    any    `json:"value,omitempty"`
	Class    string `json:"class,omitempty"`
	On       *bool  `json:"on,omitempty"`
	Behavior string `json:"behavior,omitempty"`
	Block    string `json:"block,omitempty"`
	Inline   string `json:"inline,omitempty"`
}

func (DOMActionEffect) isEffect() {}
