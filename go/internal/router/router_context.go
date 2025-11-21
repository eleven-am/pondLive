package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/route"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type Location = route.Location

var LocationCtx = runtime.CreateContext[Location](Location{Path: "/"})
var ParamsCtx = runtime.CreateContext[map[string]string](map[string]string{})

func UseLocation(ctx runtime.Ctx) Location {
	controller := UseRouterState(ctx)
	state := controller.Get()
	return cloneLocation(state.Location)
}

func UseParams(ctx runtime.Ctx) map[string]string {
	controller := UseRouterState(ctx)
	state := controller.Get()
	params := state.Params
	if len(params) == 0 {
		return map[string]string{}
	}
	cp := make(map[string]string, len(params))
	for k, v := range params {
		cp[k] = v
	}
	return cp
}

func UseParam(ctx runtime.Ctx, key string) string {
	if key == "" {
		return ""
	}
	params := UseParams(ctx)
	return params[key]
}

func UseSearch(ctx runtime.Ctx) url.Values {
	loc := UseLocation(ctx)
	return cloneValues(loc.Query)
}

func UseSearchParam(ctx runtime.Ctx, key string) (func() []string, func([]string)) {
	controller := UseRouterState(ctx)
	lower := key
	get := func() []string {
		state := controller.Get()
		values := state.Location.Query[lower]
		if len(values) == 0 {
			return nil
		}
		out := make([]string, len(values))
		copy(out, values)
		return out
	}
	set := func(values []string) {
		state := controller.Get()
		next := cloneLocation(state.Location)
		next.Query = SetSearch(next.Query, lower, values...)
		controller.SetLocation(next)
	}
	return get, set
}
