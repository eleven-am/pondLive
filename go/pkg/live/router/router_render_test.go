package router

import (
	"testing"

	"github.com/eleven-am/go/pondlive/internal/diff"
	handlers "github.com/eleven-am/go/pondlive/internal/handlers"
	"github.com/eleven-am/go/pondlive/internal/render"
	runtime "github.com/eleven-am/go/pondlive/internal/runtime"
	ui "github.com/eleven-am/go/pondlive/pkg/live"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

type emptyProps struct{}

func settingsLayout(ctx ui.Ctx, _ Match) ui.Node {
	return h.Div(
		h.H1(h.Text("Settings")),
		asItem(Outlet(ctx)),
	)
}

func settingsPage(label string) ui.Component[Match] {
	return func(ctx ui.Ctx, _ Match) ui.Node {
		return h.Div(h.Text(label))
	}
}

func settingsApp(ctx ui.Ctx, _ emptyProps) ui.Node {
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

func linkApp(ctx ui.Ctx, _ emptyProps) ui.Node {
	return Router(ctx,
		Link(ctx, LinkProps{To: "/same", Children: []h.Item{h.Text("Same")}}),
	)
}

func asItem(node ui.Node) h.Item {
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
	sess := runtime.NewSession(settingsApp, emptyProps{})
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
	sess := runtime.NewSession(linkApp, emptyProps{})
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
