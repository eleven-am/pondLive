package script

import (
	"strings"
	"testing"
)

type fakeKey struct {
	*Def[none, none]
	key string
}

func (f fakeKey) Key() string { return f.key }

func TestNewScriptAfterSealPanics(t *testing.T) {
	r := NewRegistry()
	NewIn[struct{}, struct{}](r, 0, "before")
	if err := r.Seal(); err != nil {
		t.Fatal(err)
	}
	if !r.Sealed() {
		t.Fatal("not sealed")
	}
	defer func() {
		v := recover()
		if v == nil {
			t.Fatal("declaring after the seal did not panic")
		}
		if !strings.Contains(v.(string), "registry_test.go:") {
			t.Fatalf("panic does not name the declaration: %v", v)
		}
	}()
	NewIn[struct{}, struct{}](r, 0, "after")
}

func TestSealIsIdempotent(t *testing.T) {
	r := NewRegistry()
	a := fakeKey{Def: NewIn[struct{}, struct{}](NewRegistry(), 0, "a"), key: "k"}
	b := fakeKey{Def: NewIn[struct{}, struct{}](NewRegistry(), 0, "b"), key: "k"}
	r.Register(a)
	r.Register(b)
	first := r.Seal()
	second := r.Seal()
	if first == nil || second == nil || first.Error() != second.Error() {
		t.Fatalf("seal not idempotent: %v / %v", first, second)
	}
}

func TestKeyCollisionWithDifferentSourceFails(t *testing.T) {
	r := NewRegistry()
	a := fakeKey{Def: NewIn[struct{}, struct{}](NewRegistry(), 0, "source A"), key: "abc123abc123"}
	b := fakeKey{Def: NewIn[struct{}, struct{}](NewRegistry(), 0, "source B"), key: "abc123abc123"}
	r.Register(a)
	r.Register(b)
	err := r.Seal()
	if err == nil || !strings.Contains(err.Error(), "collides") {
		t.Fatalf("collision not reported: %v", err)
	}
	if _, err := r.Bundle(Options{}); err == nil {
		t.Fatal("Bundle ran on a registry with a collision")
	}
	if got, _ := r.Lookup("abc123abc123"); got.Source() != "source A" {
		t.Fatalf("last write won: %q", got.Source())
	}
}

func TestEntriesAreSortedByKey(t *testing.T) {
	r := NewRegistry()
	for i := 0; i < 20; i++ {
		NewIn[struct{}, struct{}](r, 0, strings.Repeat("x", i+1))
	}
	es := r.Entries()
	for i := 1; i < len(es); i++ {
		if es[i-1].Key() >= es[i].Key() {
			t.Fatal("entries not sorted")
		}
	}
}
