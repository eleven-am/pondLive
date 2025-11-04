package handlers

import "testing"

func TestRegistryEnsureStableID(t *testing.T) {
	reg := NewRegistry()
	handler := func(Event) Updates { return nil }
	id1 := reg.Ensure(handler)
	id2 := reg.Ensure(handler)
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
	id := reg.Ensure(handler)
	if _, ok := reg.Get(id); !ok {
		t.Fatalf("expected handler %s to be registered", id)
	}
	reg.Remove(id)
	if _, ok := reg.Get(id); ok {
		t.Fatalf("expected handler %s to be removed", id)
	}
	// ensure new registration can reuse pointer without stale id
	id2 := reg.Ensure(handler)
	if id2 == id {
		t.Fatalf("expected new id after removal, got same %s", id2)
	}
}
