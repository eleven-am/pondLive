package router

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/dom"
	render "github.com/eleven-am/pondlive/go/internal/render"
	"github.com/eleven-am/pondlive/go/internal/route"
	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type routerIntegrationProps struct{}

var lastUserParam string
var capturedStore *RouterStore

func routerUserPage(ctx runtime.Ctx, match Match) h.Node {
	if match.Params["id"] != "" {
		lastUserParam = match.Params["id"]
	}
	return h.Div()
}

func routerUseParamsPage(ctx runtime.Ctx, match Match) h.Node {
	params := UseParams(ctx)
	if params["id"] != "" {
		lastUserParam = params["id"]
	}
	return routerUserPage(ctx, match)
}

func routerUsersApp(ctx runtime.Ctx, _ routerIntegrationProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/users/:id", Component: routerUseParamsPage}),
		),
	)
}

func TestRouter2UseParamsUpdatesAfterNavigation(t *testing.T) {
	lastUserParam = ""
	capturedStore = nil
	storeCaptureHook = func(s *RouterStore) { capturedStore = s }
	defer func() { storeCaptureHook = nil }()
	sess := runtime.NewSession(routerUsersApp, routerIntegrationProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	runtime.InternalSeedSessionLocation(sess, route.ParseHref("/users/1"))
	sess.InitialStructured()
	if capturedStore == nil {
		t.Fatal("expected router store capture")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush error: %v", err)
	}

	if lastUserParam != "1" {
		t.Fatalf("expected initial param 1, got %q", lastUserParam)
	}

	capturedStore.RecordNavigation(NavKindReplace, Location{Path: "/users/2"})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	if lastUserParam != "2" {
		t.Fatalf("expected param to update to 2, got %q", lastUserParam)
	}
}

var searchRenderCount int

func routerSearchComponent(ctx runtime.Ctx, _ Match) h.Node {
	searchRenderCount++
	_, set := UseSearchParam(ctx, "tab")
	return h.Button(
		h.On("click", func(h.Event) h.Updates {
			set([]string{"profile"})
			return nil
		}),
		h.Text("toggle"),
	)
}

func routerSearchApp(ctx runtime.Ctx, _ routerIntegrationProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/settings", Component: routerSearchComponent}),
		),
	)
}

func TestRouter2UseSearchParamSetter(t *testing.T) {
	searchRenderCount = 0
	sess := runtime.NewSession(routerSearchApp, routerIntegrationProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	runtime.InternalSeedSessionLocation(sess, route.ParseHref("/settings?tab=overview"))
	structured := sess.InitialStructured()
	if searchRenderCount != 1 {
		t.Fatalf("expected initial render count 1, got %d", searchRenderCount)
	}

	runtime.InternalClearNavHistory(sess)
	handlerID := findClickHandlerID(structured)
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}

	if searchRenderCount != 2 {
		t.Fatalf("expected render count 2, got %d", searchRenderCount)
	}

	loc := runtime.InternalCurrentLocation(sess)
	if loc.Query.Get("tab") != "profile" {
		t.Fatalf("expected tab to become profile, got %q", loc.Query.Get("tab"))
	}
}

var (
	routerNavLastCount  int
	routerNavLastUserID string
	metadataRenderCount int
)

func routerNavHome(ctx runtime.Ctx, _ Match) h.Node {
	count, setCount := runtime.UseState(ctx, 0)
	return h.Div(
		h.Span(h.Text(fmt.Sprintf("count:%d", count()))),
		h.Button(
			h.On("click", func(h.Event) h.Updates {
				next := count() + 1
				routerNavLastCount = next
				setCount(next)
				Navigate(ctx, fmt.Sprintf("/users/%d", next))
				return nil
			}),
			h.Text("Increment & Navigate"),
		),
	)
}

func routerNavUser(ctx runtime.Ctx, match Match) h.Node {
	routerNavLastUserID = match.Params["id"]
	return h.Div()
}

func routerNavApp(ctx runtime.Ctx, _ routerIntegrationProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/", Component: routerNavHome}),
			Route(ctx, RouteProps{Path: "/users/:id", Component: routerNavUser}),
		),
	)
}

func routerMetadataPage(ctx runtime.Ctx, match Match) h.Node {
	metadataRenderCount++
	runtime.UseMetadata(ctx, &runtime.Meta{
		Title: "User " + match.Params["id"],
		Meta:  []h.MetaTag{{Name: "description", Content: "Profile for user " + match.Params["id"]}},
	})
	return runtime.WithMetadata(
		h.Div(h.Text("metadata")),
		&runtime.Meta{Links: []h.LinkTag{{Rel: "canonical", Href: "/users/" + match.Params["id"]}}},
	)
}

func routerMetadataApp(ctx runtime.Ctx, _ routerIntegrationProps) h.Node {
	return Router(ctx,
		Routes(ctx,
			Route(ctx, RouteProps{Path: "/users/:id", Component: routerMetadataPage}),
		),
	)
}

func TestRouter2NavigateFromEventHandler(t *testing.T) {
	routerNavLastCount = 0
	routerNavLastUserID = ""
	sess := runtime.NewSession(routerNavApp, routerIntegrationProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	runtime.InternalSeedSessionLocation(sess, route.ParseHref("/"))
	structured := sess.InitialStructured()
	runtime.InternalClearNavHistory(sess)

	handlerID := findClickHandlerID(structured)
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}

	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}

	if routerNavLastCount != 1 {
		t.Fatalf("expected count 1, got %d", routerNavLastCount)
	}
	loc := runtime.InternalCurrentLocation(sess)
	if loc.Path != "/users/1" {
		t.Fatalf("expected /users/1, got %q", loc.Path)
	}
	if routerNavLastUserID != "1" {
		t.Fatalf("expected user id 1, got %q", routerNavLastUserID)
	}
	navs := runtime.InternalNavHistory(sess)
	if len(navs) != 1 || navs[0].Path != "/users/1" {
		t.Fatalf("unexpected nav history: %#v", navs)
	}
}

func routerLinkApp(ctx runtime.Ctx, _ routerIntegrationProps) h.Node {
	return Router(ctx, Link(ctx, LinkProps{To: "/same"}, h.Text("Same")))
}

func TestRouter2LinkNoNavigationOnSameHref(t *testing.T) {
	sess := runtime.NewSession(routerLinkApp, routerIntegrationProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	runtime.InternalSeedSessionLocation(sess, route.ParseHref("/same"))
	structured := sess.InitialStructured()
	runtime.InternalClearNavHistory(sess)

	handlerID := findClickHandlerID(structured)
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}
	payload := map[string]any{"currentTarget.href": "/same"}
	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click", Payload: payload}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if len(runtime.InternalNavHistory(sess)) != 0 {
		t.Fatalf("expected no navigation when href unchanged")
	}
}

func TestRouter2LinkTriggersNavigation(t *testing.T) {
	linkComp := func(ctx runtime.Ctx, _ routerIntegrationProps) h.Node {
		return Router(ctx, Link(ctx, LinkProps{To: "/next"}, h.Text("Next")))
	}
	sess := runtime.NewSession(linkComp, routerIntegrationProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	runtime.InternalSeedSessionLocation(sess, route.ParseHref("/"))
	structured := sess.InitialStructured()
	runtime.InternalClearNavHistory(sess)

	handlerID := findClickHandlerID(structured)
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}
	if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click", Payload: map[string]any{"currentTarget.href": "/next"}}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	navs := runtime.InternalNavHistory(sess)
	if len(navs) != 1 || navs[0].Path != "/next" {
		t.Fatalf("expected navigation to /next, got %#v", navs)
	}
}

func TestRouter2UseMetadataMergesRouteAndResult(t *testing.T) {
	metadataRenderCount = 0
	capturedStore = nil
	storeCaptureHook = func(s *RouterStore) { capturedStore = s }
	defer func() { storeCaptureHook = nil }()
	sess := runtime.NewSession(routerMetadataApp, routerIntegrationProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	runtime.InternalSeedSessionLocation(sess, route.ParseHref("/users/42"))
	sess.InitialStructured()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if metadataRenderCount == 0 {
		t.Fatal("expected metadata page to render")
	}
	if capturedStore == nil || capturedStore.Location().Path != "/users/42" {
		t.Fatalf("expected seeded store, got %#v", capturedStore)
	}
	meta := sess.Metadata()
	if meta == nil {
		t.Fatal("expected metadata to be captured")
	}
	if meta.Title != "User 42" {
		t.Fatalf("unexpected title %q", meta.Title)
	}
	if len(meta.Meta) != 1 || meta.Meta[0].Name != "description" {
		t.Fatalf("expected description meta tag, got %+v", meta.Meta)
	}
	if len(meta.Links) != 1 || meta.Links[0].Href != "/users/42" {
		t.Fatalf("expected canonical link, got %+v", meta.Links)
	}
}

func TestRouter2SessionsIsolated(t *testing.T) {
	sessA := runtime.NewSession(routerNavApp, routerIntegrationProps{})
	sessB := runtime.NewSession(routerNavApp, routerIntegrationProps{})
	for _, sess := range []*runtime.ComponentSession{sessA, sessB} {
		sess.SetPatchSender(func([]diff.Op) error { return nil })
	}
	runtime.InternalSeedSessionLocation(sessA, route.ParseHref("/"))
	runtime.InternalSeedSessionLocation(sessB, route.ParseHref("/"))
	structuredA := sessA.InitialStructured()
	sessB.InitialStructured()
	runtime.InternalClearNavHistory(sessA)
	runtime.InternalClearNavHistory(sessB)
	handler := findClickHandlerID(structuredA)
	if handler == "" {
		t.Fatal("expected click handler id")
	}
	if err := sessA.DispatchEvent(handler, dom.Event{Name: "click"}); err != nil {
		t.Fatalf("dispatch error: %v", err)
	}
	if len(runtime.InternalNavHistory(sessA)) != 1 {
		t.Fatalf("expected nav history for session A, got %d", len(runtime.InternalNavHistory(sessA)))
	}
	if len(runtime.InternalNavHistory(sessB)) != 0 {
		t.Fatalf("expected no nav history for session B, got %d", len(runtime.InternalNavHistory(sessB)))
	}
}

// Helpers copied from runtime router tests.

func findClickHandlerID(structured render.Structured) string {
	return findHandlerAttr(structured, "data-onclick")
}

func findHandlerAttr(structured render.Structured, attr string) string {
	event := strings.TrimPrefix(attr, "data-on")
	if idx := strings.IndexByte(event, '-'); idx != -1 {
		event = event[:idx]
	}
	event = strings.TrimSpace(event)
	if event != "" {
		for _, binding := range structured.Bindings {
			if binding.Event == event && binding.Handler != "" {
				return binding.Handler
			}
		}
	}
	return ""
}
