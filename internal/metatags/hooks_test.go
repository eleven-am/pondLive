package metatags

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestUseMetaTags_NoProvider(t *testing.T) {
	called := false

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			UseMetaTags(ctx, &Meta{
				Title: "Test Title",
			})
			called = true
			return &work.Text{Value: "test"}
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if !called {
		t.Error("expected UseMetaTags to be called")
	}
}

func TestUseMetaTags_WithProvider(t *testing.T) {
	called := false

	childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		UseMetaTags(ctx, &Meta{
			Title: "Test Title",
		})
		called = true
		return &work.Text{Value: "test"}
	}

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx, work.Component(childFn))
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if !called {
		t.Error("expected UseMetaTags to be called")
	}
}

func TestUseMetaTags_WithProviderMultipleComponents(t *testing.T) {
	child1Called := false
	child2Called := false

	child1Fn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		UseMetaTags(ctx, &Meta{
			Title: "Child 1 Title",
		})
		child1Called = true
		return &work.Text{Value: "child1"}
	}

	child2Fn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		UseMetaTags(ctx, &Meta{
			Title: "Child 2 Title",
		})
		child2Called = true
		return &work.Text{Value: "child2"}
	}

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx,
				work.Component(child1Fn),
				work.Component(child2Fn),
			)
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if !child1Called || !child2Called {
		t.Error("expected both children to call UseMetaTags")
	}
}

func TestProviderContextSetup(t *testing.T) {
	var capturedState *metaState

	childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		capturedState = metaCtx.UseContextValue(ctx)
		return &work.Text{Value: "test"}
	}

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx, work.Component(childFn))
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if capturedState == nil {
		t.Error("expected non-nil meta state from provider")
	}
}

func TestRenderWithoutProvider(t *testing.T) {
	var capturedNode work.Node

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			capturedNode = Render(ctx)
			return &work.Text{Value: "test"}
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if capturedNode == nil {
		t.Error("expected non-nil node from Render even without provider")
	}
}

func TestRenderWithProvider(t *testing.T) {
	var capturedNode work.Node

	childFn := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		capturedNode = Render(ctx)
		return &work.Text{Value: "test"}
	}

	root := &runtime.Instance{
		ID: "root",
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx, work.Component(childFn))
		},
		HookFrame: []runtime.HookSlot{},
		Children:  []*runtime.Instance{},
	}

	sess := &runtime.Session{
		Root:              root,
		Components:        map[string]*runtime.Instance{"root": root},
		Handlers:          make(map[string]work.Handler),
		MountedComponents: make(map[*runtime.Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if capturedNode == nil {
		t.Error("expected non-nil node from Render")
	}
}
