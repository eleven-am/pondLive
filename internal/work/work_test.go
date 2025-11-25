package work

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/metadata"
)

// Mock attachment for testing
type mockAttachment struct {
	id string
}

func (m mockAttachment) RefID() string {
	return m.id
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

	comp := &Component{Fn: nil, Props: nil}
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

	comp := NewComponent(mockFn, child1, child2)

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

	comp := NewPropsComponent(mockFn, props, child)

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
	comp := NewComponent(mockFn).WithKey("comp-123")

	if comp.Key != "comp-123" {
		t.Errorf("expected key='comp-123', got %s", comp.Key)
	}
}

func TestComponentAsChild(t *testing.T) {
	parent := &Element{Tag: "div"}
	mockFn := func() {}

	comp := NewComponent(mockFn, &Text{Value: "test"})
	comp.ApplyTo(parent)

	if len(parent.Children) != 1 {
		t.Errorf("expected 1 child, got %d", len(parent.Children))
	}

	if _, ok := parent.Children[0].(*Component); !ok {
		t.Error("child should be a Component")
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
		UnsafeHTML("<span>raw html</span>"),
	)

	if el.UnsafeHTML != "<span>raw html</span>" {
		t.Errorf("expected unsafe html to be set, got %s", el.UnsafeHTML)
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

// Mock ref that implements HandlerProvider
type mockHandlerRef struct {
	id       string
	handlers map[string][]Handler
}

func (m *mockHandlerRef) RefID() string {
	return m.id
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
