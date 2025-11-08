package runtime

import (
	"fmt"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/internal/html"
)

type routerHooksProps struct{}

var lastUserParam string

func userPage(ctx Ctx, match Match) h.Node {
	params := UseParams(ctx)
	lastUserParam = params["id"]
	if id := match.Params["id"]; id != "" {
		lastUserParam = id
	}
	return h.Div()
}

func usersApp(ctx Ctx, _ routerHooksProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/users/:id", Component: userPage}),
		),
	)
}

func TestUseParamsUpdatesAfterNavigation(t *testing.T) {
	lastUserParam = ""
	sess := NewSession(usersApp, routerHooksProps{})
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

func searchComponent(ctx Ctx, _ Match) h.Node {
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

func searchApp(ctx Ctx, _ routerHooksProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/settings", Component: searchComponent}),
		),
	)
}

func TestUseSearchParamSetterTriggersRender(t *testing.T) {
	searchRenderCount = 0
	sess := NewSession(searchApp, routerHooksProps{})
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

var (
	routerNavLastCount  int
	routerNavLastUserID string
)

func routerNavHome(ctx Ctx, _ Match) h.Node {
	count, setCount := UseState(ctx, 0)
	current := count()
	incrementAndNavigate := func(h.Event) h.Updates {
		next := count() + 1
		routerNavLastCount = next
		setCount(next)
		RouterNavigate(ctx, fmt.Sprintf("/users/%d", next))
		return nil
	}
	return h.Div(
		h.Span(h.Text(fmt.Sprintf("count:%d", current))),
		h.Button(
			h.On("click", incrementAndNavigate),
			h.Text("Increment & Navigate to User"),
		),
	)
}

func routerNavUser(ctx Ctx, match Match) h.Node {
	routerNavLastUserID = match.Params["id"]
	return h.Div()
}

func routerNavApp(ctx Ctx, _ routerHooksProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/", Component: routerNavHome}),
			Route(ctx, RouteProps{Path: "/users/:id", Component: routerNavUser}),
		),
	)
}

func TestRouterNavigateFromEventHandler(t *testing.T) {
	routerNavLastCount = 0
	routerNavLastUserID = ""
	sess := NewSession(routerNavApp, routerHooksProps{})
	sess.SetRegistry(handlers.NewRegistry())
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	InternalSeedSessionLocation(sess, ParseHref("/"))
	structured := sess.InitialStructured()

	clearNavHistory(sess)

	handlerID := findClickHandlerID(structured)
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}

	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}

	if routerNavLastCount != 1 {
		t.Fatalf("expected count to update to 1, got %d", routerNavLastCount)
	}

	loc := currentLocation(sess)
	if loc.Path != "/users/1" {
		t.Fatalf("expected current path /users/1, got %q", loc.Path)
	}

	if routerNavLastUserID != "1" {
		t.Fatalf("expected user page to render with id 1, got %q", routerNavLastUserID)
	}

	navs := navHistory(sess)
	if len(navs) != 1 {
		t.Fatalf("expected 1 navigation record, got %d", len(navs))
	}
	if navs[0].Path != "/users/1" {
		t.Fatalf("unexpected navigation target: %+v", navs[0])
	}
}
