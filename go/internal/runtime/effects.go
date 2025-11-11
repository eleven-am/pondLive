package runtime

type Effect interface{ isEffect() }

type Focus struct{ Selector string }

type Toast struct{ Message string }

type Push struct{ URL string }

type Replace struct{ URL string }

type ScrollTop struct{}

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

func (Focus) isEffect()     {}
func (Toast) isEffect()     {}
func (Push) isEffect()      {}
func (Replace) isEffect()   {}
func (ScrollTop) isEffect() {}

func (DOMActionEffect) isEffect() {}
