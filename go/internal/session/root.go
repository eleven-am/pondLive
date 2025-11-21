package session

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/meta"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Component represents a root component with no props, matching the pattern from examples.
// This is the typical signature for app components: func(ctx Ctx) *Node
type Component func(runtime.Ctx) *dom.StructuredNode

func documentRoot(sess *LiveSession, app Component) runtime.Component[struct{}] {
	return documentRootGeneric(sess, func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return app(ctx)
	})
}

func documentRootGeneric[P any](sess *LiveSession, app runtime.Component[P]) runtime.Component[P] {
	return func(ctx runtime.Ctx, props P) *dom.StructuredNode {
		initial := &router.RouterState{
			Location: toRouterLocation(sess.InitialLocation()),
			Matched:  false,
			Pattern:  "",
			Params:   make(map[string]string),
			Path:     "",
		}
		current, setCurrent := runtime.UseState(ctx, initial)

		controller := runtime.UseMemo(ctx, func() *router.Controller {
			return router.NewController(current, setCurrent)
		})

		sess.registerRouterState(func(loc Location) {
			controller.SetLocation(toRouterLocation(loc))
		})

		return router.ProvideRouterState(ctx, controller, func(rctx runtime.Ctx) *dom.StructuredNode {
			return HeaderContext.Provide(rctx, sess.Header(), func(hctx runtime.Ctx) *dom.StructuredNode {
				return meta.Provider(hctx, sess.clientAsset, app, props)
			})
		})
	}
}

func toRouterLocation(loc Location) router.Location {
	cp := router.Location{
		Path: loc.Path,
		Hash: loc.Hash,
	}
	if loc.Query != nil {
		cp.Query = cloneQuery(loc.Query)
	}
	return cp
}

func cloneRouterLocation(loc router.Location) router.Location {
	return router.Location{
		Path:  loc.Path,
		Query: cloneQuery(loc.Query),
		Hash:  loc.Hash,
	}
}
