package work

import "fmt"

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

func NewText(value string) *Text {
	return &Text{Value: value}
}

func NewTextf(format string, args ...any) *Text {
	return &Text{Value: fmt.Sprintf(format, args...)}
}

func NewComment(value string) *Comment {
	return &Comment{Value: value}
}

func NewFragment(children ...Node) *Fragment {
	return &Fragment{Children: children}
}
