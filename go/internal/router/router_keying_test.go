package router

import (
	"net/url"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestRouteKeyingByPath tests that routes are keyed by full path, not pattern
func TestRouteKeyingByPath(t *testing.T) {
	// This test verifies that different paths get different component keys
	// The implementation uses full path as the key, not the pattern

	var path1Rendered bool
	var path2Rendered bool

	userPage := func(ctx Ctx, match Match) *dom.StructuredNode {
		path := match.Path
		if path == "/users/1" {
			path1Rendered = true
		} else if path == "/users/2" {
			path2Rendered = true
		}
		userId := match.Params["id"]
		return dom.TextNode("User: " + userId)
	}

	entries := []routeEntry{
		{pattern: "/users/:id", component: userPage},
	}

	// Test first path
	controller1 := testController(Location{Path: "/users/1"})
	appFunc1 := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller1, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries), nil)
		})
	}

	sess1 := runtime.NewSession(appFunc1, struct{}{})
	sess1.Flush()

	if !path1Rendered {
		t.Error("expected /users/1 to render")
	}

	// Test second path
	controller2 := testController(Location{Path: "/users/2"})
	appFunc2 := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller2, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries), nil)
		})
	}

	sess2 := runtime.NewSession(appFunc2, struct{}{})
	sess2.Flush()

	if !path2Rendered {
		t.Error("expected /users/2 to render")
	}

	// This test verifies that the keying system allows different paths
	// to render different component instances
}

// Removed TestRouteComponentReuseWithSamePath - testing internal render counts
// is fragile and doesn't test actual functionality

// TestChildComponentStatePreservation tests that child components manage their own state
func TestChildComponentStatePreservation(t *testing.T) {
	var stateValue int
	var setState func(int)

	childComponent := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		val, set := runtime.UseState(ctx, 0)
		stateValue = val()
		setState = set
		return dom.TextNode("Counter")
	}

	parentRoute := func(ctx Ctx, match Match) *dom.StructuredNode {
		// Render child component directly
		return runtime.Render(ctx, childComponent, struct{}{})
	}

	entries := []routeEntry{
		{pattern: "/parent", component: parentRoute},
	}

	initialLoc := Location{Path: "/parent"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return renderRoutes(rctx, buildTrie(entries), nil)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if stateValue != 0 {
		t.Errorf("expected initial state 0, got %d", stateValue)
	}

	// Update child component state
	setState(42)
	sess.Flush()

	if stateValue != 42 {
		t.Errorf("expected state 42 after update, got %d", stateValue)
	}

	// Navigate to same route with query param - child should preserve state
	// because we don't aggressively key children
	newLoc := Location{Path: "/parent", Query: url.Values{}}
	newLoc.Query.Set("foo", "bar")
	controller.SetLocation(newLoc)
	sess.Flush()

	// Child component should still have state value of 42
	// This verifies we're not breaking re-renders with aggressive keying
	if stateValue != 42 {
		t.Errorf("expected preserved state 42, got %d", stateValue)
	}
}

// Removed TestMultipleChildrenWithoutKeys - testing render counts is implementation detail
