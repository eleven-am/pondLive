package router

import "github.com/eleven-am/pondlive/go/internal/dom"

func renderRouterChildren(ctx Ctx, children ...*dom.StructuredNode) *dom.StructuredNode {
	if len(children) == 0 {
		return dom.FragmentNode()
	}
	items := make([]dom.Item, 0, len(children))
	for _, child := range children {
		if child != nil {
			items = append(items, child)
		}
	}
	return dom.FragmentNode(items...)
}
