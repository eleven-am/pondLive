package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/view/diff"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestFlushRendersRootComponent(t *testing.T) {

	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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

	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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

	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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

	childComponent := func(ctx *Ctx, props any, _ []work.Node) work.Node {
		msg := props.(string)
		return work.BuildElement("span", work.NewText(msg))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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

	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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

	childComponent := func(ctx *Ctx, props any, _ []work.Node) work.Node {
		childRenderCount++
		msg := props.(string)
		return work.BuildElement("span", work.NewText(msg))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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

	childComponent := func(ctx *Ctx, props any, _ []work.Node) work.Node {
		childRenderCount++
		msg := props.(string)
		return work.BuildElement("span", work.NewText(msg))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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

	childComponent := func(ctx *Ctx, props any, _ []work.Node) work.Node {
		childRenderCount++
		p := props.(ChildProps)
		return work.BuildElement("span", work.NewTextf("%s: %d", p.Name, p.Count))
	}

	parentComponent := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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
	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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
	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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
	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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
	component := func(ctx *Ctx, _ any, _ []work.Node) work.Node {
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
