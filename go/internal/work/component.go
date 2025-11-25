package work

// Component creates a component node with no props, only children.
// fn should be func(ctx, []Node) Node
func Component(fn any, children ...Node) *ComponentNode {
	return &ComponentNode{
		Fn:            fn,
		Props:         nil,
		InputChildren: children,
	}
}

// PropsComponent creates a component node with props and children.
// fn should be func(ctx, P, []Node) Node where P is the props type
func PropsComponent[P any](fn any, props P, children ...Node) *ComponentNode {
	return &ComponentNode{
		Fn:            fn,
		Props:         props,
		InputChildren: children,
	}
}

// WithKey sets the reconciliation key on a component.
func (c *ComponentNode) WithKey(key string) *ComponentNode {
	c.Key = key
	return c
}
