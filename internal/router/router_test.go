package router_test

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
	"golang.org/x/net/html"
)

type harness struct {
	*runtimetest.App
	req *headers.RequestState

	mu   sync.Mutex
	navs []protocol.RouterNavPayload
	acts []string
}

func newRequest(path string, live bool) *headers.RequestState {
	u, _ := url.Parse(path)
	info := &headers.RequestInfo{Path: u.Path, Query: u.Query(), Headers: http.Header{}}
	req := headers.NewRequestState(info)
	req.SetIsLive(live)
	return req
}

func start(t *testing.T, path string, live bool, body func() work.Node) *harness {
	t.Helper()
	h := &harness{req: newRequest(path, live)}
	root := func(*runtime.Ctx) work.Node {
		return headers.Provider(headers.ProviderProps{
			State: h.req,
			Children: router.Provider(router.ProviderProps{
				Children: work.BuildElement("main", body()),
			}),
		})
	}
	h.App = runtimetest.New(t, root)
	if live {
		h.On(func() { h.S.SetConnected(true) })
		h.Sync()
	}
	h.S.Bus.SubscribeAll(func(topic protocol.Topic, event string, data interface{}) {
		if topic != protocol.RouteHandler {
			return
		}
		h.mu.Lock()
		defer h.mu.Unlock()
		h.acts = append(h.acts, event)
		if payload, ok := data.(protocol.RouterNavPayload); ok {
			h.navs = append(h.navs, payload)
		}
	})
	return h
}

func (h *harness) idOf(tag, attr, contains string) uint32 {
	h.T.Helper()
	for _, id := range h.Model.IDs() {
		n := h.Model.Node(id)
		if n == nil || n.Type != html.ElementNode || n.Data != tag {
			continue
		}
		for _, a := range n.Attr {
			if a.Key == attr && strings.Contains(a.Val, contains) {
				return id
			}
		}
	}
	h.T.Fatalf("no <%s %s~%q> in %s", tag, attr, contains, h.Model.HTML())
	return 0
}

func (h *harness) navigations() []protocol.RouterNavPayload {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]protocol.RouterNavPayload(nil), h.navs...)
}

type probe struct {
	mounts int
	router *router.Router
}

func mounted(ctx *runtime.Ctx, n *int) {
	runtime.UseEffect(ctx, func() func() {
		*n++
		return nil
	})
}

func page(name string, p *probe) func() work.Node {
	return runtime.Component(func(ctx *runtime.Ctx) work.Node {
		mounted(ctx, &p.mounts)
		p.router = router.UseRouter(ctx)
		return work.BuildElement("section", work.Attribute("id", name),
			work.BuildElement("p", work.TextOf(runtime.UseComputed(ctx, func() string {
				return name + ":" + p.router.Params.Get()["id"]
			}))),
		)
	})
}

func TestParamsChangeWithoutRemount(t *testing.T) {
	var user probe
	h := start(t, "/users/1", true, func() work.Node {
		return router.Routes(
			router.Route("/users/:id", page("user", &user)),
		)
	})
	if !strings.Contains(h.SSR, "user:1") {
		t.Fatalf("ssr = %s", h.SSR)
	}
	h.Do(func() { user.router.Navigate("/users/2") })
	if !strings.Contains(h.Normalized(), "user:2") {
		t.Fatalf("dom = %s", h.Normalized())
	}
	if user.mounts != 1 {
		t.Fatalf("component remounted: %d mounts", user.mounts)
	}
	navs := h.navigations()
	if len(navs) != 1 || navs[0].Path != "/users/2" || navs[0].Replace {
		t.Fatalf("navigations = %+v", navs)
	}
}

func TestPatternSwitchRemounts(t *testing.T) {
	var home, about probe
	h := start(t, "/", true, func() work.Node {
		return router.Routes(
			router.Route("/", page("home", &home)),
			router.Route("/about", page("about", &about)),
		)
	})
	h.Do(func() { home.router.Navigate("/about") })
	n := h.Normalized()
	if strings.Contains(n, "home:") || !strings.Contains(n, "about:") {
		t.Fatalf("dom = %s", n)
	}
	h.Do(func() { about.router.Navigate("/") })
	if home.mounts != 2 || about.mounts != 1 {
		t.Fatalf("mounts home=%d about=%d", home.mounts, about.mounts)
	}
}

func TestSetQueryReplaces(t *testing.T) {
	var p probe
	h := start(t, "/?a=1", true, func() work.Node {
		return router.Routes(
			router.Route("/", runtime.Component(func(ctx *runtime.Ctx) work.Node {
				p.router = router.UseRouter(ctx)
				return work.BuildElement("p", work.TextOf(runtime.UseComputed(ctx, func() string {
					return p.router.Query.Get().Get("q")
				})))
			})),
		)
	})
	h.Do(func() { p.router.SetQuery("q", "hello") })
	navs := h.navigations()
	if len(navs) != 1 || !navs[0].Replace || !strings.Contains(navs[0].Query, "q=hello") || !strings.Contains(navs[0].Query, "a=1") {
		t.Fatalf("navigations = %+v", navs)
	}
	if !strings.Contains(h.Normalized(), ">hello<") {
		t.Fatalf("dom = %s", h.Normalized())
	}
	h.Do(func() { p.router.SetQuery("q", "") })
	if got := h.navigations()[1].Query; strings.Contains(got, "q=") {
		t.Fatalf("query after clear = %q", got)
	}
}

func TestSetQueryDoesNotMutateTheValuesAnEarlierPeekReturned(t *testing.T) {
	var p probe
	h := start(t, "/?a=1&q=old", true, func() work.Node {
		return router.Routes(
			router.Route("/", runtime.Component(func(ctx *runtime.Ctx) work.Node {
				p.router = router.UseRouter(ctx)
				return work.BuildElement("p", work.TextOf(runtime.UseComputed(ctx, func() string {
					return p.router.Query.Get().Get("q")
				})))
			})),
		)
	})
	var peeked, location url.Values
	var peekedLocation route.Location
	h.Do(func() {
		peeked = p.router.Query.Peek()
		peekedLocation = p.router.Location.Peek()
		location = peekedLocation.Query
		p.router.SetQuery("q", "new")
		p.router.SetQuery("a", "")
	})
	if got := peeked.Encode(); got != "a=1&q=old" {
		t.Fatalf("the Query map returned by Peek was mutated in place: %q", got)
	}
	if got := location.Encode(); got != "a=1&q=old" {
		t.Fatalf("the Location.Query map returned by Peek was mutated in place: %q", got)
	}
	if got := h.navigations(); len(got) != 2 || got[1].Query != "q=new" {
		t.Fatalf("navigations = %+v", got)
	}
	if !strings.Contains(h.Normalized(), ">new<") {
		t.Fatalf("dom = %s", h.Normalized())
	}
}

func TestARouteBeingLeftNeverSeesTheNextRoutesParams(t *testing.T) {
	var seen []string
	var left probe
	leaving := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		left.router = router.UseRouter(ctx)
		runtime.UseEffect(ctx, func() func() {
			seen = append(seen, fmt.Sprintf("effect:%v/%v", left.router.Params.Get(), left.router.Matched.Get()))
			return nil
		})
		label := runtime.UseComputed(ctx, func() string {
			seen = append(seen, fmt.Sprintf("computed:%v/%v", left.router.Params.Get(), left.router.Matched.Get()))
			return "x"
		})
		return work.BuildElement("section", work.Attribute("id", "leaving"), work.Attribute("data-label", label))
	})
	h := start(t, "/a/1", true, func() work.Node {
		return router.Routes(
			router.Route("/a/:id", leaving),
			router.Route("/b/:other", page("b", &probe{})),
		)
	})
	h.Do(func() { left.router.Navigate("/b/2") })
	for _, s := range seen {
		if strings.Contains(s, "other") || strings.Contains(s, "false") {
			t.Fatalf("the route being left observed %q, which belongs to the next route; saw %v", s, seen)
		}
	}
	if len(seen) == 0 || !strings.Contains(h.Normalized(), "b:") {
		t.Fatalf("seen = %v, dom = %s", seen, h.Normalized())
	}
}

func TestPopstateUpdatesLocationCanonically(t *testing.T) {
	var user probe
	h := start(t, "/users/1", true, func() work.Node {
		return router.Routes(
			router.Route("/users/:id", page("user", &user)),
		)
	})
	h.Do(func() {
		h.S.Bus.Publish(protocol.RouteHandler, string(protocol.RouterPopstateAction), protocol.RouterNavPayload{Path: "/users//5/ ", Query: "", Hash: "##x"})
	})
	if !strings.Contains(h.Normalized(), "user:5") {
		t.Fatalf("dom = %s", h.Normalized())
	}
	var hash string
	h.On(func() { hash = user.router.Hash.Peek() })
	if hash != "x" {
		t.Fatalf("hash = %q", hash)
	}
	if user.mounts != 1 {
		t.Fatalf("mounts = %d", user.mounts)
	}
}

func TestLinkClickNavigatesAndNavLinkMarksActive(t *testing.T) {
	var a, b probe
	h := start(t, "/a", true, func() work.Node {
		return work.BuildElement("div",
			router.Link(router.LinkProps{To: "/b", Children: work.NewText("go b")}),
			router.NavLink(router.NavLinkProps{To: "/a", ClassName: "nav", ActiveClass: "on", End: true, Children: work.NewText("a")}),
			router.NavLink(router.NavLinkProps{To: "/b", ClassName: "nav", ActiveClass: "on", End: true, Children: work.NewText("b")}),
			router.Routes(
				router.Route("/a", page("a", &a)),
				router.Route("/b", page("b", &b)),
			),
		)
	})
	navA, navB := h.idOf("a", "href", "/a"), h.idOf("a", "href", "/b")
	_ = navB
	node := h.Model.Node(navA)
	classOf := func(n *html.Node) string {
		for _, at := range n.Attr {
			if at.Key == "class" {
				return at.Val
			}
		}
		return ""
	}
	if classOf(node) != "nav on" {
		t.Fatalf("active class = %q", classOf(node))
	}
	links := 0
	var link uint32
	for _, id := range h.Model.IDs() {
		n := h.Model.Node(id)
		if n.Type == html.ElementNode && n.Data == "a" {
			for _, at := range n.Attr {
				if at.Key == "href" && at.Val == "/b" && classOf(n) == "" {
					link = id
					links++
				}
			}
		}
	}
	if links != 1 {
		t.Fatalf("plain link count = %d", links)
	}
	if err := h.Dispatch(link, "click", nil); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(h.Normalized(), "b:") || strings.Contains(h.Normalized(), "a:") {
		t.Fatalf("dom = %s", h.Normalized())
	}
	if got := classOf(h.Model.Node(navA)); got != "nav" {
		t.Fatalf("inactive class = %q", got)
	}
	navs := h.navigations()
	if len(navs) != 1 || navs[0].Path != "/b" {
		t.Fatalf("navigations = %+v", navs)
	}
}

func TestRedirectDuringSSRSetsHTTPRedirect(t *testing.T) {
	req := newRequest("/old", false)
	root := func(*runtime.Ctx) work.Node {
		return headers.Provider(headers.ProviderProps{State: req, Children: router.Provider(router.ProviderProps{
			Children: work.BuildElement("main", router.Redirect(router.RedirectProps{To: "/new"})),
		})})
	}
	s := runtime.NewSession(runtime.Config{SessionID: "ssr", Root: root, DevMode: true})
	s.DeferEffects()
	runtimetest.Start(t, s)
	err := s.Ask(5*time.Second, func() error {
		_, err := s.RenderSnapshot()
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	url, code, ok := req.Redirect()
	if !ok || url != "/new" || code != http.StatusSeeOther {
		t.Fatalf("redirect = %q %d %v", url, code, ok)
	}
}

func TestRedirectWhileLiveNavigates(t *testing.T) {
	var home probe
	var show *runtime.Signal[bool]
	h := start(t, "/", true, func() work.Node {
		return work.BuildElement("div",
			runtime.Component(func(ctx *runtime.Ctx) work.Node {
				show = runtime.UseSignal(ctx, false)
				return work.BuildElement("i", runtime.Show(show, func() work.Node {
					return work.BuildElement("b", router.Redirect(router.RedirectProps{To: "/dest"}))
				}))
			})(),
			router.Routes(
				router.Route("/", page("home", &home)),
				router.Route("/dest", page("dest", &probe{})),
			),
		)
	})
	h.Do(func() { show.Set(true) })
	navs := h.navigations()
	if len(navs) != 1 || navs[0].Path != "/dest" || !navs[0].Replace {
		t.Fatalf("navigations = %+v", navs)
	}
	if !strings.Contains(h.Normalized(), "dest:") {
		t.Fatalf("dom = %s", h.Normalized())
	}
}

func TestNavigateWhileNotLiveRedirects(t *testing.T) {
	var p probe
	h := start(t, "/start", false, func() work.Node {
		return router.Routes(
			router.Route("/start", page("s", &p)),
		)
	})
	h.Do(func() { p.router.Navigate("/elsewhere?x=1") })
	url, code, ok := h.req.Redirect()
	if !ok || url != "/elsewhere?x=1" || code != http.StatusFound {
		t.Fatalf("redirect = %q %d %v", url, code, ok)
	}
	if len(h.navigations()) != 0 {
		t.Fatalf("unexpected navigation frames: %+v", h.navigations())
	}
}

func TestBackAndForwardPublishCommands(t *testing.T) {
	var p probe
	h := start(t, "/start", true, func() work.Node {
		return router.Routes(
			router.Route("/start", page("s", &p)),
		)
	})
	h.Do(func() { p.router.Back(); p.router.Forward() })
	h.mu.Lock()
	acts := append([]string(nil), h.acts...)
	h.mu.Unlock()
	if len(acts) != 2 || acts[0] != "back" || acts[1] != "forward" {
		t.Fatalf("acts = %v", acts)
	}
}

func TestMatchedAndParamsOutsideRoute(t *testing.T) {
	var out *router.Router
	h := start(t, "/x", true, func() work.Node {
		return runtime.Component(func(ctx *runtime.Ctx) work.Node {
			out = router.UseRouter(ctx)
			return work.BuildElement("span")
		})()
	})
	var matched bool
	var params map[string]string
	h.On(func() { matched, params = out.Matched.Peek(), out.Params.Peek() })
	if matched || len(params) != 0 {
		t.Fatalf("matched = %v params = %v", matched, params)
	}
}

func TestRouterAndRequestURLFollowNavigateReplaceAndPopstate(t *testing.T) {
	var p probe
	var req *headers.Request
	h := start(t, "/start?x=1", true, func() work.Node {
		return work.BuildElement("div",
			runtime.Component(func(ctx *runtime.Ctx) work.Node {
				req = headers.UseRequest(ctx)
				r := router.UseRouter(ctx)
				p.router = r
				line := runtime.UseComputed(ctx, func() string {
					return r.Path.Get() + "|" + r.Query.Get().Get("x") + "|" + req.URL().Get().Path
				})
				return work.BuildElement("em", work.TextOf(line))
			})(),
			router.Routes(
				router.Route("/start", page("s", &probe{})),
				router.Route("/b/:id", page("b", &probe{})),
				router.Route("/c", page("c", &probe{})),
			),
		)
	})
	if !strings.Contains(h.SSR, ">/start|1|/start<") {
		t.Fatalf("ssr = %s", h.SSR)
	}
	if edits := h.Do(func() { p.router.Navigate("/b/9?x=2") }); len(edits) == 0 {
		t.Fatal("navigate produced no edit frame")
	}
	if n := h.Normalized(); !strings.Contains(n, ">/b/9|2|/b/9<") || !strings.Contains(n, "b:9") {
		t.Fatalf("after navigate: %s", n)
	}
	h.Do(func() { p.router.Replace("/c") })
	if n := h.Normalized(); !strings.Contains(n, ">/c||/c<") || !strings.Contains(n, "c:") || strings.Contains(n, "b:9") {
		t.Fatalf("after replace: %s", n)
	}
	h.Do(func() {
		h.S.Bus.Publish(protocol.RouteHandler, string(protocol.RouterPopstateAction), protocol.RouterNavPayload{Path: "/b/3", Query: "x=7"})
	})
	if n := h.Normalized(); !strings.Contains(n, ">/b/3|7|/b/3<") || !strings.Contains(n, "b:3") {
		t.Fatalf("after popstate: %s", n)
	}
	var cur string
	h.On(func() { cur = req.URL().Peek().Path })
	if cur != "/b/3" || h.req.URL().Path != "/b/3" {
		t.Fatalf("request url = %q / %q", cur, h.req.URL().Path)
	}
}

func TestResetAfterNavigationRendersTheNewRoute(t *testing.T) {
	var p probe
	h := start(t, "/start", true, func() work.Node {
		return router.Routes(
			router.Route("/start", page("s", &probe{})),
			router.Route("/next", page("n", &p)),
		)
	})
	h.Do(func() {
		h.S.Bus.Publish(protocol.RouteHandler, string(protocol.RouterPopstateAction), protocol.RouterNavPayload{Path: "/next"})
	})
	if html := h.ResetHTML(); !strings.Contains(html, "n:") || strings.Contains(html, "s:") {
		t.Fatalf("reset html = %s", html)
	}
}

func TestNavigationFromAnotherGoroutineUpdatesTheURL(t *testing.T) {
	var p probe
	h := start(t, "/start", true, func() work.Node {
		return router.Routes(
			router.Route("/start", page("s", &p)),
			router.Route("/other", page("o", &probe{})),
		)
	})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() { defer wg.Done(); p.router.Navigate("/other") }()
	wg.Wait()
	h.Sync()
	if !strings.Contains(h.Normalized(), "o:") {
		t.Fatalf("dom = %s", h.Normalized())
	}
	if h.req.URL().Path != "/other" {
		t.Fatalf("state url = %+v", h.req.URL())
	}
}

func runScript(t *testing.T, script func(r *router.Router, req *headers.Request)) (frames []string, acts []string, navs []protocol.RouterNavPayload, dom string) {
	t.Helper()
	var r *router.Router
	var req *headers.Request
	h := start(t, "/start", true, func() work.Node {
		return work.BuildElement("div",
			runtime.Component(func(ctx *runtime.Ctx) work.Node {
				req = headers.UseRequest(ctx)
				r = router.UseRouter(ctx)
				line := runtime.UseComputed(ctx, func() string { return req.URL().Get().Path + "?" + req.URL().Get().Query.Encode() })
				return work.BuildElement("em", work.TextOf(line))
			})(),
			router.Routes(
				router.Route("/start", page("s", &probe{})),
				router.Route("/:id", page("p", &probe{})),
			),
		)
	})
	frames = append(frames, fmt.Sprint(h.Do(func() { script(r, req) })))
	h.mu.Lock()
	acts = append([]string(nil), h.acts...)
	h.mu.Unlock()
	return frames, acts, h.navigations(), h.Normalized()
}

func TestRouterMethodsAndRequestMethodsProduceIdenticalFrames(t *testing.T) {
	withQuery := func(k, v string) func(router.Location) router.Location {
		return func(l router.Location) router.Location {
			l.Query.Set(k, v)
			return l
		}
	}
	viaRouter := func(r *router.Router, _ *headers.Request) {
		r.Navigate("/a?x=1")
		r.Replace("/b")
		r.NavigateWith(withQuery("m", "1"))
		r.ReplaceWith(func(l router.Location) router.Location { l.Path = "/d"; return l })
		r.SetQuery("k", "v")
		r.SetQuery("k", "")
		r.Back()
		r.Forward()
		r.Replace("https://other.example/z")
		r.Replace("/c", headers.Status(http.StatusMovedPermanently))
	}
	viaRequest := func(_ *router.Router, req *headers.Request) {
		req.Push("/a?x=1")
		req.Replace("/b")
		req.Push("/b?m=1")
		req.Replace("/d?m=1")
		req.Replace("/d?k=v&m=1")
		req.Replace("/d?m=1")
		req.Back()
		req.Forward()
		req.Replace("https://other.example/z")
		req.Replace("/c", headers.Status(http.StatusMovedPermanently))
	}
	f1, a1, n1, d1 := runScript(t, viaRouter)
	f2, a2, n2, d2 := runScript(t, viaRequest)
	if fmt.Sprint(f1) != fmt.Sprint(f2) || fmt.Sprint(a1) != fmt.Sprint(a2) || fmt.Sprint(n1) != fmt.Sprint(n2) || d1 != d2 {
		t.Fatalf("router and request diverge\nframes %v\n   vs %v\nacts %v vs %v\nnavs %v vs %v\ndom %s\n vs %s", f1, f2, a1, a2, n1, n2, d1, d2)
	}
	if len(a1) != 10 || a1[8] != "redirect" || a1[9] != "replace" {
		t.Fatalf("acts = %v", a1)
	}
}

func TestRedirectComponentPassesItsStatusToReplace(t *testing.T) {
	req := newRequest("/old", false)
	root := func(*runtime.Ctx) work.Node {
		return headers.Provider(headers.ProviderProps{State: req, Children: router.Provider(router.ProviderProps{
			Children: work.BuildElement("main", router.Redirect(router.RedirectProps{To: "/new", Status: http.StatusMovedPermanently})),
		})})
	}
	s := runtime.NewSession(runtime.Config{SessionID: "ssr", Root: root, DevMode: true})
	s.DeferEffects()
	runtimetest.Start(t, s)
	if err := s.Ask(5*time.Second, func() error { _, err := s.RenderSnapshot(); return err }); err != nil {
		t.Fatal(err)
	}
	if url, code, ok := req.Redirect(); !ok || url != "/new" || code != http.StatusMovedPermanently {
		t.Fatalf("redirect = %q %d %v", url, code, ok)
	}
}

func TestRedirectPushIsOptIn(t *testing.T) {
	t.Run("live", func(t *testing.T) {
		var show *runtime.Signal[bool]
		h := start(t, "/", true, func() work.Node {
			return work.BuildElement("div",
				runtime.Component(func(ctx *runtime.Ctx) work.Node {
					show = runtime.UseSignal(ctx, false)
					return work.BuildElement("i", runtime.Show(show, func() work.Node {
						return router.Redirect(router.RedirectProps{To: "/dest", Push: true})
					}))
				})(),
				router.Routes(
					router.Route("/", page("home", &probe{})),
					router.Route("/dest", page("dest", &probe{})),
				),
			)
		})
		h.Do(func() { show.Set(true) })
		navs := h.navigations()
		if len(navs) != 1 || navs[0].Path != "/dest" || navs[0].Replace {
			t.Fatalf("navigations = %+v", navs)
		}
	})
	t.Run("first render", func(t *testing.T) {
		req := newRequest("/old", false)
		root := func(*runtime.Ctx) work.Node {
			return headers.Provider(headers.ProviderProps{State: req, Children: router.Provider(router.ProviderProps{
				Children: work.BuildElement("main", router.Redirect(router.RedirectProps{To: "/new", Push: true})),
			})})
		}
		s := runtime.NewSession(runtime.Config{SessionID: "ssr-push", Root: root, DevMode: true})
		s.DeferEffects()
		runtimetest.Start(t, s)
		if err := s.Ask(5*time.Second, func() error { _, err := s.RenderSnapshot(); return err }); err != nil {
			t.Fatal(err)
		}
		if url, code, ok := req.Redirect(); !ok || url != "/new" || code != http.StatusFound {
			t.Fatalf("redirect = %q %d %v", url, code, ok)
		}
	})
}

func TestRedirectDefaultsToReplaceSoBackIsNotTrapped(t *testing.T) {
	var show *runtime.Signal[bool]
	h := start(t, "/", true, func() work.Node {
		return work.BuildElement("div",
			runtime.Component(func(ctx *runtime.Ctx) work.Node {
				show = runtime.UseSignal(ctx, false)
				return work.BuildElement("i", runtime.Show(show, func() work.Node {
					return router.Redirect(router.RedirectProps{To: "/dest"})
				}))
			})(),
			router.Routes(
				router.Route("/", page("home", &probe{})),
				router.Route("/dest", page("dest", &probe{})),
			),
		)
	})
	h.Do(func() { show.Set(true) })
	navs := h.navigations()
	if len(navs) != 1 || !navs[0].Replace {
		t.Fatalf("navigations = %+v", navs)
	}
}
