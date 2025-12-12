package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

var Provide = runtime.Component(func(ctx *runtime.Ctx, children []work.Item) work.Node {
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

	loc, setLoc := locationCtx.UseProvider(ctx, canonicalizeLocation(initialLoc))
	routeBaseCtx.UseProvider(ctx, "/")

	emitterRef := runtime.UseRef(ctx, NewRouterEventEmitter())
	emitterCtx.UseProvider(ctx, emitterRef.Current)

	prevLocRef := runtime.UseRef(ctx, loc)

	runtime.UseEffect(ctx, func() func() {
		prev := prevLocRef.Current
		if !locationEqual(prev, loc) {
			emitterRef.Current.Emit("navigated", NavigationEvent{
				From:         prev,
				To:           loc,
				PathChanged:  prev.Path != loc.Path,
				HashChanged:  prev.Hash != loc.Hash,
				QueryChanged: prev.Query.Encode() != loc.Query.Encode(),
				Replace:      false,
			})
			prevLocRef.Current = loc
		}
		return nil
	}, loc)

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
			newLoc = canonicalizeLocation(newLoc)

			if !route.LocEqual(loc, newLoc) {
				emitterRef.Current.Emit("beforeNavigate", NavigationEvent{
					From:         loc,
					To:           newLoc,
					PathChanged:  loc.Path != newLoc.Path,
					HashChanged:  loc.Hash != newLoc.Hash,
					QueryChanged: loc.Query.Encode() != newLoc.Query.Encode(),
					Replace:      false,
				})
				setLoc(newLoc)
			}
		})

		return func() {
			sub.Unsubscribe()
		}
	}, bus, loc)

	nodes := work.ItemsToNodes(children)
	return &work.Fragment{Children: nodes}
})
