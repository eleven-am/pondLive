package router

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestRouterContextValue tests that the unified RouterContextValue works correctly
func TestRouterContextValue(t *testing.T) {
	var capturedPattern string
	var capturedChildrenCount int

	layout := func(ctx Ctx, match Match) *dom.StructuredNode {
		routerCtx := routerContextKey.Use(ctx)
		if routerCtx != nil {
			capturedPattern = routerCtx.Pattern
			capturedChildrenCount = len(routerCtx.Children)
		}
		return Outlet(ctx)
	}

	child := func(ctx Ctx, match Match) *dom.StructuredNode {
		return dom.TextNode("child")
	}

	initialLoc := Location{Path: "/dashboard/settings"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return Routes(rctx,
				Route(rctx, RouteProps{
					Path:      "/dashboard/*",
					Component: layout,
				},
					Route(rctx, RouteProps{
						Path:      "./settings",
						Component: child,
					}),
				),
			)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if capturedPattern != "/dashboard/*" {
		t.Errorf("expected pattern '/dashboard/*', got %q", capturedPattern)
	}

	if capturedChildrenCount != 1 {
		t.Errorf("expected 1 child route, got %d", capturedChildrenCount)
	}
}

// TestRouterContextNilSafety tests that router context handles nil values safely
func TestRouterContextNilSafety(t *testing.T) {
	var didRender bool

	page := func(ctx Ctx, match Match) *dom.StructuredNode {
		didRender = true
		routerCtx := routerContextKey.Use(ctx)
		if routerCtx == nil {
			return dom.TextNode("no context")
		}
		return Outlet(ctx)
	}

	initialLoc := Location{Path: "/page"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return Routes(rctx,
				Route(rctx, RouteProps{
					Path:      "/page",
					Component: page,
				}),
			)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if !didRender {
		t.Error("expected page to render")
	}
}

// TestOutletWithEmptyChildren tests that Outlet handles empty children correctly
func TestOutletWithEmptyChildren(t *testing.T) {
	var outletRendered bool

	layout := func(ctx Ctx, match Match) *dom.StructuredNode {
		outletRendered = true
		outlet := Outlet(ctx)
		return dom.ElementNode("div").WithChildren(outlet)
	}

	initialLoc := Location{Path: "/empty"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return Routes(rctx,
				Route(rctx, RouteProps{
					Path:      "/empty",
					Component: layout,
				}),
			)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if !outletRendered {
		t.Error("expected outlet to be called")
	}
}

// TestNestedRouterContextProvision tests that nested routes get correct context
func TestNestedRouterContextProvision(t *testing.T) {
	var patterns []string

	capturePattern := func(level string) func(Ctx, Match) *dom.StructuredNode {
		return func(ctx Ctx, match Match) *dom.StructuredNode {
			routerCtx := routerContextKey.Use(ctx)
			if routerCtx != nil {
				patterns = append(patterns, level+":"+routerCtx.Pattern)
			}
			return Outlet(ctx)
		}
	}

	initialLoc := Location{Path: "/a/b/c"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return Routes(rctx,
				Route(rctx, RouteProps{
					Path:      "/a/*",
					Component: capturePattern("level1"),
				},
					Route(rctx, RouteProps{
						Path:      "./b/*",
						Component: capturePattern("level2"),
					},
						Route(rctx, RouteProps{
							Path:      "./c",
							Component: capturePattern("level3"),
						}),
					),
				),
			)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	expectedPatterns := []string{
		"level1:/a/*",
		"level2:/a/b/*",
		"level3:/a/b/c",
	}

	if len(patterns) != len(expectedPatterns) {
		t.Fatalf("expected %d patterns, got %d: %v", len(expectedPatterns), len(patterns), patterns)
	}

	for i, expected := range expectedPatterns {
		if patterns[i] != expected {
			t.Errorf("pattern[%d]: expected %q, got %q", i, expected, patterns[i])
		}
	}
}
