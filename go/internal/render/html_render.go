package render

import (
	"html"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/internal/html"
)

// RenderHTML renders to HTML while registering handlers.
func RenderHTML(n h.Node, reg handlers.Registry) string {
	if n == nil {
		return ""
	}
	ToStructuredWithHandlers(n, reg)
	var b strings.Builder
	renderNode(&b, n)
	return b.String()
}

func renderNode(b *strings.Builder, n h.Node) {
	switch v := n.(type) {
	case *h.TextNode:
		b.WriteString(html.EscapeString(v.Value))
	case *h.Element:
		renderElement(b, v)
	case *h.FragmentNode:
		for _, child := range v.Children {
			if child == nil {
				continue
			}
			renderNode(b, child)
		}
	case *h.CommentNode:
		renderComment(b, v.Value)
	case *h.ComponentNode:
		renderComment(b, h.ComponentStartMarker(v.ID))
		if v.Child != nil {
			renderNode(b, v.Child)
		}
		renderComment(b, h.ComponentEndMarker(v.ID))
	}
}

func renderComment(b *strings.Builder, value string) {
	b.WriteString("<!--")
	b.WriteString(strings.ReplaceAll(value, "--", "- -"))
	b.WriteString("-->")
}

func renderElement(b *strings.Builder, e *h.Element) {
	if e == nil {
		return
	}
	b.WriteByte('<')
	b.WriteString(e.Tag)
	if len(e.Attrs) > 0 {
		keys := make([]string, 0, len(e.Attrs))
		for k := range e.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			v := e.Attrs[k]
			if v == "" {
				continue
			}
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString("=\"")
			b.WriteString(html.EscapeString(v))
			b.WriteString("\"")
		}
	}
	if dom.IsVoidElement(e.Tag) {
		b.WriteByte('>')
		return
	}
	b.WriteByte('>')
	if e.Unsafe != nil {
		b.WriteString(*e.Unsafe)
		b.WriteString("</")
		b.WriteString(e.Tag)
		b.WriteByte('>')
		return
	}
	for _, child := range e.Children {
		if child == nil {
			continue
		}
		renderNode(b, child)
	}
	b.WriteString("</")
	b.WriteString(e.Tag)
	b.WriteByte('>')
}

func renderFinalizedNode(n h.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	renderNode(&b, n)
	return b.String()
}
