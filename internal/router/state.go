package router

import (
	"maps"
	"net/url"
	"slices"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type routerState struct {
	location runtime.ReadSignal[Location]
	path     runtime.ReadSignal[string]
	hash     runtime.ReadSignal[string]
	query    runtime.ReadSignal[url.Values]
	req      *headers.Request
}

type matchInfo struct {
	state   runtime.ReadSignal[*MatchState]
	params  runtime.ReadSignal[map[string]string]
	matched runtime.ReadSignal[bool]
	rest    runtime.ReadSignal[string]
}

var (
	routerCtx = runtime.NewContext[*routerState]()
	matchCtx  = runtime.NewContext[*matchInfo]()
)

func paramsEqual(a, b map[string]string) bool { return maps.Equal(a, b) }

func valuesEqual(a, b url.Values) bool { return maps.EqualFunc(a, b, slices.Equal[[]string]) }

func matchStateEqual(a, b *MatchState) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Matched == b.Matched && a.Pattern == b.Pattern && a.Path == b.Path && a.Rest == b.Rest && paramsEqual(a.Params, b.Params)
}

var Provider = runtime.PropsComponent(func(ctx *runtime.Ctx, props ProviderProps) work.Node {
	req := headers.UseRequest(ctx)
	location := req.URL()
	path := runtime.UseComputed(ctx, func() string { return location.Get().Path })
	hash := runtime.UseComputed(ctx, func() string { return location.Get().Hash })
	query := runtime.UseComputed(ctx, func() url.Values { return location.Get().Query }, runtime.WithEqual(valuesEqual))

	st := runtime.UseSignal(ctx, &routerState{
		location: location,
		path:     path,
		hash:     hash,
		query:    query,
		req:      req,
	})
	runtime.Provide(ctx, routerCtx, st)
	runtime.Provide(ctx, matchCtx, runtime.UseSignal(ctx, &matchInfo{
		state:   runtime.Const(&MatchState{}),
		params:  runtime.Const(map[string]string{}),
		matched: runtime.Const(false),
		rest:    path,
	}))

	if props.Children == nil {
		return work.NewFragment()
	}
	return props.Children
})
