package router2

import (
	h "github.com/eleven-am/pondlive/go/pkg/live/html"

	"github.com/eleven-am/pondlive/go/internal/dom"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

var (
	routerStoreCtx = runtime.NewContext((*RouterStore)(nil))
	routeBaseCtx   = runtime.NewContext("/")
)

type routerProps struct {
	Children []h.Node
}

// Router provides router state to its descendants and resolves all router2
// placeholders before returning the rendered node tree.
func Router(ctx runtime.Ctx, children ...h.Node) h.Node {
	return runtime.Render(ctx, routerComponent, routerProps{Children: children})
}

func routerComponent(ctx runtime.Ctx, props routerProps) h.Node {
	store := useRouterStore(ctx)
	return routerStoreCtx.Provide(ctx, store, func() h.Node {
		resolved := make([]dom.Node, 0, len(props.Children))
		for _, child := range props.Children {
			if child == nil {
				continue
			}
			resolved = append(resolved, ResolveTree(child, store))
		}
		if len(resolved) == 0 {
			return h.Fragment()
		}
		return h.Fragment(resolved...)
	})
}

func useRouterStore(ctx runtime.Ctx) *RouterStore {
	getter, setter := runtime.UseState(ctx, (*RouterStore)(nil))
	store := getter()
	if store == nil {
		store = NewStore(Location{Path: "/"})
		setter(store)
	}
	return store
}

// Routes constructs a resolvable placeholder using the provided route
// definitions. Nested Routes will render child definitions relative to the
// currently active base path.
func Routes(ctx runtime.Ctx, defs ...*RouteDef) h.Node {
	base := routeBaseCtx.Use(ctx)
	return routeBaseCtx.Provide(ctx, base, func() h.Node {
		return RoutesNode(base, defs...)
	})
}

// WithBase adjusts the base path for nested route definitions.
func WithBase(ctx runtime.Ctx, base string, render func() h.Node) h.Node {
	if render == nil {
		return h.Fragment()
	}
	return routeBaseCtx.Provide(ctx, base, render)
}

// Link renders a router-aware anchor element that resolves relative URLs using
// the active RouterStore.
func Link(ctx runtime.Ctx, props LinkProps, children ...h.Node) h.Node {
	store := routerStoreCtx.Use(ctx)
	if store == nil {
		panic("router2: Link used outside Router")
	}
	return LinkNode(props, children...)
}
