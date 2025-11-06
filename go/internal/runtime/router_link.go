package runtime

import (
	"sync"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type LinkProps struct {
	To      string
	Replace bool
}

func RouterLink(ctx Ctx, p LinkProps, children ...h.Item) h.Node {
	state := routerStateCtx.Use(ctx)
	if sessionRendering(ctx.Session()) && state.getLoc != nil {
		return renderLink(ctx, p, children...)
	}
	return newLinkPlaceholder(p, children)
}

type linkNode struct {
	*h.FragmentNode
	props    LinkProps
	children []h.Item
}

var linkPlaceholders sync.Map // *h.FragmentNode -> *linkNode

func newLinkPlaceholder(p LinkProps, children []h.Item) *linkNode {
	node := &linkNode{FragmentNode: h.Fragment(), props: p, children: children}
	linkPlaceholders.Store(node.FragmentNode, node)
	return node
}

func consumeLinkPlaceholder(f *h.FragmentNode) (*linkNode, bool) {
	if f == nil {
		return nil, false
	}
	if value, ok := linkPlaceholders.LoadAndDelete(f); ok {
		if node, okCast := value.(*linkNode); okCast {
			return node, true
		}
	}
	return nil, false
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
		if LocEqual(state.getLoc(), target) {
			return nil
		}
		if p.Replace {
			performLocationUpdate(ctx, target, true, true)
		} else {
			performLocationUpdate(ctx, target, false, true)
		}
		return nil
	}

	items := []h.Item{h.Href(href), h.On("click", handler)}
	if len(children) > 0 {
		items = append(items, children...)
	}
	return h.A(items...)
}
