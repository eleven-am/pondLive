package dom2

import (
	"sync"
	"testing"
)

type fakeDescriptor struct{ tag string }

func (f fakeDescriptor) TagName() string { return f.tag }

type testRefUpdates string

func (testRefUpdates) isUpdates() {}

func TestElementRefBindingSnapshotIsolation(t *testing.T) {
	ref := NewElementRef("ref-1", fakeDescriptor{tag: "div"})
	ref.Bind("input", EventBinding{
		Listen: []string{"change"},
		Props:  []string{"target.value"},
	})

	snapshot := ref.BindingSnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 binding in snapshot, got %d", len(snapshot))
	}
	clone := snapshot["input"]
	clone.Listen[0] = "mutated"
	clone.Props = append(clone.Props, "target.checked")
	snapshot["input"] = clone

	latest := ref.BindingSnapshot()
	if latest["input"].Listen[0] != "change" {
		t.Fatalf("binding snapshot mutated underlying data: %v", latest["input"].Listen)
	}
	if len(latest["input"].Props) != 1 || latest["input"].Props[0] != "target.value" {
		t.Fatalf("binding snapshot props mutated underlying data: %v", latest["input"].Props)
	}
}

func TestElementRefAddListenerDispatchAndReset(t *testing.T) {
	ref := NewElementRef("ref-1", fakeDescriptor{tag: "button"})

	var mu sync.Mutex
	var calls []string
	record := func(label string) {
		mu.Lock()
		defer mu.Unlock()
		calls = append(calls, label)
	}

	ref.AddListener("click", func(Event) Updates {
		record("first")
		return nil
	}, []string{"target.value", "target.value"})

	ref.AddListener("click", func(Event) Updates {
		record("second")
		return testRefUpdates("second")
	}, []string{"target.value", "target.checked"})

	if got := ref.dispatchEvent("click", Event{}); got != testRefUpdates("second") {
		t.Fatalf("dispatchEvent returned %v, want %v", got, testRefUpdates("second"))
	}

	if len(calls) != 2 || calls[0] != "first" || calls[1] != "second" {
		t.Fatalf("listener invocation order incorrect: %v", calls)
	}

	snapshot := ref.BindingSnapshot()
	if props := snapshot["click"].Props; len(props) != 2 {
		t.Fatalf("expected deduped props, got %v", props)
	}

	ref.ResetAttachment()
	calls = calls[:0]

	ref.AddListener("click", func(Event) Updates {
		record("after-reset")
		return nil
	}, nil)

	if got := ref.dispatchEvent("click", Event{}); got != nil {
		t.Fatalf("expected nil result after reset dispatch, got %v", got)
	}

	if len(calls) != 1 || calls[0] != "after-reset" {
		t.Fatalf("listeners from previous generation leaked: %v", calls)
	}
}

func TestElementRefRef(t *testing.T) {
	ref := NewElementRef("ref-1", fakeDescriptor{tag: "div"})

	got := ref.Ref()
	if got != ref {
		t.Errorf("Ref() returned different instance, want same instance")
	}

	// Test nil ref
	var nilRef *ElementRef[fakeDescriptor]
	if nilRef.Ref() != nil {
		t.Errorf("Ref() on nil receiver returned non-nil, want nil")
	}
}

func TestElementRefOn(t *testing.T) {
	ref := NewElementRef("ref-1", fakeDescriptor{tag: "button"})

	var called bool
	var receivedEvent Event

	ref.On("customEvent", func(evt Event) Updates {
		called = true
		receivedEvent = evt
		return testRefUpdates("handled")
	})

	snapshot := ref.BindingSnapshot()
	if len(snapshot) != 1 {
		t.Fatalf("expected 1 binding, got %d", len(snapshot))
	}

	binding, exists := snapshot["customEvent"]
	if !exists {
		t.Fatal("customEvent binding not found")
	}

	if len(binding.Props) != 1 || binding.Props[0] != CaptureAllProperties {
		t.Errorf("Props = %v, want [%s]", binding.Props, CaptureAllProperties)
	}

	testEvent := Event{Payload: map[string]any{"detail": "test"}}
	result := ref.dispatchEvent("customEvent", testEvent)

	if !called {
		t.Error("handler was not called")
	}
	if result != testRefUpdates("handled") {
		t.Errorf("result = %v, want %v", result, testRefUpdates("handled"))
	}
	if receivedEvent.Payload == nil {
		t.Error("receivedEvent.Payload is nil")
	}
}

func TestElementRefOn_NilRef(t *testing.T) {
	var nilRef *ElementRef[fakeDescriptor]

	nilRef.On("click", func(Event) Updates {
		return nil
	})
}

func TestElementRefOn_NilHandler(t *testing.T) {
	ref := NewElementRef("ref-1", fakeDescriptor{tag: "button"})

	ref.On("click", nil)

	snapshot := ref.BindingSnapshot()
	if len(snapshot) != 0 {
		t.Errorf("expected no bindings for nil handler, got %d", len(snapshot))
	}
}
