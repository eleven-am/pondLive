package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ProvideRouter sets up the router context hierarchy.
// It provides:
// - LocationContext (mutable URL state)
// - MatchContext (current route match)
// - outletSlotCtx (outlet slot distribution)
//
// And subscribes to Bus for navigation events (live mode only).
var ProvideRouter = html.Component(func(ctx *runtime.Ctx, children []work.Node) work.Node {
	requestState := headers.UseRequestState(ctx)
	bus := getBus(ctx)

	initialLocation := &Location{
		Path:  "/",
		Query: url.Values{},
	}

	if requestState != nil {
		initialLocation = &Location{
			Path:  requestState.Path(),
			Query: requestState.Query(),
			Hash:  requestState.Hash(),
		}
	}

	_, setLocation := LocationContext.UseProvider(ctx, initialLocation)

	runtime.UseEffect(ctx, func() func() {
		if bus == nil {
			return nil
		}

		sub := bus.Subscribe(protocol.RouteHandler, func(event string, data interface{}) {
			switch event {
			case "popstate":
				nav := parseNavPayload(data)
				if nav == nil {
					return
				}
				newLoc := canonicalizeLocation(navPayloadToLocation(nav))
				setLocation(newLoc)
			}
		})

		return sub.Unsubscribe
	}, bus)

	return outletSlotCtx.ProvideWithoutDefault(ctx, children)
})

// parseNavPayload converts interface{} data from Bus to protocol.RouterNavPayload.
func parseNavPayload(data interface{}) *protocol.RouterNavPayload {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case protocol.RouterNavPayload:
		return &v
	case *protocol.RouterNavPayload:
		return v
	case map[string]interface{}:
		nav := &protocol.RouterNavPayload{}
		if path, ok := v["path"].(string); ok {
			nav.Path = path
		}
		if query, ok := v["query"].(string); ok {
			nav.Query = query
		}
		if hash, ok := v["hash"].(string); ok {
			nav.Hash = hash
		}
		if replace, ok := v["replace"].(bool); ok {
			nav.Replace = replace
		}
		return nav
	default:
		return nil
	}
}

// navPayloadToLocation converts protocol.RouterNavPayload to Location.
func navPayloadToLocation(nav *protocol.RouterNavPayload) *Location {
	if nav == nil {
		return nil
	}
	query, _ := url.ParseQuery(nav.Query)
	return &Location{
		Path:  nav.Path,
		Query: query,
		Hash:  nav.Hash,
	}
}
