package work

import "fmt"

func BuildElement(tag string, items ...Item) *Element {
	el := &Element{Tag: tag}
	if len(items) > 0 {
		el.Children = make([]Node, 0, len(items))
	}
	for _, item := range items {
		if item != nil {
			item.applyTo(el)
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

func NewFragment(items ...Item) *Fragment {
	children, attrs := SplitItems(items)
	return &Fragment{Children: children, Attrs: attrs}
}

func SplitItems(items []Item) ([]Node, []Item) {
	var children []Node
	var attrs []Item
	for _, item := range items {
		if node, ok := item.(Node); ok {
			children = append(children, node)
		} else if item != nil {
			attrs = append(attrs, item)
		}
	}
	return children, attrs
}

func NewPortal(children ...Item) *PortalNode {
	return &PortalNode{Children: ItemsToNodes(children)}
}

func NewPortalTarget() *PortalTarget { return &PortalTarget{} }
