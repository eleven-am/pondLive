package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestNestedComponentReceivesUpdatedProps(t *testing.T) {
	nestedRenderCount := 0
	var receivedProps []bool

	nestedChild := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		nestedRenderCount++
		isEditMode := props.(bool)
		receivedProps = append(receivedProps, isEditMode)
		if isEditMode {
			return work.BuildElement("span", work.NewText("Edit Mode"))
		}
		return work.BuildElement("span", work.NewText("View Mode"))
	}

	wrapperRenderCount := 0

	wrapper := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		wrapperRenderCount++
		return work.BuildElement("div", children...)
	}

	currentEditMode := false

	page := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("main",
			work.PropsComponent(nestedChild, currentEditMode),
			work.Component(wrapper,
				work.PropsComponent(nestedChild, currentEditMode),
			),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        page,
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

	if nestedRenderCount != 2 {
		t.Errorf("expected nested to render twice (direct + wrapped), got %d", nestedRenderCount)
	}

	for i, prop := range receivedProps {
		if prop != false {
			t.Errorf("render %d: expected false, got %v", i+1, prop)
		}
	}

	nestedRenderCount = 0
	receivedProps = nil
	wrapperRenderCount = 0

	currentEditMode = true
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	if len(receivedProps) < 2 {
		t.Fatalf("expected at least 2 renders of nested child, got %d", len(receivedProps))
	}

	for i, prop := range receivedProps {
		if prop != true {
			t.Errorf("render %d after state change: expected true, got %v (STALE PROPS BUG)", i+1, prop)
		}
	}

	mainElem := sess.View.(*view.Element)
	if mainElem.Tag != "main" {
		t.Fatalf("expected main element, got %s", mainElem.Tag)
	}

	if len(mainElem.Children) < 2 {
		t.Fatalf("expected at least 2 children, got %d", len(mainElem.Children))
	}

	directChild := mainElem.Children[0].(*view.Element)
	if len(directChild.Children) == 0 {
		t.Fatal("direct child has no text")
	}
	directText := directChild.Children[0].(*view.Text)
	if directText.Text != "Edit Mode" {
		t.Errorf("direct child: expected 'Edit Mode', got '%s'", directText.Text)
	}

	wrapperDiv := mainElem.Children[1].(*view.Element)
	if len(wrapperDiv.Children) == 0 {
		t.Fatal("wrapper has no children")
	}
	wrappedChild := wrapperDiv.Children[0].(*view.Element)
	if len(wrappedChild.Children) == 0 {
		t.Fatal("wrapped child has no text")
	}
	wrappedText := wrappedChild.Children[0].(*view.Text)
	if wrappedText.Text != "Edit Mode" {
		t.Errorf("wrapped child: expected 'Edit Mode', got '%s' (STALE PROPS BUG)", wrappedText.Text)
	}
}

func TestWrapperWithUnchangedPropsPassesFreshChildProps(t *testing.T) {
	type ChildProps struct {
		Value int
	}

	childRenderCount := 0
	var childReceivedValues []int

	child := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		childRenderCount++
		p := props.(ChildProps)
		childReceivedValues = append(childReceivedValues, p.Value)
		return work.BuildElement("span", work.NewTextf("Value: %d", p.Value))
	}

	type WrapperProps struct {
		Title string
	}

	wrapperRenderCount := 0

	wrapper := func(ctx *Ctx, props any, children []work.Item) work.Node {
		wrapperRenderCount++
		p := props.(WrapperProps)
		return work.BuildElement("div",
			work.BuildElement("h1", work.NewText(p.Title)),
			work.NewFragment(children...),
		)
	}

	currentValue := 1

	page := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("main",
			work.PropsComponent(wrapper, WrapperProps{Title: "Static Title"},
				work.PropsComponent(child, ChildProps{Value: currentValue}),
			),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        page,
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
	if len(childReceivedValues) != 1 || childReceivedValues[0] != 1 {
		t.Errorf("expected child to receive Value=1, got %v", childReceivedValues)
	}

	childRenderCount = 0
	childReceivedValues = nil
	wrapperRenderCount = 0

	currentValue = 42
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	if childRenderCount == 0 {
		t.Error("child should have re-rendered when its props changed")
	}

	foundNewValue := false
	for _, v := range childReceivedValues {
		if v == 42 {
			foundNewValue = true
			break
		}
	}
	if !foundNewValue {
		t.Errorf("child never received updated Value=42, got values: %v (STALE PROPS BUG)", childReceivedValues)
	}
}

func TestDeeplyNestedComponentReceivesUpdatedProps(t *testing.T) {
	deepChildRenderCount := 0
	var deepChildReceivedProps []string

	deepChild := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		deepChildRenderCount++
		msg := props.(string)
		deepChildReceivedProps = append(deepChildReceivedProps, msg)
		return work.BuildElement("span", work.NewText(msg))
	}

	middleWrapper := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		return work.BuildElement("section", children...)
	}

	outerWrapper := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		return work.BuildElement("article", children...)
	}

	currentMessage := "Initial"

	page := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("main",
			work.Component(outerWrapper,
				work.Component(middleWrapper,
					work.PropsComponent(deepChild, currentMessage),
				),
			),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        page,
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

	if deepChildRenderCount != 1 {
		t.Errorf("expected deep child to render once, got %d", deepChildRenderCount)
	}
	if len(deepChildReceivedProps) != 1 || deepChildReceivedProps[0] != "Initial" {
		t.Errorf("expected deep child to receive 'Initial', got %v", deepChildReceivedProps)
	}

	deepChildRenderCount = 0
	deepChildReceivedProps = nil

	currentMessage = "Updated"
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	foundUpdated := false
	for _, msg := range deepChildReceivedProps {
		if msg == "Updated" {
			foundUpdated = true
			break
		}
	}
	if !foundUpdated {
		t.Errorf("deep child never received 'Updated', got: %v (STALE PROPS BUG - props stuck at wrapper level)", deepChildReceivedProps)
	}

	mainElem := sess.View.(*view.Element)
	articleElem := mainElem.Children[0].(*view.Element)
	sectionElem := articleElem.Children[0].(*view.Element)
	spanElem := sectionElem.Children[0].(*view.Element)
	textNode := spanElem.Children[0].(*view.Text)

	if textNode.Text != "Updated" {
		t.Errorf("rendered text should be 'Updated', got '%s' (STALE PROPS BUG)", textNode.Text)
	}
}

func TestMultipleChildrenWithDifferentPropsUpdates(t *testing.T) {
	type ItemProps struct {
		ID    int
		Label string
	}

	itemRenderCounts := make(map[int]int)
	itemReceivedLabels := make(map[int][]string)

	itemComponent := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		p := props.(ItemProps)
		itemRenderCounts[p.ID]++
		itemReceivedLabels[p.ID] = append(itemReceivedLabels[p.ID], p.Label)
		return work.BuildElement("li", work.NewTextf("%d: %s", p.ID, p.Label))
	}

	listWrapper := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		return work.BuildElement("ul", children...)
	}

	items := []ItemProps{
		{ID: 1, Label: "First"},
		{ID: 2, Label: "Second"},
		{ID: 3, Label: "Third"},
	}

	page := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		var children []work.Item
		for _, item := range items {
			children = append(children, work.PropsComponent(itemComponent, item))
		}
		return work.BuildElement("main",
			work.Component(listWrapper, children...),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        page,
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

	for id := 1; id <= 3; id++ {
		if itemRenderCounts[id] != 1 {
			t.Errorf("item %d: expected 1 render, got %d", id, itemRenderCounts[id])
		}
	}

	itemRenderCounts = make(map[int]int)
	itemReceivedLabels = make(map[int][]string)

	items = []ItemProps{
		{ID: 1, Label: "First Updated"},
		{ID: 2, Label: "Second Updated"},
		{ID: 3, Label: "Third Updated"},
	}

	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	for id := 1; id <= 3; id++ {
		labels := itemReceivedLabels[id]
		foundUpdated := false
		for _, label := range labels {
			if label == items[id-1].Label {
				foundUpdated = true
				break
			}
		}
		if !foundUpdated {
			t.Errorf("item %d: never received updated label '%s', got: %v (STALE PROPS BUG)",
				id, items[id-1].Label, labels)
		}
	}
}

func TestKeyChangeTriggersRerender(t *testing.T) {
	childRenderCount := 0

	child := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		childRenderCount++
		msg := props.(string)
		return work.BuildElement("span", work.NewText(msg))
	}

	wrapper := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		return work.BuildElement("div", children...)
	}

	currentKey := "key-1"

	page := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("main",
			work.Component(wrapper,
				work.PropsComponent(child, "Hello").WithKey(currentKey),
			),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        page,
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

	childRenderCount = 0

	currentKey = "key-2"
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	if childRenderCount == 0 {
		t.Error("child should have re-rendered when key changed")
	}
}

func TestElementAttrChangeTriggersRerender(t *testing.T) {
	wrapperRenderCount := 0

	wrapper := func(ctx *Ctx, _ any, children []work.Item) work.Node {
		wrapperRenderCount++
		return work.BuildElement("div", children...)
	}

	currentClass := "class-a"

	page := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		elem := work.BuildElement("span", work.NewText("Hello"))
		elem.Attrs = map[string][]string{"class": {currentClass}}
		return work.BuildElement("main",
			work.Component(wrapper, elem),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        page,
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

	initialWrapperCount := wrapperRenderCount

	currentClass = "class-b"
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	if wrapperRenderCount <= initialWrapperCount {
		t.Error("wrapper should have re-rendered when child element attrs changed")
	}
}

func TestDirectChildPropsUpdateCorrectly(t *testing.T) {
	childRenderCount := 0
	var receivedValues []int

	child := func(ctx *Ctx, props any, _ []work.Item) work.Node {
		childRenderCount++
		value := props.(int)
		receivedValues = append(receivedValues, value)
		return work.BuildElement("span", work.NewTextf("%d", value))
	}

	currentValue := 10

	page := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		return work.BuildElement("div",
			work.PropsComponent(child, currentValue),
		)
	}

	root := &Instance{
		ID:        "root",
		Fn:        page,
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

	if childRenderCount != 1 || receivedValues[0] != 10 {
		t.Errorf("initial render: expected value 10, got %v", receivedValues)
	}

	childRenderCount = 0
	receivedValues = nil

	currentValue = 20
	sess.MarkDirty(root)
	err = sess.Flush()
	if err != nil {
		t.Fatalf("Flush 2 failed: %v", err)
	}

	if childRenderCount != 1 {
		t.Errorf("expected child to re-render once, got %d renders", childRenderCount)
	}

	if len(receivedValues) == 0 || receivedValues[0] != 20 {
		t.Errorf("expected child to receive 20, got %v", receivedValues)
	}

	divElem := sess.View.(*view.Element)
	spanElem := divElem.Children[0].(*view.Element)
	textNode := spanElem.Children[0].(*view.Text)
	if textNode.Text != "20" {
		t.Errorf("expected '20', got '%s'", textNode.Text)
	}
}
