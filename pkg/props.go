package pkg

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/work"
)

func Attr(name, value string) Item { return work.Attr(name, value) }

func ID(id string) Item         { return work.ID(id) }
func Href(url string) Item      { return work.Href(url) }
func Src(path string) Item      { return work.Src(path) }
func Target(v string) Item      { return work.Target(v) }
func Rel(v string) Item         { return work.Rel(v) }
func Title(v string) Item       { return work.Title(v) }
func Alt(v string) Item         { return work.Alt(v) }
func Type(v string) Item        { return work.Type(v) }
func Value(v string) Item       { return work.Value(v) }
func Name(v string) Item        { return work.Name(v) }
func Placeholder(v string) Item { return work.Placeholder(v) }
func Data(k, v string) Item     { return work.Data(k, v) }
func Aria(k, v string) Item     { return work.Aria(k, v) }

func Class(classes ...string) Item { return work.Class(classes...) }

func Style(property, value string) Item          { return work.Style(property, value) }
func InlineStyles(styles map[string]string) Item { return work.Styles(styles) }

func Key(key string) Item { return work.Key(key) }

func On(event string, fn func(work.Event) work.Updates) Item {
	return work.On(event, fn)
}

func OnWith(event string, options metadata.EventOptions, fn func(work.Event) work.Updates) Item {
	return work.OnWith(event, options, fn)
}

func Attach(ref work.Attachment) Item { return work.Attach(ref) }

func Disabled() Item  { return work.Disabled() }
func Checked() Item   { return work.Checked() }
func Required() Item  { return work.Required() }
func Readonly() Item  { return work.Readonly() }
func Autofocus() Item { return work.Autofocus() }
func Autoplay() Item  { return work.Autoplay() }
func Controls() Item  { return work.Controls() }
func Loop() Item      { return work.Loop() }
func Muted() Item     { return work.Muted() }
func Selected() Item  { return work.Selected() }
func Multiple() Item  { return work.Multiple() }

func UnsafeHTML(html string) Item { return work.UnsafeHTML(html) }
