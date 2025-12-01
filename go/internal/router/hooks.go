package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func UseLocation(ctx *runtime.Ctx) Location {
	return locationCtx.UseContextValue(ctx)
}

func UseParams(ctx *runtime.Ctx) map[string]string {
	match := matchCtx.UseContextValue(ctx)
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

func UseMatch(ctx *runtime.Ctx) *MatchState {
	return matchCtx.UseContextValue(ctx)
}

func UseMatched(ctx *runtime.Ctx) bool {
	match := matchCtx.UseContextValue(ctx)
	return match != nil && match.Matched
}

func UseQuery(ctx *runtime.Ctx) url.Values {
	loc := UseLocation(ctx)
	return loc.Query
}
