package render

import (
	"html"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func RenderHTML(n h.Node) string {
	if n == nil {
		return ""
	}

	n = normalizeForSSR(n)
	var b strings.Builder
	renderNode(&b, n)
	return b.String()
}

// normalizeForSSR performs router-specific normalization for SSR.
// This is needed because router placeholders must be resolved before rendering.
func normalizeForSSR(n h.Node) h.Node {
	if n == nil {
		return nil
	}

	if comp, ok := n.(*h.ComponentNode); ok {
		if comp.Child != nil {
			comp.Child = normalizeForSSR(comp.Child)
		}
		return n
	}

	if elem, ok := n.(*h.Element); ok {
		if len(elem.Children) == 0 {
			return n
		}
		changed := false
		updated := make([]h.Node, len(elem.Children))
		for i, child := range elem.Children {
			normalized := normalizeForSSR(child)
			if normalized != child {
				changed = true
			}
			updated[i] = normalized
		}
		if !changed {
			return n
		}
		clone := *elem
		clone.Children = updated
		return &clone
	}

	if frag, ok := n.(*h.FragmentNode); ok {
		if len(frag.Children) == 0 {
			return n
		}
		changed := false
		updated := make([]h.Node, len(frag.Children))
		for i, child := range frag.Children {
			normalized := normalizeForSSR(child)
			if normalized != child {
				changed = true
			}
			updated[i] = normalized
		}
		if !changed {
			return n
		}
		return h.Fragment(updated...)
	}
	return n
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
		if v.Child != nil {
			renderNode(b, v.Child)
		}
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
	if len(e.Class) > 0 {
		b.WriteString(" class=\"")
		for i, class := range e.Class {
			if i > 0 {
				b.WriteByte(' ')
			}
			b.WriteString(html.EscapeString(class))
		}
		b.WriteByte('"')
	}
	if len(e.Style) > 0 {
		b.WriteString(" style=\"")
		keys := make([]string, 0, len(e.Style))
		for k := range e.Style {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for i, k := range keys {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(html.EscapeString(k))
			b.WriteString(": ")
			b.WriteString(html.EscapeString(e.Style[k]))
		}
		b.WriteByte('"')
	}
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
