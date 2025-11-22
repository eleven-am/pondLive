package meta

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// TestMetaTagsMergeChildWins tests that child components' meta tags override parent meta tags.
func TestMetaTagsMergeChildWins(testing *testing.T) {
	var layoutMeta *Meta
	var pageMeta *Meta

	var controller *Controller

	child := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		UseMetaTags(ctx, pageMeta)
		return dom.ElementNode("div").WithChildren(dom.TextNode("Child Page"))
	}

	parent := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		UseMetaTags(ctx, layoutMeta)

		if controller == nil {
			controller = metaCtx.Use(ctx)
		}
		childNode := runtime.Render(ctx, child, struct{}{})
		return dom.ElementNode("div").WithChildren(childNode)
	}

	root := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		return Provider(ctx, "/app.js", parent, struct{}{})
	}

	sess := runtime.NewSession(root, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	layoutMeta = &Meta{
		Title:       "My App",
		Description: "App description",
	}
	pageMeta = &Meta{
		Title: "Home Page",
	}

	if err := sess.Flush(); err != nil {
		testing.Fatalf("flush failed: %v", err)
	}

	merged := controller.Get()

	if merged.Title != "Home Page" {
		testing.Errorf("Expected child title 'Home Page', got %q", merged.Title)
	}

	if merged.Description != "App description" {
		testing.Errorf("Expected parent description 'App description', got %q", merged.Description)
	}
}

// TestMetaTagsCleanupOnUnmount tests that meta tags are removed when component unmounts.
func TestMetaTagsCleanupOnUnmount(testing *testing.T) {
	var setShowChild func(bool)
	var controller *Controller

	childMeta := &Meta{
		Title: "Child Page",
	}

	child := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		UseMetaTags(ctx, childMeta)
		return dom.ElementNode("div").WithChildren(dom.TextNode("Child"))
	}

	parent := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		show, setShow := runtime.UseState(ctx, true)
		setShowChild = setShow

		if controller == nil {
			controller = metaCtx.Use(ctx)
		}

		if show() {
			childNode := runtime.Render(ctx, child, struct{}{})
			return dom.ElementNode("div").WithChildren(childNode)
		}
		return dom.ElementNode("div").WithChildren(dom.TextNode("No child"))
	}

	root := func(ctx runtime.Ctx, props struct{}) *dom.StructuredNode {
		return Provider(ctx, "/app.js", parent, struct{}{})
	}

	sess := runtime.NewSession(root, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		testing.Fatalf("initial flush failed: %v", err)
	}

	merged := controller.Get()

	if merged.Title != "Child Page" {
		testing.Errorf("Expected title 'Child Page', got %q", merged.Title)
	}

	setShowChild(false)

	if err := sess.Flush(); err != nil {
		testing.Fatalf("flush after unmount failed: %v", err)
	}

	merged = controller.Get()

	if merged.Title != "PondLive Application" {
		testing.Errorf("Expected default title 'PondLive Application' after unmount, got %q", merged.Title)
	}
}
