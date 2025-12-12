package work

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/metadata"
)

type mockAttachment struct {
	id string
}

func (m mockAttachment) AttachTo(el *Element) {
	el.RefID = m.id
}

func TestNodeApplyTo(t *testing.T) {
	parent := &Element{Tag: "div"}

	child := &Element{Tag: "span"}
	child.ApplyTo(parent)

	if len(parent.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(parent.Children))
	}

	text := &Text{Value: "hello"}
	text.ApplyTo(parent)

	if len(parent.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(parent.Children))
	}

	comp := &ComponentNode{Fn: nil, Props: nil}
	comp.ApplyTo(parent)

	if len(parent.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(parent.Children))
	}
}

func TestAttrItem(t *testing.T) {
	el := &Element{Tag: "div"}

	attr := attrItem{name: "id", value: "test"}
	attr.ApplyTo(el)

	if el.Attrs["id"][0] != "test" {
		t.Errorf("expected id='test', got %v", el.Attrs["id"])
	}
}

func TestClassItem(t *testing.T) {
	el := &Element{Tag: "div"}

	class1 := Class("foo")
	class2 := Class("bar")

	class1.ApplyTo(el)
	class2.ApplyTo(el)

	if len(el.Attrs["class"]) != 2 {
		t.Errorf("expected 2 classes, got %d", len(el.Attrs["class"]))
	}
	if el.Attrs["class"][0] != "foo" || el.Attrs["class"][1] != "bar" {
		t.Errorf("unexpected class values: %v", el.Attrs["class"])
	}
}

func TestStyleItem(t *testing.T) {
	el := &Element{Tag: "div"}

	style := styleItem{property: "color", value: "red"}
	style.ApplyTo(el)

	if el.Style["color"] != "red" {
		t.Errorf("expected color=red, got %v", el.Style["color"])
	}
}

func TestKeyItem(t *testing.T) {
	el := &Element{Tag: "div"}

	key := keyItem{value: "unique-key"}
	key.ApplyTo(el)

	if el.Key != "unique-key" {
		t.Errorf("expected key='unique-key', got %s", el.Key)
	}
}

func TestEventItem(t *testing.T) {
	el := &Element{Tag: "button"}

	handler := Handler{
		Fn: func(e Event) Updates {
			return nil
		},
	}

	event := eventItem{event: "click", handler: handler}
	event.ApplyTo(el)

	if el.Handlers["click"].Fn == nil {
		t.Error("expected click handler to be set")
	}
}

func TestAttachItem(t *testing.T) {
	el := &Element{Tag: "div"}

	ref := mockAttachment{id: "ref-123"}
	attach := attachItem{ref: ref}
	attach.ApplyTo(el)

	if el.RefID != "ref-123" {
		t.Errorf("expected refID='ref-123', got %s", el.RefID)
	}
}

func TestMultipleItems(t *testing.T) {
	el := &Element{Tag: "div"}

	ID("test").ApplyTo(el)
	Class("card").ApplyTo(el)
	Style("padding", "16px").ApplyTo(el)
	Key("card-1").ApplyTo(el)

	(&Text{Value: "Hello"}).ApplyTo(el)
	(&Element{Tag: "span"}).ApplyTo(el)

	if el.Attrs["id"][0] != "test" {
		t.Error("id not set")
	}
	if el.Attrs["class"][0] != "card" {
		t.Error("class not set")
	}
	if el.Style["padding"] != "16px" {
		t.Error("style not set")
	}
	if el.Key != "card-1" {
		t.Error("key not set")
	}
	if len(el.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(el.Children))
	}
}

func TestNewComponent(t *testing.T) {
	mockFn := func() {}
	child1 := &Text{Value: "hello"}
	child2 := &Element{Tag: "span"}

	comp := Component(mockFn, child1, child2)

	if comp.Fn == nil {
		t.Error("component function not set")
	}
	if comp.Props != nil {
		t.Error("props should be nil for no-props component")
	}
	if len(comp.InputChildren) != 2 {
		t.Errorf("expected 2 children, got %d", len(comp.InputChildren))
	}
}

func TestNewPropsComponent(t *testing.T) {
	type MyProps struct {
		Title string
		Count int
	}

	mockFn := func() {}
	props := MyProps{Title: "Test", Count: 42}
	child := &Text{Value: "content"}

	comp := PropsComponent(mockFn, props, child)

	if comp.Fn == nil {
		t.Error("component function not set")
	}
	if comp.Props == nil {
		t.Error("props should be set")
	}
	if p, ok := comp.Props.(MyProps); !ok || p.Title != "Test" || p.Count != 42 {
		t.Error("props not set correctly")
	}
	if len(comp.InputChildren) != 1 {
		t.Errorf("expected 1 child, got %d", len(comp.InputChildren))
	}
}

func TestComponentWithKey(t *testing.T) {
	mockFn := func() {}
	comp := Component(mockFn).WithKey("comp-123")

	if comp.Key != "comp-123" {
		t.Errorf("expected key='comp-123', got %s", comp.Key)
	}
}

func TestComponentAsChild(t *testing.T) {
	parent := &Element{Tag: "div"}
	mockFn := func() {}

	comp := Component(mockFn, &Text{Value: "test"})
	comp.ApplyTo(parent)

	if len(parent.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(parent.Children))
	}

	if _, ok := parent.Children[0].(*ComponentNode); !ok {
		t.Error("child should be a ComponentNode")
	}
}

func TestBuildElement(t *testing.T) {
	el := BuildElement("div",
		ID("test"),
		Class("card"),
		Style("color", "red"),
		NewText("Hello"),
	)

	if el.Tag != "div" {
		t.Errorf("expected tag='div', got %s", el.Tag)
	}
	if el.Attrs["id"][0] != "test" {
		t.Error("id not set")
	}
	if el.Attrs["class"][0] != "card" {
		t.Error("class not set")
	}
	if el.Style["color"] != "red" {
		t.Error("style not set")
	}
	if len(el.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(el.Children))
	}
}

func TestNewTextHelper(t *testing.T) {
	text := NewText("hello")
	if text.Value != "hello" {
		t.Errorf("expected 'hello', got %s", text.Value)
	}
}

func TestNewTextfHelper(t *testing.T) {
	text := NewTextf("Count: %d", 42)
	if text.Value != "Count: 42" {
		t.Errorf("expected 'Count: 42', got %s", text.Value)
	}
}

func TestNewCommentHelper(t *testing.T) {
	comment := NewComment("TODO")
	if comment.Value != "TODO" {
		t.Errorf("expected 'TODO', got %s", comment.Value)
	}
}

func TestNewFragmentHelper(t *testing.T) {
	frag := NewFragment(NewText("a"), NewText("b"))
	if len(frag.Children) != 2 {
		t.Errorf("expected 2 children, got %d", len(frag.Children))
	}
}

func TestUnsafeHTML(t *testing.T) {
	el := BuildElement("div",
		UnsafeHTML("<span>raw pkg</span>"),
	)

	if el.UnsafeHTML != "<span>raw pkg</span>" {
		t.Errorf("expected unsafe pkg to be set, got %s", el.UnsafeHTML)
	}
}

func TestEventHandlerProxy_MultipleHandlers(t *testing.T) {
	el := &Element{Tag: "button"}

	var calls []string

	handler1 := Handler{
		Fn: func(e Event) Updates {
			calls = append(calls, "handler1")
			return nil
		},
	}
	handler2 := Handler{
		Fn: func(e Event) Updates {
			calls = append(calls, "handler")
			return nil
		},
	}
	handler3 := Handler{
		Fn: func(e Event) Updates {
			calls = append(calls, "handler3")
			return nil
		},
	}

	eventItem{event: "click", handler: handler1}.ApplyTo(el)
	eventItem{event: "click", handler: handler2}.ApplyTo(el)
	eventItem{event: "click", handler: handler3}.ApplyTo(el)

	if len(el.Handlers) != 1 {
		t.Errorf("expected 1 handler entry, got %d", len(el.Handlers))
	}

	el.Handlers["click"].Fn(Event{Name: "click"})

	if len(calls) != 3 {
		t.Errorf("expected 3 calls, got %d", len(calls))
	}
	if calls[0] != "handler1" || calls[1] != "handler" || calls[2] != "handler3" {
		t.Errorf("handlers called in wrong order: %v", calls)
	}
}

func TestEventHandlerProxy_MergesOptions(t *testing.T) {
	el := &Element{Tag: "button"}

	handler1 := Handler{
		EventOptions: metadata.EventOptions{
			Prevent: true,
			Props:   []string{"target"},
		},
		Fn: func(e Event) Updates { return nil },
	}
	handler2 := Handler{
		EventOptions: metadata.EventOptions{
			Stop:  true,
			Props: []string{"value", "target"},
		},
		Fn: func(e Event) Updates { return nil },
	}

	eventItem{event: "click", handler: handler1}.ApplyTo(el)
	eventItem{event: "click", handler: handler2}.ApplyTo(el)

	opts := el.Handlers["click"].EventOptions

	if !opts.Prevent {
		t.Error("expected Prevent to be true")
	}
	if !opts.Stop {
		t.Error("expected Stop to be true")
	}

	if len(opts.Props) != 2 {
		t.Errorf("expected 2 props, got %d: %v", len(opts.Props), opts.Props)
	}
}

func TestEventHandlerProxy_DebounceThrottle(t *testing.T) {
	el := &Element{Tag: "input"}

	handler1 := Handler{
		EventOptions: metadata.EventOptions{
			Debounce: 500,
			Throttle: 1000,
		},
		Fn: func(e Event) Updates { return nil },
	}
	handler2 := Handler{
		EventOptions: metadata.EventOptions{
			Debounce: 200,
			Throttle: 300,
		},
		Fn: func(e Event) Updates { return nil },
	}

	eventItem{event: "input", handler: handler1}.ApplyTo(el)
	eventItem{event: "input", handler: handler2}.ApplyTo(el)

	opts := el.Handlers["input"].EventOptions

	if opts.Debounce != 200 {
		t.Errorf("expected Debounce=200, got %d", opts.Debounce)
	}
	if opts.Throttle != 300 {
		t.Errorf("expected Throttle=300, got %d", opts.Throttle)
	}
}

type mockHandlerRef struct {
	id       string
	handlers map[string][]Handler
}

func (m *mockHandlerRef) AttachTo(el *Element) {
	if m == nil || el == nil {
		return
	}
	el.RefID = m.id
	for _, event := range m.Events() {
		handler := m.ProxyHandler(event)
		eventItem{event: event, handler: handler}.ApplyTo(el)
	}
}

func (m *mockHandlerRef) Events() []string {
	events := make([]string, 0, len(m.handlers))
	for event, handlers := range m.handlers {
		if len(handlers) > 0 {
			events = append(events, event)
		}
	}
	return events
}

func (m *mockHandlerRef) ProxyHandler(event string) Handler {
	handlers := m.handlers[event]
	if len(handlers) == 0 {
		return Handler{}
	}

	var merged metadata.EventOptions
	for _, h := range handlers {
		merged = MergeEventOptions(merged, h.EventOptions)
	}

	return Handler{
		EventOptions: merged,
		Fn: func(evt Event) Updates {
			var result Updates
			for _, h := range handlers {
				if h.Fn != nil {
					result = h.Fn(evt)
				}
			}
			return result
		},
	}
}

func TestAttach_TransfersRefHandlers(t *testing.T) {
	var calls []string

	ref := &mockHandlerRef{
		id: "ref-123",
		handlers: map[string][]Handler{
			"click": {
				{Fn: func(e Event) Updates {
					calls = append(calls, "ref-click")
					return nil
				}},
			},
			"focus": {
				{Fn: func(e Event) Updates {
					calls = append(calls, "ref-focus")
					return nil
				}},
			},
		},
	}

	el := &Element{Tag: "button"}
	attachItem{ref: ref}.ApplyTo(el)

	if el.RefID != "ref-123" {
		t.Errorf("expected refID='ref-123', got %s", el.RefID)
	}

	if len(el.Handlers) != 2 {
		t.Errorf("expected 2 handlers, got %d", len(el.Handlers))
	}

	if el.Handlers["click"].Fn != nil {
		el.Handlers["click"].Fn(Event{Name: "click"})
	}
	if el.Handlers["focus"].Fn != nil {
		el.Handlers["focus"].Fn(Event{Name: "focus"})
	}

	if len(calls) != 2 {
		t.Errorf("expected 2 calls, got %d", len(calls))
	}
}

func TestAttach_MergesWithExistingHandlers(t *testing.T) {
	var calls []string

	el := &Element{Tag: "button"}
	eventItem{
		event: "click",
		handler: Handler{
			Fn: func(e Event) Updates {
				calls = append(calls, "direct-click")
				return nil
			},
		},
	}.ApplyTo(el)

	ref := &mockHandlerRef{
		id: "ref-123",
		handlers: map[string][]Handler{
			"click": {
				{Fn: func(e Event) Updates {
					calls = append(calls, "ref-click")
					return nil
				}},
			},
		},
	}

	attachItem{ref: ref}.ApplyTo(el)

	if len(el.Handlers) != 1 {
		t.Errorf("expected 1 handler entry, got %d", len(el.Handlers))
	}

	el.Handlers["click"].Fn(Event{Name: "click"})

	if len(calls) != 2 {
		t.Errorf("expected 2 calls, got %d: %v", len(calls), calls)
	}
	if calls[0] != "direct-click" || calls[1] != "ref-click" {
		t.Errorf("handlers called in wrong order: %v", calls)
	}
}

func TestAttach_NoHandlerProvider(t *testing.T) {

	ref := mockAttachment{id: "simple-ref"}
	el := &Element{Tag: "div"}

	attachItem{ref: ref}.ApplyTo(el)

	if el.RefID != "simple-ref" {
		t.Errorf("expected refID='simple-ref', got %s", el.RefID)
	}
	if len(el.Handlers) != 0 {
		t.Errorf("expected 0 handlers, got %d", len(el.Handlers))
	}
}

func TestMergeEventOptions(t *testing.T) {
	a := metadata.EventOptions{
		Prevent:  true,
		Debounce: 500,
		Props:    []string{"target", "value"},
		Listen:   []string{"input"},
	}
	b := metadata.EventOptions{
		Stop:     true,
		Capture:  true,
		Debounce: 200,
		Throttle: 100,
		Props:    []string{"value", "checked"},
		Listen:   []string{"change"},
	}

	merged := MergeEventOptions(a, b)

	if !merged.Prevent {
		t.Error("Prevent should be true")
	}
	if !merged.Stop {
		t.Error("Stop should be true")
	}
	if !merged.Capture {
		t.Error("Capture should be true")
	}

	if merged.Debounce != 200 {
		t.Errorf("expected Debounce=200, got %d", merged.Debounce)
	}
	if merged.Throttle != 100 {
		t.Errorf("expected Throttle=100, got %d", merged.Throttle)
	}

	if len(merged.Props) != 3 {
		t.Errorf("expected 3 props, got %d: %v", len(merged.Props), merged.Props)
	}
	if len(merged.Listen) != 2 {
		t.Errorf("expected 2 listen, got %d: %v", len(merged.Listen), merged.Listen)
	}
}

func TestDeduplicateStrings(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name:     "empty",
			input:    []string{},
			expected: nil,
		},
		{
			name:     "no duplicates",
			input:    []string{"a", "b", "c"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with duplicates",
			input:    []string{"a", "b", "a", "c", "b"},
			expected: []string{"a", "b", "c"},
		},
		{
			name:     "with empty strings",
			input:    []string{"a", "", "b", ""},
			expected: []string{"a", "b"},
		},
		{
			name:     "all empty",
			input:    []string{"", "", ""},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := deduplicateStrings(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("expected length %d, got %d", len(tt.expected), len(result))
			}
			for i, v := range tt.expected {
				if result[i] != v {
					t.Errorf("expected %s at index %d, got %s", v, i, result[i])
				}
			}
		})
	}
}

func TestIf(t *testing.T) {
	node := NewText("hello")

	result := If(true, node)
	if _, ok := result.(*Text); !ok {
		t.Error("expected node when condition is true")
	}

	result = If(false, node)
	if _, ok := result.(noopItem); !ok {
		t.Error("expected noopItem when condition is false")
	}
}

func TestIfFn(t *testing.T) {
	called := false
	fn := func() Item {
		called = true
		return NewText("hello")
	}

	result := IfFn(true, fn)
	if !called {
		t.Error("expected function to be called when condition is true")
	}
	if _, ok := result.(*Text); !ok {
		t.Error("expected Text when condition is true")
	}

	called = false
	result = IfFn(false, fn)
	if called {
		t.Error("expected function not to be called when condition is false")
	}
	if _, ok := result.(noopItem); !ok {
		t.Error("expected noopItem when condition is false")
	}

	result = IfFn(true, nil)
	if _, ok := result.(noopItem); !ok {
		t.Error("expected noopItem when function is nil")
	}
}

func TestTernary(t *testing.T) {
	trueNode := NewText("true")
	falseNode := NewText("false")

	result := Ternary(true, trueNode, falseNode)
	if text, ok := result.(*Text); !ok || text.Value != "true" {
		t.Error("expected trueNode when condition is true")
	}

	result = Ternary(false, trueNode, falseNode)
	if text, ok := result.(*Text); !ok || text.Value != "false" {
		t.Error("expected falseNode when condition is false")
	}

	result = Ternary(true, nil, falseNode)
	if text, ok := result.(*Text); !ok || text.Value != "false" {
		t.Error("expected falseNode when trueNode is nil")
	}

	result = Ternary(false, trueNode, nil)
	if _, ok := result.(noopItem); !ok {
		t.Error("expected noopItem when falseNode is nil")
	}
}

func TestTernaryFn(t *testing.T) {
	trueFn := func() Item { return NewText("true") }
	falseFn := func() Item { return NewText("false") }

	result := TernaryFn(true, trueFn, falseFn)
	if text, ok := result.(*Text); !ok || text.Value != "true" {
		t.Error("expected true result when condition is true")
	}

	result = TernaryFn(false, trueFn, falseFn)
	if text, ok := result.(*Text); !ok || text.Value != "false" {
		t.Error("expected false result when condition is false")
	}

	result = TernaryFn(true, nil, falseFn)
	if text, ok := result.(*Text); !ok || text.Value != "false" {
		t.Error("expected false result when trueFn is nil")
	}

	result = TernaryFn(false, trueFn, nil)
	if _, ok := result.(noopItem); !ok {
		t.Error("expected noopItem when falseFn is nil")
	}
}

func TestMap(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := Map(items, func(s string) Item {
		return NewText(s)
	})

	if len(result.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(result.Children))
	}

	result = Map([]string{}, func(s string) Item {
		return NewText(s)
	})
	if len(result.Children) != 0 {
		t.Error("expected empty fragment for empty slice")
	}

	result = Map(items, nil)
	if len(result.Children) != 0 {
		t.Error("expected empty fragment for nil render function")
	}

	result = Map(items, func(s string) Item {
		if s == "b" {
			return nil
		}
		return NewText(s)
	})
	if len(result.Children) != 2 {
		t.Errorf("expected 2 children (skipping nil), got %d", len(result.Children))
	}
}

func TestMapIdx(t *testing.T) {
	items := []string{"a", "b", "c"}
	result := MapIdx(items, func(i int, s string) Item {
		return NewTextf("%d:%s", i, s)
	})

	if len(result.Children) != 3 {
		t.Errorf("expected 3 children, got %d", len(result.Children))
	}

	result = MapIdx([]string{}, func(i int, s string) Item {
		return NewText(s)
	})
	if len(result.Children) != 0 {
		t.Error("expected empty fragment for empty slice")
	}

	result = MapIdx(items, nil)
	if len(result.Children) != 0 {
		t.Error("expected empty fragment for nil render function")
	}
}

func TestNodesToItems(t *testing.T) {
	nodes := []Node{NewText("a"), NewText("b")}
	items := NodesToItems(nodes)

	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
}

func TestItemsToNodes(t *testing.T) {
	items := []Item{NewText("a"), ID("test"), NewText("b")}
	nodes := ItemsToNodes(items)

	if len(nodes) != 2 {
		t.Errorf("expected 2 nodes (excluding non-Node items), got %d", len(nodes))
	}
}

func TestAttrHelpers(t *testing.T) {
	tests := []struct {
		name     string
		item     Item
		attrName string
		expected string
	}{
		{"Href", Href("/path"), "href", "/path"},
		{"Src", Src("image.png"), "src", "image.png"},
		{"Target", Target("_blank"), "target", "_blank"},
		{"Rel", Rel("noopener"), "rel", "noopener"},
		{"Title", Title("test"), "title", "test"},
		{"Alt", Alt("description"), "alt", "description"},
		{"Type", Type("text"), "type", "text"},
		{"Value", Value("input"), "value", "input"},
		{"Name", Name("field"), "name", "field"},
		{"Placeholder", Placeholder("Enter..."), "placeholder", "Enter..."},
		{"Data", Data("id", "123"), "data-id", "123"},
		{"Aria", Aria("label", "Button"), "aria-label", "Button"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &Element{Tag: "div"}
			tt.item.ApplyTo(el)
			if el.Attrs[tt.attrName][0] != tt.expected {
				t.Errorf("expected %s=%s, got %v", tt.attrName, tt.expected, el.Attrs[tt.attrName])
			}
		})
	}
}

func TestBoolAttrHelpers(t *testing.T) {
	tests := []struct {
		name     string
		item     Item
		attrName string
	}{
		{"Disabled", Disabled(), "disabled"},
		{"Checked", Checked(), "checked"},
		{"Required", Required(), "required"},
		{"Readonly", Readonly(), "readonly"},
		{"Autofocus", Autofocus(), "autofocus"},
		{"Autoplay", Autoplay(), "autoplay"},
		{"Controls", Controls(), "controls"},
		{"Loop", Loop(), "loop"},
		{"Muted", Muted(), "muted"},
		{"Selected", Selected(), "selected"},
		{"Multiple", Multiple(), "multiple"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			el := &Element{Tag: "input"}
			tt.item.ApplyTo(el)
			if _, exists := el.Attrs[tt.attrName]; !exists {
				t.Errorf("expected %s attribute to be set", tt.attrName)
			}
		})
	}
}

func TestStylesItem(t *testing.T) {
	el := &Element{Tag: "div"}
	styles := Styles(map[string]string{
		"color":      "red",
		"background": "blue",
	})
	styles.ApplyTo(el)

	if el.Style["color"] != "red" {
		t.Error("expected color=red")
	}
	if el.Style["background"] != "blue" {
		t.Error("expected background=blue")
	}
}

func TestOnEvent(t *testing.T) {
	el := &Element{Tag: "button"}
	called := false

	On("click", func(e Event) Updates {
		called = true
		return nil
	}).ApplyTo(el)

	if el.Handlers["click"].Fn == nil {
		t.Error("expected click handler")
	}

	el.Handlers["click"].Fn(Event{Name: "click"})
	if !called {
		t.Error("expected handler to be called")
	}
}

func TestOnWithEvent(t *testing.T) {
	el := &Element{Tag: "button"}

	OnWith("click", metadata.EventOptions{
		Prevent: true,
		Stop:    true,
	}, func(e Event) Updates {
		return nil
	}).ApplyTo(el)

	if !el.Handlers["click"].EventOptions.Prevent {
		t.Error("expected Prevent option")
	}
	if !el.Handlers["click"].EventOptions.Stop {
		t.Error("expected Stop option")
	}
}

func TestAttachNil(t *testing.T) {
	result := Attach(nil)
	if _, ok := result.(noopItem); !ok {
		t.Error("expected noopItem for nil attachment")
	}
}

func TestClassWithEmptyStrings(t *testing.T) {
	el := &Element{Tag: "div"}
	Class("", "foo", "  ", "bar", "").ApplyTo(el)

	if len(el.Attrs["class"]) != 2 {
		t.Errorf("expected 2 classes (excluding empty), got %d", len(el.Attrs["class"]))
	}
}

func TestNoopItemApplyTo(t *testing.T) {
	el := &Element{Tag: "div", Children: []Node{}}
	noop := noopItem{}
	noop.ApplyTo(el)

	if len(el.Children) != 0 {
		t.Error("noopItem should not add children")
	}
}

func TestSlotMarker(t *testing.T) {
	child := NewText("content")
	marker := SlotMarker("default", child)

	frag, ok := marker.(*Fragment)
	if !ok {
		t.Fatal("expected SlotMarker to return *Fragment")
	}
	if frag.Metadata["slot:name"] != "default" {
		t.Errorf("expected slot:name 'default', got %v", frag.Metadata["slot:name"])
	}
	if len(frag.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(frag.Children))
	}
}

func TestScopedSlotMarker(t *testing.T) {
	fn := func(data string) Node {
		return NewText(data)
	}
	marker := ScopedSlotMarker("items", fn)

	frag, ok := marker.(*Fragment)
	if !ok {
		t.Fatal("expected ScopedSlotMarker to return *Fragment")
	}
	if frag.Metadata["scoped-slot:name"] != "items" {
		t.Errorf("expected scoped-slot:name 'items', got %v", frag.Metadata["scoped-slot:name"])
	}
	if frag.Metadata["scoped-slot:fn"] == nil {
		t.Error("expected scoped-slot:fn to be set")
	}
}

func TestFragmentApplyTo(t *testing.T) {
	parent := &Element{Tag: "div"}
	frag := &Fragment{Children: []Node{NewText("a"), NewText("b")}}
	frag.ApplyTo(parent)

	if len(parent.Children) != 1 {
		t.Errorf("expected 1 child (fragment), got %d", len(parent.Children))
	}
}

func TestCommentApplyTo(t *testing.T) {
	parent := &Element{Tag: "div"}
	comment := &Comment{Value: "test comment"}
	comment.ApplyTo(parent)

	if len(parent.Children) != 1 {
		t.Errorf("expected 1 child (comment), got %d", len(parent.Children))
	}
}

func TestMergeUpdates(t *testing.T) {
	t.Run("nil a", func(t *testing.T) {
		result := mergeUpdates(nil, "b")
		if result != "b" {
			t.Error("expected b when a is nil")
		}
	})

	t.Run("nil b", func(t *testing.T) {
		result := mergeUpdates("a", nil)
		if result != "a" {
			t.Error("expected a when b is nil")
		}
	})

	t.Run("both slices", func(t *testing.T) {
		a := []Updates{"a1", "a2"}
		b := []Updates{"b1", "b2"}
		result := mergeUpdates(a, b)
		if slice, ok := result.([]Updates); !ok || len(slice) != 4 {
			t.Error("expected merged slice of 4")
		}
	})

	t.Run("a slice b single", func(t *testing.T) {
		a := []Updates{"a1", "a2"}
		result := mergeUpdates(a, "b")
		if slice, ok := result.([]Updates); !ok || len(slice) != 3 {
			t.Error("expected merged slice of 3")
		}
	})

	t.Run("a single b slice", func(t *testing.T) {
		b := []Updates{"b1", "b2"}
		result := mergeUpdates("a", b)
		if slice, ok := result.([]Updates); !ok || len(slice) != 3 {
			t.Error("expected merged slice of 3")
		}
	})

	t.Run("both single", func(t *testing.T) {
		result := mergeUpdates("a", "b")
		if slice, ok := result.([]Updates); !ok || len(slice) != 2 {
			t.Error("expected slice of 2")
		}
	})
}
