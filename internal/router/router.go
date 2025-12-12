package router

import (
	"net/url"

	"github.com/eleven-am/pondlive/internal/runtime"
)

type Router struct {
	ctx           *runtime.Ctx
	emitter       *RouterEventEmitter
	subscriptions *runtime.Ref[[]*Subscription]
}

func UseRouter(ctx *runtime.Ctx) *Router {
	emitter := emitterCtx.UseContextValue(ctx)
	subscriptions := runtime.UseRef(ctx, []*Subscription{})

	runtime.UseEffect(ctx, func() func() {
		return func() {
			for _, sub := range subscriptions.Current {
				sub.Unsubscribe()
			}
			subscriptions.Current = nil
		}
	})

	return &Router{
		ctx:           ctx,
		emitter:       emitter,
		subscriptions: subscriptions,
	}
}

func (r *Router) Navigate(href string) {
	Navigate(r.ctx, href)
}

func (r *Router) Replace(href string) {
	Replace(r.ctx, href)
}

func (r *Router) NavigateWith(fn func(Location) Location) {
	NavigateWith(r.ctx, fn)
}

func (r *Router) ReplaceWith(fn func(Location) Location) {
	ReplaceWith(r.ctx, fn)
}

func (r *Router) Back() {
	Back(r.ctx)
}

func (r *Router) Forward() {
	Forward(r.ctx)
}

func (r *Router) Location() Location {
	return UseLocation(r.ctx)
}

func (r *Router) Params() map[string]string {
	return UseParams(r.ctx)
}

func (r *Router) Param(key string) (string, error) {
	params := r.Params()
	if params == nil {
		return "", ErrParamNotFound
	}
	val, ok := params[key]
	if !ok {
		return "", ErrParamNotFound
	}
	return val, nil
}

func (r *Router) Match() *MatchState {
	return UseMatch(r.ctx)
}

func (r *Router) Matched() bool {
	return UseMatched(r.ctx)
}

func (r *Router) SearchParams() url.Values {
	return UseSearchParams(r.ctx)
}

func (r *Router) SearchParam(key string) (string, func(string)) {
	return UseSearchParam(r.ctx, key)
}

func (r *Router) OnBeforeNavigate(fn func(NavigationEvent)) {
	if r.emitter == nil {
		return
	}
	sub := r.emitter.Subscribe("beforeNavigate", fn)
	r.subscriptions.Current = append(r.subscriptions.Current, sub)
}

func (r *Router) OnNavigated(fn func(NavigationEvent)) {
	if r.emitter == nil {
		return
	}
	sub := r.emitter.Subscribe("navigated", fn)
	r.subscriptions.Current = append(r.subscriptions.Current, sub)
}
