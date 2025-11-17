package router

import (
	h "github.com/eleven-am/pondlive/go/internal/html"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

type routerChildrenProps struct {
	Children []h.Node
}

func renderRouterChildren(ctx Ctx, children ...h.Node) h.Node {
	return runtime.Render(ctx, routerChildrenComponent, routerChildrenProps{Children: children})
}

func routerChildrenComponent(ctx Ctx, props routerChildrenProps) h.Node {
	if len(props.Children) == 0 {
		return h.Fragment()
	}
	sess := ctx.Session()
	normalized := make([]h.Node, 0, len(props.Children))
	for _, child := range props.Children {
		normalized = append(normalized, normalizeRouterNode(ctx, sess, child))
	}
	return h.Fragment(normalized...)
}

func normalizeRouterNode(ctx Ctx, sess *runtime.ComponentSession, node h.Node) h.Node {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *routesNode:
		clearRoutesPlaceholder(sess, v.FragmentNode)
		entries := v.entries
		if len(v.children) > 0 {
			entries = collectRouteEntries(v.children, routeBaseCtx.Use(ctx))
		}
		return normalizeRouterNode(ctx, sess, renderRoutes(ctx, entries))
	case *h.Element:
		if v == nil || len(v.Children) == 0 || v.Unsafe != nil {
			return node
		}
		children := v.Children
		updated := make([]h.Node, len(children))
		changed := false
		for i, child := range children {
			normalized := normalizeRouterNode(ctx, sess, child)
			if normalized != child {
				changed = true
			}
			updated[i] = normalized
		}
		if !changed {
			return node
		}
		clone := *v
		clone.Children = updated
		return &clone
	case *h.FragmentNode:
		if placeholder, ok := consumeRoutesPlaceholder(sess, v); ok {
			entries := placeholder.entries
			if len(placeholder.children) > 0 {
				entries = collectRouteEntries(placeholder.children, routeBaseCtx.Use(ctx))
			}
			return normalizeRouterNode(ctx, sess, renderRoutes(ctx, entries))
		}
		if v == nil || len(v.Children) == 0 {
			return node
		}
		children := v.Children
		updated := make([]h.Node, len(children))
		changed := false
		for i, child := range children {
			normalized := normalizeRouterNode(ctx, sess, child)
			if normalized != child {
				changed = true
			}
			updated[i] = normalized
		}
		if !changed {
			return node
		}
		return h.Fragment(updated...)
	default:
		return node
	}
}
