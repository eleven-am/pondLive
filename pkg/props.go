package pkg

import (
	"fmt"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type AttrOpt = work.AttrOpt

func OmitEmpty() AttrOpt { return work.OmitEmpty() }

func Attr[T any](name string, v T, opts ...AttrOpt) Item { return work.Attribute(name, v, opts...) }

func ID[T any](v T) Item          { return Attr("id", v) }
func Href[T any](v T) Item        { return Attr("href", v) }
func Src[T any](v T) Item         { return Attr("src", v) }
func Target[T any](v T) Item      { return Attr("target", v) }
func Rel[T any](v T) Item         { return Attr("rel", v) }
func Title[T any](v T) Item       { return Attr("title", v) }
func Alt[T any](v T) Item         { return Attr("alt", v) }
func Type[T any](v T) Item        { return Attr("type", v) }
func Name[T any](v T) Item        { return Attr("name", v) }
func Placeholder[T any](v T) Item { return Attr("placeholder", v) }
func Data[T any](k string, v T) Item {
	return Attr("data-"+k, v)
}
func Aria[T any](k string, v T) Item {
	return Attr("aria-"+k, v)
}

func Class(parts ...any) Item { return work.Class(parts...) }

func Style[T any](property string, v T) Item { return work.Style(property, v) }
func Styles(styles map[string]any) Item      { return work.Styles(styles) }

func On(event string, fn func(Event)) Item {
	return work.On(event, fn)
}

func OnWith(event string, options EventOptions, fn func(Event)) Item {
	return work.OnWith(event, options, fn)
}

type Attachable interface {
	*ElementRef | *InputRef | *FormRef | *MediaRef | *DialogRef | *CanvasRef | *DetailsRef | Script | *UploadHandle
}

func Attach[T Attachable](target T) Item {
	switch t := any(target).(type) {
	case *ElementRef:
		return runtime.AttachItem(t.ref)
	case *InputRef:
		return runtime.AttachItem(t.ref)
	case *FormRef:
		return runtime.AttachItem(t.ref)
	case *MediaRef:
		return runtime.AttachItem(t.ref)
	case *DialogRef:
		return runtime.AttachItem(t.ref)
	case *CanvasRef:
		return runtime.AttachItem(t.ref)
	case *DetailsRef:
		return runtime.AttachItem(t.ref)
	}
	return runtime.AttachItem(target)
}

func boolAttr[T any](name string, v T) Item {
	switch any(v).(type) {
	case bool, ReadSignal[bool]:
		return Attr(name, v)
	}
	panic(&work.UnsupportedValueError{Type: fmt.Sprintf("%T", v)})
}

func Disabled[T any](v T) Item  { return boolAttr("disabled", v) }
func Required[T any](v T) Item  { return boolAttr("required", v) }
func Readonly[T any](v T) Item  { return boolAttr("readonly", v) }
func Autofocus[T any](v T) Item { return boolAttr("autofocus", v) }
func Autoplay[T any](v T) Item  { return boolAttr("autoplay", v) }
func Controls[T any](v T) Item  { return boolAttr("controls", v) }
func Loop[T any](v T) Item      { return boolAttr("loop", v) }
func Muted[T any](v T) Item     { return boolAttr("muted", v) }
func Multiple[T any](v T) Item  { return boolAttr("multiple", v) }

func UnsafeHTML[T any](v T) Item { return work.UnsafeHTML(v) }
