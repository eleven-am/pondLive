package router2

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

func TestRouteCacheReuse(t *testing.T) {
	defs := []*RouteDef{
		Route("/", func(RouteRenderContext) dom.Node { return &dom.TextNode{} }),
	}
	first := compileRoutesCached("/", defs...)
	second := compileRoutesCached("/", defs...)
	if first != second {
		t.Fatalf("expected cached tree reuse")
	}
}
