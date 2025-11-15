package runtime

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	handlers "github.com/eleven-am/pondlive/go/internal/handlers"
	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type routerRenderProps struct{}

func settingsLayout(ctx Ctx, _ Match) h.Node {
	return h.Div(
		h.H1(h.Text("Settings")),
		asItem(Outlet(ctx)),
	)
}

func settingsPage(label string) Component[Match] {
	return func(ctx Ctx, _ Match) h.Node {
		return h.Div(h.Text(label))
	}
}

func placeholderRouter(ctx Ctx, _ routerRenderProps) h.Node {
	return Router(ctx,
		h.Main(
			h.H1(h.Text("Placeholder Shell")),
			Routes(ctx,
				Route(ctx, RouteProps{Path: "/", Component: placeholderPage("home")}),
				Route(ctx, RouteProps{Path: "/refs", Component: placeholderPage("refs")}),
			),
		),
	)
}

func placeholderPage(label string) Component[Match] {
	return func(ctx Ctx, _ Match) h.Node {
		handler := func(h.Event) h.Updates { return nil }
		return h.Section(
			h.H2(h.Text(label+" route")),
			h.P(h.Text("content:"+label)),
			h.Button(
				h.Type("button"),
				h.On("click", handler),
				h.Text("action "+label),
			),
		)
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
		h.Nav(
			h.Div(
				RouterLink(ctx, LinkProps{To: "/nested"}, h.Text("Nested")),
			),
		),
	)
}

func seededSSRApp(ctx Ctx, _ routerRenderProps) h.Node {
	return Router(ctx,
		Render(ctx, seededNavigation, struct{}{}),
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/", Component: seededPage("home")}),
			Route(ctx, RouteProps{Path: "/about", Component: seededPage("about")}),
		),
	)
}

func seededNavigation(ctx Ctx, _ struct{}) h.Node {
	loc := UseLocation(ctx)
	return h.Div(h.Text("nav:" + loc.Path))
}

func seededPage(label string) Component[Match] {
	return func(ctx Ctx, _ Match) h.Node {
		loc := UseLocation(ctx)
		return h.Div(h.Text("page:" + label + ":" + loc.Path))
	}
}

func asItem(node h.Node) h.Item {
	if item, ok := node.(h.Item); ok {
		return item
	}
	return h.Fragment()
}

func findClickHandlerID(structured render.Structured) handlers.ID {
	return findHandlerAttr(structured, "data-onclick")
}

func TestRouterOutletRerender(t *testing.T) {
	sess := NewSession(settingsApp, routerRenderProps{})
	sess.SetRegistry(handlers.NewRegistry())
	var ops []diff.Op
	sess.SetPatchSender(func(o []diff.Op) error {
		ops = append([]diff.Op{}, o...)
		return nil
	})

	InternalSeedSessionLocation(sess, ParseHref("/settings/profile"))
	sess.InitialStructured()

	clearNavHistory(sess)
	ops = nil

	InternalHandleNav(sess, NavMsg{T: "nav", Path: "/settings/security"})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	update := sess.consumeTemplateUpdate()
	updates := sess.consumeComponentBoots()
	if update != nil {
		if !strings.Contains(update.html, "Security") {
			t.Fatalf("expected template update html to contain Security, got %q", update.html)
		}
		if len(ops) != 0 {
			t.Fatalf("expected no diff ops when template update emitted, got %d", len(ops))
		}
		if len(updates) > 0 {
			found := false
			for _, component := range updates {
				if strings.Contains(component.html, "Security") {
					found = true
					break
				}
			}
			if !found {
				t.Fatalf("expected component boot html to contain Security, got %+v", updates)
			}
		}
		return
	}

	if len(updates) > 0 {
		found := false
		for _, update := range updates {
			if strings.Contains(update.html, "Security") {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("expected component boot html to contain Security, got %+v", updates)
		}
		if len(ops) != 0 {
			t.Fatalf("expected no diff ops when component boot is emitted, got %d", len(ops))
		}
		return
	}

	if len(ops) != 1 {
		t.Fatalf("expected single diff op, got %d", len(ops))
	}
	set, ok := ops[0].(diff.SetText)
	if !ok {
		t.Fatalf("expected SetText op, got %T", ops[0])
	}
	if set.Text != "Security" {
		t.Fatalf("expected Security label, got %q", set.Text)
	}
}

func TestLinkNoNavigationForSameHref(t *testing.T) {
	sess := NewSession(linkApp, routerRenderProps{})
	sess.SetRegistry(handlers.NewRegistry())
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

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}); err != nil {
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
	sess := NewSession(nestedLinkApp, routerRenderProps{})
	sess.SetRegistry(handlers.NewRegistry())

	InternalSeedSessionLocation(sess, ParseHref("/root"))

	node := sess.RenderNode()
	html := render.RenderHTML(node, sess.Registry())

	if !strings.Contains(html, "<a ") {
		t.Fatalf("expected anchor element in SSR output, got %q", html)
	}
	if !strings.Contains(html, "href=\"/nested\"") {
		t.Fatalf("expected nested link href in SSR output, got %q", html)
	}
	if strings.Contains(html, "data-onclick") {
		t.Fatalf("expected sanitized SSR output without inline handler metadata, got %q", html)
	}
	structured, err := render.ToStructuredWithHandlers(node, render.StructuredOptions{Handlers: sess.Registry()})
	if err != nil {
		t.Fatalf("ToStructuredWithHandlers failed: %v", err)
	}
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
	session := NewLiveSession(SessionID("seeded"), 1, seededSSRApp, routerRenderProps{}, nil)

	InternalSeedSessionLocation(session.ComponentSession(), ParseHref("/about"))

	node := session.RenderRoot()
	html := render.RenderHTML(node, session.Registry())

	if !strings.Contains(html, "page:about:/about") {
		t.Fatalf("expected active route to render seeded location, got %q", html)
	}
	if strings.Contains(html, "page:home:/") {
		t.Fatalf("expected home route to be inactive, got %q", html)
	}
}

func TestRouterLinkRendersStaticAnchorWithoutRouter(t *testing.T) {
	sess := NewSession(func(ctx Ctx, _ routerRenderProps) h.Node {
		return RouterLink(ctx, LinkProps{To: "../settings"}, h.Text("Settings"))
	}, routerRenderProps{})
	sess.SetRegistry(handlers.NewRegistry())

	InternalSeedSessionLocation(sess, ParseHref("/users/123/"))

	node := sess.RenderNode()
	html := render.RenderHTML(node, sess.Registry())

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

func TestRouterSSRPreservesDeveloperMarkup(t *testing.T) {
	sess := NewSession(placeholderRouter, routerRenderProps{})
	sess.SetRegistry(handlers.NewRegistry())

	InternalSeedSessionLocation(sess, ParseHref("/refs"))

	node := sess.RenderNode()
	html := render.RenderHTML(node, sess.Registry())

	if !strings.Contains(html, "refs route") {
		t.Fatalf("expected refs route content, got %q", html)
	}
	if strings.Contains(html, "home route") {
		t.Fatalf("expected home route to be inactive, got %q", html)
	}
	if strings.Contains(html, "data-row-key") {
		t.Fatalf("expected SSR output to omit framework data attributes, got %q", html)
	}
	if strings.Contains(html, "data-onclick") {
		t.Fatalf("expected SSR output to omit inline handler metadata, got %q", html)
	}

	structured, err := render.ToStructuredWithHandlers(node, render.StructuredOptions{Handlers: sess.Registry()})
	if err != nil {
		t.Fatalf("ToStructuredWithHandlers failed: %v", err)
	}
	if len(structured.Bindings) == 0 {
		t.Fatalf("expected handler bindings to be recorded, got %+v", structured.Bindings)
	}
	foundClick := false
	for _, binding := range structured.Bindings {
		if binding.Event == "click" {
			foundClick = true
			break
		}
	}
	if !foundClick {
		t.Fatalf("expected click binding, got %+v", structured.Bindings)
	}
	if len(structured.SlotPaths) == 0 {
		t.Fatalf("expected slot paths for dynamic button, got %+v", structured.SlotPaths)
	}
	if len(structured.ComponentPaths) == 0 {
		t.Fatalf("expected component paths to be recorded, got %+v", structured.ComponentPaths)
	}
	if len(structured.Components) < 2 {
		t.Fatalf("expected multiple component spans to be tracked, got %+v", structured.Components)
	}
}
