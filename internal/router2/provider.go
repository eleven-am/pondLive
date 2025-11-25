package router2

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/headers2"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ProvideRouter2 sets up the router context hierarchy.
// It provides:
// - LocationContext (mutable URL state)
// - MatchContext (current route match)
// - outletSlotCtx (outlet slot distribution)
//
// And subscribes to Bus for navigation events (live mode only).
func ProvideRouter2(ctx *runtime2.Ctx, children []work.Node) work.Node {
	requestState := headers2.UseRequestState(ctx)
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

	location, setLocation := LocationContext.UseProvider(ctx, initialLocation)
	_ = location

	runtime2.UseEffect(ctx, func() func() {
		if bus == nil {
			return nil
		}

		sub := bus.Subscribe("router", func(event string, data interface{}) {
			switch event {
			case "navigate":
				nav := parseNavPayload(data)
				if nav == nil {
					return
				}
				newLoc := canonicalizeLocation(nav.ToLocation())
				setLocation(newLoc)

				if nav.Replace {
					bus.Publish("router", "replaced", NavResponse{
						Path:    newLoc.Path,
						Query:   newLoc.Query.Encode(),
						Hash:    newLoc.Hash,
						Replace: true,
					})
				} else {
					bus.Publish("router", "navigated", NavResponse{
						Path:    newLoc.Path,
						Query:   newLoc.Query.Encode(),
						Hash:    newLoc.Hash,
						Replace: false,
					})
				}

			case "popstate":

				nav := parseNavPayload(data)
				if nav == nil {
					return
				}
				newLoc := canonicalizeLocation(nav.ToLocation())
				setLocation(newLoc)
			}
		})

		return sub.Unsubscribe
	}, bus)

	return outletSlotCtx.ProvideWithoutDefault(ctx, children)
}

// parseNavPayload converts interface{} data from Bus to NavPayload.
func parseNavPayload(data interface{}) *NavPayload {
	if data == nil {
		return nil
	}

	switch v := data.(type) {
	case NavPayload:
		return &v
	case *NavPayload:
		return v
	case map[string]interface{}:
		nav := &NavPayload{}
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
