package dom

import (
	"testing"
)

// testDescriptor implements ElementDescriptor for testing
type testDescriptor struct{ tag string }

func (t testDescriptor) TagName() string { return t.tag }

// mockDispatcher implements the Dispatcher interface for testing
type mockDispatcher struct {
	actions []DOMActionEffect
	calls   []struct {
		ref    string
		method string
		args   []any
	}
	gets []struct {
		ref       string
		selectors []string
	}
}

func (m *mockDispatcher) EnqueueDOMAction(effect DOMActionEffect) {
	m.actions = append(m.actions, effect)
}

func (m *mockDispatcher) DOMGet(ref string, selectors ...string) (map[string]any, error) {
	m.gets = append(m.gets, struct {
		ref       string
		selectors []string
	}{ref, selectors})
	return nil, nil
}

func (m *mockDispatcher) DOMAsyncCall(ref string, method string, args ...any) (any, error) {
	m.calls = append(m.calls, struct {
		ref    string
		method string
		args   []any
	}{ref, method, args})
	return "result", nil
}

func TestDOMCall(t *testing.T) {
	tests := []struct {
		name   string
		method string
		args   []any
		want   *DOMActionEffect
	}{
		{
			name:   "simple method call",
			method: "focus",
			args:   nil,
			want: &DOMActionEffect{
				Type:   domEffectType,
				Kind:   domActionCall,
				Ref:    "test-ref",
				Method: "focus",
			},
		},
		{
			name:   "method with args",
			method: "scrollTo",
			args:   []any{0, 100},
			want: &DOMActionEffect{
				Type:   domEffectType,
				Kind:   domActionCall,
				Ref:    "test-ref",
				Method: "scrollTo",
				Args:   []any{0, 100},
			},
		},
		{
			name:   "method with whitespace trimmed",
			method: "  blur  ",
			args:   nil,
			want: &DOMActionEffect{
				Type:   domEffectType,
				Kind:   domActionCall,
				Ref:    "test-ref",
				Method: "blur",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDispatcher{}
			ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

			DOMCall(mock, ref, tt.method, tt.args...)

			if len(mock.actions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(mock.actions))
			}

			got := mock.actions[0]
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Kind != tt.want.Kind {
				t.Errorf("Kind = %q, want %q", got.Kind, tt.want.Kind)
			}
			if got.Ref != tt.want.Ref {
				t.Errorf("Ref = %q, want %q", got.Ref, tt.want.Ref)
			}
			if got.Method != tt.want.Method {
				t.Errorf("Method = %q, want %q", got.Method, tt.want.Method)
			}
			if len(got.Args) != len(tt.want.Args) {
				t.Errorf("Args length = %d, want %d", len(got.Args), len(tt.want.Args))
			}
		})
	}
}

func TestDOMCall_EmptyMethod(t *testing.T) {
	mock := &mockDispatcher{}
	ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

	DOMCall(mock, ref, "")
	DOMCall(mock, ref, "   ")

	if len(mock.actions) != 0 {
		t.Errorf("expected no actions for empty method, got %d", len(mock.actions))
	}
}

func TestDOMCall_NilContext(t *testing.T) {
	ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

	DOMCall(nil, ref, "focus")
}

func TestDOMCall_NilRef(t *testing.T) {
	mock := &mockDispatcher{}

	DOMCall(mock, (*ElementRef[testDescriptor])(nil), "focus")

	if len(mock.actions) != 0 {
		t.Errorf("expected no actions for nil ref, got %d", len(mock.actions))
	}
}

func TestDOMSet(t *testing.T) {
	tests := []struct {
		name  string
		prop  string
		value any
		want  *DOMActionEffect
	}{
		{
			name:  "set string property",
			prop:  "value",
			value: "hello",
			want: &DOMActionEffect{
				Type:  domEffectType,
				Kind:  domActionSet,
				Ref:   "test-ref",
				Prop:  "value",
				Value: "hello",
			},
		},
		{
			name:  "set numeric property",
			prop:  "scrollTop",
			value: 100,
			want: &DOMActionEffect{
				Type:  domEffectType,
				Kind:  domActionSet,
				Ref:   "test-ref",
				Prop:  "scrollTop",
				Value: 100,
			},
		},
		{
			name:  "set boolean property",
			prop:  "disabled",
			value: true,
			want: &DOMActionEffect{
				Type:  domEffectType,
				Kind:  domActionSet,
				Ref:   "test-ref",
				Prop:  "disabled",
				Value: true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDispatcher{}
			ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

			DOMSet(mock, ref, tt.prop, tt.value)

			if len(mock.actions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(mock.actions))
			}

			got := mock.actions[0]
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Kind != tt.want.Kind {
				t.Errorf("Kind = %q, want %q", got.Kind, tt.want.Kind)
			}
			if got.Prop != tt.want.Prop {
				t.Errorf("Prop = %q, want %q", got.Prop, tt.want.Prop)
			}
			if got.Value != tt.value {
				t.Errorf("Value = %v, want %v", got.Value, tt.value)
			}
		})
	}
}

func TestDOMSet_EmptyProp(t *testing.T) {
	mock := &mockDispatcher{}
	ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

	DOMSet(mock, ref, "", "value")
	DOMSet(mock, ref, "   ", "value")

	if len(mock.actions) != 0 {
		t.Errorf("expected no actions for empty prop, got %d", len(mock.actions))
	}
}

func TestDOMToggle(t *testing.T) {
	tests := []struct {
		name string
		prop string
		on   bool
		want *DOMActionEffect
	}{
		{
			name: "toggle on",
			prop: "disabled",
			on:   true,
			want: &DOMActionEffect{
				Type:  domEffectType,
				Kind:  domActionToggle,
				Ref:   "test-ref",
				Prop:  "disabled",
				Value: true,
			},
		},
		{
			name: "toggle off",
			prop: "checked",
			on:   false,
			want: &DOMActionEffect{
				Type:  domEffectType,
				Kind:  domActionToggle,
				Ref:   "test-ref",
				Prop:  "checked",
				Value: false,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDispatcher{}
			ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

			DOMToggle(mock, ref, tt.prop, tt.on)

			if len(mock.actions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(mock.actions))
			}

			got := mock.actions[0]
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Kind != tt.want.Kind {
				t.Errorf("Kind = %q, want %q", got.Kind, tt.want.Kind)
			}
			if got.Prop != tt.want.Prop {
				t.Errorf("Prop = %q, want %q", got.Prop, tt.want.Prop)
			}
			if got.Value != tt.on {
				t.Errorf("Value = %v, want %v", got.Value, tt.on)
			}
		})
	}
}

func TestDOMToggleClass(t *testing.T) {
	tests := []struct {
		name  string
		class string
		on    bool
		want  *DOMActionEffect
	}{
		{
			name:  "add class",
			class: "active",
			on:    true,
			want: &DOMActionEffect{
				Type:  domEffectType,
				Kind:  domActionClass,
				Ref:   "test-ref",
				Class: "active",
			},
		},
		{
			name:  "remove class",
			class: "hidden",
			on:    false,
			want: &DOMActionEffect{
				Type:  domEffectType,
				Kind:  domActionClass,
				Ref:   "test-ref",
				Class: "hidden",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDispatcher{}
			ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

			DOMToggleClass(mock, ref, tt.class, tt.on)

			if len(mock.actions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(mock.actions))
			}

			got := mock.actions[0]
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Kind != tt.want.Kind {
				t.Errorf("Kind = %q, want %q", got.Kind, tt.want.Kind)
			}
			if got.Class != tt.want.Class {
				t.Errorf("Class = %q, want %q", got.Class, tt.want.Class)
			}
			if got.On == nil {
				t.Fatal("On is nil")
			}
			if *got.On != tt.on {
				t.Errorf("On = %v, want %v", *got.On, tt.on)
			}
		})
	}
}

func TestDOMToggleClass_EmptyClass(t *testing.T) {
	mock := &mockDispatcher{}
	ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

	DOMToggleClass(mock, ref, "", true)
	DOMToggleClass(mock, ref, "   ", false)

	if len(mock.actions) != 0 {
		t.Errorf("expected no actions for empty class, got %d", len(mock.actions))
	}
}

func TestDOMScrollIntoView(t *testing.T) {
	tests := []struct {
		name string
		opts ScrollOptions
		want *DOMActionEffect
	}{
		{
			name: "smooth scroll to center",
			opts: ScrollOptions{
				Behavior: "smooth",
				Block:    "center",
				Inline:   "nearest",
			},
			want: &DOMActionEffect{
				Type:     domEffectType,
				Kind:     domActionScroll,
				Ref:      "test-ref",
				Behavior: "smooth",
				Block:    "center",
				Inline:   "nearest",
			},
		},
		{
			name: "instant scroll to start",
			opts: ScrollOptions{
				Behavior: "instant",
				Block:    "start",
			},
			want: &DOMActionEffect{
				Type:     domEffectType,
				Kind:     domActionScroll,
				Ref:      "test-ref",
				Behavior: "instant",
				Block:    "start",
			},
		},
		{
			name: "empty options",
			opts: ScrollOptions{},
			want: &DOMActionEffect{
				Type: domEffectType,
				Kind: domActionScroll,
				Ref:  "test-ref",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDispatcher{}
			ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

			DOMScrollIntoView(mock, ref, tt.opts)

			if len(mock.actions) != 1 {
				t.Fatalf("expected 1 action, got %d", len(mock.actions))
			}

			got := mock.actions[0]
			if got.Type != tt.want.Type {
				t.Errorf("Type = %q, want %q", got.Type, tt.want.Type)
			}
			if got.Kind != tt.want.Kind {
				t.Errorf("Kind = %q, want %q", got.Kind, tt.want.Kind)
			}
			if got.Behavior != tt.want.Behavior {
				t.Errorf("Behavior = %q, want %q", got.Behavior, tt.want.Behavior)
			}
			if got.Block != tt.want.Block {
				t.Errorf("Block = %q, want %q", got.Block, tt.want.Block)
			}
			if got.Inline != tt.want.Inline {
				t.Errorf("Inline = %q, want %q", got.Inline, tt.want.Inline)
			}
		})
	}
}

func TestDOMAsyncCall(t *testing.T) {
	tests := []struct {
		name   string
		method string
		args   []any
		want   any
	}{
		{
			name:   "call without args",
			method: "getBoundingClientRect",
			args:   nil,
			want:   "result",
		},
		{
			name:   "call with args",
			method: "querySelector",
			args:   []any{".selector"},
			want:   "result",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mock := &mockDispatcher{}
			ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

			result, err := DOMAsyncCall(mock, ref, tt.method, tt.args...)

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result != tt.want {
				t.Errorf("result = %v, want %v", result, tt.want)
			}

			if len(mock.calls) != 1 {
				t.Fatalf("expected 1 call, got %d", len(mock.calls))
			}

			call := mock.calls[0]
			if call.ref != "test-ref" {
				t.Errorf("ref = %q, want %q", call.ref, "test-ref")
			}
			if call.method != tt.method {
				t.Errorf("method = %q, want %q", call.method, tt.method)
			}
		})
	}
}

func TestDOMAsyncCall_NilContext(t *testing.T) {
	ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

	result, err := DOMAsyncCall(nil, ref, "method")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestDOMAsyncCall_NilRef(t *testing.T) {
	mock := &mockDispatcher{}

	result, err := DOMAsyncCall(mock, (*ElementRef[testDescriptor])(nil), "method")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result, got %v", result)
	}
}

func TestDOMAsyncCall_EmptyMethod(t *testing.T) {
	mock := &mockDispatcher{}
	ref := NewElementRef("test-ref", testDescriptor{tag: "div"})

	result, err := DOMAsyncCall(mock, ref, "")

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil result for empty method, got %v", result)
	}

	if len(mock.calls) != 0 {
		t.Errorf("expected no calls for empty method, got %d", len(mock.calls))
	}
}

func TestBoolRef(t *testing.T) {
	trueVal := boolRef(true)
	if trueVal == nil {
		t.Fatal("boolRef(true) returned nil")
	}
	if !*trueVal {
		t.Error("boolRef(true) = false, want true")
	}

	falseVal := boolRef(false)
	if falseVal == nil {
		t.Fatal("boolRef(false) returned nil")
	}
	if *falseVal {
		t.Error("boolRef(false) = true, want false")
	}
}
