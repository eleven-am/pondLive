package runtime

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	handlers "github.com/eleven-am/pondlive/go/internal/handlers"
	"github.com/eleven-am/pondlive/go/internal/render"
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
	for _, dyn := range structured.D {
		if dyn.Kind != render.DynAttrs {
			continue
		}
		if id := strings.TrimSpace(dyn.Attrs["data-onclick"]); id != "" {
			return handlers.ID(id)
		}
	}
	combined := strings.Join(structured.S, "")
	needle := "data-onclick=\""
	if idx := strings.Index(combined, needle); idx >= 0 {
		start := idx + len(needle)
		if end := strings.Index(combined[start:], "\""); end >= 0 {
			return handlers.ID(combined[start : start+end])
		}
	}
	return ""
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

	switch len(ops) {
	case 1:
		set, ok := ops[0].(diff.SetText)
		if !ok {
			t.Fatalf("expected SetText op, got %T", ops[0])
		}
		if set.Text != "Security" {
			t.Fatalf("expected Security label, got %q", set.Text)
		}
	case 2:
		attrs, ok := ops[0].(diff.SetAttrs)
		if !ok {
			t.Fatalf("expected SetAttrs as first op, got %T", ops[0])
		}
		if attrs.Upsert["data-row-key"] != "/settings/security" {
			t.Fatalf("expected data-row-key to update to /settings/security, got %q", attrs.Upsert["data-row-key"])
		}
		set, ok := ops[1].(diff.SetText)
		if !ok {
			t.Fatalf("expected SetText as second op, got %T", ops[1])
		}
		if set.Text != "Security" {
			t.Fatalf("expected Security label, got %q", set.Text)
		}
	default:
		t.Fatalf("expected 1 or 2 diff ops, got %d", len(ops))
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
	if !strings.Contains(html, "data-onclick") {
		t.Fatalf("expected click handler attribute for nested link, got %q", html)
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
