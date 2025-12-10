package runtime

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestRegisterHandlerUnsubscribesPrevious(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		Handlers: make(map[string]work.Handler),
	}

	var mu sync.Mutex
	handler1Called := false
	handler2Called := false
	done := make(chan struct{})

	handler1 := work.Handler{
		Fn: func(e work.Event) work.Updates {
			mu.Lock()
			handler1Called = true
			mu.Unlock()
			return nil
		},
	}

	handler2 := work.Handler{
		Fn: func(e work.Event) work.Updates {
			mu.Lock()
			handler2Called = true
			mu.Unlock()
			close(done)
			return nil
		},
	}

	handlerID1 := sess.registerHandler(inst, elem, "click", handler1)
	inst.NextHandlerIndex = 0

	handlerID2 := sess.registerHandler(inst, elem, "click", handler2)

	if handlerID1 != handlerID2 {
		t.Errorf("expected same handler ID, got %q and %q", handlerID1, handlerID2)
	}

	sess.Bus.PublishHandlerInvoke(handlerID1, map[string]any{"type": "click"})

	select {
	case <-done:
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for handler to be called")
	}

	mu.Lock()
	h1Called := handler1Called
	h2Called := handler2Called
	mu.Unlock()

	if h1Called {
		t.Error("expected old handler to NOT be called after re-registration")
	}

	if !h2Called {
		t.Error("expected new handler to be called")
	}
}

func TestRegisterHandlerSubscriptionCountStable(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		RefID:    "btn-ref",
		Handlers: make(map[string]work.Handler),
	}

	handler := work.Handler{
		Fn: func(e work.Event) work.Updates { return nil },
	}

	for i := 0; i < 100; i++ {
		sess.registerHandler(inst, elem, "click", handler)
	}

	sess.handlerIDsMu.Lock()
	subCount := len(sess.allHandlerSubs)
	sess.handlerIDsMu.Unlock()

	if subCount != 1 {
		t.Errorf("expected 1 subscription after 100 re-registrations, got %d", subCount)
	}

	handlerID := "btn-ref:click"
	busSubCount := sess.Bus.SubscriberCount(protocol.Topic(handlerID))

	if busSubCount != 1 {
		t.Errorf("expected 1 bus subscriber, got %d", busSubCount)
	}
}

func TestRegisterHandlerWithRefID(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		RefID:    "my-button",
		Handlers: make(map[string]work.Handler),
	}

	handler := work.Handler{
		Fn: func(e work.Event) work.Updates { return nil },
	}

	handlerID := sess.registerHandler(inst, elem, "click", handler)

	expected := "my-button:click"
	if handlerID != expected {
		t.Errorf("expected handler ID %q, got %q", expected, handlerID)
	}
}

func TestRegisterHandlerWithoutRefID(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	inst := &Instance{
		ID:               "test-inst",
		HookFrame:        []HookSlot{},
		NextHandlerIndex: 0,
	}

	elem := &work.Element{
		Tag:      "button",
		Handlers: make(map[string]work.Handler),
	}

	handler := work.Handler{
		Fn: func(e work.Event) work.Updates { return nil },
	}

	handlerID := sess.registerHandler(inst, elem, "click", handler)

	expected := "test-inst:h0"
	if handlerID != expected {
		t.Errorf("expected handler ID %q, got %q", expected, handlerID)
	}

	if inst.NextHandlerIndex != 1 {
		t.Errorf("expected NextHandlerIndex to be 1, got %d", inst.NextHandlerIndex)
	}
}

func TestConvertPortalNodeCollectsViews(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
	}

	portalNode := &work.PortalNode{
		Children: []work.Node{
			&work.Text{Value: "first"},
			&work.Text{Value: "second"},
		},
	}

	result := sess.convertPortalNode(portalNode, parent)

	if result != nil {
		t.Errorf("expected convertPortalNode to return nil, got %T", result)
	}

	if len(sess.PortalViews) != 2 {
		t.Fatalf("expected 2 portal views, got %d", len(sess.PortalViews))
	}

	text1, ok := sess.PortalViews[0].(*view.Text)
	if !ok {
		t.Fatalf("expected first view to be Text, got %T", sess.PortalViews[0])
	}
	if text1.Text != "first" {
		t.Errorf("expected 'first', got '%s'", text1.Text)
	}

	text2, ok := sess.PortalViews[1].(*view.Text)
	if !ok {
		t.Fatalf("expected second view to be Text, got %T", sess.PortalViews[1])
	}
	if text2.Text != "second" {
		t.Errorf("expected 'second', got '%s'", text2.Text)
	}
}

func TestConvertPortalTargetReturnsCollectedViews(t *testing.T) {
	sess := &Session{
		PortalViews: []view.Node{
			&view.Text{Text: "a"},
			&view.Text{Text: "b"},
			&view.Text{Text: "c"},
		},
	}

	result := sess.convertPortalTarget()

	frag, ok := result.(*view.Fragment)
	if !ok {
		t.Fatalf("expected Fragment, got %T", result)
	}

	if len(frag.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(frag.Children))
	}

	if sess.PortalViews != nil {
		t.Errorf("expected PortalViews to be cleared, got %d items", len(sess.PortalViews))
	}
}

func TestConvertPortalTargetReturnsSingleViewUnwrapped(t *testing.T) {
	sess := &Session{
		PortalViews: []view.Node{
			&view.Text{Text: "only"},
		},
	}

	result := sess.convertPortalTarget()

	text, ok := result.(*view.Text)
	if !ok {
		t.Fatalf("expected Text (unwrapped), got %T", result)
	}

	if text.Text != "only" {
		t.Errorf("expected 'only', got '%s'", text.Text)
	}

	if sess.PortalViews != nil {
		t.Errorf("expected PortalViews to be cleared")
	}
}

func TestConvertPortalTargetReturnsNilWhenEmpty(t *testing.T) {
	sess := &Session{
		PortalViews: nil,
	}

	result := sess.convertPortalTarget()

	if result != nil {
		t.Errorf("expected nil when no portal views, got %T", result)
	}
}

func TestConvertPortalNodePreservesParentForContext(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	testCtx := CreateContext("default")

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Providers: map[any]any{
			testCtx.id: "parent-value",
		},
		renderCtx: context.Background(),
	}

	var capturedValue string
	childFn := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedValue = testCtx.UseContextValue(ctx)
		return &work.Text{Value: capturedValue}
	}

	portalNode := &work.PortalNode{
		Children: []work.Node{
			work.Component(childFn),
		},
	}

	sess.convertPortalNode(portalNode, parent)

	if capturedValue != "parent-value" {
		t.Errorf("expected child to access parent context 'parent-value', got '%s'", capturedValue)
	}
}

func TestConvertWorkToViewHandlesPortalNode(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
	}

	portalNode := &work.PortalNode{
		Children: []work.Node{
			&work.Text{Value: "portal content"},
		},
	}

	result := sess.convertWorkToView(portalNode, parent)

	if result != nil {
		t.Errorf("expected nil from portal node conversion, got %T", result)
	}

	if len(sess.PortalViews) != 1 {
		t.Fatalf("expected 1 portal view, got %d", len(sess.PortalViews))
	}
}

func TestConvertWorkToViewHandlesPortalTarget(t *testing.T) {
	sess := &Session{
		PortalViews: []view.Node{
			&view.Text{Text: "collected"},
		},
	}

	target := &work.PortalTarget{}

	result := sess.convertWorkToView(target, nil)

	text, ok := result.(*view.Text)
	if !ok {
		t.Fatalf("expected Text, got %T", result)
	}

	if text.Text != "collected" {
		t.Errorf("expected 'collected', got '%s'", text.Text)
	}
}

func TestPortalIntegrationWithFlush(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	rootFn := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return &work.Fragment{
			Children: []work.Node{
				&work.Element{
					Tag: "div",
					Children: []work.Node{
						&work.Text{Value: "before"},
						&work.PortalNode{
							Children: []work.Node{
								&work.Element{Tag: "span", Children: []work.Node{
									&work.Text{Value: "teleported"},
								}},
							},
						},
						&work.Text{Value: "after"},
					},
				},
				&work.PortalTarget{},
			},
		}
	}

	sess.Root = &Instance{
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
		t.Fatalf("expected 2 children (div + portal content), got %d", len(frag.Children))
	}

	div, ok := frag.Children[0].(*view.Element)
	if !ok {
		t.Fatalf("expected first child to be Element, got %T", frag.Children[0])
	}
	if div.Tag != "div" {
		t.Errorf("expected div, got %s", div.Tag)
	}

	if len(div.Children) != 2 {
		t.Errorf("expected div to have 2 children (before + after), got %d", len(div.Children))
	}

	span, ok := frag.Children[1].(*view.Element)
	if !ok {
		t.Fatalf("expected portal content to be Element, got %T", frag.Children[1])
	}
	if span.Tag != "span" {
		t.Errorf("expected span at target, got %s", span.Tag)
	}
}

func TestPortalContextPreservationIntegration(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	themeCtx := CreateContext("light")
	var capturedTheme string

	themedComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		capturedTheme = themeCtx.UseContextValue(ctx)
		return &work.Text{Value: capturedTheme}
	}

	providerComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		themeCtx.UseProvider(ctx, "dark")
		return &work.PortalNode{
			Children: []work.Node{
				work.Component(themedComponent),
			},
		}
	}

	rootFn := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return &work.Fragment{
			Children: []work.Node{
				work.Component(providerComponent),
				&work.PortalTarget{},
			},
		}
	}

	sess.Root = &Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if capturedTheme != "dark" {
		t.Errorf("expected context value 'dark', got '%s' - context not preserved through portal", capturedTheme)
	}
}

func TestMultiplePortalsCollectInOrder(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	rootFn := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return &work.Fragment{
			Children: []work.Node{
				&work.PortalNode{Children: []work.Node{&work.Text{Value: "1"}}},
				&work.PortalNode{Children: []work.Node{&work.Text{Value: "2"}}},
				&work.PortalNode{Children: []work.Node{&work.Text{Value: "3"}}},
				&work.PortalTarget{},
			},
		}
	}

	sess.Root = &Instance{
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

	if len(frag.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(frag.Children))
	}

	expected := []string{"1", "2", "3"}
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

func TestPortalViewsClearedBetweenFlushes(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	renderCount := 0
	rootFn := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		renderCount++
		content := "render-1"
		if renderCount > 1 {
			content = "render-2"
		}
		return &work.Fragment{
			Children: []work.Node{
				&work.PortalNode{Children: []work.Node{&work.Text{Value: content}}},
				&work.PortalTarget{},
			},
		}
	}

	sess.Root = &Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("first flush failed: %v", err)
	}

	text1, ok := sess.View.(*view.Text)
	if !ok {
		t.Fatalf("first render: expected Text, got %T", sess.View)
	}
	if text1.Text != "render-1" {
		t.Errorf("first render: expected 'render-1', got '%s'", text1.Text)
	}

	sess.Root.WorkTree = nil
	sess.MarkDirty(sess.Root)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	text2, ok := sess.View.(*view.Text)
	if !ok {
		t.Fatalf("second render: expected Text, got %T", sess.View)
	}
	if text2.Text != "render-2" {
		t.Errorf("second render: expected 'render-2', got '%s'", text2.Text)
	}

	if sess.PortalViews != nil && len(sess.PortalViews) > 0 {
		t.Errorf("expected PortalViews to be empty after flush, got %d items", len(sess.PortalViews))
	}
}
