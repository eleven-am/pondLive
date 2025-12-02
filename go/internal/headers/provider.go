package headers

import (
	"encoding/json"
	"io"
	"net/http"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type pendingCookie struct {
	name    string
	value   string
	options *CookieOptions
}

type providerState struct {
	requestState   *RequestState
	script         runtime.ScriptHandle
	handler        runtime.HandlerHandle
	handlerURL     string
	pendingCookies map[string]*pendingCookie
	cookieMu       sync.Mutex
}

func (p *providerState) storeCookie(token string, cookie *pendingCookie) {
	p.cookieMu.Lock()
	if p.pendingCookies == nil {
		p.pendingCookies = make(map[string]*pendingCookie)
	}
	p.pendingCookies[token] = cookie
	p.cookieMu.Unlock()
}

func (p *providerState) consumeCookie(token string) *pendingCookie {
	p.cookieMu.Lock()
	defer p.cookieMu.Unlock()
	cookie, exists := p.pendingCookies[token]
	if exists {
		delete(p.pendingCookies, token)
		return cookie
	}
	return nil
}

var providerCtx = runtime.CreateContext[*providerState](nil)

type tokenPayload struct {
	Token string `json:"token"`
}

var Provider = runtime.PropsComponent(func(ctx *runtime.Ctx, requestState *RequestState, children []work.Item) work.Node {
	requestCtx.UseProvider(ctx, requestState)

	pState := &providerState{
		requestState:   requestState,
		pendingCookies: make(map[string]*pendingCookie),
	}

	handler := runtime.UseHandler(ctx, "POST", func(w http.ResponseWriter, r *http.Request) error {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil
		}
		defer r.Body.Close()

		var payload tokenPayload
		if err := json.Unmarshal(body, &payload); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return nil
		}

		pending := pState.consumeCookie(payload.Token)
		if pending == nil {
			w.WriteHeader(http.StatusForbidden)
			return nil
		}

		cookie := &http.Cookie{
			Name:  pending.name,
			Value: pending.value,
		}

		if pending.options != nil {
			cookie.Path = pending.options.Path
			cookie.Domain = pending.options.Domain
			cookie.MaxAge = pending.options.MaxAge
			cookie.Expires = pending.options.Expires
			cookie.Secure = pending.options.Secure
			cookie.HttpOnly = pending.options.HttpOnly
			cookie.SameSite = pending.options.SameSite
		}

		if cookie.Path == "" {
			cookie.Path = "/"
		}

		http.SetCookie(w, cookie)
		w.WriteHeader(http.StatusNoContent)
		return nil
	})

	script := runtime.UseScript(ctx, `function(element,transport){transport.on('setCookie',function(data){fetch(data.url,{method:'POST',credentials:'include',headers:{'Content-Type':'application/json'},body:JSON.stringify({token:data.token})})})}`)

	pState.handler = handler
	pState.script = script
	pState.handlerURL = handler.URL()

	providerCtx.UseProvider(ctx, pState)

	nodes := work.ItemsToNodes(children)
	return &work.Fragment{Children: nodes}
})

var Render = runtime.Component(func(ctx *runtime.Ctx, _ []work.Item) work.Node {
	pState := providerCtx.UseContextValue(ctx)
	if pState == nil {
		return nil
	}

	scriptNode := &work.Element{
		Tag: "script",
	}

	pState.script.AttachTo(scriptNode)
	return scriptNode
})

func useProviderState(ctx *runtime.Ctx) *providerState {
	return providerCtx.UseContextValue(ctx)
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
		"url":   pState.handlerURL,
		"token": token,
	})
}
