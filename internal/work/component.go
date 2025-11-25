package work

// NewComponent creates a component node with no props, only children.
// fn should be func(ctx, []Node) Node
func NewComponent(fn any, children ...Node) *Component {
	return &Component{
		Fn:            fn,
		Props:         nil,
		InputChildren: children,
	}
}

// NewPropsComponent creates a component node with props and children.
// fn should be func(ctx, P, []Node) Node where P is the props type
func NewPropsComponent[P any](fn any, props P, children ...Node) *Component {
	return &Component{
		Fn:            fn,
		Props:         props,
		InputChildren: children,
	}
}

// WithKey sets the reconciliation key on a component.
func (c *Component) WithKey(key string) *Component {
	c.Key = key
	return c
}
