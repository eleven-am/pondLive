package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

// Mirrors the tabs pattern: choose between two content components rendered via a
// wrapper component with a keyed Render call. Switching tabs should produce patches.
func TestTabsLikeComponentSwitchProducesPatches(t *testing.T) {
	signin := Component[struct{}](func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode("signin-content"))
	})

	signup := Component[struct{}](func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode("signup-content"))
	})

	var setActive func(string)

	tabs := Component[struct{}](func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		active, set := UseState(ctx, "signin")
		setActive = set

		var content Component[struct{}]
		var key string
		if active() == "signin" {
			content = signin
			key = "tab-content-signin"
		} else {
			content = signup
			key = "tab-content-signup"
		}

		// Wrapper component rendered with a stable key, as in the tabs implementation.
		wrapper := func(wrapperCtx Ctx, _ struct{}) *dom.StructuredNode {
			return dom.ElementNode("div").
				WithAttr("data-slot", "tabs-content").
				WithChildren(content(wrapperCtx, struct{}{}))
		}

		return Render(ctx, wrapper, struct{}{}, WithKey(key))
	})

	sess := NewSession(tabs, struct{}{})
	sess.SetPatchSender(func(p []dom2diff.Patch) error { return nil })

	// Initial render
	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}
	prev := sess.Tree().Flatten()

	// Switch active tab
	setActive("signup")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after switch failed: %v", err)
	}
	next := sess.Tree().Flatten()

	patches := dom2diff.Diff(prev, next)
	if len(patches) == 0 {
		t.Fatalf("expected patches when switching tabs; prev=%q next=%q", prev.ToHTML(), next.ToHTML())
	}

	// Basic content sanity: new tree should contain signup text
	if next.ToHTML() == prev.ToHTML() || !containsText(next, "signup-content") {
		t.Fatalf("expected signup content in next tree; prev=%q next=%q", prev.ToHTML(), next.ToHTML())
	}
}

// This variant mirrors the real tabs structure: TabContent functions are ui.Component,
// and the active content is rendered via Render(ctx, wrapper, ...) where wrapper renders
// tc.Content(wrapperCtx). This reproduces the component-boundary flattening path.
func TestTabsLikeWithComponentChildren(t *testing.T) {
	// Child content as ui.Component (adds a component boundary)
	signin := Component[[]dom.Item](func(ctx Ctx, _ []dom.Item) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode("signin-content"))
	})

	signup := Component[[]dom.Item](func(ctx Ctx, _ []dom.Item) *dom.StructuredNode {
		return dom.ElementNode("div").WithChildren(dom.TextNode("signup-content"))
	})

	var setActive func(string)

	tabs := Component[struct{}](func(ctx Ctx, _ struct{}) *dom.StructuredNode {
		active, set := UseState(ctx, "signin")
		setActive = set

		var content Component[[]dom.Item]
		var key string
		if active() == "signin" {
			content = signin
			key = "tab-content-signin"
		} else {
			content = signup
			key = "tab-content-signup"
		}

		// Wrapper component rendered with a key, mimicking common.Tabs logic.
		wrapper := func(wrapperCtx Ctx, _ struct{}) *dom.StructuredNode {
			return dom.ElementNode("div").
				WithAttr("data-slot", "tabs-content").
				WithChildren(content(wrapperCtx, nil))
		}

		return Render(ctx, wrapper, struct{}{}, WithKey(key))
	})

	sess := NewSession(tabs, struct{}{})
	sess.SetPatchSender(func(p []dom2diff.Patch) error { return nil })

	// Initial render
	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}
	prev := sess.Tree().Flatten()

	// Switch active tab
	setActive("signup")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after switch failed: %v", err)
	}
	next := sess.Tree().Flatten()

	patches := dom2diff.Diff(prev, next)
	if len(patches) == 0 {
		t.Fatalf("expected patches when switching tabs with component children; prev=%q next=%q", prev.ToHTML(), next.ToHTML())
	}

	if next.ToHTML() == prev.ToHTML() || !containsText(next, "signup-content") {
		t.Fatalf("expected signup content in next tree; prev=%q next=%q", prev.ToHTML(), next.ToHTML())
	}
}

func containsText(node *dom.StructuredNode, target string) bool {
	if node == nil {
		return false
	}
	if node.Text == target {
		return true
	}
	for _, child := range node.Children {
		if containsText(child, target) {
			return true
		}
	}
	return false
}
