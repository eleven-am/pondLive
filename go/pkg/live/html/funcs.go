package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
	internalhtml "github.com/eleven-am/pondlive/go/internal/html"
)

var (
	// Core builders from internal/html
	Text          = internalhtml.Text
	Textf         = internalhtml.Textf
	Comment       = internalhtml.Comment
	WrapComponent = internalhtml.WrapComponent
	If            = internalhtml.If
	IfFn          = internalhtml.IfFn
	Ternary       = internalhtml.Ternary
	TernaryFn     = internalhtml.TernaryFn

	// Props from dom2
	Attr   = dom2.Attr
	ID     = dom2.ID
	Href   = dom2.Href
	Src    = dom2.Src
	Target = dom2.Target
	Rel    = dom2.Rel
	Title  = dom2.Title
	Alt    = dom2.Alt
	Type   = dom2.Type
	Value  = dom2.Value
	Name   = dom2.Name
	Data   = dom2.Data
	Aria   = dom2.Aria
	Class  = dom2.Class
	Style  = dom2.Style
	Key    = dom2.Key
	Upload = dom2.Upload

	// Event and state
	Rerender = dom2.Rerender
)

// Prop type from dom2
type Prop = dom2.Prop

// Map renders a slice into a fragment using render.
func Map[T any](xs []T, render func(T) Node) Node {
	return internalhtml.Map(xs, func(t T) dom2.Item {
		return render(t)
	})
}

// MapIdx renders a slice with index-aware render function.
func MapIdx[T any](xs []T, render func(int, T) Node) Node {
	return internalhtml.MapIdx(xs, func(i int, t T) dom2.Item {
		return render(i, t)
	})
}

// RenderHTML renders a node to HTML string.
func RenderHTML(n Node) string {
	return n.ToHTML()
}

// Attach binds an element ref to the element. This is a wrapper around dom2.Attach.
func Attach[T dom2.ElementDescriptor](ref *dom2.ElementRef[T]) Prop {
	return dom2.Attach(ref)
}

// Fragment creates a fragment node.
func Fragment(children ...Item) Node {
	return internalhtml.Fragment(children...)
}
