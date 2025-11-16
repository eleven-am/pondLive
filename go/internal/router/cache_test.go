package router

import (
	"testing"

	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestRouteCacheReuse(t *testing.T) {
	defs := []*RouteDef{
		DefineRoute("/", func(runtime.Ctx, RouteRenderContext) h.Node { return h.Text("ok") }),
	}
	first := compileRoutesCached("/", defs...)
	second := compileRoutesCached("/", defs...)
	if first != second {
		t.Fatalf("expected cached tree reuse")
	}
}

func TestRouteCacheStats(t *testing.T) {
	before := RouteCacheStats()
	defs := []*RouteDef{
		DefineRoute("/stats", func(runtime.Ctx, RouteRenderContext) h.Node { return h.Text("stats") }),
	}
	compileRoutesCached("/", defs...)
	compileRoutesCached("/", defs...)
	after := RouteCacheStats()
	if after.Misses-before.Misses != 1 {
		t.Fatalf("expected one miss delta, got before=%#v after=%#v", before, after)
	}
	if after.Hits-before.Hits != 1 {
		t.Fatalf("expected one hit delta, got before=%#v after=%#v", before, after)
	}
}
