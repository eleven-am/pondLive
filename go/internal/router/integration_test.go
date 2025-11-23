package router

import (
	"net/url"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Test component that renders match information
func testComponent(label string) runtime.Component[Match] {
	return func(ctx runtime.Ctx, match Match) *dom.StructuredNode {
		node := dom.ElementNode("div")
		node.Attrs = map[string][]string{
			"data-component": {label},
			"data-path":      {match.Path},
			"data-pattern":   {match.Pattern},
		}
		return node
	}
}

func TestRoutes_BasicMatching(t *testing.T) {

	reqController := headers.NewRequestController()
	reqController.SetInitialLocation("/users/123", nil, "")

	var finalMatchState *MatchState

	rootComponent := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return headers.ProvideRequestController(ctx, reqController, func(headersCtx runtime.Ctx) *dom.StructuredNode {
			return ProvideRouter(headersCtx, func(routerCtx runtime.Ctx) *dom.StructuredNode {

				routes := Routes(routerCtx, RoutesProps{Outlet: "default"},
					Route(routerCtx, RouteProps{
						Path:      "/users/:id",
						Component: testComponent("user-detail"),
					}),
					Route(routerCtx, RouteProps{
						Path:      "/posts",
						Component: testComponent("post-list"),
					}),
				)

				controller := useRouterController(routerCtx)
				if controller != nil {
					finalMatchState = controller.GetMatch()
				}

				outlet := Outlet(routerCtx)

				container := dom.ElementNode("div")
				container.WithChildren(routes, outlet)
				return container
			})
		})
	}

	sess := runtime.NewSession(rootComponent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if finalMatchState == nil {
		t.Fatal("expected match state to be set")
	}
	if !finalMatchState.Matched {
		t.Error("expected route to match")
	}
	if finalMatchState.Pattern != "/users/:id" {
		t.Errorf("expected pattern /users/:id, got %q", finalMatchState.Pattern)
	}
	if finalMatchState.Params["id"] != "123" {
		t.Errorf("expected id=123, got %q", finalMatchState.Params["id"])
	}
}

func TestRoutes_NoMatch(t *testing.T) {
	reqController := headers.NewRequestController()
	reqController.SetInitialLocation("/unknown", nil, "")

	var finalMatchState *MatchState

	rootComponent := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return headers.ProvideRequestController(ctx, reqController, func(headersCtx runtime.Ctx) *dom.StructuredNode {
			return ProvideRouter(headersCtx, func(routerCtx runtime.Ctx) *dom.StructuredNode {
				controller := useRouterController(routerCtx)
				if controller != nil {
					finalMatchState = controller.GetMatch()
				}

				return Routes(routerCtx, RoutesProps{Outlet: "default"},
					Route(routerCtx, RouteProps{
						Path:      "/users/:id",
						Component: testComponent("user-detail"),
					}),
				)
			})
		})
	}

	sess := runtime.NewSession(rootComponent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if finalMatchState == nil {
		t.Fatal("expected match state to be set")
	}
	if finalMatchState.Matched {
		t.Error("expected no match")
	}
}

func TestLink_HrefGeneration(t *testing.T) {
	reqController := headers.NewRequestController()
	reqController.SetInitialLocation("/users", url.Values{"page": []string{"1"}}, "")

	var linkNode *dom.StructuredNode

	rootComponent := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return headers.ProvideRequestController(ctx, reqController, func(headersCtx runtime.Ctx) *dom.StructuredNode {
			return ProvideRouter(headersCtx, func(routerCtx runtime.Ctx) *dom.StructuredNode {
				linkNode = Link(routerCtx, LinkProps{
					To:      "/posts/456",
					Replace: false,
				}, dom.TextNode("View Post"))
				return linkNode
			})
		})
	}

	sess := runtime.NewSession(rootComponent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if linkNode == nil {
		t.Fatal("expected link node")
	}

	if linkNode.Tag != "a" {
		t.Fatalf("expected 'a' element, got %q", linkNode.Tag)
	}

	hrefs, ok := linkNode.Attrs["href"]
	if !ok || len(hrefs) == 0 {
		t.Fatal("expected href attribute")
	}

	if hrefs[0] != "/posts/456" {
		t.Errorf("expected href /posts/456, got %q", hrefs[0])
	}

	if linkNode.Router == nil {
		t.Fatal("expected router metadata")
	}

	if linkNode.Router.PathValue != "/posts/456" {
		t.Errorf("expected router path /posts/456, got %q", linkNode.Router.PathValue)
	}
}

func TestLink_RelativeNavigation(t *testing.T) {
	reqController := headers.NewRequestController()
	reqController.SetInitialLocation("/users/123", nil, "")

	var linkNode *dom.StructuredNode

	rootComponent := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return headers.ProvideRequestController(ctx, reqController, func(headersCtx runtime.Ctx) *dom.StructuredNode {
			return ProvideRouter(headersCtx, func(routerCtx runtime.Ctx) *dom.StructuredNode {
				linkNode = Link(routerCtx, LinkProps{
					To: "./edit",
				}, dom.TextNode("Edit"))
				return linkNode
			})
		})
	}

	sess := runtime.NewSession(rootComponent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if linkNode == nil {
		t.Fatal("expected link node")
	}

	hrefs := linkNode.Attrs["href"]

	if hrefs[0] != "/users/edit" {
		t.Errorf("expected href /users/edit, got %q", hrefs[0])
	}
}

func TestController_GetLocation(t *testing.T) {
	reqController := headers.NewRequestController()
	reqController.SetInitialLocation("/test", url.Values{"key": []string{"value"}}, "section")

	matchState := &MatchState{Matched: false}
	controller := NewController(reqController, func() *MatchState { return matchState }, func(m *MatchState) {})

	loc := controller.GetLocation()

	if loc.Path != "/test" {
		t.Errorf("expected path /test, got %q", loc.Path)
	}
	if loc.Query.Get("key") != "value" {
		t.Errorf("expected query key=value, got %q", loc.Query.Get("key"))
	}
	if loc.Hash != "section" {
		t.Errorf("expected hash section, got %q", loc.Hash)
	}
}

func TestController_SetMatch(t *testing.T) {
	reqController := headers.NewRequestController()
	reqController.SetInitialLocation("/", nil, "")

	matchState := &MatchState{Matched: false}
	setMatchState := func(m *MatchState) {
		matchState = m
	}

	controller := NewController(reqController, func() *MatchState { return matchState }, setMatchState)

	params := map[string]string{"id": "123"}
	controller.SetMatch("/users/:id", params, "/users/123")

	if !matchState.Matched {
		t.Error("expected matched=true")
	}
	if matchState.Pattern != "/users/:id" {
		t.Errorf("expected pattern /users/:id, got %q", matchState.Pattern)
	}
	if matchState.Params["id"] != "123" {
		t.Errorf("expected id=123, got %q", matchState.Params["id"])
	}
	if matchState.Path != "/users/123" {
		t.Errorf("expected path /users/123, got %q", matchState.Path)
	}

	oldMatchState := matchState
	controller.SetMatch("/users/:id", params, "/users/123")
	if matchState != oldMatchState {
		t.Error("expected no update for identical match")
	}
}

func TestController_ClearMatch(t *testing.T) {
	reqController := headers.NewRequestController()
	reqController.SetInitialLocation("/", nil, "")

	matchState := &MatchState{
		Matched: true,
		Pattern: "/users/:id",
		Params:  map[string]string{"id": "123"},
		Path:    "/users/123",
	}
	setMatchState := func(m *MatchState) {
		matchState = m
	}

	controller := NewController(reqController, func() *MatchState { return matchState }, setMatchState)

	controller.ClearMatch()

	if matchState.Matched {
		t.Error("expected matched=false after clear")
	}
	if matchState.Pattern != "" {
		t.Errorf("expected empty pattern after clear, got %q", matchState.Pattern)
	}
	if len(matchState.Params) != 0 {
		t.Error("expected empty params after clear")
	}
}
