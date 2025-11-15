package router2

import "github.com/eleven-am/pondlive/go/internal/dom"

// resolvableNode represents a placeholder node that must be resolved using a
// RouterStore before rendering.
type resolvableNode interface {
	dom.Node
	resolve(*RouterStore) dom.Node
}

// ResolveTree walks the provided DOM node and resolves any router placeholders
// using the supplied store.
func ResolveTree(node dom.Node, store *RouterStore) dom.Node {
	if node == nil || store == nil {
		if node == nil {
			return &dom.FragmentNode{}
		}
		return node
	}
	switch v := node.(type) {
	case resolvableNode:
		return ResolveTree(v.resolve(store), store)
	case *dom.Element:
		clone := *v
		if len(v.Children) > 0 {
			clone.Children = make([]dom.Node, len(v.Children))
			for i, child := range v.Children {
				clone.Children[i] = ResolveTree(child, store)
			}
		}
		return &clone
	case *dom.FragmentNode:
		if len(v.Children) == 0 {
			return v
		}
		children := make([]dom.Node, len(v.Children))
		for i, child := range v.Children {
			children[i] = ResolveTree(child, store)
		}
		return &dom.FragmentNode{Children: children}
	case *dom.ComponentNode:
		clone := *v
		if v.Child != nil {
			clone.Child = ResolveTree(v.Child, store)
		}
		return &clone
	case *dom.TextNode, *dom.CommentNode:
		return node
	default:
		return node
	}
}
