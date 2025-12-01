package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/route"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

var Provide = runtime.Component(func(ctx *runtime.Ctx, children []work.Node) work.Node {
	requestState := headers.UseRequestState(ctx)

	initialLoc := Location{Path: "/"}
	if requestState != nil {
		initialLoc = Location{
			Path:  requestState.Path(),
			Query: requestState.Query(),
			Hash:  requestState.Hash(),
		}
	}

	if initialLoc.Path == "" {
		initialLoc.Path = "/"
	}

	loc, setLoc := runtime.UseState(ctx, initialLoc)

	bus := runtime.GetBus(ctx)
	runtime.UseEffect(ctx, func() func() {
		if bus == nil {
			return nil
		}

		sub := bus.SubscribeToRouterPopstate(func(payload protocol.RouterNavPayload) {
			query, _ := url.ParseQuery(payload.Query)
			newLoc := Location{
				Path:  payload.Path,
				Query: query,
				Hash:  payload.Hash,
			}
			if newLoc.Path == "" {
				newLoc.Path = "/"
			}
			if !route.LocEqual(loc, newLoc) {
				setLoc(newLoc)
			}
		})

		return func() {
			sub.Unsubscribe()
		}
	}, bus)

	locationCtx.UseProvider(ctx, loc)
	routeBaseCtx.UseProvider(ctx, "/")

	return &work.Fragment{Children: children}
})
