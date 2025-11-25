package router2

import (
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// Outlet renders the matched route component for a named outlet.
// If no route is matched, renders the fallback children.
//
// Usage:
//
//	router.Outlet(ctx, "main", fallbackComponent)
func Outlet(ctx *runtime2.Ctx, name string, fallback ...work.Node) work.Node {
	if name == "" {
		name = "default"
	}
	return outletSlotCtx.Render(ctx, name, fallback...)
}

// OutletDefault is a convenience function for the default outlet.
//
// Usage:
//
//	router.OutletDefault(ctx, fallbackComponent)
func OutletDefault(ctx *runtime2.Ctx, fallback ...work.Node) work.Node {
	return Outlet(ctx, "default", fallback...)
}

// HasOutlet checks if a named outlet has matched content.
//
// Usage:
//
//	if router.HasOutlet(ctx, "sidebar") {
//	    // sidebar route is matched
//	}
func HasOutlet(ctx *runtime2.Ctx, name string) bool {
	if name == "" {
		name = "default"
	}
	return outletSlotCtx.Has(ctx, name)
}

// OutletNames returns all outlet names that have content.
func OutletNames(ctx *runtime2.Ctx) []string {
	return outletSlotCtx.Names(ctx)
}
