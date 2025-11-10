package handlers

import "testing"

func TestRegistryEnsureStableID(t *testing.T) {
	reg := NewRegistry()
	handler := func(Event) Updates { return nil }
	id1 := reg.Ensure(handler, "")
	id2 := reg.Ensure(handler, "")
	if id1 == "" || id2 == "" {
		t.Fatal("expected handler ids to be assigned")
	}
	if id1 != id2 {
		t.Fatalf("expected stable id, got %s and %s", id1, id2)
	}
}

func TestRegistryRemove(t *testing.T) {
	reg := NewRegistry()
	handler := func(Event) Updates { return nil }
	id := reg.Ensure(handler, "")
	if _, ok := reg.Get(id); !ok {
		t.Fatalf("expected handler %s to be registered", id)
	}
	reg.Remove(id)
	if _, ok := reg.Get(id); ok {
		t.Fatalf("expected handler %s to be removed", id)
	}

	id2 := reg.Ensure(handler, "")
	if id2 == id {
		t.Fatalf("expected new id after removal, got same %s", id2)
	}
}

func TestRegistryKeyedEnsure(t *testing.T) {
	reg := NewRegistry()
	handlerA := func(Event) Updates { return nil }
	handlerB := func(Event) Updates { return nil }
	first := reg.Ensure(handlerA, "ref:r1/input")
	second := reg.Ensure(handlerB, "ref:r1/input")
	if first != second {
		t.Fatalf("expected keyed handlers to reuse id, got %s vs %s", first, second)
	}
	third := reg.Ensure(handlerB, "ref:r2/input")
	if third == first {
		t.Fatalf("expected distinct key to produce new id, reused %s", third)
	}
}

func TestRegistryKeyedVsPointerBased(t *testing.T) {
	reg := NewRegistry()

	makeHandler := func(id int) Handler {
		return func(Event) Updates {
			_ = id
			return nil
		}
	}

	handler1 := makeHandler(1)
	handler2 := makeHandler(2)

	id1 := reg.Ensure(handler1, "")
	id2 := reg.Ensure(handler2, "")

	if id1 != id2 {
		t.Log("Note: Handlers with identical code may or may not share function pointers depending on Go compiler optimization")
	}

	id3 := reg.Ensure(handler1, "handler-1")
	id4 := reg.Ensure(handler2, "handler-2")

	if id3 == id4 {
		t.Errorf("expected different handler IDs with different keys, got same ID %s", id3)
	}

	id5 := reg.Ensure(handler1, "handler-1")
	if id5 != id3 {
		t.Errorf("expected same ID for same key, got %s vs %s", id3, id5)
	}
}

func TestRegistryKeyPreventsPointerDeduplication(t *testing.T) {
	reg := NewRegistry()

	handler := func(Event) Updates { return nil }

	id1 := reg.Ensure(handler, "")

	id2 := reg.Ensure(handler, "my-key")

	if id1 == id2 {
		t.Errorf("expected different IDs for keyed vs non-keyed handler, got same ID %s", id1)
	}

	id3 := reg.Ensure(handler, "my-key")
	if id3 != id2 {
		t.Errorf("expected same ID for same key, got %s vs %s", id2, id3)
	}

	id4 := reg.Ensure(handler, "")
	if id4 != id1 {
		t.Errorf("expected same ID for pointer-based lookup, got %s vs %s", id1, id4)
	}
}

func TestRegistryMultipleRefsWithSameEvent(t *testing.T) {
	reg := NewRegistry()

	makeRefHandler := func(refID string) Handler {
		return func(evt Event) Updates {
			_ = refID
			return nil
		}
	}

	handler1 := makeRefHandler("ref:0")
	handler2 := makeRefHandler("ref:1")

	id1 := reg.Ensure(handler1, "ref:ref:0/click")
	id2 := reg.Ensure(handler2, "ref:ref:1/click")

	if id1 == id2 {
		t.Errorf("expected different IDs for different ref keys, got same ID %s", id1)
	}

	h1, ok1 := reg.Get(id1)
	if !ok1 || h1 == nil {
		t.Errorf("expected to retrieve handler for %s", id1)
	}

	h2, ok2 := reg.Get(id2)
	if !ok2 || h2 == nil {
		t.Errorf("expected to retrieve handler for %s", id2)
	}
}
