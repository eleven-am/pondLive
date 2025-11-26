package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// LocationContext provides the current location to the component tree.
// Location is mutable via navigation events.
var LocationContext = runtime.CreateContext[*Location](nil)

// MatchContext provides the current route match to the component tree.
// Updated by Routes component when a match is found.
var MatchContext = runtime.CreateContext[*MatchState](nil)

// outletSlotCtx manages outlet slot content distribution.
var outletSlotCtx = runtime.CreateSlotContext()

// getBus accesses Bus from context (keeps Session private).
func getBus(ctx *runtime.Ctx) *protocol.Bus {
	return runtime.GetBus(ctx)
}

// UseLocation returns the current location from context.
// Returns nil if no router provider is present.
func UseLocation(ctx *runtime.Ctx) *Location {
	return LocationContext.UseContextValue(ctx)
}

// UseParams returns all route parameters from the current match.
// Returns nil if no route is matched.
func UseParams(ctx *runtime.Ctx) map[string]string {
	match := MatchContext.UseContextValue(ctx)
	if match == nil {
		return nil
	}
	return match.Params
}

// UseParam returns a single route parameter by key.
// Returns empty string if the parameter doesn't exist or no route is matched.
func UseParam(ctx *runtime.Ctx, key string) string {
	params := UseParams(ctx)
	if params == nil {
		return ""
	}
	return params[key]
}

// UseQuery returns query parameters from the current location.
// Returns nil if no router provider is present.
func UseQuery(ctx *runtime.Ctx) url.Values {
	loc := UseLocation(ctx)
	if loc == nil {
		return nil
	}
	return loc.Query
}

// UseSearchParam returns getter/setter for a single query parameter.
// The setter triggers navigation to update the URL.
func UseSearchParam(ctx *runtime.Ctx, key string) (func() []string, func([]string)) {
	loc := UseLocation(ctx)

	getter := func() []string {
		if loc == nil || loc.Query == nil {
			return nil
		}
		return loc.Query[key]
	}

	setter := func(values []string) {
		if loc == nil {
			return
		}
		newQuery := cloneValues(loc.Query)
		if len(values) == 0 {
			delete(newQuery, key)
		} else {
			newQuery[key] = values
		}
		href := buildHref(loc.Path, newQuery, loc.Hash)
		Navigate(ctx, href)
	}

	return getter, setter
}

// UseSearchParams returns getter/setter for all query parameters.
// The setter triggers navigation to update the URL.
func UseSearchParams(ctx *runtime.Ctx) (func() url.Values, func(url.Values)) {
	loc := UseLocation(ctx)

	getter := func() url.Values {
		if loc == nil {
			return nil
		}
		return cloneValues(loc.Query)
	}

	setter := func(values url.Values) {
		if loc == nil {
			return
		}
		href := buildHref(loc.Path, values, loc.Hash)
		Navigate(ctx, href)
	}

	return getter, setter
}

// UseMatch returns the current route match state.
// Returns nil if no route is matched.
func UseMatch(ctx *runtime.Ctx) *MatchState {
	return MatchContext.UseContextValue(ctx)
}

// UseMatched returns whether any route is currently matched.
func UseMatched(ctx *runtime.Ctx) bool {
	match := MatchContext.UseContextValue(ctx)
	return match != nil && match.Matched
}
