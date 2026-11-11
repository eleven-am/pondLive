package headers

import (
	"cmp"
	"net/http"
	"net/url"
	"strings"

	"github.com/eleven-am/pondlive/internal/effectguard"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
)

type navOptions struct{ status int }

type NavOption func(*navOptions)

func Status(code int) NavOption { return func(o *navOptions) { o.status = code } }

func (r *Request) Push(href string) {
	r.navigate("req.Push", func() { r.redirect(false, href, http.StatusFound) })
}

func (r *Request) Replace(href string, opts ...NavOption) {
	o := navOptions{status: http.StatusSeeOther}
	for _, opt := range opts {
		opt(&o)
	}
	r.navigate("req.Replace", func() { r.redirect(true, href, o.status) })
}

func (r *Request) Back() {
	r.navigate("req.Back", func() { r.history((*protocol.Bus).PublishRouterBack) })
}

func (r *Request) Forward() {
	r.navigate("req.Forward", func() { r.history((*protocol.Bus).PublishRouterForward) })
}

func (r *Request) navigate(op string, fn func()) {
	if effectguard.Active() && r.root.connected.Peek() && effectguard.Refuse(op) {
		return
	}
	fn()
}

func (r *Request) history(publish func(*protocol.Bus)) {
	root := r.root
	r.onActor(func() {
		if root.bus != nil && root.connected.Peek() {
			publish(root.bus)
		}
	})
}

func (r *Request) reject(href string) {
	runtime.ReportOn(r.ctx, runtime.Report{
		Severity: runtime.SeverityWarning,
		Phase:    "navigation:unsafe-url",
		Code:     "NAVIGATION_UNSAFE_URL",
		Message:  "URL is not an http, https or relative URL",
		Meta:     map[string]any{"href": href},
	})
}

func (r *Request) redirect(replace bool, href string, code int) {
	u, ok := parseRedirect(href)
	if !ok || code < 300 || code > 399 {
		r.reject(href)
		return
	}
	root := r.root
	external := !sameOrigin(u, root.state.snapshot())
	if !external && u.Host != "" {
		local := url.URL{Path: u.Path, RawQuery: u.RawQuery, Fragment: u.Fragment}
		if local.Path == "" {
			local.Path = "/"
		}
		href = local.String()
	}
	r.onActor(func() {
		if root.bus == nil || !root.connected.Peek() {
			if external {
				root.state.SetRedirect(href, code)
				return
			}
			root.state.SetRedirect(route.BuildHref(route.ResolveHref(root.url.Peek(), href)), code)
			return
		}
		if external {
			root.bus.PublishRouterRedirect(protocol.RouterRedirectPayload{URL: href})
			return
		}
		r.apply(replace, href)
	})
}

func parseRedirect(target string) (*url.URL, bool) {
	u, err := url.Parse(target)
	if err != nil || !protocol.SafeRedirect(target, u.Scheme) {
		return nil, false
	}
	return u, true
}

func sameOrigin(u *url.URL, info *RequestInfo) bool {
	if u.Host == "" {
		return true
	}
	if info == nil || info.Host == "" {
		return false
	}
	scheme := info.scheme()
	return originOf(cmp.Or(strings.ToLower(u.Scheme), scheme), u.Host) == originOf(scheme, info.Host)
}

var defaultPorts = map[string]string{"http": "80", "https": "443"}

func originOf(scheme, host string) string {
	parsed := url.URL{Host: host}
	port := parsed.Port()
	if port == "" {
		port = defaultPorts[scheme]
	}
	return scheme + "://" + strings.ToLower(parsed.Hostname()) + ":" + port
}

func (r *Request) onActor(fn func()) {
	if runtime.CtxOnActor(r.ctx) {
		fn()
		return
	}
	runtime.RunOnActor(r.ctx, fn)
}

func (r *Request) apply(replace bool, href string) {
	root := r.root
	target := route.Canonicalize(route.ResolveHref(root.url.Peek(), href))
	payload := protocol.RouterNavPayload{
		Path:    target.Path,
		Query:   target.Query.Encode(),
		Hash:    target.Hash,
		Replace: replace,
	}
	root.setURL(target)
	if replace {
		root.bus.PublishRouterReplace(payload)
		return
	}
	root.bus.PublishRouterPush(payload)
}
