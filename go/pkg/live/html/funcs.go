package html

import internalhtml "github.com/eleven-am/pondlive/go/internal/html"

var (
	Text                   = internalhtml.Text
	Textf                  = internalhtml.Textf
	Fragment               = internalhtml.Fragment
	Comment                = internalhtml.Comment
	WrapComponent          = internalhtml.WrapComponent
	If                     = internalhtml.If
	IfFn                   = internalhtml.IfFn
	Ternary                = internalhtml.Ternary
	TernaryFn              = internalhtml.TernaryFn
	Attr                   = internalhtml.Attr
	MutableAttr            = internalhtml.MutableAttr
	ID                     = internalhtml.ID
	Href                   = internalhtml.Href
	Src                    = internalhtml.Src
	Target                 = internalhtml.Target
	Rel                    = internalhtml.Rel
	Title                  = internalhtml.Title
	Alt                    = internalhtml.Alt
	Type                   = internalhtml.Type
	Value                  = internalhtml.Value
	Name                   = internalhtml.Name
	Data                   = internalhtml.Data
	Aria                   = internalhtml.Aria
	Class                  = internalhtml.Class
	Style                  = internalhtml.Style
	Key                    = internalhtml.Key
	UnsafeHTML             = internalhtml.UnsafeHTML
	On                     = internalhtml.On
	OnWith                 = internalhtml.OnWith
	Rerender               = internalhtml.Rerender
	RenderHTML             = internalhtml.RenderHTML
	ComponentStartMarker   = internalhtml.ComponentStartMarker
	ComponentEndMarker     = internalhtml.ComponentEndMarker
	ComponentCommentPrefix = internalhtml.ComponentCommentPrefix
	MetaTags               = internalhtml.MetaTags
	LinkTags               = internalhtml.LinkTags
	ScriptTags             = internalhtml.ScriptTags
	PayloadString          = internalhtml.PayloadString
)

func Map[T any](xs []T, render func(T) Node) Item {
	return internalhtml.Map(xs, render)
}

func MapIdx[T any](xs []T, render func(int, T) Node) Item {
	return internalhtml.MapIdx(xs, render)
}

func NewElementRef[T ElementDescriptor](id string, descriptor T) *ElementRef[T] {
	return internalhtml.NewElementRef(id, descriptor)
}

func Attach[T ElementDescriptor, Target internalhtml.AttachTarget[T]](target Target) Prop {
	return internalhtml.Attach[T](target)
}
