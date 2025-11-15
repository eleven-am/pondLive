package dom

import "testing"

type testDescriptor struct{}

func (testDescriptor) TagName() string { return "test" }

func TestAttachElementRefPanicsWhenReused(t *testing.T) {
	ref := NewElementRef("ref:1", testDescriptor{})
	first := &Element{Tag: "test", Descriptor: testDescriptor{}}
	AttachElementRef[testDescriptor](ref, first)

	second := &Element{Tag: "test", Descriptor: testDescriptor{}}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("AttachElementRef should panic when reusing a ref, but it did not")
		}
	}()
	AttachElementRef[testDescriptor](ref, second)
}

func TestAttachElementRefAllowsReattachAfterReset(t *testing.T) {
	ref := NewElementRef("ref:2", testDescriptor{})
	first := &Element{Tag: "test", Descriptor: testDescriptor{}}
	AttachElementRef[testDescriptor](ref, first)

	ref.ResetAttachment()

	second := &Element{Tag: "test", Descriptor: testDescriptor{}}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AttachElementRef panicked after ResetAttachment: %v", r)
		}
	}()
	AttachElementRef[testDescriptor](ref, second)
}

func TestAttachElementRefSetsRefID(t *testing.T) {
	ref := NewElementRef("ref:3", testDescriptor{})
	el := &Element{Tag: "test", Descriptor: testDescriptor{}}

	AttachElementRef[testDescriptor](ref, el)

	if got, want := el.RefID, "ref:3"; got != want {
		t.Fatalf("AttachElementRef should set element ref ID to %q, got %q", want, got)
	}
}

func TestAddListenerReplacesHandlersPerEvent(t *testing.T) {
	ref := NewElementRef("ref:4", testDescriptor{})

	var firstCalled, secondCalled int
	first := func(Event) Updates {
		firstCalled++
		return nil
	}
	second := func(Event) Updates {
		secondCalled++
		return nil
	}

	ref.AddListener("click", first, nil)
	ref.ResetAttachment()
	ref.AddListener("click", second, nil)

	ref.dispatchEvent("click", Event{Name: "click"})

	if firstCalled != 0 {
		t.Fatalf("expected previous handler to be replaced, but it was invoked %d times", firstCalled)
	}
	if secondCalled != 1 {
		t.Fatalf("expected new handler to be invoked once, got %d", secondCalled)
	}

	ref.dispatchEvent("click", Event{Name: "click"})

	if secondCalled != 2 {
		t.Fatalf("expected handler to remain registered across dispatches, got %d invocations", secondCalled)
	}

	bucket := ref.listeners["click"]
	if bucket == nil {
		t.Fatalf("expected listener bucket for event to exist")
	}
	if len(bucket.handlers) != 1 {
		t.Fatalf("expected a single handler in bucket, got %d", len(bucket.handlers))
	}
}

func TestRefEventBindingKeyGeneration(t *testing.T) {
	tests := []struct {
		name     string
		id       string
		event    string
		expected string
	}{
		{
			name:     "basic click event",
			id:       "ref:0",
			event:    "click",
			expected: "ref:0/click",
		},
		{
			name:     "input event",
			id:       "ref:1",
			event:    "input",
			expected: "ref:1/input",
		},
		{
			name:     "uppercase event is lowercased",
			id:       "ref:2",
			event:    "MouseDown",
			expected: "ref:2/mousedown",
		},
		{
			name:     "empty id returns empty",
			id:       "",
			event:    "click",
			expected: "",
		},
		{
			name:     "empty event returns empty",
			id:       "ref:0",
			event:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := refEventBindingKey(tt.id, tt.event)
			if got != tt.expected {
				t.Errorf("refEventBindingKey(%q, %q) = %q, want %q", tt.id, tt.event, got, tt.expected)
			}
		})
	}
}
