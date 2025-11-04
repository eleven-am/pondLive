package router

import (
	"testing"

	"github.com/eleven-am/go/pondlive/internal/diff"
	"github.com/eleven-am/go/pondlive/internal/handlers"
	"github.com/eleven-am/go/pondlive/internal/runtime"
	ui "github.com/eleven-am/go/pondlive/pkg/live"
	h "github.com/eleven-am/go/pondlive/pkg/live/html"
)

var lastUserParam string

func userPage(ctx ui.Ctx, match Match) ui.Node {
	params := UseParams(ctx)
	lastUserParam = params["id"]
	if id := match.Params["id"]; id != "" {
		lastUserParam = id
	}
	return h.Div()
}

func usersApp(ctx ui.Ctx, _ emptyProps) ui.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/users/:id", Component: userPage}),
		),
	)
}

func TestUseParamsUpdatesAfterNavigation(t *testing.T) {
	lastUserParam = ""
	sess := runtime.NewSession(usersApp, emptyProps{})
	sess.SetRegistry(handlers.NewRegistry())
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	InternalSeedSessionLocation(sess, ParseHref("/users/1"))
	sess.InitialStructured()

	if lastUserParam != "1" {
		t.Fatalf("expected initial param 1, got %q", lastUserParam)
	}

	InternalHandleNav(sess, NavMsg{T: "nav", Path: "/users/2"})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	if lastUserParam != "2" {
		t.Fatalf("expected param to update to 2, got %q", lastUserParam)
	}
}

var searchRenderCount int

func searchComponent(ctx ui.Ctx, _ Match) ui.Node {
	searchRenderCount++
	get, set := UseSearchParam(ctx, "tab")
	_ = get
	return h.Button(
		h.On("click", func(h.Event) h.Updates {
			set([]string{"profile"})
			return nil
		}),
		h.Text("toggle"),
	)
}

func searchApp(ctx ui.Ctx, _ emptyProps) ui.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/settings", Component: searchComponent}),
		),
	)
}

func TestUseSearchParamSetterTriggersRender(t *testing.T) {
	searchRenderCount = 0
	sess := runtime.NewSession(searchApp, emptyProps{})
	sess.SetRegistry(handlers.NewRegistry())
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	InternalSeedSessionLocation(sess, ParseHref("/settings?tab=overview"))
	structured := sess.InitialStructured()
	if searchRenderCount != 1 {
		t.Fatalf("expected initial render count 1, got %d", searchRenderCount)
	}

	clearNavHistory(sess)

	handlerID := findClickHandlerID(structured)
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}

	if searchRenderCount != 2 {
		t.Fatalf("expected render count 2, got %d", searchRenderCount)
	}

	loc := currentLocation(sess)
	if loc.Query.Get("tab") != "profile" {
		t.Fatalf("expected search param to update to profile, got %q", loc.Query.Get("tab"))
	}
}
