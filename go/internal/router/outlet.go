package router

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type OutletProps struct {
	Name string
}

func nodesToItems(nodes []work.Node) []work.Item {
	items := make([]work.Item, len(nodes))
	for i, n := range nodes {
		items[i] = n
	}
	return items
}

var Outlet = runtime.PropsComponent(func(ctx *runtime.Ctx, props OutletProps, fallback []work.Node) work.Node {
	name := props.Name
	if name == "" {
		name = "default"
	}

	childRoutes := childRoutesCtx.UseContextValue(ctx)
	if len(childRoutes) > 0 {
		return &work.Fragment{
			Children: []work.Node{
				Routes(ctx, RoutesProps{Outlet: name}, nodesToItems(childRoutes)...),
				outletSlotCtx.Render(ctx, name, fallback...),
			},
		}
	}

	return outletSlotCtx.Render(ctx, name, fallback...)
})

var OutletDefault = runtime.Component(func(ctx *runtime.Ctx, fallback []work.Node) work.Node {
	return Outlet(ctx, OutletProps{Name: "default"}, nodesToItems(fallback)...)
})

func HasOutlet(ctx *runtime.Ctx, name string) bool {
	if name == "" {
		name = "default"
	}
	return outletSlotCtx.Has(ctx, name)
}

func OutletNames(ctx *runtime.Ctx) []string {
	return outletSlotCtx.Names(ctx)
}
