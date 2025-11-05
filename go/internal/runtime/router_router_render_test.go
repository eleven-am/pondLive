package runtime

import (
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
					Route(ctx, RouteProps{Path: "/settings/profile", Component: settingsPage("Profile")}),
					Route(ctx, RouteProps{Path: "/settings/security", Component: settingsPage("Security")}),
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
		if id, ok := dyn.Attrs["data-onclick"]; ok && id != "" {
			return handlers.ID(id)
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
