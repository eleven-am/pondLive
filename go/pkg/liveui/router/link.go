package router

import (
	ui "github.com/eleven-am/liveui/pkg/liveui"
	h "github.com/eleven-am/liveui/pkg/liveui/html"
)

type LinkProps struct {
	To       string
	Replace  bool
	Children []h.Item
}

func Link(ctx ui.Ctx, p LinkProps) ui.Node {
	state := routerStateCtx.Use(ctx)
	if sessionRendering(ctx.Session()) && state.getLoc != nil {
		return renderLink(ctx, p)
	}
	return &linkNode{FragmentNode: h.Fragment(), props: p}
}

type linkNode struct {
	*h.FragmentNode
	props LinkProps
}

func renderLink(ctx ui.Ctx, p LinkProps) ui.Node {
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
	if len(p.Children) > 0 {
		items = append(items, p.Children...)
	}
	return h.A(items...)
}
