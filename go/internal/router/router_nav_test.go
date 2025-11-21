package router

import (
	"net/url"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestNavigateEnqueuesNavDelta verifies that Navigate() queues a push navigation
func TestNavigateEnqueuesNavDelta(t *testing.T) {
	initialLoc := Location{Path: "/start", Query: url.Values{}}
	state := &RouterState{Location: initialLoc}
	controller := NewController(
		func() *RouterState { return state },
		func(s *RouterState) { state = s },
	)

	var capturedNav *runtime.NavDelta
	appFunc := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			Navigate(rctx, "/destination?key=value")
			return dom.TextNode("test")
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()
	capturedNav = sess.TakeNavDelta()

	if capturedNav == nil {
		t.Fatal("expected navigation delta to be enqueued")
	}
	if capturedNav.Push != "/destination?key=value" {
		t.Errorf("expected Push to be '/destination?key=value', got %q", capturedNav.Push)
	}
	if capturedNav.Replace != "" {
		t.Error("expected Replace to be empty for Navigate()")
	}
}

// TestReplaceEnqueuesNavDelta verifies that Replace() queues a replace navigation
func TestReplaceEnqueuesNavDelta(t *testing.T) {
	initialLoc := Location{Path: "/start", Query: url.Values{}}
	state := &RouterState{Location: initialLoc}
	controller := NewController(
		func() *RouterState { return state },
		func(s *RouterState) { state = s },
	)

	var capturedNav *runtime.NavDelta
	appFunc := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			Replace(rctx, "/replaced")
			return dom.TextNode("test")
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()
	capturedNav = sess.TakeNavDelta()

	if capturedNav == nil {
		t.Fatal("expected navigation delta to be enqueued")
	}
	if capturedNav.Replace != "/replaced" {
		t.Errorf("expected Replace to be '/replaced', got %q", capturedNav.Replace)
	}
	if capturedNav.Push != "" {
		t.Error("expected Push to be empty for Replace()")
	}
}

// TestNavigateWithSearchEnqueuesNavDelta verifies NavigateWithSearch queues push navigation
func TestNavigateWithSearchEnqueuesNavDelta(t *testing.T) {
	initialLoc := Location{Path: "/current", Query: url.Values{"existing": {"value"}}}
	state := &RouterState{Location: initialLoc}
	controller := NewController(
		func() *RouterState { return state },
		func(s *RouterState) { state = s },
	)

	var capturedNav *runtime.NavDelta
	appFunc := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			NavigateWithSearch(rctx, func(q url.Values) url.Values {
				q.Set("new", "param")
				return q
			})
			return dom.TextNode("test")
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()
	capturedNav = sess.TakeNavDelta()

	if capturedNav == nil {
		t.Fatal("expected navigation delta to be enqueued")
	}
	if capturedNav.Push == "" {
		t.Fatal("expected Push to be set")
	}

	if capturedNav.Replace != "" {
		t.Error("expected Replace to be empty for NavigateWithSearch()")
	}
}

// TestReplaceWithSearchEnqueuesNavDelta verifies ReplaceWithSearch queues replace navigation
func TestReplaceWithSearchEnqueuesNavDelta(t *testing.T) {
	initialLoc := Location{Path: "/current", Query: url.Values{}}
	state := &RouterState{Location: initialLoc}
	controller := NewController(
		func() *RouterState { return state },
		func(s *RouterState) { state = s },
	)

	var capturedNav *runtime.NavDelta
	appFunc := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			ReplaceWithSearch(rctx, func(q url.Values) url.Values {
				q.Set("filter", "active")
				return q
			})
			return dom.TextNode("test")
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()
	capturedNav = sess.TakeNavDelta()

	if capturedNav == nil {
		t.Fatal("expected navigation delta to be enqueued")
	}
	if capturedNav.Replace == "" {
		t.Fatal("expected Replace to be set")
	}
	if capturedNav.Push != "" {
		t.Error("expected Push to be empty for ReplaceWithSearch()")
	}
}

// TestNavigateToSameLocationNoNavDelta verifies no nav delta when navigating to same location
func TestNavigateToSameLocationNoNavDelta(t *testing.T) {
	initialLoc := Location{Path: "/same", Query: url.Values{}}
	state := &RouterState{Location: initialLoc}
	controller := NewController(
		func() *RouterState { return state },
		func(s *RouterState) { state = s },
	)

	appFunc := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			Navigate(rctx, "/same")
			return dom.TextNode("test")
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()
	capturedNav := sess.TakeNavDelta()

	if capturedNav != nil {
		t.Error("expected no navigation delta when navigating to same location")
	}
}

// TestNavigateWithHashEnqueuesNavDelta verifies hash-only navigation works
func TestNavigateWithHashEnqueuesNavDelta(t *testing.T) {
	initialLoc := Location{Path: "/page", Query: url.Values{}}
	state := &RouterState{Location: initialLoc}
	controller := NewController(
		func() *RouterState { return state },
		func(s *RouterState) { state = s },
	)

	var capturedNav *runtime.NavDelta
	appFunc := func(ctx runtime.Ctx, _ struct{}) *dom.StructuredNode {
		return ProvideRouterState(ctx, controller, func(rctx Ctx) *dom.StructuredNode {
			Navigate(rctx, "#section")
			return dom.TextNode("test")
		})
	}

	sess := runtime.NewSession(appFunc, struct{}{})
	sess.Flush()
	capturedNav = sess.TakeNavDelta()

	if capturedNav == nil {
		t.Fatal("expected navigation delta to be enqueued for hash navigation")
	}
	if capturedNav.Push == "" {
		t.Fatal("expected Push to be set for hash navigation")
	}
}
