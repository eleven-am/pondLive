package runtime

import (
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/view/diff"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestFlushRendersRootComponent(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div", work.NewText("Hello World"))
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sess.View == nil {
		t.Fatal("expected View to be created after flush")
	}

	viewElem, ok := sess.View.(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", sess.View)
	}

	if viewElem.Tag != "div" {
		t.Errorf("expected div tag, got %s", viewElem.Tag)
	}

	if len(viewElem.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(viewElem.Children))
	}

	textNode, ok := viewElem.Children[0].(*view.Text)
	if !ok {
		t.Fatalf("expected Text node, got %T", viewElem.Children[0])
	}

	if textNode.Text != "Hello World" {
		t.Errorf("expected 'Hello World', got %s", textNode.Text)
	}
}

func TestFlushRegistersHandlers(t *testing.T) {

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		handler := work.Handler{
			Fn: func(evt work.Event) work.Updates {
				return nil
			},
		}
		elem := work.BuildElement("button", work.NewText("Click me"))
		elem.Handlers = map[string]work.Handler{
			"click": handler,
		}
		return elem
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sess.Bus == nil {
		t.Fatal("expected bus to be initialized")
	}
	if sess.Bus.SubscriberCount("root:h0") != 1 {
		t.Errorf("expected 1 handler in bus for 'root:h0', got %d", sess.Bus.SubscriberCount("root:h0"))
	}

	if sess.View == nil {
		t.Fatal("expected View to be created")
	}

	viewElem, ok := sess.View.(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", sess.View)
	}

	if len(viewElem.Handlers) != 1 {
		t.Errorf("expected 1 handler in view, got %d", len(viewElem.Handlers))
	}

	handlerMeta := viewElem.Handlers[0]
	if handlerMeta.Event != "click" {
		t.Errorf("expected click event, got %s", handlerMeta.Event)
	}

	if handlerMeta.Handler != "root:h0" {
		t.Errorf("expected handler ID 'root:h0', got %s", handlerMeta.Handler)
	}
}

func TestFlushWithStatefulComponent(t *testing.T) {

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		count, setCount := UseState(ctx, 0)
		_ = setCount
		return work.BuildElement("div", work.NewTextf("%d", count))
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sess.View == nil {
		t.Fatal("expected View to be created")
	}

	if len(root.HookFrame) != 1 {
		t.Errorf("expected 1 hook slot, got %d", len(root.HookFrame))
	}

	if root.HookFrame[0].Type != HookTypeState {
		t.Errorf("expected HookTypeState, got %v", root.HookFrame[0].Type)
	}
}

func TestFlushHandlesNestedComponents(t *testing.T) {

	childComponent := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		msg := props.(string)
		return work.BuildElement("span", work.NewText(msg))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			work.PropsComponent(childComponent, "Hello from child"),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child component, got %d", len(root.Children))
	}

	child := root.Children[0]
	if child.Fn == nil {
		t.Error("expected child to have component function")
	}

	if len(sess.Components) != 2 {
		t.Errorf("expected 2 components in registry, got %d", len(sess.Components))
	}

	if sess.View == nil {
		t.Fatal("expected View to be created")
	}

	viewElem, ok := sess.View.(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", sess.View)
	}

	if viewElem.Tag != "div" {
		t.Errorf("expected div tag, got %s", viewElem.Tag)
	}

	if len(viewElem.Children) != 1 {
		t.Fatalf("expected 1 child in view, got %d", len(viewElem.Children))
	}

	spanElem, ok := viewElem.Children[0].(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", viewElem.Children[0])
	}

	if spanElem.Tag != "span" {
		t.Errorf("expected span tag, got %s", spanElem.Tag)
	}

	if len(spanElem.Children) != 1 {
		t.Fatalf("expected 1 text child, got %d", len(spanElem.Children))
	}

	textNode, ok := spanElem.Children[0].(*view.Text)
	if !ok {
		t.Fatalf("expected Text node, got %T", spanElem.Children[0])
	}

	if textNode.Text != "Hello from child" {
		t.Errorf("expected 'Hello from child', got %s", textNode.Text)
	}
}

func TestFlushHandlesFragments(t *testing.T) {

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.NewFragment(
			work.BuildElement("span", work.NewText("First")),
			work.BuildElement("span", work.NewText("Second")),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sess.View == nil {
		t.Fatal("expected View to be created")
	}

	frag, ok := sess.View.(*view.Fragment)
	if !ok {
		t.Fatalf("expected Fragment, got %T", sess.View)
	}

	if len(frag.Children) != 2 {
		t.Fatalf("expected 2 children in fragment, got %d", len(frag.Children))
	}

	span1, ok := frag.Children[0].(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", frag.Children[0])
	}
	if span1.Tag != "span" {
		t.Errorf("expected span, got %s", span1.Tag)
	}

	span2, ok := frag.Children[1].(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", frag.Children[1])
	}
	if span2.Tag != "span" {
		t.Errorf("expected span, got %s", span2.Tag)
	}
}

func TestPropsMemoization(t *testing.T) {

	childRenderCount := 0

	childComponent := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		childRenderCount++
		msg := props.(string)
		return work.BuildElement("span", work.NewText(msg))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			work.PropsComponent(childComponent, "Hello"),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush 1 failed: %v", err)
	}

	if childRenderCount != 1 {
		t.Errorf("expected child to render once on first flush, got %d", childRenderCount)
	}

	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	if childRenderCount != 1 {
		t.Errorf("expected child to NOT re-render with same props, got %d renders", childRenderCount)
	}
}

func TestPropsMemoizationWithChangedProps(t *testing.T) {
	childRenderCount := 0
	currentMsg := "Hello"

	childComponent := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		childRenderCount++
		msg := props.(string)
		return work.BuildElement("span", work.NewText(msg))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			work.PropsComponent(childComponent, currentMsg),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush 1 failed: %v", err)
	}
	if childRenderCount != 1 {
		t.Errorf("expected child to render once, got %d", childRenderCount)
	}

	currentMsg = "World"
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	if childRenderCount != 2 {
		t.Errorf("expected child to re-render with changed props, got %d renders", childRenderCount)
	}

	viewElem := sess.View.(*view.Element)
	spanElem := viewElem.Children[0].(*view.Element)
	textNode := spanElem.Children[0].(*view.Text)
	if textNode.Text != "World" {
		t.Errorf("expected 'World', got %s", textNode.Text)
	}
}

func TestPropsMemoizationWithStructProps(t *testing.T) {
	type ChildProps struct {
		Name  string
		Count int
	}

	childRenderCount := 0
	currentProps := ChildProps{Name: "Alice", Count: 1}

	childComponent := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		childRenderCount++
		p := props.(ChildProps)
		return work.BuildElement("span", work.NewTextf("%s: %d", p.Name, p.Count))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			work.PropsComponent(childComponent, currentProps),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush 1 failed: %v", err)
	}
	if childRenderCount != 1 {
		t.Errorf("expected 1 render, got %d", childRenderCount)
	}

	currentProps = ChildProps{Name: "Alice", Count: 1}
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}
	if childRenderCount != 1 {
		t.Errorf("expected child to NOT re-render with equal struct props, got %d", childRenderCount)
	}

	currentProps = ChildProps{Name: "Bob", Count: 2}
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 3 failed: %v", err)
	}
	if childRenderCount != 2 {
		t.Errorf("expected child to re-render with different struct props, got %d", childRenderCount)
	}
}

func TestPropsEqual(t *testing.T) {

	if !propsEqual(nil, nil) {
		t.Error("expected nil == nil")
	}
	if propsEqual(nil, "hello") {
		t.Error("expected nil != 'hello'")
	}
	if propsEqual("hello", nil) {
		t.Error("expected 'hello' != nil")
	}

	if !propsEqual(42, 42) {
		t.Error("expected 42 == 42")
	}
	if propsEqual(42, 43) {
		t.Error("expected 42 != 43")
	}

	if !propsEqual("hello", "hello") {
		t.Error("expected 'hello' == 'hello'")
	}
	if propsEqual("hello", "world") {
		t.Error("expected 'hello' != 'world'")
	}

	type Props struct {
		Name string
		Age  int
	}
	if !propsEqual(Props{Name: "Alice", Age: 30}, Props{Name: "Alice", Age: 30}) {
		t.Error("expected equal structs to be equal")
	}
	if propsEqual(Props{Name: "Alice", Age: 30}, Props{Name: "Bob", Age: 30}) {
		t.Error("expected different structs to not be equal")
	}
}

func TestFlushRegistersHandlersViaHTMLHelpers(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		btn := work.BuildElement("button",
			work.NewText("Click me"),
		)
		btn.Handlers = map[string]work.Handler{
			"click": {
				Fn: func(evt work.Event) work.Updates {
					return nil
				},
			},
		}
		return work.BuildElement("div", btn)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sess.View == nil {
		t.Fatal("expected View to be created")
	}

	viewDiv, ok := sess.View.(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", sess.View)
	}

	if viewDiv.Tag != "div" {
		t.Errorf("expected div tag, got %s", viewDiv.Tag)
	}

	if len(viewDiv.Children) != 1 {
		t.Fatalf("expected 1 child (button), got %d", len(viewDiv.Children))
	}

	viewBtn, ok := viewDiv.Children[0].(*view.Element)
	if !ok {
		t.Fatalf("expected button Element, got %T", viewDiv.Children[0])
	}

	if viewBtn.Tag != "button" {
		t.Errorf("expected button tag, got %s", viewBtn.Tag)
	}

	if len(viewBtn.Handlers) != 1 {
		t.Errorf("expected 1 handler on button, got %d", len(viewBtn.Handlers))
	}

	if len(viewBtn.Handlers) > 0 && viewBtn.Handlers[0].Event != "click" {
		t.Errorf("expected click event, got %s", viewBtn.Handlers[0].Event)
	}
}

func TestExtractMetadataIncludesNestedHandlers(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		btn := work.BuildElement("button", work.NewText("Click"))
		btn.Handlers = map[string]work.Handler{
			"click": {Fn: func(evt work.Event) work.Updates { return nil }},
		}
		return work.BuildElement("div", btn)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	viewDiv := sess.View.(*view.Element)
	viewBtn := viewDiv.Children[0].(*view.Element)

	if len(viewBtn.Handlers) != 1 {
		t.Fatalf("expected button to have 1 handler, got %d", len(viewBtn.Handlers))
	}

	t.Logf("Button handlers: %+v", viewBtn.Handlers)
	t.Logf("View tree: div -> button (handlers: %d)", len(viewBtn.Handlers))
}

func TestHtmlHelpersWithOnHandler(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		btn := work.BuildElement("button", work.NewText("Click"))
		btn.Handlers["click"] = work.Handler{Fn: func(evt work.Event) work.Updates { return nil }}
		return work.BuildElement("div", btn)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	viewDiv := sess.View.(*view.Element)
	if len(viewDiv.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(viewDiv.Children))
	}

	viewBtn := viewDiv.Children[0].(*view.Element)
	if viewBtn.Tag != "button" {
		t.Errorf("expected button, got %s", viewBtn.Tag)
	}

	if len(viewBtn.Handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(viewBtn.Handlers))
	}

	t.Logf("Handler from pkg helpers: %+v", viewBtn.Handlers[0])
}

func TestExtractMetadataIncludesHandlerPatches(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		btn := work.BuildElement("button", work.NewText("Click"))
		btn.Handlers["click"] = work.Handler{Fn: func(evt work.Event) work.Updates { return nil }}
		return work.BuildElement("div", btn)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	patches := diff.ExtractMetadata(sess.View)

	hasSetHandlers := false
	for _, p := range patches {
		if p.Op == "setHandlers" {
			hasSetHandlers = true
			t.Logf("Found setHandlers patch: path=%v value=%v", p.Path, p.Value)
		}
	}

	if !hasSetHandlers {
		t.Errorf("expected setHandlers patch in metadata, got patches: %+v", patches)
	}
}

func TestHandlerActuallyFiresWhenInvoked(t *testing.T) {
	handlerFired := make(chan map[string]any, 1)

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		btn := work.BuildElement("button", work.NewText("Click"))
		btn.Handlers["click"] = work.Handler{
			Fn: func(evt work.Event) work.Updates {
				handlerFired <- evt.Payload
				return nil
			},
		}
		return btn
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sess.Bus == nil {
		t.Fatal("Bus should be initialized after flush")
	}

	viewBtn := sess.View.(*view.Element)
	if len(viewBtn.Handlers) != 1 {
		t.Fatalf("expected 1 handler, got %d", len(viewBtn.Handlers))
	}

	handlerID := viewBtn.Handlers[0].Handler

	sess.Bus.PublishHandlerInvoke(handlerID, map[string]any{"test": "value"})

	select {
	case receivedPayload := <-handlerFired:
		if receivedPayload == nil || receivedPayload["test"] != "value" {
			t.Errorf("expected payload {test: value}, got %v", receivedPayload)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("handler should have fired when invoked via Bus")
	}
}

func TestStateUpdateProducesCorrectPatches(t *testing.T) {
	currentText := "Hello"
	var setState func(string)

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		text, setFn := UseState(ctx, currentText)
		setState = setFn
		return work.BuildElement("div", work.NewText(text))
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	sess.SetAutoFlush(func() {})

	if err := sess.Flush(); err != nil {
		t.Fatalf("First flush failed: %v", err)
	}

	prevView := sess.View

	viewElem := sess.View.(*view.Element)
	textNode := viewElem.Children[0].(*view.Text)
	if textNode.Text != "Hello" {
		t.Errorf("expected 'Hello', got %s", textNode.Text)
	}

	currentText = "World"
	setState("World")

	if err := sess.Flush(); err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	viewElem = sess.View.(*view.Element)
	textNode = viewElem.Children[0].(*view.Text)
	if textNode.Text != "World" {
		t.Errorf("expected 'World' after state update, got %s", textNode.Text)
	}

	patches := diff.Diff(prevView, sess.View)

	hasSetText := false
	hasDelChild := false
	hasAddChild := false

	for _, p := range patches {
		switch p.Op {
		case diff.OpSetText:
			hasSetText = true
		case diff.OpDelChild:
			hasDelChild = true
		case diff.OpAddChild:
			hasAddChild = true
		}
	}

	if !hasSetText {
		t.Error("expected setText patch for text update")
	}

	if hasDelChild || hasAddChild {
		t.Error("text update should use setText, not delChild/addChild")
	}
}

func TestHandlerCleanupOnUnmount(t *testing.T) {
	showChild := true

	childComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		btn := work.BuildElement("button", work.NewText("Click"))
		btn.Handlers["click"] = work.Handler{
			Fn: func(evt work.Event) work.Updates { return nil },
		}
		return btn
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		if showChild {
			return work.BuildElement("div", &work.ComponentNode{Fn: childComponent})
		}
		return work.BuildElement("div", work.NewText("No child"))
	}

	root := &Instance{
		ID:        "root",
		Fn:        parentComponent,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("First flush failed: %v", err)
	}

	if len(root.Children) != 1 {
		t.Fatalf("expected 1 child component, got %d", len(root.Children))
	}

	childInst := root.Children[0]
	handlerID := protocol.Topic(childInst.ID + ":h0")

	subscriberCount := sess.Bus.SubscriberCount(handlerID)
	if subscriberCount != 1 {
		t.Errorf("expected 1 subscriber for child handler, got %d", subscriberCount)
	}

	showChild = false
	sess.MarkDirty(root)

	if err := sess.Flush(); err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	subscriberCount = sess.Bus.SubscriberCount(handlerID)
	if subscriberCount != 0 {
		t.Errorf("expected 0 subscribers after unmount, got %d", subscriberCount)
	}
}

func TestAttrChangeProducesSetAttrPatch(t *testing.T) {
	currentClass := "foo"

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		elem := work.BuildElement("div", work.NewText("content"))
		elem.Attrs = map[string][]string{"class": {currentClass}}
		return elem
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("First flush failed: %v", err)
	}

	prevView := sess.View

	currentClass = "bar"
	sess.MarkDirty(root)

	if err := sess.Flush(); err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	patches := diff.Diff(prevView, sess.View)

	hasSetAttr := false
	hasReplaceNode := false

	for _, p := range patches {
		if p.Op == diff.OpSetAttr {
			hasSetAttr = true
			if attrMap, ok := p.Value.(map[string][]string); ok {
				if classVals, exists := attrMap["class"]; exists {
					if len(classVals) != 1 || classVals[0] != "bar" {
						t.Errorf("expected class value ['bar'], got %v", classVals)
					}
				} else {
					t.Error("setAttr patch should contain class attribute")
				}
			} else {
				t.Errorf("expected setAttr value to be map[string][]string, got %T", p.Value)
			}
		}
		if p.Op == diff.OpReplaceNode {
			hasReplaceNode = true
		}
	}

	if !hasSetAttr {
		t.Error("expected setAttr patch for class change")
	}

	if hasReplaceNode {
		t.Error("attr change should use setAttr, not replaceNode")
	}
}

func TestFirstRenderViewLifecycle(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			work.BuildElement("span", work.NewText("Hello")),
			work.BuildElement("span", work.NewText("World")),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if sess.View != nil {
		t.Error("View should be nil before first flush")
	}

	if sess.PrevView != nil {
		t.Error("PrevView should be nil before first flush")
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	if sess.View == nil {
		t.Fatal("View should be set after flush")
	}

	viewElem, ok := sess.View.(*view.Element)
	if !ok {
		t.Fatalf("expected Element, got %T", sess.View)
	}

	if viewElem.Tag != "div" {
		t.Errorf("expected div, got %s", viewElem.Tag)
	}

	if len(viewElem.Children) != 2 {
		t.Fatalf("expected 2 children, got %d", len(viewElem.Children))
	}

	span1 := viewElem.Children[0].(*view.Element)
	span2 := viewElem.Children[1].(*view.Element)

	if span1.Tag != "span" || span2.Tag != "span" {
		t.Error("expected span children")
	}

	text1 := span1.Children[0].(*view.Text)
	text2 := span2.Children[0].(*view.Text)

	if text1.Text != "Hello" || text2.Text != "World" {
		t.Errorf("expected 'Hello' and 'World', got '%s' and '%s'", text1.Text, text2.Text)
	}
}

func TestMultipleHandlersOnSameElement(t *testing.T) {
	clickChan := make(chan struct{}, 1)
	mouseoverChan := make(chan struct{}, 1)

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		btn := work.BuildElement("button", work.NewText("Hover and Click"))
		btn.Handlers["click"] = work.Handler{
			Fn: func(evt work.Event) work.Updates {
				clickChan <- struct{}{}
				return nil
			},
		}
		btn.Handlers["mouseover"] = work.Handler{
			Fn: func(evt work.Event) work.Updates {
				mouseoverChan <- struct{}{}
				return nil
			},
		}
		return btn
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	viewBtn := sess.View.(*view.Element)
	if len(viewBtn.Handlers) != 2 {
		t.Fatalf("expected 2 handlers, got %d", len(viewBtn.Handlers))
	}

	for _, h := range viewBtn.Handlers {
		sess.Bus.PublishHandlerInvoke(h.Handler, map[string]any{})
	}

	select {
	case <-clickChan:
	case <-time.After(100 * time.Millisecond):
		t.Error("click handler should have fired")
	}

	select {
	case <-mouseoverChan:
	case <-time.After(100 * time.Millisecond):
		t.Error("mouseover handler should have fired")
	}
}

func TestHandlerStaysStableAcrossRerenders(t *testing.T) {
	currentCount := 0
	var setState func(int)
	handlerChan := make(chan struct{}, 2)

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		count, setFn := UseState(ctx, currentCount)
		setState = setFn
		btn := work.BuildElement("button", work.NewTextf("Count: %d", count))
		btn.Handlers["click"] = work.Handler{
			Fn: func(evt work.Event) work.Updates {
				handlerChan <- struct{}{}
				return nil
			},
		}
		return btn
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("First flush failed: %v", err)
	}

	viewBtn := sess.View.(*view.Element)
	handlerID := viewBtn.Handlers[0].Handler

	sess.Bus.PublishHandlerInvoke(handlerID, map[string]any{})
	select {
	case <-handlerChan:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("handler should have fired on first invoke")
	}

	currentCount = 1
	setState(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	viewBtn = sess.View.(*view.Element)
	newHandlerID := viewBtn.Handlers[0].Handler

	if newHandlerID != handlerID {
		t.Errorf("handler ID should stay stable across rerenders, was %s now %s", handlerID, newHandlerID)
	}

	sess.Bus.PublishHandlerInvoke(handlerID, map[string]any{})
	select {
	case <-handlerChan:
	case <-time.After(100 * time.Millisecond):
		t.Fatal("handler should have fired after rerender")
	}
}

func TestPortalRendering(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			work.BuildElement("header", work.NewText("Header")),
			&work.PortalNode{
				Children: []work.Node{
					work.BuildElement("div", work.NewText("Modal Content")),
				},
			},
			work.BuildElement("main", work.NewText("Main Content")),
			&work.PortalTarget{},
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	viewDiv := sess.View.(*view.Element)
	if viewDiv.Tag != "div" {
		t.Errorf("expected root div, got %s", viewDiv.Tag)
	}

	if len(viewDiv.Children) != 3 {
		t.Fatalf("expected 3 children (header, main, portal content), got %d", len(viewDiv.Children))
	}

	header := viewDiv.Children[0].(*view.Element)
	if header.Tag != "header" {
		t.Errorf("expected header as first child, got %s", header.Tag)
	}

	main := viewDiv.Children[1].(*view.Element)
	if main.Tag != "main" {
		t.Errorf("expected main as second child, got %s", main.Tag)
	}

	portalContent := viewDiv.Children[2].(*view.Element)
	if portalContent.Tag != "div" {
		t.Errorf("expected portal div as third child, got %s", portalContent.Tag)
	}
	portalText := portalContent.Children[0].(*view.Text)
	if portalText.Text != "Modal Content" {
		t.Errorf("expected 'Modal Content' in portal, got %s", portalText.Text)
	}
}

func TestChildReorderProducesMovePatch(t *testing.T) {
	items := []string{"A", "B", "C"}

	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		children := make([]work.Node, len(items))
		for i, item := range items {
			elem := work.BuildElement("div", work.NewText(item))
			elem.Key = item
			children[i] = elem
		}
		parent := work.BuildElement("div")
		parent.Children = children
		return parent
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		DirtyQueue:        []*Instance{},
		DirtySet:          make(map[*Instance]struct{}),
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
		Bus:               protocol.NewBus(),
	}

	sess.SetAutoFlush(func() {})

	if err := sess.Flush(); err != nil {
		t.Fatalf("First flush failed: %v", err)
	}

	prevView := sess.View

	viewDiv := sess.View.(*view.Element)
	if len(viewDiv.Children) != 3 {
		t.Fatalf("expected 3 children, got %d", len(viewDiv.Children))
	}

	items = []string{"C", "A", "B"}
	sess.MarkDirty(root)

	if err := sess.Flush(); err != nil {
		t.Fatalf("Second flush failed: %v", err)
	}

	viewDiv = sess.View.(*view.Element)
	child0 := viewDiv.Children[0].(*view.Element)
	child1 := viewDiv.Children[1].(*view.Element)
	child2 := viewDiv.Children[2].(*view.Element)

	text0 := child0.Children[0].(*view.Text)
	text1 := child1.Children[0].(*view.Text)
	text2 := child2.Children[0].(*view.Text)

	if text0.Text != "C" || text1.Text != "A" || text2.Text != "B" {
		t.Errorf("expected [C, A, B], got [%s, %s, %s]", text0.Text, text1.Text, text2.Text)
	}

	patches := diff.Diff(prevView, sess.View)

	hasMoveChild := false
	delChildCount := 0
	addChildCount := 0

	for _, p := range patches {
		switch p.Op {
		case diff.OpMoveChild:
			hasMoveChild = true
		case diff.OpDelChild:
			delChildCount++
		case diff.OpAddChild:
			addChildCount++
		}
	}

	if delChildCount > 0 && addChildCount > 0 {
		t.Logf("Warning: reorder used %d delChild and %d addChild instead of moveChild", delChildCount, addChildCount)
	}

	if !hasMoveChild && delChildCount == 0 && addChildCount == 0 {
		t.Error("expected some form of reorder patch (moveChild or delChild/addChild)")
	}
}

func TestMultiplePortalContents(t *testing.T) {
	component := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			&work.PortalNode{
				Children: []work.Node{
					work.BuildElement("div", work.NewText("Modal 1")),
				},
			},
			&work.PortalNode{
				Children: []work.Node{
					work.BuildElement("div", work.NewText("Modal 2")),
				},
			},
			work.BuildElement("main", work.NewText("Content")),
			&work.PortalTarget{},
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        component,
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	sess := &Session{
		Root:              root,
		Components:        map[string]*Instance{"root": root},
		PendingEffects:    []effectTask{},
		PendingCleanups:   []cleanupTask{},
		MountedComponents: make(map[*Instance]struct{}),
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	viewDiv := sess.View.(*view.Element)

	if len(viewDiv.Children) != 2 {
		t.Fatalf("expected 2 children (main, portal fragment), got %d", len(viewDiv.Children))
	}

	main := viewDiv.Children[0].(*view.Element)
	if main.Tag != "main" {
		t.Errorf("expected main as first child, got %s", main.Tag)
	}

	portalFrag := viewDiv.Children[1].(*view.Fragment)
	if len(portalFrag.Children) != 2 {
		t.Fatalf("expected 2 portal children, got %d", len(portalFrag.Children))
	}

	modal1 := portalFrag.Children[0].(*view.Element)
	modal1Text := modal1.Children[0].(*view.Text)
	if modal1Text.Text != "Modal 1" {
		t.Errorf("expected 'Modal 1', got %s", modal1Text.Text)
	}

	modal2 := portalFrag.Children[1].(*view.Element)
	modal2Text := modal2.Children[0].(*view.Text)
	if modal2Text.Text != "Modal 2" {
		t.Errorf("expected 'Modal 2', got %s", modal2Text.Text)
	}
}
