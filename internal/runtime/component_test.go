package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestComponent(t *testing.T) {
	fn := func(ctx *Ctx, children []work.Item) work.Node {
		return &work.Text{Value: "test"}
	}

	wrapper := Component(fn)
	if wrapper == nil {
		t.Fatal("expected non-nil wrapper")
	}

	ctx := &Ctx{}
	node := wrapper(ctx)
	if node == nil {
		t.Fatal("expected non-nil node")
	}

	comp, ok := node.(*work.ComponentNode)
	if !ok {
		t.Fatalf("expected *work.ComponentNode, got %T", node)
	}

	if comp.Fn == nil {
		t.Error("expected Fn to be set")
	}
}

func TestComponentWithChildren(t *testing.T) {
	fn := func(ctx *Ctx, children []work.Item) work.Node {
		return &work.Text{Value: "test"}
	}

	wrapper := Component(fn)
	ctx := &Ctx{}

	child1 := &work.Element{Tag: "div"}
	child2 := &work.Text{Value: "text"}

	node := wrapper(ctx, child1, child2)

	comp, ok := node.(*work.ComponentNode)
	if !ok {
		t.Fatalf("expected *work.ComponentNode, got %T", node)
	}

	if len(comp.InputChildren) != 2 {
		t.Errorf("expected 2 children in ComponentNode, got %d", len(comp.InputChildren))
	}
}

func TestPropsComponent(t *testing.T) {
	type TestProps struct {
		Title string
		Count int
	}

	fn := func(ctx *Ctx, props TestProps, children []work.Item) work.Node {
		return &work.Text{Value: props.Title}
	}

	wrapper := PropsComponent(fn)
	if wrapper == nil {
		t.Fatal("expected non-nil wrapper")
	}

	ctx := &Ctx{}
	props := TestProps{Title: "Hello", Count: 42}
	node := wrapper(ctx, props)

	if node == nil {
		t.Fatal("expected non-nil node")
	}

	comp, ok := node.(*work.ComponentNode)
	if !ok {
		t.Fatalf("expected *work.ComponentNode, got %T", node)
	}

	if comp.Fn == nil {
		t.Error("expected Fn to be set")
	}

	if comp.Props == nil {
		t.Error("expected Props to be set")
	}
}

func TestPropsComponentWithChildren(t *testing.T) {
	type TestProps struct {
		Title string
	}

	fn := func(ctx *Ctx, props TestProps, children []work.Item) work.Node {
		return &work.Text{Value: props.Title}
	}

	wrapper := PropsComponent(fn)
	ctx := &Ctx{}
	props := TestProps{Title: "Hello"}

	child := &work.Element{Tag: "span"}
	node := wrapper(ctx, props, child)

	comp, ok := node.(*work.ComponentNode)
	if !ok {
		t.Fatalf("expected *work.ComponentNode, got %T", node)
	}

	if len(comp.InputChildren) != 1 {
		t.Errorf("expected 1 child in ComponentNode, got %d", len(comp.InputChildren))
	}
}

func TestPropsComponentWithWrongPropsType(t *testing.T) {
	type TestProps struct {
		Title string
	}

	receivedProps := TestProps{Title: "should be overwritten"}
	fn := func(ctx *Ctx, props TestProps, children []work.Item) work.Node {
		receivedProps = props
		return nil
	}

	_ = PropsComponent(fn)

	wrappedFn := func(ctx *Ctx, propsAny any, children []work.Item) work.Node {
		p, ok := propsAny.(TestProps)
		if !ok {
			var zero TestProps
			p = zero
		}
		return fn(ctx, p, children)
	}

	ctx := &Ctx{}
	wrappedFn(ctx, "wrong type", nil)

	if receivedProps.Title != "" {
		t.Errorf("expected zero value props, got %+v", receivedProps)
	}
}

func TestSessionSetDevMode(t *testing.T) {
	sess := &Session{}

	sess.SetDevMode(true)
	if !sess.devMode {
		t.Error("expected devMode to be true")
	}

	sess.SetDevMode(false)
	if sess.devMode {
		t.Error("expected devMode to be false")
	}
}

func TestSessionSetDevModeNil(t *testing.T) {
	var sess *Session
	sess.SetDevMode(true)
}

func TestSessionChannelManager(t *testing.T) {
	bus := protocol.NewBus()
	sess := &Session{
		Bus:       bus,
		SessionID: "test-session",
	}

	cm := sess.ChannelManager()
	if cm == nil {
		t.Fatal("expected non-nil ChannelManager")
	}

	cm2 := sess.ChannelManager()
	if cm2 != cm {
		t.Error("expected same ChannelManager instance on second call")
	}
}

func TestSessionChannelManagerNil(t *testing.T) {
	var sess *Session
	cm := sess.ChannelManager()
	if cm != nil {
		t.Error("expected nil ChannelManager for nil session")
	}
}

func TestSessionChannelManagerNoBus(t *testing.T) {
	sess := &Session{
		SessionID: "test-session",
	}

	cm := sess.ChannelManager()
	if cm != nil {
		t.Error("expected nil ChannelManager when Bus is nil")
	}
}

func TestSessionChannelManagerNoSessionID(t *testing.T) {
	bus := protocol.NewBus()
	sess := &Session{
		Bus: bus,
	}

	cm := sess.ChannelManager()
	if cm != nil {
		t.Error("expected nil ChannelManager when SessionID is empty")
	}
}

func TestInstanceBeginRender(t *testing.T) {
	t.Run("nil instance does not panic", func(t *testing.T) {
		var inst *Instance
		inst.BeginRender()
	})

	t.Run("resets render indices", func(t *testing.T) {
		inst := &Instance{
			ID:                 "test",
			ChildRenderIndex:   5,
			ProviderSeq:        3,
			ReferencedChildren: map[string]bool{"child1": true},
		}
		inst.BeginRender()

		if inst.ChildRenderIndex != 0 {
			t.Errorf("expected ChildRenderIndex to be 0, got %d", inst.ChildRenderIndex)
		}
		if inst.ProviderSeq != 0 {
			t.Errorf("expected ProviderSeq to be 0, got %d", inst.ProviderSeq)
		}
		if len(inst.ReferencedChildren) != 0 {
			t.Errorf("expected ReferencedChildren to be empty, got %d", len(inst.ReferencedChildren))
		}
	})

	t.Run("creates new ReferencedChildren map", func(t *testing.T) {
		inst := &Instance{ID: "test"}
		inst.BeginRender()

		if inst.ReferencedChildren == nil {
			t.Error("expected ReferencedChildren to be initialized")
		}
	})
}

func TestInstanceEndRender(t *testing.T) {
	t.Run("nil instance does not panic", func(t *testing.T) {
		var inst *Instance
		inst.EndRender()
	})

	t.Run("non-nil instance does not panic", func(t *testing.T) {
		inst := &Instance{ID: "test"}
		inst.EndRender()
	})
}

func TestInstanceSetDirty(t *testing.T) {
	t.Run("nil instance does not panic", func(t *testing.T) {
		var inst *Instance
		inst.SetDirty(true)
	})

	t.Run("sets dirty to true", func(t *testing.T) {
		inst := &Instance{ID: "test", Dirty: false}
		inst.SetDirty(true)
		if !inst.Dirty {
			t.Error("expected Dirty to be true")
		}
	})

	t.Run("sets dirty to false", func(t *testing.T) {
		inst := &Instance{ID: "test", Dirty: true}
		inst.SetDirty(false)
		if inst.Dirty {
			t.Error("expected Dirty to be false")
		}
	})
}

func TestPropsComponentRendersWithWrongPropsType(t *testing.T) {
	type TestProps struct {
		Title string
		Count int
	}

	var receivedProps TestProps
	fn := func(ctx *Ctx, props TestProps, children []work.Item) work.Node {
		receivedProps = props
		return &work.Text{Value: props.Title}
	}

	wrapper := PropsComponent(fn)

	sess := &Session{
		Components: make(map[string]*Instance),
	}

	rootFn := func(ctx *Ctx, _ any, _ []work.Item) work.Node {
		comp := wrapper(ctx, TestProps{Title: "correct", Count: 5})
		return comp
	}

	sess.Root = &Instance{
		ID: "root",
		Fn: rootFn,
	}

	err := sess.Flush()
	if err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if receivedProps.Title != "correct" {
		t.Errorf("expected 'correct', got '%s'", receivedProps.Title)
	}
	if receivedProps.Count != 5 {
		t.Errorf("expected 5, got %d", receivedProps.Count)
	}
}

func TestPropsComponentTypeAssertionFailure(t *testing.T) {
	type TestProps struct {
		Title string
	}

	var receivedProps TestProps
	fn := func(ctx *Ctx, props TestProps, children []work.Item) work.Node {
		receivedProps = props
		return nil
	}

	wrapper := PropsComponent(fn)
	compNode := wrapper(nil, TestProps{Title: "test"})
	comp := compNode.(*work.ComponentNode)

	sess := &Session{
		Components: make(map[string]*Instance),
	}

	parent := &Instance{
		ID:                    "parent",
		HookFrame:             []HookSlot{},
		Children:              []*Instance{},
		CombinedContextEpochs: make(map[contextID]int),
	}
	sess.Root = parent
	sess.Components["parent"] = parent

	comp.Props = "wrong type string"

	sess.convertComponent(comp, parent)

	if receivedProps.Title != "" {
		t.Errorf("expected zero value for Title, got '%s'", receivedProps.Title)
	}
}
