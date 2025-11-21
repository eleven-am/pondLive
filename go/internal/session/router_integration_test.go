package session

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestRouterStateInRoot verifies that router location state is managed
// by documentRoot via UseState and provided through RouterStateCtx
func TestRouterStateInRoot(t *testing.T) {
	var lastPath string

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		loc := router.UseLocation(ctx)
		lastPath = loc.Path

		return dom.ElementNode("div").WithChildren(
			dom.TextNode("Path: " + loc.Path),
		)
	}

	transport := &mockTransport{}
	sess := NewLiveSession(
		SessionID("router-test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/users/123")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if lastPath != "/users/123" {
		t.Errorf("expected path '/users/123', got %q", lastPath)
	}

	if lastPath != "/users/123" {
		t.Errorf("location not updated correctly in component")
	}
}

// TestRouterStateWithQuery verifies router handles query parameters correctly
func TestRouterStateWithQuery(t *testing.T) {
	var lastPath string
	var lastTab string

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		loc := router.UseLocation(ctx)
		lastPath = loc.Path
		lastTab = loc.Query.Get("tab")

		return dom.ElementNode("div").WithChildren(
			dom.TextNode("Path: " + loc.Path + " Tab: " + lastTab),
		)
	}

	transport := &mockTransport{}
	sess := NewLiveSession(
		SessionID("router-query-test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/settings?tab=profile&view=details")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if lastPath != "/settings" {
		t.Errorf("expected path '/settings', got %q", lastPath)
	}

	if lastTab != "profile" {
		t.Errorf("expected tab 'profile', got %q", lastTab)
	}

	if lastTab != "profile" {
		t.Errorf("query parameter not read correctly")
	}
}

// TestRouterStateWithHash verifies router handles hash fragments correctly
func TestRouterStateWithHash(t *testing.T) {
	var lastHash string

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		loc := router.UseLocation(ctx)
		lastHash = loc.Hash

		return dom.ElementNode("div").WithChildren(
			dom.TextNode("Hash: " + loc.Hash),
		)
	}

	transport := &mockTransport{}
	sess := NewLiveSession(
		SessionID("router-hash-test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/docs#section-intro")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if lastHash != "section-intro" {
		t.Errorf("expected hash 'section-intro', got %q", lastHash)
	}
}

// TestRouterParams verifies UseParams hook works with router state from documentRoot
func TestRouterParams(t *testing.T) {
	var lastUserID string

	userPage := func(ctx runtime.Ctx, match router.Match) *dom.StructuredNode {
		params := router.UseParams(ctx)
		lastUserID = params["id"]

		return dom.ElementNode("div").WithChildren(
			dom.TextNode("User: " + lastUserID),
		)
	}

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return router.Router(ctx,
			router.Routes(ctx,
				router.Route(ctx, router.RouteProps{
					Path:      "/users/:id",
					Component: userPage,
				}),
			),
		)
	}

	transport := &mockTransport{}
	sess := NewLiveSession(
		SessionID("router-params-test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/users/42")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if lastUserID != "42" {
		t.Errorf("expected user id '42', got %q", lastUserID)
	}
}

// TestRouterSearchParams verifies UseSearchParam hook works correctly
func TestRouterSearchParams(t *testing.T) {
	var lastFilter string

	listPage := func(ctx runtime.Ctx, match router.Match) *dom.StructuredNode {
		get, _ := router.UseSearchParam(ctx, "filter")
		values := get()
		if len(values) > 0 {
			lastFilter = values[0]
		}

		return dom.ElementNode("div").WithChildren(
			dom.TextNode("Filter: " + lastFilter),
		)
	}

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return router.Router(ctx,
			router.Routes(ctx,
				router.Route(ctx, router.RouteProps{
					Path:      "/items",
					Component: listPage,
				}),
			),
		)
	}

	transport := &mockTransport{}
	sess := NewLiveSession(
		SessionID("router-search-test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/items?filter=active&sort=name")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if lastFilter != "active" {
		t.Errorf("expected filter 'active', got %q", lastFilter)
	}
}

// TestRouterNestedRoutes verifies nested routing works with session integration
func TestRouterNestedRoutes(t *testing.T) {
	var layoutRendered bool
	var profileRendered bool

	layout := func(ctx runtime.Ctx, _ router.Match) *dom.StructuredNode {
		layoutRendered = true
		return dom.ElementNode("div").WithChildren(
			dom.TextNode("Settings Layout"),
			router.Outlet(ctx),
		)
	}

	profilePage := func(ctx runtime.Ctx, _ router.Match) *dom.StructuredNode {
		profileRendered = true
		return dom.ElementNode("div").WithChildren(
			dom.TextNode("Profile Page"),
		)
	}

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return router.Router(ctx,
			router.Routes(ctx,
				router.Route(ctx, router.RouteProps{
					Path:      "/settings/*",
					Component: layout,
				},
					router.Routes(ctx,
						router.Route(ctx, router.RouteProps{
							Path:      "./profile",
							Component: profilePage,
						}),
					),
				),
			),
		)
	}

	transport := &mockTransport{}
	sess := NewLiveSession(
		SessionID("router-nested-test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/settings/profile")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if !layoutRendered {
		t.Error("expected layout to be rendered")
	}

	if !profileRendered {
		t.Error("expected profile page to be rendered")
	}
}

// TestRouterWithHeaderContext verifies router and header contexts both work
func TestRouterWithHeaderContext(t *testing.T) {
	var lastPath string
	var lastUserAgent string

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		loc := router.UseLocation(ctx)
		headers := UseHeader(ctx)

		lastPath = loc.Path
		lastUserAgent, _ = headers.GetHeader("User-Agent")

		return dom.ElementNode("div").WithChildren(
			dom.TextNode("Path: " + lastPath + " UA: " + lastUserAgent),
		)
	}

	transport := &mockTransport{}
	sess := NewLiveSession(
		SessionID("router-header-test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/api/v1/users")
	req.Header.Set("User-Agent", "TestClient/3.0")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if lastPath != "/api/v1/users" {
		t.Errorf("expected path '/api/v1/users', got %q", lastPath)
	}

	if lastUserAgent != "TestClient/3.0" {
		t.Errorf("expected user agent 'TestClient/3.0', got %q", lastUserAgent)
	}
}

// TestRouterStateIsolation verifies each session has isolated router state
func TestRouterStateIsolation(t *testing.T) {
	var path1, path2 string

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		loc := router.UseLocation(ctx)
		return dom.ElementNode("div").WithChildren(
			dom.TextNode(loc.Path),
		)
	}

	transport1 := &mockTransport{}
	sess1 := NewLiveSession(SessionID("session-1"), 1, app, &Config{Transport: transport1})
	req1 := newRequest("/users/1")
	sess1.MergeRequest(req1)
	if err := sess1.Flush(); err != nil {
		t.Fatalf("session1 flush failed: %v", err)
	}
	path1 = sess1.InitialLocation().Path

	transport2 := &mockTransport{}
	sess2 := NewLiveSession(SessionID("session-2"), 1, app, &Config{Transport: transport2})
	req2 := newRequest("/users/2")
	sess2.MergeRequest(req2)
	if err := sess2.Flush(); err != nil {
		t.Fatalf("session2 flush failed: %v", err)
	}
	path2 = sess2.InitialLocation().Path

	if path1 != "/users/1" {
		t.Errorf("session1 expected '/users/1', got %q", path1)
	}

	if path2 != "/users/2" {
		t.Errorf("session2 expected '/users/2', got %q", path2)
	}

	if path1 == path2 {
		t.Error("sessions should have isolated router state")
	}
}
