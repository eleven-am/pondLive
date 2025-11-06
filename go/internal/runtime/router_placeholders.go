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

func normalizeRouterNode(ctx Ctx, sess *ComponentSession, node h.Node) h.Node {
	if node == nil {
		return nil
	}
	switch v := node.(type) {
	case *routesNode:
		removeRoutesPlaceholder(sess, v.FragmentNode)
		return normalizeRouterNode(ctx, sess, renderRoutes(ctx, v.entries))
	case *linkNode:
		removeLinkPlaceholder(sess, v.FragmentNode)
		return renderLink(ctx, v.props, v.children...)
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
			return renderLink(ctx, placeholder.props, placeholder.children...)
		}
		if placeholder, ok := consumeRoutesPlaceholder(sess, v); ok {
			return normalizeRouterNode(ctx, sess, renderRoutes(ctx, placeholder.entries))
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

func removeLinkPlaceholder(sess *ComponentSession, frag *h.FragmentNode) {
	if sess == nil || frag == nil {
		return
	}
	if state := sess.loadRouterState(); state != nil {
		state.linkPlaceholders.Delete(frag)
	}
}

func removeRoutesPlaceholder(sess *ComponentSession, frag *h.FragmentNode) {
	if sess == nil || frag == nil {
		return
	}
	if state := sess.loadRouterState(); state != nil {
		state.routesPlaceholders.Delete(frag)
	}
}
