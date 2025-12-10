package portal

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestPortalRendersAtTarget(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	rootFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		return &work.Fragment{
			Children: []work.Node{
				&work.Element{Tag: "div", Children: []work.Node{
					&work.Text{Value: "before"},
					Portal(&work.Element{Tag: "span", Children: []work.Node{
						&work.Text{Value: "portal content"},
					}}),
					&work.Text{Value: "after"},
				}},
				Target(),
			},
		}
	}

	sess.Root = &runtime.Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	frag, ok := sess.View.(*view.Fragment)
	if !ok {
		t.Fatalf("expected Fragment, got %T", sess.View)
	}

	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children (div + portal target content), got %d", len(frag.Children))
	}

	div, ok := frag.Children[0].(*view.Element)
	if !ok {
		t.Fatalf("expected first child to be Element, got %T", frag.Children[0])
	}

	if div.Tag != "div" {
		t.Errorf("expected div tag, got %s", div.Tag)
	}

	if len(div.Children) != 2 {
		t.Fatalf("expected div to have 2 children (before + after, portal removed), got %d", len(div.Children))
	}

	span, ok := frag.Children[1].(*view.Element)
	if !ok {
		t.Fatalf("expected portal target content to be Element, got %T", frag.Children[1])
	}

	if span.Tag != "span" {
		t.Errorf("expected span tag at target, got %s", span.Tag)
	}
}

func TestPortalPreservesContext(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	testCtx := runtime.CreateContext("default")
	var capturedValue string

	childComponent := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		capturedValue = testCtx.UseContextValue(ctx)
		return &work.Text{Value: capturedValue}
	}

	providerComponent := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		testCtx.UseProvider(ctx, "provided-value")
		return &work.Fragment{
			Children: []work.Node{
				Portal(work.Component(childComponent)),
			},
		}
	}

	rootFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		return &work.Fragment{
			Children: []work.Node{
				work.Component(providerComponent),
				Target(),
			},
		}
	}

	sess.Root = &runtime.Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if capturedValue != "provided-value" {
		t.Errorf("expected context value 'provided-value', got '%s'", capturedValue)
	}
}

func TestMultiplePortalsRenderInOrder(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	rootFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		return &work.Fragment{
			Children: []work.Node{
				Portal(&work.Text{Value: "first"}),
				Portal(&work.Text{Value: "second"}),
				Portal(&work.Text{Value: "third"}),
				Target(),
			},
		}
	}

	sess.Root = &runtime.Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	frag, ok := sess.View.(*view.Fragment)
	if !ok {
		t.Fatalf("expected Fragment at target, got %T", sess.View)
	}

	if len(frag.Children) != 3 {
		t.Fatalf("expected 3 portal contents, got %d", len(frag.Children))
	}

	expected := []string{"first", "second", "third"}
	for i, exp := range expected {
		text, ok := frag.Children[i].(*view.Text)
		if !ok {
			t.Errorf("child %d: expected Text, got %T", i, frag.Children[i])
			continue
		}
		if text.Text != exp {
			t.Errorf("child %d: expected '%s', got '%s'", i, exp, text.Text)
		}
	}
}

func TestEmptyPortalRendersNothing(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	rootFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		return &work.Fragment{
			Children: []work.Node{
				&work.Text{Value: "content"},
				Target(),
			},
		}
	}

	sess.Root = &runtime.Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	text, ok := sess.View.(*view.Text)
	if !ok {
		t.Fatalf("expected Text (single child unwrapped), got %T", sess.View)
	}

	if text.Text != "content" {
		t.Errorf("expected 'content', got '%s'", text.Text)
	}
}

func TestPortalClearsAfterFlush(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	callCount := 0
	rootFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		callCount++
		if callCount == 1 {
			return &work.Fragment{
				Children: []work.Node{
					Portal(&work.Text{Value: "first render"}),
					Target(),
				},
			}
		}
		return &work.Fragment{
			Children: []work.Node{
				Portal(&work.Text{Value: "second render"}),
				Target(),
			},
		}
	}

	sess.Root = &runtime.Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("first flush failed: %v", err)
	}

	text, ok := sess.View.(*view.Text)
	if !ok {
		t.Fatalf("first render: expected Text, got %T", sess.View)
	}
	if text.Text != "first render" {
		t.Errorf("first render: expected 'first render', got '%s'", text.Text)
	}

	sess.Root.WorkTree = nil
	sess.MarkDirty(sess.Root)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	text, ok = sess.View.(*view.Text)
	if !ok {
		t.Fatalf("second render: expected Text, got %T", sess.View)
	}
	if text.Text != "second render" {
		t.Errorf("second render: expected 'second render', got '%s'", text.Text)
	}
}
