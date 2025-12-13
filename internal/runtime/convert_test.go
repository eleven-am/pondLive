package runtime

import (
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

func TestTrackConvertRenderErrorNilSession(t *testing.T) {
	var sess *Session
	inst := &Instance{
		ID:          "test",
		RenderError: &Error{Message: "test error"},
	}
	sess.trackConvertRenderError(inst)
}

func TestTrackConvertRenderErrorNilInstance(t *testing.T) {
	sess := &Session{}
	sess.trackConvertRenderError(nil)
	if len(sess.convertRenderErrors) != 0 {
		t.Error("expected no errors tracked for nil instance")
	}
}

func TestTrackConvertRenderErrorNilRenderError(t *testing.T) {
	sess := &Session{}
	inst := &Instance{
		ID:          "test",
		RenderError: nil,
	}
	sess.trackConvertRenderError(inst)
	if len(sess.convertRenderErrors) != 0 {
		t.Error("expected no errors tracked when RenderError is nil")
	}
}

func TestTrackConvertRenderErrorTracksInstance(t *testing.T) {
	sess := &Session{}
	inst := &Instance{
		ID:          "test",
		RenderError: &Error{Message: "test error"},
	}
	sess.trackConvertRenderError(inst)

	if len(sess.convertRenderErrors) != 1 {
		t.Fatalf("expected 1 error tracked, got %d", len(sess.convertRenderErrors))
	}
	if sess.convertRenderErrors[0] != inst {
		t.Error("expected tracked instance to match")
	}
}

func TestTrackConvertRenderErrorTracksMultiple(t *testing.T) {
	sess := &Session{}
	inst1 := &Instance{
		ID:          "test1",
		RenderError: &Error{Message: "error 1"},
	}
	inst2 := &Instance{
		ID:          "test2",
		RenderError: &Error{Message: "error 2"},
	}

	sess.trackConvertRenderError(inst1)
	sess.trackConvertRenderError(inst2)

	if len(sess.convertRenderErrors) != 2 {
		t.Fatalf("expected 2 errors tracked, got %d", len(sess.convertRenderErrors))
	}
}

func TestConvertComponentTracksRenderError(t *testing.T) {
	sess := &Session{
		Components: make(map[string]*Instance),
	}

	panicComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		panic("component panic")
	}

	parent := &Instance{
		ID:                    "parent",
		Fn:                    func(*Ctx, any, []work.Item) work.Node { return nil },
		HookFrame:             []HookSlot{},
		Children:              []*Instance{},
		CombinedContextEpochs: make(map[contextID]int),
	}
	sess.Root = parent
	sess.Components["parent"] = parent

	compNode := &work.ComponentNode{
		Fn:    panicComponent,
		Key:   "",
		Props: nil,
	}

	sess.convertComponent(compNode, parent)

	if len(sess.convertRenderErrors) != 1 {
		t.Fatalf("expected 1 render error tracked, got %d", len(sess.convertRenderErrors))
	}

	trackedInst := sess.convertRenderErrors[0]
	if trackedInst.RenderError == nil {
		t.Error("expected tracked instance to have RenderError set")
	}
	if trackedInst.RenderError.Message != "component panic" {
		t.Errorf("expected error message 'component panic', got '%s'", trackedInst.RenderError.Message)
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

func TestConvertComponentSkipsErroredNodeWithAncestorBoundary(t *testing.T) {
	sess := &Session{
		Bus: protocol.NewBus(),
	}

	grandparent := &Instance{
		ID:        "grandparent",
		HookFrame: []HookSlot{{Type: HookTypeErrorBoundary, Value: nil}},
		Children:  []*Instance{},
	}

	parent := &Instance{
		ID:        "parent",
		Parent:    grandparent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	grandparent.Children = append(grandparent.Children, parent)

	childFn := func(ctx *Ctx) work.Node {
		t.Fatal("errored component should not be re-rendered")
		return &work.Text{Value: "should not render"}
	}

	childID := buildComponentID(parent, childFn, "testkey")

	child := &Instance{
		ID:                     childID,
		Parent:                 parent,
		errorHandledByBoundary: true,
		HookFrame:              []HookSlot{},
		Children:               []*Instance{},
		Key:                    "testkey",
		Fn:                     childFn,
	}
	parent.Children = append(parent.Children, child)

	sess.Root = grandparent
	sess.Components = map[string]*Instance{
		"grandparent": grandparent,
		"parent":      parent,
		childID:       child,
	}

	comp := &work.ComponentNode{
		Fn:   childFn,
		Key:  "testkey",
		Name: "child",
	}

	parent.ReferencedChildren = make(map[string]bool)

	result := sess.convertComponent(comp, parent)

	if result != nil {
		t.Errorf("expected nil result for errored component, got %v", result)
	}

	if len(sess.convertRenderErrors) != 0 {
		t.Errorf("expected no errors tracked (already handled by boundary), got %d", len(sess.convertRenderErrors))
	}
}

func TestNodeChangedCommentNodes(t *testing.T) {
	t.Run("both nil returns false", func(t *testing.T) {
		if nodeChanged(nil, nil) {
			t.Error("expected false for both nil")
		}
	})

	t.Run("prev nil returns true", func(t *testing.T) {
		if !nodeChanged(nil, &work.Comment{Value: "test"}) {
			t.Error("expected true when prev is nil")
		}
	})

	t.Run("curr nil returns true", func(t *testing.T) {
		if !nodeChanged(&work.Comment{Value: "test"}, nil) {
			t.Error("expected true when curr is nil")
		}
	})

	t.Run("same comment returns false", func(t *testing.T) {
		prev := &work.Comment{Value: "test"}
		curr := &work.Comment{Value: "test"}
		if nodeChanged(prev, curr) {
			t.Error("expected false for identical comments")
		}
	})

	t.Run("different comment returns true", func(t *testing.T) {
		prev := &work.Comment{Value: "test1"}
		curr := &work.Comment{Value: "test2"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different comments")
		}
	})

	t.Run("comment vs text returns true", func(t *testing.T) {
		prev := &work.Comment{Value: "test"}
		curr := &work.Text{Value: "test"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different node types")
		}
	})
}

func TestNodeChangedFragmentNodes(t *testing.T) {
	t.Run("same fragment returns false", func(t *testing.T) {
		prev := &work.Fragment{Children: []work.Node{&work.Text{Value: "a"}}}
		curr := &work.Fragment{Children: []work.Node{&work.Text{Value: "a"}}}
		if nodeChanged(prev, curr) {
			t.Error("expected false for identical fragments")
		}
	})

	t.Run("different fragment children returns true", func(t *testing.T) {
		prev := &work.Fragment{Children: []work.Node{&work.Text{Value: "a"}}}
		curr := &work.Fragment{Children: []work.Node{&work.Text{Value: "b"}}}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different fragment children")
		}
	})

	t.Run("fragment vs element returns true", func(t *testing.T) {
		prev := &work.Fragment{Children: []work.Node{}}
		curr := &work.Element{Tag: "div"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different node types")
		}
	})
}

func TestNodeChangedTextNodes(t *testing.T) {
	t.Run("same text returns false", func(t *testing.T) {
		prev := &work.Text{Value: "hello"}
		curr := &work.Text{Value: "hello"}
		if nodeChanged(prev, curr) {
			t.Error("expected false for identical text")
		}
	})

	t.Run("different text returns true", func(t *testing.T) {
		prev := &work.Text{Value: "hello"}
		curr := &work.Text{Value: "world"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different text")
		}
	})

	t.Run("text vs comment returns true", func(t *testing.T) {
		prev := &work.Text{Value: "test"}
		curr := &work.Comment{Value: "test"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different node types")
		}
	})
}

func TestNodeChangedElementNodes(t *testing.T) {
	t.Run("same element returns false", func(t *testing.T) {
		prev := &work.Element{Tag: "div", Key: "k1", Attrs: map[string][]string{"class": {"foo"}}}
		curr := &work.Element{Tag: "div", Key: "k1", Attrs: map[string][]string{"class": {"foo"}}}
		if nodeChanged(prev, curr) {
			t.Error("expected false for identical elements")
		}
	})

	t.Run("different tag returns true", func(t *testing.T) {
		prev := &work.Element{Tag: "div"}
		curr := &work.Element{Tag: "span"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different tags")
		}
	})

	t.Run("different key returns true", func(t *testing.T) {
		prev := &work.Element{Tag: "div", Key: "k1"}
		curr := &work.Element{Tag: "div", Key: "k2"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different keys")
		}
	})

	t.Run("different attrs returns true", func(t *testing.T) {
		prev := &work.Element{Tag: "div", Attrs: map[string][]string{"class": {"a"}}}
		curr := &work.Element{Tag: "div", Attrs: map[string][]string{"class": {"b"}}}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different attrs")
		}
	})

	t.Run("different style returns true", func(t *testing.T) {
		prev := &work.Element{Tag: "div", Style: map[string]string{"color": "red"}}
		curr := &work.Element{Tag: "div", Style: map[string]string{"color": "blue"}}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different styles")
		}
	})

	t.Run("different unsafeHTML returns true", func(t *testing.T) {
		prev := &work.Element{Tag: "div", UnsafeHTML: "<b>a</b>"}
		curr := &work.Element{Tag: "div", UnsafeHTML: "<b>b</b>"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different unsafeHTML")
		}
	})

	t.Run("different children returns true", func(t *testing.T) {
		prev := &work.Element{Tag: "div", Children: []work.Node{&work.Text{Value: "a"}}}
		curr := &work.Element{Tag: "div", Children: []work.Node{&work.Text{Value: "b"}}}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different children")
		}
	})

	t.Run("element vs fragment returns true", func(t *testing.T) {
		prev := &work.Element{Tag: "div"}
		curr := &work.Fragment{}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different node types")
		}
	})
}

func TestNodeChangedComponentNodes(t *testing.T) {
	fn := func(*Ctx, any, []work.Item) work.Node { return nil }

	t.Run("same component returns false", func(t *testing.T) {
		prev := &work.ComponentNode{Key: "k1", Props: "props"}
		curr := &work.ComponentNode{Key: "k1", Props: "props"}
		if nodeChanged(prev, curr) {
			t.Error("expected false for identical components")
		}
	})

	t.Run("different key returns true", func(t *testing.T) {
		prev := &work.ComponentNode{Fn: fn, Key: "k1"}
		curr := &work.ComponentNode{Fn: fn, Key: "k2"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different keys")
		}
	})

	t.Run("different props returns true", func(t *testing.T) {
		prev := &work.ComponentNode{Fn: fn, Props: "a"}
		curr := &work.ComponentNode{Fn: fn, Props: "b"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different props")
		}
	})

	t.Run("different input children returns true", func(t *testing.T) {
		prev := &work.ComponentNode{Fn: fn, InputChildren: []work.Node{&work.Text{Value: "a"}}}
		curr := &work.ComponentNode{Fn: fn, InputChildren: []work.Node{&work.Text{Value: "b"}}}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different input children")
		}
	})

	t.Run("component vs element returns true", func(t *testing.T) {
		prev := &work.ComponentNode{Fn: fn}
		curr := &work.Element{Tag: "div"}
		if !nodeChanged(prev, curr) {
			t.Error("expected true for different node types")
		}
	})
}

func TestInputChildrenChanged(t *testing.T) {
	t.Run("different lengths returns true", func(t *testing.T) {
		prev := []work.Node{&work.Text{Value: "a"}}
		curr := []work.Node{&work.Text{Value: "a"}, &work.Text{Value: "b"}}
		if !inputChildrenChanged(prev, curr) {
			t.Error("expected true for different lengths")
		}
	})

	t.Run("same children returns false", func(t *testing.T) {
		prev := []work.Node{&work.Text{Value: "a"}, &work.Text{Value: "b"}}
		curr := []work.Node{&work.Text{Value: "a"}, &work.Text{Value: "b"}}
		if inputChildrenChanged(prev, curr) {
			t.Error("expected false for identical children")
		}
	})

	t.Run("empty slices returns false", func(t *testing.T) {
		if inputChildrenChanged(nil, nil) {
			t.Error("expected false for both nil")
		}
		if inputChildrenChanged([]work.Node{}, []work.Node{}) {
			t.Error("expected false for both empty")
		}
	})
}

func TestCleanupStaleHandlers(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	sub1 := sess.Bus.SubscribeToHandlerInvoke("handler1", func(data interface{}) {})
	sub2 := sess.Bus.SubscribeToHandlerInvoke("handler2", func(data interface{}) {})
	sub3 := sess.Bus.SubscribeToHandlerInvoke("handler3", func(data interface{}) {})

	sess.allHandlerSubs["handler1"] = sub1
	sess.allHandlerSubs["handler2"] = sub2
	sess.allHandlerSubs["handler3"] = sub3

	sess.currentHandlerIDs["handler1"] = true
	sess.currentHandlerIDs["handler3"] = true

	sess.cleanupStaleHandlers()

	if _, exists := sess.allHandlerSubs["handler1"]; !exists {
		t.Error("handler1 should still exist (was current)")
	}
	if _, exists := sess.allHandlerSubs["handler2"]; exists {
		t.Error("handler2 should be removed (was stale)")
	}
	if _, exists := sess.allHandlerSubs["handler3"]; !exists {
		t.Error("handler3 should still exist (was current)")
	}
}

func TestCleanupStaleHandlersNilSub(t *testing.T) {
	sess := &Session{
		Bus:               protocol.NewBus(),
		currentHandlerIDs: make(map[string]bool),
		allHandlerSubs:    make(map[string]*protocol.Subscription),
	}

	sess.allHandlerSubs["handler1"] = nil

	sess.cleanupStaleHandlers()

	if _, exists := sess.allHandlerSubs["handler1"]; exists {
		t.Error("handler1 should be removed even with nil subscription")
	}
}

func TestResetRefsForComponentWithElementRef(t *testing.T) {
	sess := &Session{}

	ref := &ElementRef{attached: true}
	inst := &Instance{
		ID: "test",
		HookFrame: []HookSlot{
			{Type: HookTypeElement, Value: ref},
		},
	}

	if !ref.attached {
		t.Error("ref should be attached before reset")
	}

	sess.resetRefsForComponent(inst)

	if ref.attached {
		t.Error("ref should be detached after reset")
	}
}

func TestResetRefsForComponentNilInstance(t *testing.T) {
	sess := &Session{}
	sess.resetRefsForComponent(nil)
}

func TestResetRefsForComponentNoElementRefs(t *testing.T) {
	sess := &Session{}
	inst := &Instance{
		ID: "test",
		HookFrame: []HookSlot{
			{Type: HookTypeState, Value: nil},
			{Type: HookTypeEffect, Value: nil},
		},
	}

	sess.resetRefsForComponent(inst)
}

func TestResetRefsForComponentNonElementRefValue(t *testing.T) {
	sess := &Session{}
	inst := &Instance{
		ID: "test",
		HookFrame: []HookSlot{
			{Type: HookTypeElement, Value: "not an element ref"},
		},
	}

	sess.resetRefsForComponent(inst)
}

func TestHasAncestorErrorBoundary(t *testing.T) {
	sess := &Session{}

	root := &Instance{
		ID:        "root",
		HookFrame: []HookSlot{},
	}

	parent := &Instance{
		ID:        "parent",
		Parent:    root,
		HookFrame: []HookSlot{{Type: HookTypeErrorBoundary, Value: nil}},
	}

	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}

	grandchild := &Instance{
		ID:        "grandchild",
		Parent:    child,
		HookFrame: []HookSlot{},
	}

	if sess.hasAncestorErrorBoundary(nil) {
		t.Error("nil instance should return false")
	}

	if sess.hasAncestorErrorBoundary(root) {
		t.Error("root with no parent should return false")
	}

	if !sess.hasAncestorErrorBoundary(child) {
		t.Error("child with parent having error boundary should return true")
	}

	if !sess.hasAncestorErrorBoundary(grandchild) {
		t.Error("grandchild with ancestor having error boundary should return true")
	}

	parentNoEB := &Instance{
		ID:        "parentNoEB",
		Parent:    root,
		HookFrame: []HookSlot{{Type: HookTypeState, Value: nil}},
	}

	childNoEB := &Instance{
		ID:        "childNoEB",
		Parent:    parentNoEB,
		HookFrame: []HookSlot{},
	}

	if sess.hasAncestorErrorBoundary(childNoEB) {
		t.Error("child with no ancestor error boundary should return false")
	}
}
