package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestUseSlots_BasicSlotExtraction(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-1",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		work.SlotMarker("header",
			&work.Element{Tag: "h1"},
			&work.Element{Tag: "p"},
		),
		&work.Element{Tag: "div"},
		work.SlotMarker("footer",
			&work.Element{Tag: "button"},
		),
	}

	slots := UseSlots(ctx, children)

	if !slots.Has("header") {
		t.Error("expected header slot to exist")
	}

	if !slots.Has("footer") {
		t.Error("expected footer slot to exist")
	}

	if !slots.Has("default") {
		t.Error("expected default slot to exist")
	}

	headerNode := slots.Render("header")
	if frag, ok := headerNode.(*work.Fragment); ok {
		if len(frag.Children) != 2 {
			t.Errorf("expected 2 children in header slot, got %d", len(frag.Children))
		}
	} else {
		t.Error("expected Fragment for header slot")
	}

	defaultNode := slots.Render("default")
	if elem, ok := defaultNode.(*work.Element); ok {
		if elem.Tag != "div" {
			t.Errorf("expected div in default slot, got %s", elem.Tag)
		}
	} else {
		t.Error("expected Element for default slot")
	}
}

func TestUseSlots_EmptySlots(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-2",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{}

	slots := UseSlots(ctx, children)

	if slots.Has("header") {
		t.Error("expected no header slot")
	}

	rendered := slots.Render("header")
	if frag, ok := rendered.(*work.Fragment); ok {
		if len(frag.Children) != 0 {
			t.Error("expected empty fragment for non-existent slot")
		}
	} else {
		t.Error("expected Fragment for non-existent slot")
	}
}

func TestUseSlots_MultipleNodesPerSlot(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-3",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		work.SlotMarker("header",
			&work.Element{Tag: "h1"},
		),
		work.SlotMarker("header",
			&work.Element{Tag: "h2"},
		),
	}

	slots := UseSlots(ctx, children)

	headerNode := slots.Render("header")
	if frag, ok := headerNode.(*work.Fragment); ok {
		if len(frag.Children) != 2 {
			t.Errorf("expected 2 children from appended slots, got %d", len(frag.Children))
		}
	} else {
		t.Error("expected Fragment for appended header slots")
	}
}

func TestUseSlots_SlotOrder(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-4",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
		&work.Element{Tag: "div"},
		work.SlotMarker("footer", &work.Element{Tag: "footer"}),
	}

	slots := UseSlots(ctx, children)

	names := slots.Names()
	expected := []string{"header", "footer", "default"}

	if len(names) != len(expected) {
		t.Errorf("expected %d slot names, got %d", len(expected), len(names))
	}

	for i, name := range expected {
		if i >= len(names) || names[i] != name {
			t.Errorf("expected slot %d to be %s, got %s", i, name, names[i])
		}
	}
}

func TestUseScopedSlots_BasicExtraction(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-5",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	type TestData struct {
		Name string
	}

	children := []work.Item{
		work.ScopedSlotMarker("actions", func(data TestData) work.Node {
			return &work.Element{
				Tag: "button",
				Children: []work.Node{
					&work.Text{Value: data.Name},
				},
			}
		}),
	}

	slots := UseScopedSlots[TestData](ctx, children)

	if !slots.Has("actions") {
		t.Error("expected actions scoped slot to exist")
	}

	data := TestData{Name: "Click Me"}
	rendered := slots.Render("actions", data)

	if elem, ok := rendered.(*work.Element); ok {
		if elem.Tag != "button" {
			t.Errorf("expected button tag, got %s", elem.Tag)
		}
		if len(elem.Children) != 1 {
			t.Errorf("expected 1 child, got %d", len(elem.Children))
		}
		if txt, ok := elem.Children[0].(*work.Text); ok {
			if txt.Value != "Click Me" {
				t.Errorf("expected 'Click Me', got %s", txt.Value)
			}
		} else {
			t.Error("expected Text node as child")
		}
	} else {
		t.Error("expected Element from scoped slot")
	}
}

func TestUseScopedSlots_MultipleFunctions(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-6",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	type TestData struct {
		Value int
	}

	children := []work.Item{
		work.ScopedSlotMarker("actions", func(data TestData) work.Node {
			return &work.Element{Tag: "button"}
		}),
		work.ScopedSlotMarker("actions", func(data TestData) work.Node {
			return &work.Element{Tag: "a"}
		}),
	}

	slots := UseScopedSlots[TestData](ctx, children)

	data := TestData{Value: 42}
	rendered := slots.Render("actions", data)

	if frag, ok := rendered.(*work.Fragment); ok {
		if len(frag.Children) != 2 {
			t.Errorf("expected 2 children from multiple scoped slots, got %d", len(frag.Children))
		}
	} else {
		t.Error("expected Fragment for multiple scoped slot functions")
	}
}

func TestFingerprintChildren(t *testing.T) {
	children1 := []work.Node{
		&work.Element{Tag: "div"},
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
	}

	children2 := []work.Node{
		&work.Element{Tag: "div"},
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
	}

	fp1 := fingerprintChildren(children1)
	fp2 := fingerprintChildren(children2)

	if fp1 != fp2 {
		t.Error("expected same fingerprint for structurally identical children")
	}

	children3 := []work.Node{
		&work.Element{Tag: "div"},
		work.SlotMarker("footer", &work.Element{Tag: "h1"}),
	}

	fp3 := fingerprintChildren(children3)

	if fp1 == fp3 {
		t.Error("expected different fingerprint for different slot names")
	}
}

func TestUseSlots_ExtractsCorrectly(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-7",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
		&work.Element{Tag: "div"},
	}

	slots := UseSlots(ctx, children)

	if !slots.Has("header") {
		t.Error("expected header slot")
	}

	if !slots.Has("default") {
		t.Error("expected default slot")
	}
}

func TestUseSlots_NilChildren(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-8",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		nil,
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
		nil,
	}

	slots := UseSlots(ctx, children)

	if !slots.Has("header") {
		t.Error("expected header slot despite nil children")
	}

	if slots.Has("default") {
		t.Error("expected no default slot when only nil children are unmarked")
	}
}

func TestUseSlots_FallbackSingle(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-9",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{}
	slots := UseSlots(ctx, children)

	fallbackNode := &work.Element{Tag: "h1"}
	rendered := slots.Render("header", fallbackNode)

	if elem, ok := rendered.(*work.Element); ok {
		if elem.Tag != "h1" {
			t.Errorf("expected h1 fallback, got %s", elem.Tag)
		}
	} else {
		t.Error("expected Element from single fallback")
	}
}

func TestUseSlots_FallbackMultiple(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-10",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{}
	slots := UseSlots(ctx, children)

	fallback1 := &work.Element{Tag: "h1"}
	fallback2 := &work.Element{Tag: "p"}
	rendered := slots.Render("header", fallback1, fallback2)

	if frag, ok := rendered.(*work.Fragment); ok {
		if len(frag.Children) != 2 {
			t.Errorf("expected 2 fallback children, got %d", len(frag.Children))
		}
	} else {
		t.Error("expected Fragment from multiple fallbacks")
	}
}

func TestUseSlots_FallbackIgnoredWhenSlotProvided(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-11",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		work.SlotMarker("header", &work.Element{Tag: "h2"}),
	}
	slots := UseSlots(ctx, children)

	fallbackNode := &work.Element{Tag: "h1"}
	rendered := slots.Render("header", fallbackNode)

	if elem, ok := rendered.(*work.Element); ok {
		if elem.Tag != "h2" {
			t.Error("expected provided slot content (h2), not fallback (h1)")
		}
	} else {
		t.Error("expected Element from provided slot")
	}
}

func TestUseScopedSlots_Fallback(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-12",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	type TestData struct {
		Value int
	}

	children := []work.Item{}
	slots := UseScopedSlots[TestData](ctx, children)

	fallbackNode := &work.Element{Tag: "span"}
	rendered := slots.Render("actions", TestData{Value: 42}, fallbackNode)

	if elem, ok := rendered.(*work.Element); ok {
		if elem.Tag != "span" {
			t.Errorf("expected span fallback, got %s", elem.Tag)
		}
	} else {
		t.Error("expected Element from scoped slot fallback")
	}
}

func TestSlotContext_ProvideAndRender(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{
		work.SlotMarker("default", &work.Element{Tag: "h1"}),
		work.SlotMarker("sidebar", &work.Element{Tag: "nav"}),
	}

	slotCtx.Provide(providerCtx, children)

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	consumerResult := slotCtx.Render(consumerCtx, "default")

	if consumerResult == nil {
		t.Fatal("expected consumer to receive slot content")
	}

	if elem, ok := consumerResult.(*work.Element); ok {
		if elem.Tag != "h1" {
			t.Errorf("expected h1 element, got %s", elem.Tag)
		}
	} else {
		t.Fatalf("expected Element, got %T", consumerResult)
	}
}

func TestSlotContext_ProvideWithoutDefault(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{
		&work.Element{Tag: "div"},
		work.SlotMarker("named", &work.Element{Tag: "aside"}),
	}

	result := slotCtx.ProvideWithoutDefault(providerCtx, children)

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	if slotCtx.Has(consumerCtx, "default") {
		t.Error("expected default slot to remain empty")
	}

	fallback := &work.Element{Tag: "fallback"}
	defaultResult := slotCtx.Render(consumerCtx, "default", fallback)

	if elem, ok := defaultResult.(*work.Element); ok {
		if elem.Tag != "fallback" {
			t.Errorf("expected fallback element, got %s", elem.Tag)
		}
	} else {
		t.Fatalf("expected fallback element for empty default, got %T", defaultResult)
	}

	namedResult := slotCtx.Render(consumerCtx, "named")

	if elem, ok := namedResult.(*work.Element); ok {
		if elem.Tag != "aside" {
			t.Errorf("expected named slot content, got %s", elem.Tag)
		}
	} else {
		t.Fatalf("expected Element for named slot, got %T", namedResult)
	}

	if frag, ok := result.(*work.Fragment); ok {
		if len(frag.Children) != 1 {
			t.Fatalf("expected non-slot children to render, got %d children", len(frag.Children))
		}
		if elem, ok := frag.Children[0].(*work.Element); !ok || elem.Tag != "div" {
			t.Errorf("expected rendered child div, got %T", frag.Children[0])
		}
	} else {
		t.Fatalf("expected Fragment result from ProvideWithoutDefault, got %T", result)
	}
}

func TestSlotContext_Has(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
	}

	slotCtx.Provide(providerCtx, children)

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	if !slotCtx.Has(consumerCtx, "header") {
		t.Error("expected header slot to exist")
	}

	if slotCtx.Has(consumerCtx, "footer") {
		t.Error("expected footer slot to not exist")
	}
}

func TestSlotContext_Fallback(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{}

	slotCtx.Provide(providerCtx, children)

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	fallback := &work.Element{Tag: "div"}
	result := slotCtx.Render(consumerCtx, "default", fallback)

	if elem, ok := result.(*work.Element); ok {
		if elem.Tag != "div" {
			t.Errorf("expected fallback div, got %s", elem.Tag)
		}
	} else {
		t.Fatalf("expected Element fallback, got %T", result)
	}
}

func TestSlotContext_SetSlot(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{}

	slotCtx.Provide(providerCtx, children)

	slotCtx.SetSlot(providerCtx, "dynamic", &work.Element{Tag: "span"})

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	result := slotCtx.Render(consumerCtx, "dynamic")

	if elem, ok := result.(*work.Element); ok {
		if elem.Tag != "span" {
			t.Errorf("expected dynamically set span, got %s", elem.Tag)
		}
	} else {
		t.Fatalf("expected Element from SetSlot, got %T", result)
	}
}

func TestSlotContext_NoProvider(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "orphan",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	fallback := &work.Element{Tag: "fallback"}
	result := slotCtx.Render(ctx, "default", fallback)

	if elem, ok := result.(*work.Element); ok {
		if elem.Tag != "fallback" {
			t.Errorf("expected fallback element, got %s", elem.Tag)
		}
	} else {
		t.Fatalf("expected Element fallback when no provider, got %T", result)
	}

	if slotCtx.Has(ctx, "default") {
		t.Error("expected Has to return false when no provider")
	}
}

func TestSlotRenderer_RequireSlots_AllPresent(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-require-1",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
		work.SlotMarker("footer", &work.Element{Tag: "footer"}),
	}

	slots := UseSlots(ctx, children)

	err := slots.RequireSlots("header", "footer")
	if err != nil {
		t.Errorf("expected no error when all slots present, got %v", err)
	}
}

func TestSlotRenderer_RequireSlots_Missing(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-require-2",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Item{
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
	}

	slots := UseSlots(ctx, children)

	err := slots.RequireSlots("header", "footer", "sidebar")
	if err == nil {
		t.Error("expected error when slots are missing")
	}
}

func TestScopedSlotRenderer_Names(t *testing.T) {
	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	inst := &Instance{
		ID:        "test-scoped-names",
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{
		instance:  inst,
		session:   session,
		hookIndex: 0,
	}

	type TestData struct {
		Value int
	}

	children := []work.Item{
		work.ScopedSlotMarker("actions", func(data TestData) work.Node {
			return &work.Element{Tag: "button"}
		}),
		work.ScopedSlotMarker("header", func(data TestData) work.Node {
			return &work.Element{Tag: "h1"}
		}),
		work.ScopedSlotMarker("footer", func(data TestData) work.Node {
			return &work.Element{Tag: "footer"}
		}),
	}

	slots := UseScopedSlots[TestData](ctx, children)

	names := slots.Names()
	expected := []string{"actions", "header", "footer"}

	if len(names) != len(expected) {
		t.Errorf("expected %d slot names, got %d", len(expected), len(names))
	}

	for i, name := range expected {
		if i >= len(names) || names[i] != name {
			t.Errorf("expected slot %d to be %s, got %s", i, name, names[i])
		}
	}
}

func TestSlotContext_AppendSlot(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{}
	slotCtx.Provide(providerCtx, children)

	slotCtx.SetSlot(providerCtx, "items", &work.Element{Tag: "li"})
	slotCtx.AppendSlot(providerCtx, "items", &work.Element{Tag: "li"})
	slotCtx.AppendSlot(providerCtx, "items", &work.Element{Tag: "li"})

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	result := slotCtx.Render(consumerCtx, "items")

	if frag, ok := result.(*work.Fragment); ok {
		if len(frag.Children) != 3 {
			t.Errorf("expected 3 children after append, got %d", len(frag.Children))
		}
	} else {
		t.Fatalf("expected Fragment from appended slot, got %T", result)
	}
}

func TestSlotContext_AppendSlot_NewSlot(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{}
	slotCtx.Provide(providerCtx, children)

	slotCtx.AppendSlot(providerCtx, "newslot", &work.Element{Tag: "span"})

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	if !slotCtx.Has(consumerCtx, "newslot") {
		t.Error("expected newslot to exist after AppendSlot")
	}

	result := slotCtx.Render(consumerCtx, "newslot")

	if elem, ok := result.(*work.Element); ok {
		if elem.Tag != "span" {
			t.Errorf("expected span, got %s", elem.Tag)
		}
	} else {
		t.Fatalf("expected Element from new appended slot, got %T", result)
	}
}

func TestDefaultSlotName_Constant(t *testing.T) {
	if DefaultSlotName != "default" {
		t.Errorf("expected DefaultSlotName to be 'default', got %s", DefaultSlotName)
	}
}

func TestFilterRenderedChildren_NoSlots(t *testing.T) {
	children := []work.Node{
		&work.Element{Tag: "div"},
		&work.Element{Tag: "span"},
	}

	result := filterRenderedChildren(children)

	if len(result) != len(children) {
		t.Errorf("expected same length, got %d vs %d", len(result), len(children))
	}

	for i := range children {
		if result[i] != children[i] {
			t.Errorf("expected same slice returned when no slots")
			break
		}
	}
}

func TestFilterRenderedChildren_WithSlots(t *testing.T) {
	children := []work.Node{
		&work.Element{Tag: "div"},
		work.SlotMarker("header", &work.Element{Tag: "h1"}),
		&work.Element{Tag: "span"},
	}

	result := filterRenderedChildren(children)

	if len(result) != 2 {
		t.Errorf("expected 2 non-slot children, got %d", len(result))
	}
}

func TestFingerprintNode_TextContentHash(t *testing.T) {
	text1 := &work.Text{Value: "hello"}
	text2 := &work.Text{Value: "world"}
	text3 := &work.Text{Value: "hello"}

	fp1 := fingerprintNode(text1)
	fp2 := fingerprintNode(text2)
	fp3 := fingerprintNode(text3)

	if fp1 == fp2 {
		t.Error("expected different fingerprints for different text content")
	}

	if fp1 != fp3 {
		t.Error("expected same fingerprint for same text content")
	}
}

func TestSetSlot_ReplacesExisting(t *testing.T) {
	slotCtx := CreateSlotContext()

	session := &Session{
		Components: make(map[string]*Instance),
		Handlers:   make(map[string]work.Handler),
	}

	parent := &Instance{
		ID:        "provider",
		HookFrame: []HookSlot{},
		Providers: make(map[any]any),
	}

	child := &Instance{
		ID:        "consumer",
		HookFrame: []HookSlot{},
		Parent:    parent,
	}

	parent.Children = []*Instance{child}

	providerCtx := &Ctx{
		instance:  parent,
		session:   session,
		hookIndex: 0,
	}

	children := []work.Node{}
	slotCtx.Provide(providerCtx, children)

	slotCtx.SetSlot(providerCtx, "test", &work.Element{Tag: "div"})
	slotCtx.SetSlot(providerCtx, "test", &work.Element{Tag: "span"})

	consumerCtx := &Ctx{
		instance:  child,
		session:   session,
		hookIndex: 0,
	}

	result := slotCtx.Render(consumerCtx, "test")

	if elem, ok := result.(*work.Element); ok {
		if elem.Tag != "span" {
			t.Errorf("expected SetSlot to replace, got %s instead of span", elem.Tag)
		}
	} else {
		t.Fatalf("expected Element, got %T", result)
	}

	names := slotCtx.Names(consumerCtx)
	count := 0
	for _, n := range names {
		if n == "test" {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected slot name to appear once in order, got %d", count)
	}
}
