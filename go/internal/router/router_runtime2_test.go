package router

import (
	"net/url"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Helper to create a test controller with given location
func testController(loc Location) *Controller {
	state := &State{Location: loc}
	return NewController(
		func() *State { return state },
		func(s *State) { state = s },
	)
}

// Test route matching and rendering
func TestRouteMatching(t *testing.T) {
	var matchedRoute string

	homePage := func(ctx Ctx, _ Match) *dom.StructuredNode {
		matchedRoute = "home"
		return dom.TextNode("Home")
	}

	userPage := func(ctx Ctx, _ Match) *dom.StructuredNode {
		matchedRoute = "user"
		return dom.TextNode("User")
	}

	entries := []routeEntry{
		{pattern: "/", component: homePage},
		{pattern: "/users/:id", component: userPage},
	}

	matchedRoute = ""
	initialLoc := Location{Path: "/"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries))
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if matchedRoute != "home" {
		t.Errorf("expected home route to match, got %q", matchedRoute)
	}

	matchedRoute = ""
	initialLoc2 := Location{Path: "/users/123"}
	controller2 := testController(initialLoc2)

	appFunc2 := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller2, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries))
		})
	}

	sess2 := runtime.NewSession(appFunc2, struct{}{})
	sess2.Flush()

	if matchedRoute != "user" {
		t.Errorf("expected user route to match, got %q", matchedRoute)
	}
}

// Test UseLocation provides correct location
func TestUseLocationContext(t *testing.T) {
	testPath := "/test/path"
	testQuery := url.Values{"foo": {"bar"}}
	testHash := "section"

	initialLoc := Location{
		Path:  testPath,
		Query: testQuery,
		Hash:  testHash,
	}

	var capturedLoc Location

	page := func(ctx Ctx, _ Match) *dom.StructuredNode {
		capturedLoc = UseLocation(ctx)
		return dom.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/test/path", component: page},
	}

	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return LocationCtx.Provide(rctx, initialLoc, func(lctx Ctx) *dom.StructuredNode {
				return renderRoutes(lctx, buildTrie(entries))
			})
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if capturedLoc.Path != testPath {
		t.Errorf("expected path %q, got %q", testPath, capturedLoc.Path)
	}

	if capturedLoc.Hash != testHash {
		t.Errorf("expected hash %q, got %q", testHash, capturedLoc.Hash)
	}

	if capturedLoc.Query.Get("foo") != "bar" {
		t.Errorf("expected query foo=bar, got %q", capturedLoc.Query.Get("foo"))
	}
}

// Test UseParams extracts route parameters
func TestUseParamsExtractsParams(t *testing.T) {
	var capturedID string
	var capturedSlug string

	page := func(ctx Ctx, _ Match) *dom.StructuredNode {
		params := UseParams(ctx)
		capturedID = params["id"]
		capturedSlug = params["slug"]
		return dom.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/posts/:id/:slug", component: page},
	}

	initialLoc := Location{Path: "/posts/42/hello-world"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries))
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if capturedID != "42" {
		t.Errorf("expected id '42', got %q", capturedID)
	}

	if capturedSlug != "hello-world" {
		t.Errorf("expected slug 'hello-world', got %q", capturedSlug)
	}
}

// Test UseSearchParam hook
func TestUseSearchParamReturnsValues(t *testing.T) {
	var capturedValues []string

	page := func(ctx Ctx, _ Match) *dom.StructuredNode {
		get, _ := UseSearchParam(ctx, "tab")
		capturedValues = get()
		return dom.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/settings", component: page},
	}

	initialLoc := Location{
		Path:  "/settings",
		Query: url.Values{},
	}
	initialLoc.Query.Set("tab", "profile")
	initialLoc.Query.Add("tab", "security")

	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries))
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if len(capturedValues) != 2 {
		t.Fatalf("expected 2 values, got %d", len(capturedValues))
	}

	if capturedValues[0] != "profile" {
		t.Errorf("expected first value 'profile', got %q", capturedValues[0])
	}

	if capturedValues[1] != "security" {
		t.Errorf("expected second value 'security', got %q", capturedValues[1])
	}
}

// Test route specificity (param routes beat wildcard routes)
func TestRouteSpecificity(t *testing.T) {
	var matchedRoute string

	wildcardPage := func(ctx Ctx, _ Match) *dom.StructuredNode {
		matchedRoute = "wildcard"
		return dom.FragmentNode()
	}

	paramPage := func(ctx Ctx, _ Match) *dom.StructuredNode {
		matchedRoute = "param"
		return dom.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/items/*", component: wildcardPage},
		{pattern: "/items/:id", component: paramPage},
	}

	initialLoc := Location{Path: "/items/123"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries))
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if matchedRoute != "param" {
		t.Errorf("expected param route (higher specificity), got %q", matchedRoute)
	}
}

// Test no route match returns fragment
func TestNoRouteMatchReturnsFragment(t *testing.T) {
	homePage := func(ctx Ctx, _ Match) *dom.StructuredNode {
		return dom.TextNode("Home")
	}

	entries := []routeEntry{
		{pattern: "/", component: homePage},
	}

	initialLoc := Location{Path: "/404"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries))
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	_ = sess.Flush()

}

// Test context cloning prevents mutation
func TestUseLocationClonesLocation(t *testing.T) {
	testLoc := Location{
		Path:  "/test",
		Query: url.Values{"key": {"value"}},
		Hash:  "hash",
	}

	var captured1, captured2 Location

	page := func(ctx Ctx, _ Match) *dom.StructuredNode {
		captured1 = UseLocation(ctx)
		captured2 = UseLocation(ctx)

		captured1.Path = "/modified"
		captured1.Query.Set("key", "modified")
		captured1.Hash = "modified"

		return dom.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/test", component: page},
	}

	controller := testController(testLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return LocationCtx.Provide(rctx, testLoc, func(lctx Ctx) *dom.StructuredNode {
				return renderRoutes(lctx, buildTrie(entries))
			})
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if captured2.Path != "/test" {
		t.Errorf("expected path '/test', got %q (mutation leaked)", captured2.Path)
	}

	if captured2.Query.Get("key") != "value" {
		t.Errorf("expected query value 'value', got %q (mutation leaked)", captured2.Query.Get("key"))
	}

	if captured2.Hash != "hash" {
		t.Errorf("expected hash 'hash', got %q (mutation leaked)", captured2.Hash)
	}
}

func buildTrie(entries []routeEntry) *RouterTrie {
	t := NewRouterTrie()
	for _, e := range entries {
		t.Insert(e.pattern, e)
	}
	return t
}
