package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// RenderRoutes evaluates the provided router nodes against the store and
// returns the rendered DOM tree. When no route matches, an empty fragment is
// returned. This helper mirrors runtime behavior while letting tests exercise
// router without session contexts.
func RenderRoutes(ctx runtime.Ctx, store *RouterStore, nodes ...h.Node) dom.Node {
	defs := collectRouteDefs(nodes, "/")
	return renderRoutesWithBase(ctx, store, "/", defs)
}
