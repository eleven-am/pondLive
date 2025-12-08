package work

func Component(fn any, items ...Item) *ComponentNode {
	children, attrs := splitItems(items)
	comp := &ComponentNode{
		Fn:            fn,
		Props:         nil,
		InputChildren: children,
		InputAttrs:    attrs,
	}
	applyAttrsToComponent(comp, attrs)
	return comp
}

func PropsComponent[P any](fn any, props P, items ...Item) *ComponentNode {
	children, attrs := splitItems(items)
	comp := &ComponentNode{
		Fn:            fn,
		Props:         props,
		InputChildren: children,
		InputAttrs:    attrs,
	}
	applyAttrsToComponent(comp, attrs)
	return comp
}

func (c *ComponentNode) WithKey(key string) *ComponentNode {
	c.Key = key
	return c
}

func splitItems(items []Item) ([]Node, []Item) {
	return SplitItems(items)
}

func SplitItems(items []Item) ([]Node, []Item) {
	var children []Node
	var attrs []Item
	for _, item := range items {
		if node, ok := item.(Node); ok {
			children = append(children, node)
		} else {
			attrs = append(attrs, item)
		}
	}
	return children, attrs
}

func applyAttrsToComponent(comp *ComponentNode, attrs []Item) {
	if len(attrs) == 0 {
		return
	}
	el := &Element{}
	for _, attr := range attrs {
		attr.ApplyTo(el)
	}
	if el.Key != "" {
		comp.Key = el.Key
	}
}
