package router_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func label(name string) func() work.Node {
	return runtime.Component(func(ctx *runtime.Ctx) work.Node {
		r := router.UseRouter(ctx)
		text := runtime.UseComputed(ctx, func() string {
			params := r.Params.Get()
			keys := make([]string, 0, len(params))
			for k, v := range params {
				keys = append(keys, k+"="+v)
			}
			sortStrings(keys)
			return name + "[" + strings.Join(keys, ",") + "]"
		})
		return work.BuildElement("p", work.TextOf(text))
	})
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}

func rendered(h *harness) string { return h.Normalized() }

func TestTheMostSpecificRouteWinsInAnyOrder(t *testing.T) {
	orders := [][]string{{"fallback", "id", "new", "root"}, {"root", "new", "id", "fallback"}, {"id", "fallback", "root", "new"}}
	defs := map[string]router.RouteDef{
		"fallback": router.Route("*", label("fallback")),
		"id":       router.Route("/users/:id", label("id")),
		"new":      router.Route("/users/new", label("new")),
		"root":     router.Route("/", label("root")),
	}
	cases := map[string]string{
		"/users/new": "new[]",
		"/users/5":   "id[id=5]",
		"/zzz":       "fallback[]",
		"/":          "root[]",
	}
	for _, order := range orders {
		for path, want := range cases {
			h := start(t, path, false, func() work.Node {
				list := make([]router.RouteDef, len(order))
				for i, k := range order {
					list[i] = defs[k]
				}
				return router.Routes(list...)
			})
			if !strings.Contains(h.SSR, ">"+want+"<") || strings.Count(h.SSR, "<p") != 1 {
				t.Errorf("order %v path %s: want %s in %s", order, path, want, h.SSR)
			}
		}
	}
}

func TestAnExactPathAndASlashStarPathAreDifferentRoutes(t *testing.T) {
	for _, order := range [][]string{{"exact", "star"}, {"star", "exact"}} {
		defs := map[string]router.RouteDef{"exact": router.Route("/x", label("exact")), "star": router.Route("/x/*", label("star"))}
		for path, want := range map[string]string{"/x": "exact[]", "/x/y": "star[]", "/x/y/z": "star[]"} {
			h := start(t, path, false, func() work.Node { return router.Routes(defs[order[0]], defs[order[1]]) })
			if !strings.Contains(h.SSR, ">"+want+"<") {
				t.Errorf("order %v path %s: want %s in %s", order, path, want, h.SSR)
			}
		}
	}
	h := start(t, "/x/y", false, func() work.Node { return router.Routes(router.Route("/x", label("exact"))) })
	if strings.Contains(h.SSR, "<p") {
		t.Fatalf("an exact path must not match a longer path: %s", h.SSR)
	}
	h = start(t, "/x", false, func() work.Node { return router.Routes(router.Route("/x/*", label("star"))) })
	if !strings.Contains(h.SSR, ">star[]<") {
		t.Fatalf("/x/* must also match /x: %s", h.SSR)
	}
}

type layoutProbe struct {
	mounts int
	router *router.Router
	count  *runtime.Signal[int]
}

func orgLayout(lp *layoutProbe, inner func() work.Node) func() work.Node {
	return runtime.Component(func(ctx *runtime.Ctx) work.Node {
		mounted(ctx, &lp.mounts)
		lp.router = router.UseRouter(ctx)
		lp.count = runtime.UseSignal(ctx, 0)
		shown := runtime.UseComputed(ctx, func() string { return fmt.Sprint(lp.count.Get()) })
		org := runtime.UseComputed(ctx, func() string { return "org=" + lp.router.Params.Get()["org"] })
		return work.BuildElement("div", work.Attribute("id", "layout"),
			work.BuildElement("b", work.TextOf(shown)),
			work.BuildElement("i", work.TextOf(org)),
			inner(),
		)
	})
}

func nested() func() work.Node {
	return func() work.Node {
		return router.Routes(
			router.Route("/members/:id", label("member")),
			router.Route("/settings", label("settings")),
			router.Route("*", label("inner-missing")),
		)
	}
}

func TestNestedRelativeRoutesFirstRenderAndParamsMergedAcrossLevels(t *testing.T) {
	var lp layoutProbe
	h := start(t, "/orgs/acme/members/7", false, func() work.Node {
		return router.Routes(
			router.Route("/orgs/:org/*", orgLayout(&lp, nested())),
			router.Route("*", label("outer-missing")),
		)
	})
	for _, want := range []string{`<i`, ">org=acme<", ">member[id=7,org=acme]<"} {
		if !strings.Contains(h.SSR, want) {
			t.Fatalf("missing %q in %s", want, h.SSR)
		}
	}
	if strings.Contains(h.SSR, "missing") {
		t.Fatalf("a fallback rendered for a path that matches: %s", h.SSR)
	}
}

func TestTheLayoutStaysMountedAndKeepsItsStateAcrossInnerNavigation(t *testing.T) {
	var lp layoutProbe
	h := start(t, "/orgs/acme/members/7", true, func() work.Node {
		return router.Routes(router.Route("/orgs/:org/*", orgLayout(&lp, nested())))
	})
	h.Do(func() { lp.count.Set(41) })
	h.Do(func() { lp.router.Navigate("/orgs/acme/settings") })
	if !strings.Contains(rendered(h), ">settings[org=acme]<") || strings.Contains(rendered(h), "member[") {
		t.Fatalf("dom = %s", rendered(h))
	}
	h.Do(func() { lp.router.Navigate("/orgs/acme/members/9") })
	h.Do(func() { lp.router.Navigate("/orgs/globex/members/9") })
	n := rendered(h)
	if !strings.Contains(n, ">member[id=9,org=globex]<") || !strings.Contains(n, ">org=globex<") {
		t.Fatalf("dom = %s", n)
	}
	if !strings.Contains(n, ">41<") {
		t.Fatalf("the layout lost its state: %s", n)
	}
	if lp.mounts != 1 {
		t.Fatalf("the layout was remounted %d times", lp.mounts)
	}
}

func TestEachLevelHasItsOwnFallback(t *testing.T) {
	var lp layoutProbe
	build := func() work.Node {
		return router.Routes(
			router.Route("/orgs/:org/*", orgLayout(&lp, nested())),
			router.Route("*", label("outer-missing")),
		)
	}
	h := start(t, "/nope", false, build)
	if !strings.Contains(h.SSR, ">outer-missing[]<") || strings.Contains(h.SSR, `id="layout"`) {
		t.Fatalf("outer fallback: %s", h.SSR)
	}
	h = start(t, "/orgs/acme/zzz/qqq", false, build)
	if !strings.Contains(h.SSR, `id="layout"`) || !strings.Contains(h.SSR, ">inner-missing[org=acme]<") || strings.Contains(h.SSR, "outer-missing") {
		t.Fatalf("inner fallback: %s", h.SSR)
	}
}

func TestLiveNavigationAcrossLevelsSwapsOnlyWhatChanged(t *testing.T) {
	var lp layoutProbe
	var outer probe
	h := start(t, "/orgs/acme/settings", true, func() work.Node {
		return router.Routes(
			router.Route("/orgs/:org/*", orgLayout(&lp, nested())),
			router.Route("/home", page("home", &outer)),
			router.Route("*", label("outer-missing")),
		)
	})
	h.Do(func() { lp.router.Navigate("/home") })
	if strings.Contains(rendered(h), `id="layout"`) || !strings.Contains(rendered(h), "home:") {
		t.Fatalf("dom = %s", rendered(h))
	}
	h.Do(func() { outer.router.Navigate("/orgs/acme/members/3") })
	if !strings.Contains(rendered(h), ">member[id=3,org=acme]<") || lp.mounts != 2 {
		t.Fatalf("mounts=%d dom = %s", lp.mounts, rendered(h))
	}
	h.Do(func() { outer.router.Navigate("/gone") })
	if !strings.Contains(rendered(h), ">outer-missing[]<") {
		t.Fatalf("dom = %s", rendered(h))
	}
}

func TestBackAndForwardReRouteInsideTheMountedLayout(t *testing.T) {
	var lp layoutProbe
	h := start(t, "/orgs/acme/members/1", true, func() work.Node {
		return router.Routes(router.Route("/orgs/:org/*", orgLayout(&lp, nested())))
	})
	h.Do(func() { lp.count.Set(5) })
	h.Do(func() { lp.router.Navigate("/orgs/acme/settings") })
	pop := func(path string) {
		h.Do(func() {
			h.S.Bus.Publish(protocol.RouteHandler, string(protocol.RouterPopstateAction), protocol.RouterNavPayload{Path: path})
		})
	}
	pop("/orgs/acme/members/1")
	if !strings.Contains(rendered(h), ">member[id=1,org=acme]<") || strings.Contains(rendered(h), "settings[") {
		t.Fatalf("back: %s", rendered(h))
	}
	pop("/orgs/acme/settings")
	if !strings.Contains(rendered(h), ">settings[org=acme]<") {
		t.Fatalf("forward: %s", rendered(h))
	}
	if lp.mounts != 1 || !strings.Contains(rendered(h), ">5<") {
		t.Fatalf("mounts=%d dom = %s", lp.mounts, rendered(h))
	}
	h.Do(func() { lp.router.Back(); lp.router.Forward() })
	if got := h.acts; len(got) < 2 || got[len(got)-2] != string(protocol.RouterBackAction) || got[len(got)-1] != string(protocol.RouterForwardAction) {
		t.Fatalf("history commands = %v", got)
	}
}

func TestMatchStateDescribesTheRouteAtItsLevel(t *testing.T) {
	var lp layoutProbe
	var match *router.MatchState
	probeMatch := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		r := router.UseRouter(ctx)
		runtime.UseEffect(ctx, func() func() { match = r.Match.Peek(); return nil })
		return &work.Fragment{}
	})
	start(t, "/orgs/acme/members/7", true, func() work.Node {
		return router.Routes(router.Route("/orgs/:org/*", orgLayout(&lp, func() work.Node {
			return router.Routes(router.Route("/members/:id", probeMatch))
		})))
	})
	if match == nil || !match.Matched || match.Pattern != "/members/:id" || match.Path != "/orgs/acme/members/7" || match.Params["id"] != "7" || match.Params["org"] != "acme" {
		t.Fatalf("match = %+v", match)
	}
}
