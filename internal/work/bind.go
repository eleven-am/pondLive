package work

import (
	"fmt"
	"time"

	"github.com/eleven-am/pondlive/internal/reactive"
)

type BindKind uint8

const (
	BindString BindKind = iota + 1
	BindBool
	BindStrings
)

func (k BindKind) String() string {
	switch k {
	case BindString:
		return "string"
	case BindBool:
		return "bool"
	case BindStrings:
		return "[]string"
	}
	return "unknown"
}

type BindSpec struct {
	Kind     BindKind
	Debounce time.Duration
	Throttle int
	Lazy     bool

	ConstString  string
	ConstBool    bool
	ConstStrings []string

	ReadString  reactive.Readable[string]
	ReadBool    reactive.Readable[bool]
	ReadStrings reactive.Readable[[]string]

	WriteString  *reactive.Signal[string]
	WriteBool    *reactive.Signal[bool]
	WriteStrings *reactive.Signal[[]string]
}

func (b *BindSpec) Constant() bool {
	return b.ReadString == nil && b.ReadBool == nil && b.ReadStrings == nil
}

func (b *BindSpec) Writable() bool {
	return b.WriteString != nil || b.WriteBool != nil || b.WriteStrings != nil
}

type BindOpt interface{ applyBind(*BindSpec) }

type Lazy struct{}

func (Lazy) applyBind(b *BindSpec) { b.Lazy = true }

type UnsupportedBindError struct{ Type string }

func (e *UnsupportedBindError) Error() string {
	return "work: Bind accepts a string, bool, []string, or a signal of one of them, got " + e.Type
}

func NewBind(v any, opts ...BindOpt) (*BindSpec, error) {
	b := &BindSpec{}
	switch x := v.(type) {
	case string:
		b.Kind, b.ConstString = BindString, x
	case bool:
		b.Kind, b.ConstBool = BindBool, x
	case []string:
		b.Kind, b.ConstStrings = BindStrings, x
	case *reactive.Signal[string]:
		b.Kind, b.ReadString, b.WriteString = BindString, x, x
	case *reactive.Signal[bool]:
		b.Kind, b.ReadBool, b.WriteBool = BindBool, x, x
	case *reactive.Signal[[]string]:
		b.Kind, b.ReadStrings, b.WriteStrings = BindStrings, x, x
	case reactive.Readable[string]:
		b.Kind, b.ReadString = BindString, x
	case reactive.Readable[bool]:
		b.Kind, b.ReadBool = BindBool, x
	case reactive.Readable[[]string]:
		b.Kind, b.ReadStrings = BindStrings, x
	default:
		return nil, &UnsupportedBindError{Type: fmt.Sprintf("%T", v)}
	}
	for _, o := range opts {
		o.applyBind(b)
	}
	return b, nil
}

type bindItem struct{ spec *BindSpec }

func (b bindItem) applyTo(el *Element) { el.Bind = b.spec }

func Bind(v any, opts ...BindOpt) Item {
	spec, err := NewBind(v, opts...)
	if err != nil {
		panic(err)
	}
	return bindItem{spec: spec}
}
