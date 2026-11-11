package router

import (
	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func targetOf(current Location, to string) Location {
	if current.Path == "" {
		current.Path = "/"
	}
	return route.ResolveHref(current, to)
}

func hrefSignal(ctx *runtime.Ctx, r *Router, to string) runtime.ReadSignal[string] {
	return runtime.UseComputed(ctx, func() string {
		return route.BuildHref(targetOf(r.Location.Get(), to))
	})
}

func clickHandler(r *Router, to string, replace bool) work.Item {
	return work.OnWith("click", metadata.EventOptions{Prevent: true}, func(work.Event) {
		if replace {
			r.Replace(to)
			return
		}
		r.Navigate(to)
	})
}

func childItems(node work.Node) []work.Item {
	if node == nil {
		return nil
	}
	return []work.Item{node}
}

var Link = runtime.PropsComponent(func(ctx *runtime.Ctx, props LinkProps) work.Node {
	r := UseRouter(ctx)
	href := hrefSignal(ctx, r, props.To)
	return work.BuildElement("a", append([]work.Item{
		work.Attribute("href", href),
		clickHandler(r, props.To, props.Replace),
	}, childItems(props.Children)...)...)
})

var NavLink = runtime.PropsComponent(func(ctx *runtime.Ctx, props NavLinkProps) work.Node {
	r := UseRouter(ctx)
	href := hrefSignal(ctx, r, props.To)
	active := runtime.UseComputed(ctx, func() bool {
		current := r.Location.Get()
		target := targetOf(current, props.To)
		if props.End {
			return current.Path == target.Path
		}
		return matchesPrefix(current.Path, target.Path)
	})
	ariaCurrent := runtime.UseComputed(ctx, func() string {
		if active.Get() {
			return "page"
		}
		return ""
	})
	activeClass := runtime.UseComputed(ctx, func() string {
		if active.Get() {
			return props.ActiveClass
		}
		return ""
	})
	items := []work.Item{
		work.Attribute("href", href),
		work.Attribute("aria-current", ariaCurrent),
		clickHandler(r, props.To, props.Replace),
	}
	if props.ClassName != "" || props.ActiveClass != "" {
		items = append(items, work.Class(props.ClassName, activeClass))
	}
	return work.BuildElement("a", append(items, childItems(props.Children)...)...)
})

var Redirect = runtime.PropsComponent(func(ctx *runtime.Ctx, props RedirectProps) work.Node {
	r := UseRouter(ctx)
	connected := headers.UseRequest(ctx).Connected()
	apply := func() {
		if props.Push {
			r.Navigate(props.To)
			return
		}
		var opts []headers.NavOption
		if props.Status != 0 {
			opts = append(opts, headers.Status(props.Status))
		}
		r.Replace(props.To, opts...)
	}
	runtime.UseRenderEffect(ctx, func() func() {
		if !connected.Peek() {
			apply()
		}
		return nil
	})
	runtime.UseEffect(ctx, func() func() {
		apply()
		return nil
	})
	return work.NewFragment()
})
