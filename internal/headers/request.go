package headers

import (
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

const setCookieScript = `function(element,transport){transport.on('setCookie',function(data){var body=JSON.stringify({token:data.token});var init={method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:body};fetch(data.url,init).then(function(res){if(res.ok){return fetch(data.ackUrl,init)}})})}`

type CookieOptions struct {
	Path     string
	Domain   string
	MaxAge   int
	Expires  time.Time
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

type requestRoot struct {
	state *RequestState
	ps    *providerState

	url     *runtime.Signal[route.Location]
	headers *runtime.Signal[http.Header]

	bus       *protocol.Bus
	connected runtime.ReadSignal[bool]
}

func (r *requestRoot) setURL(loc route.Location) {
	loc = route.Canonicalize(loc)
	r.state.setURL(loc)
	r.url.Set(loc)
}

func (r *requestRoot) refresh() {
	r.url.Set(r.state.URL())
	r.headers.Set(r.state.effectiveHeaders())
}

func headersEqual(a, b http.Header) bool { return maps.EqualFunc(a, b, slices.Equal[[]string]) }

var requestCtx = runtime.NewContext[*requestRoot]()

type ProviderProps struct {
	State    *RequestState
	Children work.Node
}

var Provider = runtime.PropsComponent(func(ctx *runtime.Ctx, props ProviderProps) work.Node {
	url := runtime.UseSignal(ctx, props.State.URL(), runtime.WithEqual(route.LocEqual))
	headerSet := runtime.UseSignal(ctx, props.State.effectiveHeaders(), runtime.WithEqual(headersEqual))
	rootSig := runtime.UseSignal(ctx, &requestRoot{state: props.State, url: url, headers: headerSet})
	root := rootSig.Peek()
	if root.ps == nil {
		root.ps = &providerState{
			requestState: props.State,
			holder:       &tokenHolder{onEvict: evictionReporter(func(r runtime.Report) { runtime.ReportOn(ctx, r) })},
		}
		props.State.bindNotify(root.refresh)
	}

	root.bus = runtime.GetBus(ctx)
	root.connected = runtime.SessionOf(ctx).Connected()
	runtime.UseEffect(ctx, func() func() {
		if root.bus == nil {
			return nil
		}
		sub := root.bus.SubscribeToRouterPopstate(func(payload protocol.RouterPopstatePayload) {
			runtime.Invoke(ctx, func() {
				root.setURL(route.FromNavigation(payload.Path, payload.Query, payload.Hash))
			})
		})
		return sub.Unsubscribe
	})

	ps := root.ps
	handler := runtime.UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
		ps.serveCookieSet(w, r)
		return nil
	})
	ackHandler := runtime.UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
		ps.serveCookieAck(w, r)
		return nil
	})
	script := runtime.UseScript(ctx, setCookieScript)

	ps.handler = handler
	ps.script = script
	ps.handlerURL = handler.URL()
	ps.ackHandlerURL = ackHandler.URL()

	runtime.Provide(ctx, requestCtx, rootSig)

	if props.Children == nil {
		return work.NewFragment()
	}
	return props.Children
})

var Render = runtime.Component(func(ctx *runtime.Ctx) work.Node {
	root := useRoot(ctx)
	return work.BuildElement("script", runtime.AttachItem(root.ps.script))
})

func useRoot(ctx *runtime.Ctx) *requestRoot {
	return runtime.UseContext(ctx, requestCtx).Peek()
}

func UseRequestState(ctx *runtime.Ctx) *RequestState {
	return useRoot(ctx).state
}

type Request struct {
	ctx  *runtime.Ctx
	root *requestRoot
}

func UseRequest(ctx *runtime.Ctx) *Request {
	return &Request{ctx: ctx, root: useRoot(ctx)}
}

func (r *Request) Header(name string) runtime.ReadSignal[string] {
	return runtime.Derive(r.root.headers, func(h http.Header) string { return h.Get(name) })
}

func (r *Request) Connected() runtime.ReadSignal[bool] {
	return r.root.connected
}

func (r *Request) URL() runtime.ReadSignal[route.Location] {
	return runtime.Derive(r.root.url, route.Location.Clone)
}

func (r *Request) Headers() runtime.ReadSignal[http.Header] {
	return runtime.Derive(r.root.headers, http.Header.Clone)
}

func (r *Request) Cookie(name string) runtime.ReadSignal[string] {
	return runtime.Derive(r.root.headers, func(h http.Header) string {
		value, _ := (&RequestInfo{Headers: h}).GetCookie(name)
		return value
	})
}

func (r *Request) SetCookie(name, value string, opts *CookieOptions) {
	pending := &pendingCookie{name: name, value: value, options: opts}
	received, ok := pending.received()
	if !ok {
		return
	}

	state := r.root.state
	if pending.deletes() {
		state.DeleteCookieMutation(name)
	} else {
		state.MutateCookie(name, received.Value)
	}

	if state.IsLive() && r.root.ps != nil {
		sendCookieViaScript(r.root.ps, name, value, opts)
	} else {
		state.AddResponseHeader("Set-Cookie", pending.httpCookie().String())
	}
	r.root.headers.Set(state.effectiveHeaders())
}

func sendCookieViaScript(pState *providerState, name, value string, opts *CookieOptions) {
	if pState == nil {
		return
	}

	token := pState.handler.GenerateToken()
	pState.storeCookie(token, &pendingCookie{
		name:    name,
		value:   value,
		options: opts,
	})

	pState.script.Send("setCookie", map[string]any{
		"url":    pState.handlerURL,
		"token":  token,
		"ackUrl": pState.ackHandlerURL,
	})
}
