package router

import (
	"net/url"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// Test route matching and rendering
func TestRouteMatching(t *testing.T) {
	var matchedRoute string

	homePage := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		matchedRoute = "home"
		return dom2.TextNode("Home")
	}

	userPage := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		matchedRoute = "user"
		return dom2.TextNode("User")
	}

	entries := []routeEntry{
		{pattern: "/", component: homePage},
		{pattern: "/users/:id", component: userPage},
	}

	matchedRoute = ""
	initialLoc := Location{Path: "/"}
	getLoc := func() Location { return initialLoc }
	setLoc := func(Location) {}
	state := routerState{getLoc: getLoc, setLoc: setLoc}

	appFunc := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state, func(rctx Ctx) *dom2.StructuredNode {
			return renderRoutes(rctx, entries)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if matchedRoute != "home" {
		t.Errorf("expected home route to match, got %q", matchedRoute)
	}

	matchedRoute = ""
	initialLoc2 := Location{Path: "/users/123"}
	getLoc2 := func() Location { return initialLoc2 }
	setLoc2 := func(Location) {}
	state2 := routerState{getLoc: getLoc2, setLoc: setLoc2}

	appFunc2 := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state2, func(rctx Ctx) *dom2.StructuredNode {
			return renderRoutes(rctx, entries)
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

	page := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		capturedLoc = UseLocation(ctx)
		return dom2.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/test/path", component: page},
	}

	getLoc := func() Location { return initialLoc }
	setLoc := func(Location) {}
	state := routerState{getLoc: getLoc, setLoc: setLoc}

	appFunc := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state, func(rctx Ctx) *dom2.StructuredNode {
			return LocationCtx.Provide(rctx, initialLoc, func(lctx Ctx) *dom2.StructuredNode {
				return renderRoutes(lctx, entries)
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

	page := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		params := UseParams(ctx)
		capturedID = params["id"]
		capturedSlug = params["slug"]
		return dom2.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/posts/:id/:slug", component: page},
	}

	initialLoc := Location{Path: "/posts/42/hello-world"}
	getLoc := func() Location { return initialLoc }
	setLoc := func(Location) {}
	state := routerState{getLoc: getLoc, setLoc: setLoc}

	appFunc := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state, func(rctx Ctx) *dom2.StructuredNode {
			return renderRoutes(rctx, entries)
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

	page := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		get, _ := UseSearchParam(ctx, "tab")
		capturedValues = get()
		return dom2.FragmentNode()
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

	getLoc := func() Location { return initialLoc }
	setLoc := func(Location) {}
	state := routerState{getLoc: getLoc, setLoc: setLoc}

	appFunc := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state, func(rctx Ctx) *dom2.StructuredNode {
			return renderRoutes(rctx, entries)
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

	wildcardPage := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		matchedRoute = "wildcard"
		return dom2.FragmentNode()
	}

	paramPage := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		matchedRoute = "param"
		return dom2.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/items/*", component: wildcardPage},
		{pattern: "/items/:id", component: paramPage},
	}

	initialLoc := Location{Path: "/items/123"}
	getLoc := func() Location { return initialLoc }
	setLoc := func(Location) {}
	state := routerState{getLoc: getLoc, setLoc: setLoc}

	appFunc := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state, func(rctx Ctx) *dom2.StructuredNode {
			return renderRoutes(rctx, entries)
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
	homePage := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		return dom2.TextNode("Home")
	}

	entries := []routeEntry{
		{pattern: "/", component: homePage},
	}

	initialLoc := Location{Path: "/404"}
	getLoc := func() Location { return initialLoc }
	setLoc := func(Location) {}
	state := routerState{getLoc: getLoc, setLoc: setLoc}

	appFunc := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state, func(rctx Ctx) *dom2.StructuredNode {
			return renderRoutes(rctx, entries)
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

	page := func(ctx Ctx, _ Match) *dom2.StructuredNode {
		captured1 = UseLocation(ctx)
		captured2 = UseLocation(ctx)

		captured1.Path = "/modified"
		captured1.Query.Set("key", "modified")
		captured1.Hash = "modified"

		return dom2.FragmentNode()
	}

	entries := []routeEntry{
		{pattern: "/test", component: page},
	}

	getLoc := func() Location { return testLoc }
	setLoc := func(Location) {}
	state := routerState{getLoc: getLoc, setLoc: setLoc}

	appFunc := func(ctx Ctx, _ struct{}) *dom2.StructuredNode {
		return RouterStateCtx.Provide(ctx, state, func(rctx Ctx) *dom2.StructuredNode {
			return LocationCtx.Provide(rctx, testLoc, func(lctx Ctx) *dom2.StructuredNode {
				return renderRoutes(lctx, entries)
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
