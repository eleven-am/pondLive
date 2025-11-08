package runtime

type Effect interface{ isEffect() }

type Focus struct{ Selector string }

type Toast struct{ Message string }

type Push struct{ URL string }

type Replace struct{ URL string }

type ScrollTop struct{}

type DOMCallEffect struct {
	Type   string `json:"type"`
	Ref    string `json:"ref"`
	Method string `json:"method"`
	Args   []any  `json:"args,omitempty"`
}

func (Focus) isEffect()     {}
func (Toast) isEffect()     {}
func (Push) isEffect()      {}
func (Replace) isEffect()   {}
func (ScrollTop) isEffect() {}

func (DOMCallEffect) isEffect() {}
