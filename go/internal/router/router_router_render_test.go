package router

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"

	h "github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/render"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type routerRenderProps struct{}

func settingsLayout(ctx Ctx, _ Match) h.Node {
	return testDiv(
		testH1(h.Text("Settings")),
		asItem(Outlet(ctx)),
	)
}

func settingsPage(label string) Component[Match] {
	return func(ctx Ctx, _ Match) h.Node {
		return testDiv(h.Text(label))
	}
}

func settingsApp(ctx Ctx, _ routerRenderProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/settings/*", Component: settingsLayout},
				Routes(ctx,
					Route(ctx, RouteProps{Path: "./profile", Component: settingsPage("Profile")}),
					Route(ctx, RouteProps{Path: "./security", Component: settingsPage("Security")}),
				),
			),
		),
	)
}

func linkApp(ctx Ctx, _ routerRenderProps) h.Node {
	return Router(ctx,
		RouterLink(ctx, LinkProps{To: "/same"}, h.Text("Same")),
	)
}

func nestedLinkApp(ctx Ctx, _ routerRenderProps) h.Node {
	return Router(ctx,
		testNav(
			testDiv(
				RouterLink(ctx, LinkProps{To: "/nested"}, h.Text("Nested")),
			),
		),
	)
}

func seededSSRApp(ctx Ctx, _ routerRenderProps) h.Node {
	return Router(ctx,

		Routes(ctx,
			Route(ctx, RouteProps{Path: "/", Component: seededPage("home")}),
			Route(ctx, RouteProps{Path: "/about", Component: seededPage("about")}),
		),
	)
}

func seededNavigation(ctx Ctx, _ struct{}) h.Node {
	loc := UseLocation(ctx)
	return testDiv(h.Text("nav:" + loc.Path))
}

func seededPage(label string) Component[Match] {
	return func(ctx Ctx, _ Match) h.Node {
		loc := UseLocation(ctx)
		return testDiv(h.Text("page:" + label + ":" + loc.Path))
	}
}

func asItem(node h.Node) h.Item {
	if item, ok := node.(h.Item); ok {
		return item
	}
	return h.Fragment()
}

func TestLinkNoNavigationForSameHref(t *testing.T) {
	sess := runtime.NewSession(linkApp, routerRenderProps{})
	var ops []diff.Op
	sess.SetPatchSender(func(o []diff.Op) error {
		ops = append(ops, o...)
		return nil
	})

	InternalSeedSessionLocation(sess, ParseHref("/same"))
	structured := sess.InitialStructured()

	clearNavHistory(sess)
	ops = nil

	handlerID := findClickHandlerID(structured)
	if handlerID == "" {
		t.Fatal("expected click handler to be registered")
	}

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}

	if len(ops) != 0 {
		t.Fatalf("expected no patch ops, got %d", len(ops))
	}
	if navs := navHistory(sess); len(navs) != 0 {
		t.Fatalf("expected no nav history, got %v", navs)
	}
}

func TestRouterRendersNestedLinkDuringSSR(t *testing.T) {
	sess := runtime.NewSession(nestedLinkApp, routerRenderProps{})

	InternalSeedSessionLocation(sess, ParseHref("/root"))

	node := sess.RenderNode()
	html := render.RenderHTML(node)

	if !strings.Contains(html, "<a ") {
		t.Fatalf("expected anchor element in SSR output, got %q", html)
	}
	if !strings.Contains(html, "href=\"/nested\"") {
		t.Fatalf("expected nested link href in SSR output, got %q", html)
	}
	if strings.Contains(html, "data-onclick") {
		t.Fatalf("expected sanitized SSR output without inline handler metadata, got %q", html)
	}
	structured, _ := render.ToStructuredWithOptions(node, render.StructuredOptions{
		Promotions: sess,
		Components: sess,
	})
	var found bool
	for _, binding := range structured.Bindings {
		if binding.Event == "click" && binding.Handler != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected click handler binding to be recorded, got %+v", structured.Bindings)
	}
}

func TestRouterUsesSeededLocationDuringSSR(t *testing.T) {
	session := runtime.NewLiveSession(runtime.SessionID("seeded"), 1, seededSSRApp, routerRenderProps{}, nil)

	InternalSeedSessionLocation(session.ComponentSession(), ParseHref("/about"))

	node := session.RenderRoot()
	html := render.RenderHTML(node)

	if !strings.Contains(html, "page:about:/about") {
		t.Fatalf("expected active route to render seeded location, got %q", html)
	}
	if strings.Contains(html, "page:home:/") {
		t.Fatalf("expected home route to be inactive, got %q", html)
	}
}

func TestRouterLinkRendersStaticAnchorWithoutRouter(t *testing.T) {
	sess := runtime.NewSession(func(ctx Ctx, _ routerRenderProps) h.Node {
		return RouterLink(ctx, LinkProps{To: "../settings"}, h.Text("Settings"))
	}, routerRenderProps{})

	InternalSeedSessionLocation(sess, ParseHref("/users/123/"))

	node := sess.RenderNode()
	html := render.RenderHTML(node)

	if !strings.Contains(html, "<a ") {
		t.Fatalf("expected fallback anchor in SSR output, got %q", html)
	}
	if !strings.Contains(html, "href=\"/settings\"") {
		t.Fatalf("expected fallback anchor href to resolve relative path, got %q", html)
	}
	if !strings.Contains(html, ">Settings<") {
		t.Fatalf("expected fallback anchor text, got %q", html)
	}
}
