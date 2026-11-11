package reactive

import (
	"context"
	"reflect"
	"runtime"
	"testing"
	"weak"
)

func TestNoEditAfterRemoveInSameFlush(t *testing.T) {
	g, root := setup()
	gate := NewSignal(g, true)
	text := NewSignal(g, "a")
	var edits []string
	var child *Owner
	NewEffect(root, KindRender, func() {
		if gate.Get() {
			if child == nil {
				child = root.NewUnit()
				NewEffect(child, KindRender, func() { edits = append(edits, "t:"+text.Get()) })
			}
			return
		}
		if child != nil {
			child.Dispose()
			child = nil
		}
	})
	edits = nil
	gate.Set(false)
	text.Set("b")
	flush(t, g)
	if len(edits) != 0 {
		t.Fatalf("edit emitted for a node disposed in the same flush: %v", edits)
	}
	if g.Pending(QueueRender) {
		t.Fatal("dead entry still counted")
	}
}

func TestParentRunsBeforeChild(t *testing.T) {
	g, root := setup()
	s := NewSignal(g, 0)
	var order []string
	kid := root.NewChild()
	grand := kid.NewChild()
	NewEffect(grand, KindRender, func() { s.Get(); order = append(order, "grand") })
	NewEffect(kid, KindRender, func() { s.Get(); order = append(order, "kid") })
	NewEffect(root, KindRender, func() { s.Get(); order = append(order, "root") })
	order = nil
	s.Set(1)
	flush(t, g)
	if !reflect.DeepEqual(order, []string{"root", "kid", "grand"}) {
		t.Fatalf("order %v", order)
	}
}

func TestSiblingsRunInMountOrder(t *testing.T) {
	g, root := setup()
	s := NewSignal(g, 0)
	var order []int
	for i := 0; i < 5; i++ {
		i := i
		NewEffect(root.NewChild(), KindRender, func() { s.Get(); order = append(order, i) })
	}
	order = nil
	s.Set(1)
	flush(t, g)
	if !reflect.DeepEqual(order, []int{0, 1, 2, 3, 4}) {
		t.Fatalf("%v", order)
	}
}

func TestHeapOrderWhenInsertedMidFlush(t *testing.T) {
	g, root := setup()
	trigger := NewSignal(g, 0)
	shallow := NewSignal(g, 0)
	deep := NewSignal(g, 0)
	kid := root.NewChild()
	grand := kid.NewChild()
	var order []string
	NewEffect(grand, KindRender, func() { deep.Get(); order = append(order, "grand") })
	NewEffect(kid, KindRender, func() { shallow.Get(); order = append(order, "kid") })
	NewEffect(root, KindRender, func() {
		if trigger.Get() > 0 {
			shallow.Set(shallow.Peek() + 1)
		}
		order = append(order, "root")
	})
	order = nil
	deep.Set(1)
	trigger.Set(1)
	flush(t, g)
	if !reflect.DeepEqual(order, []string{"root", "kid", "grand"}) {
		t.Fatalf("a node queued mid-flush must slot in by (depth, seq): %v", order)
	}
}

func TestDisposeDuringFlushSkipsQueuedSibling(t *testing.T) {
	g, root := setup()
	s := NewSignal(g, 0)
	var second *Effect
	ran := false
	NewEffect(root, KindRender, func() {
		if s.Get() > 0 && second != nil {
			second.Dispose()
		}
	})
	second = NewEffect(root, KindRender, func() { s.Get(); ran = true })
	ran = false
	s.Set(1)
	flush(t, g)
	if ran {
		t.Fatal("disposed effect ran")
	}
}

func TestSelfDisposingEffect(t *testing.T) {
	g, root := setup()
	s := NewSignal(g, 0)
	var e *Effect
	runs := 0
	e = NewEffect(root, KindRender, func() {
		runs++
		if s.Get() > 0 {
			e.Dispose()
		}
	})
	s.Set(1)
	flush(t, g)
	s.Set(2)
	flush(t, g)
	if runs != 2 || s.targets != nil {
		t.Fatalf("runs=%d targets=%v", runs, s.targets)
	}
}

func TestDisposeUnlinksEverything(t *testing.T) {
	g, root := setup()
	a := NewSignal(g, 1)
	b := NewSignal(g, 1)
	kid := root.NewChild()
	c := NewComputed(kid, func() int { return a.Get() + b.Get() })
	NewEffect(kid, KindRender, func() { c.Get() })
	NewEffect(kid, KindWatch, func() { a.Get() })
	if a.targets == nil || b.targets == nil {
		t.Fatal("setup")
	}
	root.Dispose()
	if a.targets != nil || b.targets != nil {
		t.Fatal("dispose left subscriptions on the sources")
	}
	if c.n.srcHead != nil || c.n.targets != nil {
		t.Fatal("computed retained links")
	}
	a.Set(2)
	flush(t, g)
}

func TestCleanupOrderLIFOChildrenFirst(t *testing.T) {
	_, root := setup()
	var order []string
	kid := root.NewChild()
	unit := root.NewUnit()
	root.OnCleanup(func() { order = append(order, "root1") })
	root.OnCleanup(func() { order = append(order, "root2") })
	kid.OnCleanup(func() { order = append(order, "kid") })
	unit.OnCleanup(func() { order = append(order, "unit") })
	root.Dispose()
	if !reflect.DeepEqual(order, []string{"kid", "unit", "root2", "root1"}) {
		t.Fatalf("%v", order)
	}
	root.Dispose()
	if len(order) != 4 {
		t.Fatal("dispose is not idempotent")
	}
}

func TestCleanupPanicDoesNotSkipOthers(t *testing.T) {
	g, root := setup()
	var order []string
	var caught []any
	g.SetCleanupPanicHandler(func(o *Owner, v any) { caught = append(caught, v) })
	root.OnCleanup(func() { order = append(order, "a") })
	root.OnCleanup(func() { panic("bad") })
	root.OnCleanup(func() { order = append(order, "c") })
	root.Dispose()
	if !reflect.DeepEqual(order, []string{"c", "a"}) || !reflect.DeepEqual(caught, []any{"bad"}) {
		t.Fatalf("%v %v", order, caught)
	}
}

func TestCleanupPanicWithoutHandlerRepanicsAfterAll(t *testing.T) {
	_, root := setup()
	ran := false
	root.OnCleanup(func() { ran = true })
	root.OnCleanup(func() { panic("bad") })
	func() {
		defer func() {
			if recover() != "bad" {
				t.Fatal("panic lost")
			}
		}()
		root.Dispose()
	}()
	if !ran || !root.Disposed() {
		t.Fatal("cleanups skipped after a panic")
	}
}

func TestUnitDisposeDetachesFromParent(t *testing.T) {
	_, root := setup()
	a, b, c := root.NewUnit(), root.NewUnit(), root.NewUnit()
	b.Dispose()
	if root.units != c || c.nextU != a || a.prevU != c || a.nextU != nil {
		t.Fatal("list corrupted")
	}
	c.Dispose()
	a.Dispose()
	if root.units != nil {
		t.Fatal("units retained")
	}
}

func TestOwnerContextCancelledOnDispose(t *testing.T) {
	_, root := setup()
	kid := root.NewChild()
	ctx := kid.Context()
	if ctx.Err() != nil {
		t.Fatal()
	}
	root.Dispose()
	if ctx.Err() != context.Canceled || kid.Context().Err() == nil {
		t.Fatal("context not cancelled")
	}
}

func TestContextLookupWalksParents(t *testing.T) {
	_, root := setup()
	type key struct{}
	k := &key{}
	root.SetContext(k, 1)
	kid := root.NewChild()
	grand := kid.NewUnit()
	kid.SetContext(k, 2)
	if v, ok := grand.Lookup(k); !ok || v != 2 {
		t.Fatal(v, ok)
	}
	if v, _ := root.Lookup(k); v != 1 {
		t.Fatal(v)
	}
	if _, ok := root.Lookup(&key{}); ok {
		t.Fatal("unprovided key found")
	}
}

func TestCellsMountThenSerialize(t *testing.T) {
	g, root := setup()
	o := root.NewChild()
	type A struct{ n int }
	type B struct{ s string }
	a := Cell(o, func() *A { return &A{1} })
	b := Cell(o, func() *B { return &B{"x"} })
	o.Seal()
	mustPanic(t, SealedOwner, func() { Cell(o, func() *A { return &A{} }) })
	g.Serialize(func() {
		o.ResetCursor()
		if Cell[A](o, nil) != a || Cell[B](o, nil) != b {
			t.Fatal("serialize must resolve the mounted cells")
		}
		if err := o.EndBody(); err != nil {
			t.Fatal(err)
		}
		o.ResetCursor()
		_ = Cell[A](o, nil)
		if err := o.EndBody(); err == nil {
			t.Fatal("fewer cells must be a determinism error")
		}
	})
}

func TestSerializeCellMismatches(t *testing.T) {
	g, root := setup()
	o := root.NewChild()
	type A struct{}
	type B struct{}
	Cell(o, func() *A { return &A{} })
	expect := func(fn func()) {
		t.Helper()
		defer func() {
			if _, ok := recover().(*DeterminismError); !ok {
				t.Fatal("expected a determinism error")
			}
		}()
		fn()
	}
	g.Serialize(func() {
		o.ResetCursor()
		expect(func() { Cell[B](o, nil) })
		o.ResetCursor()
		Cell[A](o, nil)
		expect(func() { Cell[A](o, nil) })
	})
}

func TestTransientOwnerHasNoCells(t *testing.T) {
	g, root := setup()
	mustTransient := func() {
		defer func() {
			if recover() == nil {
				t.Fatal("transient owners are for serialize passes only")
			}
		}()
		g.NewTransient(root)
	}
	mustTransient()
	type A struct{}
	g.Serialize(func() {
		tr := g.NewTransient(root)
		defer func() {
			if _, ok := recover().(*DeterminismError); !ok {
				t.Fatal("a transient owner must reject hooks")
			}
		}()
		Cell[A](tr, nil)
	})
}

func TestOwnerDropping(t *testing.T) {
	_, root := setup()
	static := root.NewChild()
	if !static.Droppable() || !static.Drop() || root.Kids()[0] != nil {
		t.Fatal("a bare owner must drop and leave a nil slot")
	}
	if static.Drop() {
		t.Fatal("double drop")
	}
	type A struct{}
	for name, mark := range map[string]func(o *Owner){
		"cell":    func(o *Owner) { Cell(o, func() *A { return &A{} }) },
		"block":   func(o *Owner) { o.N = 2 },
		"region":  func(o *Owner) { o.Regions = append(o.Regions, 1) },
		"cleanup": func(o *Owner) { o.OnCleanup(func() {}) },
		"context": func(o *Owner) { o.SetContext(1, 1) },
		"unit":    func(o *Owner) { o.NewUnit() },
		"effect":  func(o *Owner) { NewEffect(o, KindWatch, func() {}) },
		"kid": func(o *Owner) {
			k := o.NewChild()
			Cell(k, func() *A { return &A{} })
		},
	} {
		o := root.NewChild()
		mark(o)
		if o.Droppable() || o.Drop() {
			t.Fatalf("%s: owner with retained state was dropped", name)
		}
	}
	parent := root.NewChild()
	parent.NewChild().Drop()
	if !parent.Droppable() {
		t.Fatal("an owner whose kids were all dropped is droppable")
	}
}

func TestOwnerLeakAfterDispose(t *testing.T) {
	g, root := setup()
	s := NewSignal(g, 0)
	var ow weak.Pointer[Owner]
	var cw weak.Pointer[Computed[int]]
	var ew weak.Pointer[Effect]
	func() {
		u := root.NewUnit()
		c := NewComputed(u, func() int { return s.Get() })
		e := NewEffect(u, KindRender, func() { c.Get() })
		ow, cw, ew = weak.Make(u), weak.Make(c), weak.Make(e)
		u.Dispose()
	}()
	for i := 0; i < 3; i++ {
		runtime.GC()
	}
	if ow.Value() != nil || cw.Value() != nil || ew.Value() != nil {
		t.Fatal("disposed owner, computed or effect is still reachable from the live graph")
	}
	if s.targets != nil {
		t.Fatal("signal still links to disposed nodes")
	}
}

func TestManyUnitsChurnLeavesNoResidue(t *testing.T) {
	g, root := setup()
	s := NewSignal(g, 0)
	for i := 0; i < 2000; i++ {
		u := root.NewUnit()
		NewEffect(u, KindRender, func() { s.Get() })
		c := NewComputed(u, func() int { return s.Get() })
		NewEffect(u, KindWatch, func() { c.Get() })
		if i%3 != 0 {
			u.Dispose()
		}
	}
	live := 0
	for u := root.units; u != nil; u = u.nextU {
		live++
	}
	links := 0
	for l := s.targets; l != nil; l = l.nextS {
		links++
	}
	if live != 667 || links != 667*2 {
		t.Fatalf("live=%d links=%d", live, links)
	}
	root.Dispose()
	if s.targets != nil || root.units != nil {
		t.Fatal("residue")
	}
}
