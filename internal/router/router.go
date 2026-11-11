package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
)

type Router struct {
	st *routerState

	Location runtime.ReadSignal[Location]
	Path     runtime.ReadSignal[string]
	Hash     runtime.ReadSignal[string]
	Query    runtime.ReadSignal[url.Values]
	Params   runtime.ReadSignal[map[string]string]
	Matched  runtime.ReadSignal[bool]
	Match    runtime.ReadSignal[*MatchState]
}

func UseRouter(ctx *runtime.Ctx) *Router {
	st := runtime.UseContext(ctx, routerCtx).Peek()
	info := runtime.UseContext(ctx, matchCtx).Peek()
	return &Router{
		st:       st,
		Location: st.location,
		Path:     st.path,
		Hash:     st.hash,
		Query:    st.query,
		Params:   info.params,
		Matched:  info.matched,
		Match:    info.state,
	}
}

func (r *Router) Navigate(href string) { r.st.req.Push(href) }

func (r *Router) Replace(href string, opts ...headers.NavOption) { r.st.req.Replace(href, opts...) }

func (r *Router) NavigateWith(fn func(Location) Location) { r.st.req.Push(r.hrefFrom(fn)) }

func (r *Router) ReplaceWith(fn func(Location) Location, opts ...headers.NavOption) {
	r.st.req.Replace(r.hrefFrom(fn), opts...)
}

func (r *Router) hrefFrom(fn func(Location) Location) string {
	return route.BuildHref(fn(r.Location.Peek()))
}

func (r *Router) SetQuery(key, value string) {
	r.ReplaceWith(func(current Location) Location {
		current = route.DelSearch(current, key)
		if value != "" {
			current = route.AddSearch(current, key, value)
		}
		return current
	})
}

func (r *Router) Back() { r.st.req.Back() }

func (r *Router) Forward() { r.st.req.Forward() }
