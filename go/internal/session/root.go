package session

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Component represents a root component with no props, matching the pattern from examples.
// This is the typical signature for app components: func(ctx Ctx) *Node
type Component func(runtime.Ctx) *dom2.StructuredNode

// documentRoot wraps a no-props component with context providers and adapts it to runtime2.Component[struct{}].
// This is internal and used by New() and NewLiveSession().
// It provides HeaderContext and RouterState at the root level.
func documentRoot(sess *LiveSession, app Component) runtime.Component[struct{}] {
	return func(ctx runtime.Ctx, _ struct{}) *dom2.StructuredNode {
		initial := toRouterLocation(sess.InitialLocation())
		getLoc, setLoc := runtime.UseState(ctx, initial, runtime.WithEqual(router.LocEqual))

		stateProvider := router.StateProvider{
			Get: func() router.Location { return cloneRouterLocation(getLoc()) },
			Set: func(loc router.Location) { setLoc(loc) },
		}

		sess.registerRouterState(func(loc Location) {
			setLoc(toRouterLocation(loc))
		})

		return router.ProvideState(ctx, stateProvider, func(rctx runtime.Ctx) *dom2.StructuredNode {
			return HeaderContext.Provide(rctx, sess.Header(), func(hctx runtime.Ctx) *dom2.StructuredNode {
				return app(hctx)
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
