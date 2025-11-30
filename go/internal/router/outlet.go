package router

import (
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func Outlet(ctx *runtime.Ctx, name string, fallback ...work.Node) work.Node {
	if name == "" {
		name = "default"
	}
	return outletSlotCtx.Render(ctx, name, fallback...)
}

func OutletDefault(ctx *runtime.Ctx, fallback ...work.Node) work.Node {
	return Outlet(ctx, "default", fallback...)
}

func HasOutlet(ctx *runtime.Ctx, name string) bool {
	if name == "" {
		name = "default"
	}
	return outletSlotCtx.Has(ctx, name)
}

func OutletNames(ctx *runtime.Ctx) []string {
	return outletSlotCtx.Names(ctx)
}
