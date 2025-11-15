package runtime

import (
	"strconv"

	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type LinkProps struct {
	To      string
	Replace bool
}

func RouterLink(ctx Ctx, p LinkProps, children ...h.Item) h.Node {
	state := routerStateCtx.Use(ctx)
	sess := ctx.Session()
	if sessionRendering(sess) && state.getLoc != nil {
		return renderLink(ctx, p, children...)
	}
	return newLinkPlaceholder(sess, p, children)
}

type linkNode struct {
	*h.FragmentNode
	props    LinkProps
	children []h.Item
}

func newLinkPlaceholder(sess *ComponentSession, p LinkProps, children []h.Item) h.Node {
	fragment := h.Fragment()
	node := &linkNode{FragmentNode: fragment, props: p, children: children}
	if sess != nil {
		sess.storeLinkPlaceholder(fragment, node)
	}
	fragment.Children = append(fragment.Children, renderStaticLink(sess, p, children...))
	return fragment
}

func consumeLinkPlaceholder(sess *ComponentSession, f *h.FragmentNode) (*linkNode, bool) {
	if sess == nil {
		return nil, false
	}
	return sess.takeLinkPlaceholder(f)
}

func renderLink(ctx Ctx, p LinkProps, children ...h.Item) h.Node {
	state := requireRouterState(ctx)
	base := state.getLoc()
	target := resolveHref(base, p.To)
	href := BuildHref(target.Path, target.Query, target.Hash)

	handler := func(ev h.Event) h.Updates {
		if ev.Mods.Ctrl || ev.Mods.Meta || ev.Mods.Shift || ev.Mods.Alt || ev.Mods.Button != 0 {
			return nil
		}
		state := requireRouterState(ctx)
		current := state.getLoc()
		next := extractTargetFromEvent(ev, current)
		if next.Path == "" {
			return nil
		}
		replace := p.Replace

		if LocEqual(current, next) {
			return nil
		}
		performLocationUpdate(ctx, next, replace, true)
		return nil
	}

	replaceAttr := strconv.FormatBool(p.Replace)
	encodedQuery := encodeQuery(target.Query)
	items := []h.Item{
		h.Href(href),
		h.OnWith("click", h.EventOptions{Props: []string{
			"currentTarget.href",
		}}, handler),
	}
	if len(children) > 0 {
		items = append(items, children...)
	}
	link := h.A(items...)
	attachRouterMeta(link, dom.RouterMeta{
		Path:    target.Path,
		Query:   encodedQuery,
		Hash:    target.Hash,
		Replace: replaceAttr,
	})
	return link
}

func renderStaticLink(sess *ComponentSession, p LinkProps, children ...h.Item) h.Node {
	href := fallbackLinkHref(sess, p)
	items := make([]h.Item, 0, len(children)+1)
	items = append(items, h.Href(href))
	items = append(items, children...)
	link := h.A(items...)
	target := Location{Path: p.To}
	if sess != nil {
		base := currentSessionLocation(sess)
		resolved := resolveHref(base, p.To)
		target = resolved
	}
	attachRouterMeta(link, dom.RouterMeta{
		Path:    target.Path,
		Query:   encodeQuery(target.Query),
		Hash:    target.Hash,
		Replace: strconv.FormatBool(p.Replace),
	})
	return link
}

func fallbackLinkHref(sess *ComponentSession, p LinkProps) string {
	if sess == nil {
		return p.To
	}
	base := currentSessionLocation(sess)
	target := resolveHref(base, p.To)
	return BuildHref(target.Path, target.Query, target.Hash)
}

func extractTargetFromEvent(ev h.Event, base Location) Location {
	href := h.PayloadString(ev.Payload, "currentTarget.href", "")
	if href == "" {
		return Location{}
	}
	return resolveHref(base, href)
}

func attachRouterMeta(node h.Node, meta dom.RouterMeta) {
	if el, ok := node.(*dom.Element); ok {
		m := meta
		el.RouterMeta = &m
	}
}
