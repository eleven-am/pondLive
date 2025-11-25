package work

import "fmt"

// BuildElement constructs an element with the given tag and items.
func BuildElement(tag string, items ...Item) *Element {
	el := &Element{
		Tag:      tag,
		Attrs:    make(map[string][]string),
		Style:    make(map[string]string),
		Handlers: make(map[string]Handler),
		Children: make([]Node, 0),
	}

	for _, item := range items {
		if item != nil {
			item.ApplyTo(el)
		}
	}

	return el
}

// NewText creates a text node.
func NewText(value string) *Text {
	return &Text{Value: value}
}

// NewTextf creates a formatted text node.
func NewTextf(format string, args ...any) *Text {
	return &Text{Value: fmt.Sprintf(format, args...)}
}

// NewComment creates a comment node.
func NewComment(value string) *Comment {
	return &Comment{Value: value}
}

// NewFragment creates a fragment node from children.
func NewFragment(children ...Node) *Fragment {
	return &Fragment{Children: children}
}
