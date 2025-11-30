package view

import (
	"html"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/go/internal/metadata"
)

var voidElements = map[string]bool{
	"area": true, "base": true, "br": true, "col": true,
	"embed": true, "hr": true, "img": true, "input": true,
	"link": true, "meta": true, "param": true, "source": true,
	"track": true, "wbr": true,
}

func RenderHTML(n Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	renderNode(&b, n)
	return b.String()
}

func renderNode(b *strings.Builder, n Node) {
	if n == nil {
		return
	}

	switch node := n.(type) {
	case *Element:
		renderElement(b, node)
	case *Text:
		b.WriteString(html.EscapeString(node.Text))
	case *Comment:
		b.WriteString("<!--")
		b.WriteString(html.EscapeString(node.Comment))
		b.WriteString("-->")
	case *Fragment:
		for _, child := range node.Children {
			renderNode(b, child)
		}
	}
}

func renderElement(b *strings.Builder, el *Element) {
	if el.Tag == "html" {
		b.WriteString("<!DOCTYPE html>")
	}
	b.WriteByte('<')
	b.WriteString(el.Tag)

	if len(el.Attrs) > 0 {
		keys := make([]string, 0, len(el.Attrs))
		for k := range el.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			values := el.Attrs[k]
			if len(values) == 0 {

				b.WriteByte(' ')
				b.WriteString(k)
				continue
			}
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString(`="`)
			b.WriteString(html.EscapeString(strings.Join(values, " ")))
			b.WriteByte('"')
		}
	}

	if len(el.Style) > 0 {
		keys := make([]string, 0, len(el.Style))
		for k := range el.Style {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		b.WriteString(` style="`)
		for i, k := range keys {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(html.EscapeString(el.Style[k]))
		}
		b.WriteByte('"')
	}

	b.WriteByte('>')

	if voidElements[el.Tag] {
		return
	}

	if el.UnsafeHTML != "" {
		b.WriteString(el.UnsafeHTML)
	} else {
		for _, child := range el.Children {
			renderNode(b, child)
		}
	}

	if el.Stylesheet != nil && (len(el.Stylesheet.Rules) > 0 || len(el.Stylesheet.MediaBlocks) > 0) {
		renderStylesheet(b, el.Stylesheet)
	}

	b.WriteString("</")
	b.WriteString(el.Tag)
	b.WriteByte('>')
}

func renderStylesheet(b *strings.Builder, ss *metadata.Stylesheet) {
	if ss == nil {
		return
	}

	b.WriteString("<style>")

	for _, rule := range ss.Rules {
		b.WriteString(escapeCSSForHTML(rule.Selector))
		b.WriteString("{")
		writeProps(b, rule.Props)
		b.WriteString("}")
	}

	for _, media := range ss.MediaBlocks {
		b.WriteString("@media ")
		b.WriteString(escapeCSSForHTML(media.Query))
		b.WriteString("{")
		for _, rule := range media.Rules {
			b.WriteString(escapeCSSForHTML(rule.Selector))
			b.WriteString("{")
			writeProps(b, rule.Props)
			b.WriteString("}")
		}
		b.WriteString("}")
	}

	b.WriteString("</style>")
}

func escapeCSSForHTML(s string) string {
	return strings.ReplaceAll(s, "</", "<\\/")
}

func writeProps(b *strings.Builder, props map[string]string) {
	if len(props) == 0 {
		return
	}

	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for i, k := range keys {
		if i > 0 {
			b.WriteByte(';')
		}
		b.WriteString(escapeCSSForHTML(k))
		b.WriteByte(':')
		b.WriteString(escapeCSSForHTML(props[k]))
	}
}
