package runtime

import (
	"testing"
)

// TestDepsEqual_FunctionPointers tests function pointer comparison
func TestDepsEqual_FunctionPointers(t *testing.T) {
	fn1 := func() int { return 1 }
	fn2 := func() int { return 1 }
	fn3 := fn1

	if !depsEqual([]any{fn1}, []any{fn3}) {
		t.Error("expected same function instance to be equal")
	}

	if depsEqual([]any{fn1}, []any{fn2}) {
		t.Error("expected different function instances to be unequal")
	}

	var nilFn func()
	if !depsEqual([]any{nilFn}, []any{nilFn}) {
		t.Error("expected nil functions to be equal")
	}

	nonNilFn := func() {}
	if depsEqual([]any{nilFn}, []any{nonNilFn}) {
		t.Error("expected nil and non-nil functions to be unequal")
	}
}

// TestDepsEqual_Channels tests channel comparison
func TestDepsEqual_Channels(t *testing.T) {
	ch1 := make(chan int)
	ch2 := make(chan int)
	ch3 := ch1

	if !depsEqual([]any{ch1}, []any{ch3}) {
		t.Error("expected same channel instance to be equal")
	}

	if depsEqual([]any{ch1}, []any{ch2}) {
		t.Error("expected different channel instances to be unequal")
	}

	var nilCh chan int
	if !depsEqual([]any{nilCh}, []any{nilCh}) {
		t.Error("expected nil channels to be equal")
	}

	if depsEqual([]any{nilCh}, []any{ch1}) {
		t.Error("expected nil and non-nil channels to be unequal")
	}

	ch4 := make(chan string)
	if depsEqual([]any{ch1}, []any{ch4}) {
		t.Error("expected different channel types to be unequal")
	}
}

// TestDepsEqual_Maps tests map pointer comparison
func TestDepsEqual_Maps(t *testing.T) {
	map1 := map[string]int{"a": 1}
	map2 := map[string]int{"a": 1}
	map3 := map1

	if !depsEqual([]any{map1}, []any{map3}) {
		t.Error("expected same map instance to be equal")
	}

	if depsEqual([]any{map1}, []any{map2}) {
		t.Error("expected different map instances to be unequal")
	}

	var nilMap map[string]int
	if !depsEqual([]any{nilMap}, []any{nilMap}) {
		t.Error("expected nil maps to be equal")
	}

	if depsEqual([]any{nilMap}, []any{map1}) {
		t.Error("expected nil and non-nil maps to be unequal")
	}

	emptyMap := make(map[string]int)
	if depsEqual([]any{nilMap}, []any{emptyMap}) {
		t.Error("expected nil and empty maps to be unequal")
	}
}

// TestDepsEqual_MixedTypes tests mixed dependency types
func TestDepsEqual_MixedTypes(t *testing.T) {
	fn := func() int { return 1 }
	ch := make(chan int)
	m := map[string]int{"a": 1}
	num := 42
	str := "hello"

	deps1 := []any{fn, ch, m, num, str}
	deps2 := []any{fn, ch, m, num, str}

	if !depsEqual(deps1, deps2) {
		t.Error("expected identical mixed dependencies to be equal")
	}

	deps3 := []any{fn, ch, m, 43, str}
	if depsEqual(deps1, deps3) {
		t.Error("expected different primitive value to make deps unequal")
	}

	fn2 := func() int { return 1 }
	deps4 := []any{fn2, ch, m, num, str}
	if depsEqual(deps1, deps4) {
		t.Error("expected different function to make deps unequal")
	}
}

// TestDepsEqual_NilValues tests nil handling across types
func TestDepsEqual_NilValues(t *testing.T) {

	if !depsEqual([]any{nil}, []any{nil}) {
		t.Error("expected nil values to be equal")
	}

	if depsEqual([]any{nil}, []any{42}) {
		t.Error("expected nil and non-nil to be unequal")
	}

	if !depsEqual([]any{nil, nil, nil}, []any{nil, nil, nil}) {
		t.Error("expected multiple nils to be equal")
	}

	deps1 := []any{nil, 42, nil}
	deps2 := []any{nil, 42, nil}
	if !depsEqual(deps1, deps2) {
		t.Error("expected same nil pattern to be equal")
	}

	deps3 := []any{nil, 43, nil}
	if depsEqual(deps1, deps3) {
		t.Error("expected different non-nil value to make deps unequal")
	}
}

// TestDepsEqual_TypeMismatches tests type safety
func TestDepsEqual_TypeMismatches(t *testing.T) {

	if depsEqual([]any{42}, []any{"42"}) {
		t.Error("expected int and string to be unequal")
	}

	if depsEqual([]any{42}, []any{42.0}) {
		t.Error("expected int and float to be unequal")
	}

	if depsEqual([]any{1, 2}, []any{1, 2, 3}) {
		t.Error("expected different length arrays to be unequal")
	}

	if depsEqual([]any{1}, []any{}) {
		t.Error("expected different length arrays to be unequal")
	}

	if !depsEqual([]any{}, []any{}) {
		t.Error("expected empty arrays to be equal")
	}
}

// TestDepsEqual_ComplexStructures tests structs, slices, arrays
func TestDepsEqual_ComplexStructures(t *testing.T) {
	type Person struct {
		Name string
		Age  int
	}

	p1 := Person{"Alice", 30}
	p2 := Person{"Alice", 30}
	p3 := Person{"Bob", 25}

	if !depsEqual([]any{p1}, []any{p2}) {
		t.Error("expected equal structs to be equal")
	}

	if depsEqual([]any{p1}, []any{p3}) {
		t.Error("expected different structs to be unequal")
	}

	slice1 := []int{1, 2, 3}
	slice2 := []int{1, 2, 3}
	slice3 := []int{1, 2, 4}

	if !depsEqual([]any{slice1}, []any{slice2}) {
		t.Error("expected equal slices to be equal")
	}

	if depsEqual([]any{slice1}, []any{slice3}) {
		t.Error("expected different slices to be unequal")
	}

	arr1 := [3]int{1, 2, 3}
	arr2 := [3]int{1, 2, 3}
	arr3 := [3]int{1, 2, 4}

	if !depsEqual([]any{arr1}, []any{arr2}) {
		t.Error("expected equal arrays to be equal")
	}

	if depsEqual([]any{arr1}, []any{arr3}) {
		t.Error("expected different arrays to be unequal")
	}
}

// TestDepsEqual_PointerTypes tests pointer comparison
func TestDepsEqual_PointerTypes(t *testing.T) {
	val1 := 42
	val2 := 42
	val3 := 99
	ptr1 := &val1
	ptr2 := &val2
	ptr3 := ptr1
	ptrDiff := &val3

	if !depsEqual([]any{ptr1}, []any{ptr3}) {
		t.Error("expected same pointer to be equal")
	}

	if !depsEqual([]any{ptr1}, []any{ptr2}) {
		t.Error("expected pointers to same value to be equal")
	}

	if depsEqual([]any{ptr1}, []any{ptrDiff}) {
		t.Error("expected pointers to different values to be unequal")
	}

	var nilPtr *int
	if !depsEqual([]any{nilPtr}, []any{nilPtr}) {
		t.Error("expected nil pointers to be equal")
	}

	if depsEqual([]any{nilPtr}, []any{ptr1}) {
		t.Error("expected nil and non-nil pointers to be unequal")
	}
}

// TestDepsEqual_Interfaces tests interface values
func TestDepsEqual_Interfaces(t *testing.T) {
	var i1 interface{} = 42
	var i2 interface{} = 42
	var i3 interface{} = "42"

	if !depsEqual([]any{i1}, []any{i2}) {
		t.Error("expected same interface values to be equal")
	}

	if depsEqual([]any{i1}, []any{i3}) {
		t.Error("expected different interface types to be unequal")
	}

	var nilIntf interface{}
	if !depsEqual([]any{nilIntf}, []any{nilIntf}) {
		t.Error("expected nil interfaces to be equal")
	}
}

// TestDepsEqual_NestedFunctions tests comparison of nested structures
func TestDepsEqual_NestedFunctions(t *testing.T) {

	slice1 := [][]int{{1, 2}, {3, 4}}
	slice2 := [][]int{{1, 2}, {3, 4}}
	slice3 := [][]int{{1, 2}, {3, 5}}

	if !depsEqual([]any{slice1}, []any{slice2}) {
		t.Error("expected equal nested slices to be equal")
	}

	if depsEqual([]any{slice1}, []any{slice3}) {
		t.Error("expected different nested slices to be unequal")
	}

	fn1 := func() {}
	fn2 := fn1
	fn3 := func() {}

	if !depsEqual([]any{fn1}, []any{fn2}) {
		t.Error("expected same function to be equal")
	}

	if depsEqual([]any{fn1}, []any{fn3}) {
		t.Error("expected different functions to be unequal")
	}
}

// TestDepsValueEqual_EdgeCases tests the underlying value comparison
func TestDepsValueEqual_EdgeCases(t *testing.T) {

	if !depsValueEqual(nil, nil) {
		t.Error("expected both nil to be equal")
	}

	if depsValueEqual(nil, 42) {
		t.Error("expected nil and value to be unequal")
	}

	if depsValueEqual(42, nil) {
		t.Error("expected value and nil to be unequal")
	}

	if depsValueEqual(int(42), int64(42)) {
		t.Error("expected different types to be unequal")
	}

	if !depsValueEqual(42, 42) {
		t.Error("expected same int values to be equal")
	}

	if !depsValueEqual("hello", "hello") {
		t.Error("expected same string values to be equal")
	}

	if depsValueEqual(42, 43) {
		t.Error("expected different int values to be unequal")
	}
}

// TestDepsEqual_EmptyDeps tests empty dependency arrays
func TestDepsEqual_EmptyDeps(t *testing.T) {
	if !depsEqual([]any{}, []any{}) {
		t.Error("expected empty deps to be equal")
	}

	if !depsEqual(nil, nil) {
		t.Error("expected nil deps to be equal")
	}

	if depsEqual([]any{}, []any{1}) {
		t.Error("expected empty and non-empty to be unequal")
	}

	if depsEqual(nil, []any{1}) {
		t.Error("expected nil and non-empty to be unequal")
	}
}
