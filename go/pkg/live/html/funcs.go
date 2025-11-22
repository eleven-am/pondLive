package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
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

	// Props from dom
	Attr   = dom.Attr
	ID     = dom.ID
	Href   = dom.Href
	Src    = dom.Src
	Target = dom.Target
	Rel    = dom.Rel
	Title  = dom.Title
	Alt    = dom.Alt
	Type   = dom.Type
	Value  = dom.Value
	Name   = dom.Name
	Data   = dom.Data
	Aria   = dom.Aria
	Class  = dom.Class
	Style  = dom.Style
	Key    = dom.Key

	// Boolean attributes
	Autofocus   = dom.Autofocus
	Autoplay    = dom.Autoplay
	Checked     = dom.Checked
	Controls    = dom.Controls
	Disabled    = dom.Disabled
	Loop        = dom.Loop
	Multiple    = dom.Multiple
	Muted       = dom.Muted
	Placeholder = dom.Placeholder
	Readonly    = dom.Readonly
	Required    = dom.Required
	Selected    = dom.Selected

	// Events
	On     = dom.On
	OnWith = dom.OnWith

	// State
	Rerender = dom.Rerender
)

// Prop type from dom
type Prop = dom.Prop

// Map renders a slice into a fragment using render.
func Map[T any](xs []T, render func(T) Node) Node {
	return internalhtml.Map(xs, func(t T) dom.Item {
		return render(t)
	})
}

// MapIdx renders a slice with index-aware render function.
func MapIdx[T any](xs []T, render func(int, T) Node) Node {
	return internalhtml.MapIdx(xs, func(i int, t T) dom.Item {
		return render(i, t)
	})
}

// RenderHTML renders a node to HTML string.
func RenderHTML(n Node) string {
	return n.ToHTML()
}

// Attachment is an interface for types that can be attached to elements.
type Attachment = dom.Attachment

// Attach binds an element ref to the element. This is a wrapper around dom.Attach.
// It accepts both raw ElementRef[T] and wrapper refs like *ButtonRef.
func Attach(target Attachment) Prop {
	return dom.Attach(target)
}

// Fragment creates a fragment node.
func Fragment(children ...Item) Node {
	return internalhtml.Fragment(children...)
}
