package work

func Component(fn any, children ...Node) *ComponentNode {
	return &ComponentNode{
		Fn:            fn,
		Props:         nil,
		InputChildren: children,
	}
}

func PropsComponent[P any](fn any, props P, children ...Node) *ComponentNode {
	return &ComponentNode{
		Fn:            fn,
		Props:         props,
		InputChildren: children,
	}
}

func (c *ComponentNode) WithKey(key string) *ComponentNode {
	c.Key = key
	return c
}
