package dom

import (
	"html"
	"sort"
	"strings"
)

// ToHTML renders the StructuredNode to HTML string.
// Component boundaries (ComponentID nodes) are transparent in the output.
func (n *StructuredNode) ToHTML() string {
	if n == nil {
		return ""
	}

	if n.Text != "" {
		return html.EscapeString(n.Text)
	}

	if n.Comment != "" {
		return "<!-- " + html.EscapeString(n.Comment) + " -->"
	}

	if n.Fragment {
		var b strings.Builder
		for _, child := range n.Children {
			b.WriteString(child.ToHTML())
		}
		return b.String()
	}

	if n.ComponentID != "" {
		var b strings.Builder
		for _, child := range n.Children {
			b.WriteString(child.ToHTML())
		}
		return b.String()
	}

	if n.Tag != "" {
		return n.renderElement()
	}

	return ""
}

// renderElement renders an HTML element with all its attributes and children
func (n *StructuredNode) renderElement() string {
	var b strings.Builder

	b.WriteString("<")
	b.WriteString(n.Tag)

	if len(n.Attrs) > 0 {
		keys := make([]string, 0, len(n.Attrs))
		for k := range n.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, key := range keys {
			tokens := n.Attrs[key]
			if len(tokens) == 0 {
				continue
			}
			b.WriteString(" ")
			b.WriteString(key)
			b.WriteString(`="`)
			b.WriteString(html.EscapeString(strings.Join(tokens, " ")))
			b.WriteString(`"`)
		}
	}

	if len(n.Style) > 0 {
		b.WriteString(` style="`)
		b.WriteString(serializeInlineStyles(n.Style))
		b.WriteString(`"`)
	}

	b.WriteString(">")

	if n.UnsafeHTML != "" {
		b.WriteString(n.UnsafeHTML)
	} else if n.Tag == "style" && n.Stylesheet != nil {
		b.WriteString(serializeStylesheet(n.Stylesheet))
	} else {
		for _, child := range n.Children {
			b.WriteString(child.ToHTML())
		}
	}

	b.WriteString("</")
	b.WriteString(n.Tag)
	b.WriteString(">")

	return b.String()
}

// serializeInlineStyles converts a Style map to CSS inline style string
// Example: {"color": "red", "font-size": "14px"} -> "color: red; font-size: 14px"
func serializeInlineStyles(style map[string]string) string {
	if len(style) == 0 {
		return ""
	}

	keys := make([]string, 0, len(style))
	for k := range style {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, key := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(html.EscapeString(style[key]))
	}

	return b.String()
}

// serializeStylesheet converts a Stylesheet to CSS text
func serializeStylesheet(ss *Stylesheet) string {
	if ss == nil {
		return ""
	}

	var b strings.Builder

	for _, rule := range ss.Rules {
		b.WriteString(rule.Selector)
		b.WriteString(" { ")
		b.WriteString(serializeProps(rule.Props))
		b.WriteString(" }\n")
	}

	for _, media := range ss.MediaBlocks {
		b.WriteString("@media ")
		b.WriteString(media.Query)
		b.WriteString(" {\n")
		for _, rule := range media.Rules {
			b.WriteString("  ")
			b.WriteString(rule.Selector)
			b.WriteString(" { ")
			b.WriteString(serializeProps(rule.Props))
			b.WriteString(" }\n")
		}
		b.WriteString("}\n")
	}

	return b.String()
}

// serializeProps converts a property map to CSS declarations
func serializeProps(props map[string]string) string {
	if len(props) == 0 {
		return ""
	}

	keys := make([]string, 0, len(props))
	for k := range props {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for i, key := range keys {
		if i > 0 {
			b.WriteString("; ")
		}
		b.WriteString(key)
		b.WriteString(": ")
		b.WriteString(props[key])
	}
	return b.String()
}

// TextNode creates a text node
func TextNode(text string) *StructuredNode {
	return &StructuredNode{Text: text}
}

// CommentNode creates an HTML comment node.
func CommentNode(text string) *StructuredNode {
	return &StructuredNode{Comment: text}
}

// ElementNode creates an element node with the given tag
func ElementNode(tag string) *StructuredNode {
	return &StructuredNode{Tag: tag}
}

// ComponentNode creates a component boundary node
func ComponentNode(id string) *StructuredNode {
	return &StructuredNode{ComponentID: id}
}

// FragmentNode creates a container for multiple children without a wrapper element.
// Fragments are transparent in the rendered output - only their children appear.
func FragmentNode(children ...Item) *StructuredNode {
	node := &StructuredNode{Fragment: true}
	for _, child := range children {
		if child != nil {
			child.ApplyTo(node)
		}
	}
	return node
}

// El constructs an Element with the provided descriptor, tag name, and content.
func El(desc ElementDescriptor, items ...Item) *StructuredNode {
	e := &StructuredNode{Tag: desc.TagName(), Descriptor: desc}
	for _, it := range items {
		if it != nil {
			it.ApplyTo(e)
		}
	}
	return e
}

// WithChildren adds children to a node (builder pattern helper)
func (n *StructuredNode) WithChildren(children ...*StructuredNode) *StructuredNode {
	for _, child := range children {
		if child != nil {
			n.Children = append(n.Children, child)
		}
	}
	return n
}

// WithAttr adds an attribute to a node (builder pattern helper)
func (n *StructuredNode) WithAttr(key string, tokens ...string) *StructuredNode {
	if n.Attrs == nil {
		n.Attrs = make(map[string][]string)
	}
	n.Attrs[key] = tokens
	return n
}

// WithStyle adds an inline style property to a node (builder pattern helper)
func (n *StructuredNode) WithStyle(key, value string) *StructuredNode {
	if n.Style == nil {
		n.Style = make(map[string]string)
	}
	n.Style[key] = value
	return n
}

// WithKey sets the key for stable diffing (builder pattern helper)
func (n *StructuredNode) WithKey(key string) *StructuredNode {
	n.Key = key
	return n
}

// WithRef sets the ref ID for element references (builder pattern helper)
func (n *StructuredNode) WithRef(refID string) *StructuredNode {
	n.RefID = refID
	return n
}
