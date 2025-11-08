package runtime

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
)

type stubCallable struct {
	nameStr string
	ptr     uintptr
}

func (s stubCallable) call(Ctx, any) h.Node { return nil }
func (s stubCallable) pointer() uintptr     { return s.ptr }
func (s stubCallable) name() string         { return s.nameStr }

func TestBuildComponentIDDeterministic(t *testing.T) {
	callable := stubCallable{nameStr: "example.Component", ptr: 0x01}

	first := buildComponentID(nil, callable, "item")
	second := buildComponentID(nil, callable, "item")

	if first != second {
		t.Fatalf("component IDs must be stable across renders: %q vs %q", first, second)
	}
	if len(first) == 0 || first[0] != 'c' {
		t.Fatalf("expected hashed component ID to be prefixed with 'c', got %q", first)
	}
}

func TestBuildComponentIDDifferentKeys(t *testing.T) {
	callable := stubCallable{nameStr: "example.Component", ptr: 0x01}

	a := buildComponentID(nil, callable, "a")
	b := buildComponentID(nil, callable, "b")

	if a == b {
		t.Fatalf("component IDs should differ when keys differ: %q == %q", a, b)
	}
}

func TestBuildComponentIDDifferentParents(t *testing.T) {
	parentCallable := stubCallable{nameStr: "parent.Component", ptr: 0x02}
	childCallable := stubCallable{nameStr: "example.Component", ptr: 0x01}

	parentA := &component{id: buildComponentID(nil, parentCallable, "A")}
	parentB := &component{id: buildComponentID(nil, parentCallable, "B")}

	childFromA := buildComponentID(parentA, childCallable, "child")
	childFromB := buildComponentID(parentB, childCallable, "child")

	if childFromA == childFromB {
		t.Fatalf("child IDs should differ for different parent instances: %q == %q", childFromA, childFromB)
	}
}
