package dom

import (
	"html"
	"sort"
	"strings"
)

const componentCommentPrefix = "live-component"

var voidElements = map[string]struct{}{
	"area": {}, "base": {}, "br": {}, "col": {}, "embed": {}, "hr": {}, "img": {}, "input": {},
	"link": {}, "meta": {}, "param": {}, "source": {}, "track": {}, "wbr": {},
}

// IsVoidElement reports whether the provided tag corresponds to an element that
// cannot contain children according to the HTML specification.
func IsVoidElement(tag string) bool {
	if tag == "" {
		return false
	}
	_, ok := voidElements[strings.ToLower(tag)]
	return ok
}

// FinalizeElement normalizes a node tree by folding metadata into attributes.
func FinalizeElement(e *Element) {
	if e == nil {
		return
	}
	if _, ok := voidElements[strings.ToLower(e.Tag)]; ok {
		e.Children = nil
		e.Unsafe = nil
	}
	if len(e.Class) > 0 {
		existing := ""
		if e.Attrs != nil {
			existing = e.Attrs["class"]
		}
		joined := JoinClasses(existing, e.Class)
		if joined != "" {
			if e.Attrs == nil {
				e.Attrs = map[string]string{}
			}
			e.Attrs["class"] = joined
		}
		e.Class = nil
	}
	if len(e.Style) > 0 {
		existing := ""
		if e.Attrs != nil {
			existing = e.Attrs["style"]
		}
		joined := JoinStyles(e.Style, existing)
		if joined != "" {
			if e.Attrs == nil {
				e.Attrs = map[string]string{}
			}
			e.Attrs["style"] = joined
		}
		e.Style = nil
	}
	if e.Unsafe != nil {
		e.Children = nil
	}
}

// JoinClasses merges existing class attributes with builder-provided tokens.
func JoinClasses(existing string, classes []string) string {
	seen := map[string]struct{}{}
	ordered := make([]string, 0, len(classes))
	for _, token := range strings.Fields(existing) {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		ordered = append(ordered, token)
	}
	for _, c := range classes {
		for _, token := range strings.Fields(c) {
			token = strings.TrimSpace(token)
			if token == "" {
				continue
			}
			if _, ok := seen[token]; ok {
				continue
			}
			seen[token] = struct{}{}
			ordered = append(ordered, token)
		}
	}
	return strings.Join(ordered, " ")
}

// JoinStyles merges style declarations from builder props and existing attrs.
func JoinStyles(styleMap map[string]string, existing string) string {
	if len(styleMap) == 0 && strings.TrimSpace(existing) == "" {
		return ""
	}
	merged := map[string]string{}
	if existing != "" {
		for _, decl := range strings.Split(existing, ";") {
			decl = strings.TrimSpace(decl)
			if decl == "" {
				continue
			}
			parts := strings.SplitN(decl, ":", 2)
			key := strings.TrimSpace(parts[0])
			if key == "" {
				continue
			}
			val := ""
			if len(parts) > 1 {
				val = strings.TrimSpace(parts[1])
			}
			merged[key] = val
		}
	}
	for k, v := range styleMap {
		if strings.TrimSpace(k) == "" {
			continue
		}
		merged[k] = v
	}
	if len(merged) == 0 {
		return ""
	}
	keys := make([]string, 0, len(merged))
	for k := range merged {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var b strings.Builder
	for _, k := range keys {
		b.WriteString(k)
		b.WriteString(":")
		b.WriteString(merged[k])
		b.WriteString(";")
	}
	return b.String()
}

// EscapeComment sanitizes HTML comments by preventing closing markers.
func EscapeComment(value string) string {
	return strings.ReplaceAll(value, "--", "- -")
}

// ComponentStartComment returns the comment marker used for component starts.
func ComponentStartComment(id string) string {
	return componentCommentPrefix + ":start:" + id
}

// ComponentEndComment returns the comment marker used for component ends.
func ComponentEndComment(id string) string {
	return componentCommentPrefix + ":end:" + id
}

// ComponentCommentPrefix exposes the prefix used to mark component boundaries.
func ComponentCommentPrefix() string { return componentCommentPrefix }

// RenderHTML renders a Node tree into an escaped HTML string.
func RenderHTML(n Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	renderNode(&b, n)
	return b.String()
}

func renderNode(b *strings.Builder, n Node) {
	switch v := n.(type) {
	case *TextNode:
		b.WriteString(html.EscapeString(v.Value))
	case *Element:
		renderElement(b, v)
	case *FragmentNode:
		for _, child := range v.Children {
			if child == nil {
				continue
			}
			renderNode(b, child)
		}
	case *CommentNode:
		renderComment(b, v.Value)
	case *ComponentNode:
		renderComment(b, ComponentStartComment(v.ID))
		if v.Child != nil {
			renderNode(b, v.Child)
		}
		renderComment(b, ComponentEndComment(v.ID))
	}
}

func renderComment(b *strings.Builder, value string) {
	b.WriteString("<!--")
	b.WriteString(EscapeComment(value))
	b.WriteString("-->")
}

func renderElement(b *strings.Builder, e *Element) {
	if e == nil {
		return
	}
	FinalizeElement(e)
	b.WriteByte('<')
	b.WriteString(e.Tag)
	attrKeys := make([]string, 0, len(e.Attrs))
	for k := range e.Attrs {
		attrKeys = append(attrKeys, k)
	}
	sort.Strings(attrKeys)
	for _, k := range attrKeys {
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
	b.WriteByte('>')
	if _, ok := voidElements[strings.ToLower(e.Tag)]; ok {
		return
	}
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
