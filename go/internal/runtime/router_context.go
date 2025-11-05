package runtime

import (
	"net/url"
)

type Location struct {
	Path  string
	Query url.Values
	Hash  string
}

var LocationCtx = NewContext(Location{Path: "/"})
var ParamsCtx = NewContext(map[string]string{})

func UseLocation(ctx Ctx) Location {
	loc := LocationCtx.Use(ctx)
	return cloneLocation(loc)
}

func UseParams(ctx Ctx) map[string]string {
	params := ParamsCtx.Use(ctx)
	if len(params) == 0 {
		sess := ctx.Session()
		fallback := sessionParams(sess)
		if len(fallback) == 0 {
			return map[string]string{}
		}
		return fallback
	}
	cp := make(map[string]string, len(params))
	for k, v := range params {
		cp[k] = v
	}
	return cp
}

func UseParam(ctx Ctx, key string) string {
	if key == "" {
		return ""
	}
	params := UseParams(ctx)
	return params[key]
}

func UseSearch(ctx Ctx) url.Values {
	loc := UseLocation(ctx)
	return cloneValues(loc.Query)
}

func UseSearchParam(ctx Ctx, key string) (func() []string, func([]string)) {
	state := requireRouterState(ctx)
	lower := key
	get := func() []string {
		loc := state.getLoc()
		values := loc.Query[lower]
		if len(values) == 0 {
			return nil
		}
		out := make([]string, len(values))
		copy(out, values)
		return out
	}
	set := func(values []string) {
		loc := state.getLoc()
		next := cloneLocation(loc)
		next.Query = SetSearch(next.Query, lower, values...)
		state.setLoc(next)
	}
	return get, set
}
