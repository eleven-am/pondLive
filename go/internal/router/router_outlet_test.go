package router

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestOutletWithNestedRoutes tests that layouts render with outlets and child routes render inside them
func TestOutletWithNestedRoutes(t *testing.T) {
	var layoutRendered bool
	var childRendered string

	dashboardLayout := func(ctx Ctx, match Match) *dom.StructuredNode {
		layoutRendered = true
		return dom.ElementNode("div").
			WithAttr("data-layout", "dashboard").
			WithChildren(Outlet(ctx))
	}

	settingsPage := func(ctx Ctx, match Match) *dom.StructuredNode {
		childRendered = "settings"
		return dom.ElementNode("div").
			WithAttr("data-page", "settings")
	}

	profilePage := func(ctx Ctx, match Match) *dom.StructuredNode {
		childRendered = "profile"
		return dom.ElementNode("div").
			WithAttr("data-page", "profile")
	}

	t.Run("renders layout with settings child", func(t *testing.T) {
		layoutRendered = false
		childRendered = ""

		initialLoc := Location{Path: "/dashboard/settings"}
		controller := testController(initialLoc)

		appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
			return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
				return Routes(rctx,
					Route(rctx, RouteProps{
						Path:      "/dashboard/*",
						Component: dashboardLayout,
					},
						Route(rctx, RouteProps{
							Path:      "./settings",
							Component: settingsPage,
						}),
						Route(rctx, RouteProps{
							Path:      "./profile",
							Component: profilePage,
						}),
					),
				)
			})
		}

		sess := runtime.NewSession(appFunc, struct{}{})
		sess.Flush()

		if !layoutRendered {
			t.Error("expected dashboard layout to render")
		}

		if childRendered != "settings" {
			t.Errorf("expected settings page to render, got %q", childRendered)
		}
	})

	t.Run("renders layout with profile child", func(t *testing.T) {
		layoutRendered = false
		childRendered = ""

		initialLoc := Location{Path: "/dashboard/profile"}
		controller := testController(initialLoc)

		appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
			return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
				return Routes(rctx,
					Route(rctx, RouteProps{
						Path:      "/dashboard/*",
						Component: dashboardLayout,
					},
						Route(rctx, RouteProps{
							Path:      "./settings",
							Component: settingsPage,
						}),
						Route(rctx, RouteProps{
							Path:      "./profile",
							Component: profilePage,
						}),
					),
				)
			})
		}

		sess := runtime.NewSession(appFunc, struct{}{})
		sess.Flush()

		if !layoutRendered {
			t.Error("expected dashboard layout to render")
		}

		if childRendered != "profile" {
			t.Errorf("expected profile page to render, got %q", childRendered)
		}
	})
}

// TestOutletWithoutChildren tests that outlets render empty fragments when there are no child routes
func TestOutletWithoutChildren(t *testing.T) {
	var outletCalled bool

	layout := func(ctx Ctx, match Match) *dom.StructuredNode {
		outletNode := Outlet(ctx)
		outletCalled = true
		return dom.ElementNode("div").WithChildren(outletNode)
	}

	initialLoc := Location{Path: "/dashboard"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return Routes(rctx,
				Route(rctx, RouteProps{
					Path:      "/dashboard",
					Component: layout,
				}),
			)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if !outletCalled {
		t.Error("expected outlet to be called")
	}
}

// TestNestedOutlets tests deeply nested routes with multiple levels of outlets
func TestNestedOutlets(t *testing.T) {
	var renderOrder []string

	outerLayout := func(ctx Ctx, match Match) *dom.StructuredNode {
		renderOrder = append(renderOrder, "outer-layout")
		return dom.ElementNode("div").
			WithAttr("data-layout", "outer").
			WithChildren(Outlet(ctx))
	}

	middleLayout := func(ctx Ctx, match Match) *dom.StructuredNode {
		renderOrder = append(renderOrder, "middle-layout")
		return dom.ElementNode("div").
			WithAttr("data-layout", "middle").
			WithChildren(Outlet(ctx))
	}

	innerPage := func(ctx Ctx, match Match) *dom.StructuredNode {
		renderOrder = append(renderOrder, "inner-page")
		return dom.ElementNode("div").
			WithAttr("data-page", "inner")
	}

	renderOrder = []string{}
	initialLoc := Location{Path: "/outer/middle/inner"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return Routes(rctx,
				Route(rctx, RouteProps{
					Path:      "/outer/*",
					Component: outerLayout,
				},
					Route(rctx, RouteProps{
						Path:      "./middle/*",
						Component: middleLayout,
					},
						Route(rctx, RouteProps{
							Path:      "./inner",
							Component: innerPage,
						}),
					),
				),
			)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	expectedOrder := []string{"outer-layout", "middle-layout", "inner-page"}
	if len(renderOrder) != len(expectedOrder) {
		t.Fatalf("expected %d renders, got %d: %v", len(expectedOrder), len(renderOrder), renderOrder)
	}

	for i, expected := range expectedOrder {
		if renderOrder[i] != expected {
			t.Errorf("render order[%d]: expected %q, got %q", i, expected, renderOrder[i])
		}
	}
}

// TestRelativeVsAbsolutePaths tests that ./path is relative and /path is absolute
func TestRelativeVsAbsolutePaths(t *testing.T) {
	var matchedComponent string

	parentLayout := func(ctx Ctx, match Match) *dom.StructuredNode {
		matchedComponent = "parent"
		return Outlet(ctx)
	}

	relativeChild := func(ctx Ctx, match Match) *dom.StructuredNode {
		matchedComponent = "relative-child"
		return dom.TextNode("relative")
	}

	absoluteChild := func(ctx Ctx, match Match) *dom.StructuredNode {
		matchedComponent = "absolute-child"
		return dom.TextNode("absolute")
	}

	t.Run("relative path resolves under parent", func(t *testing.T) {
		matchedComponent = ""
		initialLoc := Location{Path: "/parent/child"}
		controller := testController(initialLoc)

		appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
			return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
				return Routes(rctx,
					Route(rctx, RouteProps{
						Path:      "/parent/*",
						Component: parentLayout,
					},
						Route(rctx, RouteProps{
							Path:      "./child",
							Component: relativeChild,
						}),
					),
				)
			})
		}

		sess := runtime.NewSession(appFunc, struct{}{})
		sess.Flush()

		if matchedComponent != "relative-child" {
			t.Errorf("expected relative-child to match /parent/child, got %q", matchedComponent)
		}
	})

	t.Run("absolute path ignores parent", func(t *testing.T) {
		matchedComponent = ""
		initialLoc := Location{Path: "/absolute"}
		controller := testController(initialLoc)

		appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
			return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
				return Routes(rctx,
					Route(rctx, RouteProps{
						Path:      "/parent/*",
						Component: parentLayout,
					},

						Route(rctx, RouteProps{
							Path:      "/absolute",
							Component: absoluteChild,
						}),
					),

					Route(rctx, RouteProps{
						Path:      "/absolute",
						Component: absoluteChild,
					}),
				)
			})
		}

		sess := runtime.NewSession(appFunc, struct{}{})
		sess.Flush()

		if matchedComponent != "absolute-child" {
			t.Errorf("expected absolute-child to match /absolute, got %q", matchedComponent)
		}
	})
}

// TestOutletMatchParams tests that params are correctly passed through outlets
func TestOutletMatchParams(t *testing.T) {
	var capturedLayoutParams map[string]string
	var capturedChildParams map[string]string

	layout := func(ctx Ctx, match Match) *dom.StructuredNode {
		capturedLayoutParams = match.Params
		return Outlet(ctx)
	}

	child := func(ctx Ctx, match Match) *dom.StructuredNode {
		capturedChildParams = match.Params
		return dom.TextNode("child")
	}

	initialLoc := Location{Path: "/users/123/posts/456"}
	controller := testController(initialLoc)

	appFunc := func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			return Routes(rctx,
				Route(rctx, RouteProps{
					Path:      "/users/:userId/*",
					Component: layout,
				},
					Route(rctx, RouteProps{
						Path:      "./posts/:postId",
						Component: child,
					}),
				),
			)
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()

	if capturedLayoutParams["userId"] != "123" {
		t.Errorf("expected layout userId=123, got %q", capturedLayoutParams["userId"])
	}

	if capturedChildParams["userId"] != "123" {
		t.Errorf("expected child userId=123, got %q", capturedChildParams["userId"])
	}
	if capturedChildParams["postId"] != "456" {
		t.Errorf("expected child postId=456, got %q", capturedChildParams["postId"])
	}
}
