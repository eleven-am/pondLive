package runtime

import (
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type routerChildrenProps struct {
	Children []h.Node
}

func renderRouterChildren(ctx Ctx, children ...h.Node) h.Node {
	return Render(ctx, routerChildrenComponent, routerChildrenProps{Children: children})
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

// normalizeRouterNodeForSSR is called after component rendering for SSR to normalize any remaining router placeholders
func normalizeRouterNodeForSSR(ctx Ctx, sess *ComponentSession, node h.Node) h.Node {
	return normalizeRouterNode(ctx, sess, node)
}

func normalizeRouterNode(ctx Ctx, sess *ComponentSession, node h.Node) h.Node {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *routesNode:
		sess.clearRoutesPlaceholder(v.placeholder)
		entries := v.entries
		if len(v.children) > 0 {
			entries = collectRouteEntries(v.children, routeBaseCtx.Use(ctx))
		}
		return normalizeRouterNode(ctx, sess, renderRoutes(ctx, entries))
	case *linkNode:
		sess.clearLinkPlaceholder(v.FragmentNode)
		state := routerStateCtx.Use(ctx)
		if sessionRendering(sess) && state.getLoc != nil {
			return renderLink(ctx, v.props, v.children...)
		}
		return renderStaticLink(sess, v.props, v.children...)
	case *h.ComponentNode:

		if v.ID == "" && v.Child != nil {
			if frag, ok := v.Child.(*h.FragmentNode); ok {

				if placeholder, exists := consumeRoutesPlaceholder(sess, frag); exists {
					entries := placeholder.entries
					if len(placeholder.children) > 0 {
						entries = collectRouteEntries(placeholder.children, routeBaseCtx.Use(ctx))
					}
					return normalizeRouterNode(ctx, sess, renderRoutes(ctx, entries))
				}

				if placeholder, exists := consumeLinkPlaceholder(sess, frag); exists {
					state := routerStateCtx.Use(ctx)
					if sessionRendering(sess) && state.getLoc != nil {
						return renderLink(ctx, placeholder.props, placeholder.children...)
					}
					return renderStaticLink(sess, placeholder.props, placeholder.children...)
				}
			}
		}

		if v.Child != nil {
			normalized := normalizeRouterNode(ctx, sess, v.Child)
			if normalized != v.Child {
				clone := *v
				clone.Child = normalized
				return &clone
			}
		}
		return node
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
		if placeholder, ok := consumeLinkPlaceholder(sess, v); ok {
			state := routerStateCtx.Use(ctx)
			if sessionRendering(sess) && state.getLoc != nil {
				return renderLink(ctx, placeholder.props, placeholder.children...)
			}
			return renderStaticLink(sess, placeholder.props, placeholder.children...)
		}
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
