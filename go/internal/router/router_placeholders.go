package router

import "github.com/eleven-am/pondlive/go/internal/dom2"

func renderRouterChildren(ctx Ctx, children ...*dom2.StructuredNode) *dom2.StructuredNode {
	if len(children) == 0 {
		return dom2.FragmentNode()
	}
	items := make([]dom2.Item, 0, len(children))
	for _, child := range children {
		if child != nil {
			items = append(items, child)
		}
	}
	return dom2.FragmentNode(items...)
}
