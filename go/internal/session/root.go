package session

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/meta"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Component represents a root component with no props, matching the pattern from examples.
// This is the typical signature for app components: func(ctx Ctx) *Node
type Component func(runtime.Ctx) *dom.StructuredNode

func documentRoot(sess *LiveSession, app Component) runtime.Component[struct{}] {
	return func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return headers.ProvideRequestController(ctx, sess.requestController, func(hctx runtime.Ctx) *dom.StructuredNode {
			wrapped := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
				return app(ctx)
			}

			return headers.ProvideHeadersManager(hctx, func(mctx runtime.Ctx) *dom.StructuredNode {
				return router.ProvideRouter(mctx, func(rctx runtime.Ctx) *dom.StructuredNode {
					return meta.Provider(rctx, sess.clientAsset, wrapped, struct{}{})
				})
			})
		})
	}
}
