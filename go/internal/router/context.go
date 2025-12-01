package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

var LocationContext = runtime.CreateContext[*Location](nil)

var MatchContext = runtime.CreateContext[*MatchState](nil)

var outletSlotCtx = runtime.CreateSlotContext()

var childRoutesCtx = runtime.CreateContext[[]work.Node](nil)

func getBus(ctx *runtime.Ctx) *protocol.Bus {
	return runtime.GetBus(ctx)
}

func UseLocation(ctx *runtime.Ctx) *Location {
	return LocationContext.UseContextValue(ctx)
}

func UseParams(ctx *runtime.Ctx) map[string]string {
	match := MatchContext.UseContextValue(ctx)
	if match == nil {
		return nil
	}
	return match.Params
}

func UseParam(ctx *runtime.Ctx, key string) string {
	params := UseParams(ctx)
	if params == nil {
		return ""
	}
	return params[key]
}

func UseQuery(ctx *runtime.Ctx) url.Values {
	loc := UseLocation(ctx)
	if loc == nil {
		return nil
	}
	return loc.Query
}

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

func UseMatch(ctx *runtime.Ctx) *MatchState {
	return MatchContext.UseContextValue(ctx)
}

func UseMatched(ctx *runtime.Ctx) bool {
	match := MatchContext.UseContextValue(ctx)
	return match != nil && match.Matched
}
