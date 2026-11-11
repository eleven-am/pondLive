package reactive

import (
	"fmt"
	"reflect"
	"testing"
)

func TestDiamondRunsEffectOnce(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 1)
	b := NewComputed(o, func() int { return a.Get() + 1 })
	c := NewComputed(o, func() int { return a.Get() * 10 })
	d := NewComputed(o, func() int { return b.Get() + c.Get() })
	var seen []int
	NewEffect(o, KindRender, func() { seen = append(seen, d.Get()) })
	if !reflect.DeepEqual(seen, []int{12}) {
		t.Fatalf("initial %v", seen)
	}
	a.Set(2)
	flush(t, g)
	a.Set(3)
	flush(t, g)
	if !reflect.DeepEqual(seen, []int{12, 23, 34}) {
		t.Fatalf("effect saw %v, want one run per write with consistent values", seen)
	}
}

func TestDiamondTwoWritesOneFlushRunsOnce(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 1)
	b := NewSignal(g, 1)
	sum := NewComputed(o, func() int { return a.Get() + b.Get() })
	runs := 0
	NewEffect(o, KindRender, func() { runs++; sum.Get() })
	a.Set(5)
	b.Set(7)
	flush(t, g)
	if runs != 2 {
		t.Fatalf("runs=%d, want 2 (initial plus one)", runs)
	}
}

func TestChain(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 1)
	prev := Readable[int](a)
	for i := 0; i < 50; i++ {
		p := prev
		prev = NewComputed(o, func() int { return p.Get() + 1 })
	}
	var got int
	NewEffect(o, KindRender, func() { got = prev.Get() })
	if got != 51 {
		t.Fatalf("got %d", got)
	}
	a.Set(10)
	flush(t, g)
	if got != 60 {
		t.Fatalf("got %d", got)
	}
}

func TestDynamicDependenciesUnlinkUnused(t *testing.T) {
	g, o := setup()
	flag := NewSignal(g, true)
	left := NewSignal(g, "l")
	right := NewSignal(g, "r")
	var got string
	runs := 0
	NewEffect(o, KindRender, func() {
		runs++
		if flag.Get() {
			got = left.Get()
		} else {
			got = right.Get()
		}
	})
	if left.targets == nil || right.targets != nil {
		t.Fatal("only the taken branch may be linked")
	}
	flag.Set(false)
	flush(t, g)
	if got != "r" || left.targets != nil || right.targets == nil {
		t.Fatalf("branch switch must unlink the dropped source: got=%q", got)
	}
	left.Set("l2")
	flush(t, g)
	if runs != 2 {
		t.Fatalf("write to an unlinked source ran the effect: runs=%d", runs)
	}
}

func TestLazyComputed(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 1)
	evals := 0
	c := NewComputed(o, func() int { evals++; return a.Get() * 2 })
	if evals != 0 {
		t.Fatal("computed ran before it was read")
	}
	a.Set(2)
	a.Set(3)
	if evals != 0 {
		t.Fatal("unread computed ran on write")
	}
	if c.Get() != 6 || evals != 1 {
		t.Fatalf("evals=%d", evals)
	}
	if c.Get() != 6 || evals != 1 {
		t.Fatal("clean computed recomputed")
	}
	if a.targets != nil {
		t.Fatal("computed with no subscribers must not be subscribed to its sources")
	}
	a.Set(4)
	if c.Peek() != 8 || evals != 2 {
		t.Fatalf("evals=%d", evals)
	}
}

func TestComputedEqualityPrunesDownstream(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 1)
	parity := NewComputed(o, func() int { return a.Get() % 2 })
	runs := 0
	NewEffect(o, KindRender, func() { runs++; parity.Get() })
	a.Set(3)
	flush(t, g)
	if runs != 1 {
		t.Fatalf("effect re-ran though the computed value was equal: runs=%d", runs)
	}
	a.Set(4)
	flush(t, g)
	if runs != 2 {
		t.Fatalf("runs=%d", runs)
	}
}

type named string

type pair struct {
	a int
	b string
}

func TestEqualityPerKind(t *testing.T) {
	g, _ := setup()
	changed := func(set func(), s *source) bool {
		before := s.version
		set()
		return s.version != before
	}
	si := NewSignal(g, 1)
	if changed(func() { si.Set(1) }, &si.source) || !changed(func() { si.Set(2) }, &si.source) {
		t.Fatal("int")
	}
	ss := NewSignal(g, "a")
	if changed(func() { ss.Set("a") }, &ss.source) || !changed(func() { ss.Set("b") }, &ss.source) {
		t.Fatal("string")
	}
	sf := NewSignal(g, 1.5)
	if changed(func() { sf.Set(1.5) }, &sf.source) {
		t.Fatal("float")
	}
	sb := NewSignal(g, false)
	if changed(func() { sb.Set(false) }, &sb.source) || !changed(func() { sb.Set(true) }, &sb.source) {
		t.Fatal("bool")
	}
	sn := NewSignal(g, named("x"))
	if changed(func() { sn.Set("x") }, &sn.source) || !changed(func() { sn.Set("y") }, &sn.source) {
		t.Fatal("named string")
	}
	sp := NewSignal(g, pair{1, "a"})
	if changed(func() { sp.Set(pair{1, "a"}) }, &sp.source) || !changed(func() { sp.Set(pair{2, "a"}) }, &sp.source) {
		t.Fatal("struct")
	}
	ptr := &pair{}
	spp := NewSignal(g, ptr)
	if changed(func() { spp.Set(ptr) }, &spp.source) || !changed(func() { spp.Set(&pair{}) }, &spp.source) {
		t.Fatal("pointer")
	}
	sl := NewSignal(g, []int{1})
	if !changed(func() { sl.Set(sl.Peek()) }, &sl.source) {
		t.Fatal("slices are never equal so every write notifies")
	}
	sa := NewSignal(g, map[string]int{})
	if !changed(func() { sa.Set(nil) }, &sa.source) {
		t.Fatal("maps never equal")
	}
	var x any = 1
	sx := NewSignal(g, x)
	if changed(func() { sx.Set(1) }, &sx.source) || !changed(func() { sx.Set(2) }, &sx.source) {
		t.Fatal("interface with comparable dynamic value")
	}
	sx.Set([]int{1})
	if !changed(func() { sx.Set([]int{1}) }, &sx.source) {
		t.Fatal("interface holding a slice must not panic and never equals")
	}
	se := NewSignal(g, []int{1}, WithEqual(func(a, b []int) bool { return reflect.DeepEqual(a, b) }))
	if changed(func() { se.Set([]int{1}) }, &se.source) || !changed(func() { se.Set([]int{2}) }, &se.source) {
		t.Fatal("WithEqual")
	}
}

func TestBasicEqualAllocatesNothing(t *testing.T) {
	g, _ := setup()
	s := NewSignal(g, 1)
	if n := testing.AllocsPerRun(100, func() { s.Set(1) }); n != 0 {
		t.Fatalf("allocs=%v", n)
	}
	str := NewSignal(g, "x")
	if n := testing.AllocsPerRun(100, func() { str.Set("x") }); n != 0 {
		t.Fatalf("allocs=%v", n)
	}
}

func TestComputedWithEqualOption(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 1)
	c := NewComputed(o, func() []int { return []int{a.Get() % 2} },
		WithEqual(func(x, y []int) bool { return reflect.DeepEqual(x, y) }))
	runs := 0
	NewEffect(o, KindRender, func() { runs++; c.Get() })
	a.Set(3)
	flush(t, g)
	if runs != 1 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestComputedPanicLeavesGraphUsable(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 0)
	c := NewComputed(o, func() int {
		if a.Get() == 1 {
			panic("boom")
		}
		return a.Get()
	})
	if c.Get() != 0 {
		t.Fatal()
	}
	a.Set(1)
	func() {
		defer func() { recover() }()
		c.Get()
	}()
	if g.computing != 0 || g.observer != nil {
		t.Fatal("graph state leaked after computed panic")
	}
	a.Set(2)
	if c.Get() != 2 {
		t.Fatal("computed must recompute after a panic")
	}
}

func TestEffectPanicLeavesGraphUsable(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 0)
	NewEffect(o, KindRender, func() {
		if a.Get() == 1 {
			panic("boom")
		}
	})
	a.Set(1)
	func() {
		defer func() { recover() }()
		g.RunQueue(QueueRender)
	}()
	if g.observer != nil || g.inBody {
		t.Fatal("graph state leaked")
	}
	a.Set(2)
	flush(t, g)
}

func TestUpdate(t *testing.T) {
	g, _ := setup()
	s := NewSignal(g, 1)
	s.Update(func(v int) int { return v + 1 })
	s.Update(func(v int) int { return v + 1 })
	if s.Peek() != 3 {
		t.Fatal(s.Peek())
	}
}

func TestSteadyRerunAllocatesNothing(t *testing.T) {
	g, o := setup()
	a := NewSignal(g, 0)
	b := NewSignal(g, 0)
	flag := NewSignal(g, true)
	c := NewComputed(o, func() int { return a.Get() + 1 })
	sink := 0
	NewEffect(o, KindRender, func() {
		if flag.Get() {
			sink += c.Get()
		} else {
			sink += b.Get()
		}
	})
	i := 0
	n := testing.AllocsPerRun(200, func() {
		i++
		a.Set(i)
		if err := g.RunQueue(QueueRender); err != nil {
			t.Fatal(err)
		}
	})
	if n != 0 {
		t.Fatalf("steady-state re-run allocated %v", n)
	}
	n = testing.AllocsPerRun(200, func() {
		i++
		flag.Set(i%2 == 0)
		if err := g.RunQueue(QueueRender); err != nil {
			t.Fatal(err)
		}
	})
	if n > 1 {
		t.Fatalf("branch-toggling re-run allocated %v, want at most the one new link", n)
	}
	_ = sink
}

func TestSerializeGuard(t *testing.T) {
	g, o := setup()
	s := NewSignal(g, 1)
	g.Serialize(func() {
		mustPanic(t, WriteInSerialize, func() { s.Set(2) })
		mustPanic(t, WriteInSerialize, func() { s.Update(func(v int) int { return v }) })
		mustPanic(t, CreateInSerialize, func() { NewSignal(g, 1) })
		mustPanic(t, CreateInSerialize, func() { NewComputed(o, func() int { return 1 }) })
		mustPanic(t, CreateInSerialize, func() { NewEffect(o, KindRender, func() {}) })
		mustPanic(t, CreateInSerialize, func() { o.NewChild() })
		mustPanic(t, CreateInSerialize, func() { o.NewUnit() })
		mustPanic(t, CreateInSerialize, func() { o.OnCleanup(func() {}) })
	})
	if s.Peek() != 1 || g.Mode() != ModeMount {
		t.Fatal("serialize pass changed state or leaked its mode")
	}
	s.Set(2)
}

func TestDevErrors(t *testing.T) {
	g, o := setup()
	s := NewSignal(g, 1)
	mustPanic(t, WriteInComputed, func() {
		NewComputed(o, func() int { s.Set(2); return 0 }).Get()
	})
	if g.computing != 0 {
		t.Fatal("computing counter leaked")
	}
	g.RunBody(func() {
		mustPanic(t, WriteInBody, func() { s.Set(3) })
		mustPanic(t, GetInBody, func() { s.Get() })
		_ = s.Peek()
	})
	var self *Computed[int]
	self = NewComputed(o, func() int { return self.Get() })
	mustPanic(t, CycleDetected, func() { self.Get() })
	dead := g.NewRoot()
	dead.Dispose()
	mustPanic(t, NoLiveOwner, func() { NewEffect(dead, KindRender, func() {}) })
	mustPanic(t, NoLiveOwner, func() { NewComputed(dead, func() int { return 1 }) })
	mustPanic(t, NoLiveOwner, func() { NewEffect(nil, KindRender, func() {}) })
	if s.Peek() != 1 {
		t.Fatal("rejected writes must not apply")
	}
}

func TestBodyDoesNotBlockEffectsRunInline(t *testing.T) {
	g, o := setup()
	s := NewSignal(g, 1)
	g.RunBody(func() {
		NewEffect(o, KindRender, func() { s.Get() })
	})
}

func TestLenientViolationHandlerSkipsOperation(t *testing.T) {
	g, _ := setup()
	var got []Violation
	g.SetViolationHandler(func(e *DevError) { got = append(got, e.Violation) })
	s := NewSignal(g, 1)
	g.RunBody(func() {
		s.Set(2)
		if s.Get() != 1 {
			t.Fatal("rejected write applied")
		}
	})
	if !reflect.DeepEqual(got, []Violation{WriteInBody, GetInBody}) {
		t.Fatalf("%v", got)
	}
}

func TestBudgetRounds(t *testing.T) {
	g, o := setup()
	s := NewSignal(g, 0)
	runs := 0
	NewEffect(o, KindRender, func() { runs++; s.Set(s.Get() + 1) })
	s.Set(100)
	err := g.RunQueue(QueueRender)
	be, ok := err.(*BudgetError)
	if !ok || be.Kind != BudgetRounds || be.Owner != o {
		t.Fatalf("err=%v", err)
	}
	if runs > MaxRenderIterations+2 {
		t.Fatalf("runs=%d", runs)
	}
}

func TestBudgetCascadePasses(t *testing.T) {
	g, o := setup()
	trigger := NewSignal(g, 0)
	NewEffect(o, KindRender, func() { trigger.Get() })
	passes := 0
	NewEffect(o, KindWatch, func() {
		trigger.Get()
	})
	err := g.Flush(FlushHooks{
		Drain: func() {},
		Publish: func() {
			passes++
			trigger.Set(trigger.Peek() + 1)
		},
	})
	be, ok := err.(*BudgetError)
	if !ok || be.Kind != BudgetPasses {
		t.Fatalf("err=%v", err)
	}
	if passes != CascadeBudget {
		t.Fatalf("passes=%d, want %d", passes, CascadeBudget)
	}
}

func TestFlushSettlesAndOrdersPhases(t *testing.T) {
	g, o := setup()
	s := NewSignal(g, 0)
	var order []string
	NewEffect(o, KindRender, func() { s.Get(); order = append(order, "render") })
	NewEffect(o, KindWatch, func() { s.Get(); order = append(order, "watch") })
	order = nil
	s.Set(1)
	err := g.Flush(FlushHooks{Publish: func() { order = append(order, "publish") }})
	if err != nil || !reflect.DeepEqual(order, []string{"render", "publish", "watch"}) {
		t.Fatalf("%v %v", err, order)
	}
	order = nil
	s.Set(2)
	g.Flush(FlushHooks{Publish: func() { order = append(order, "publish") }, Deferred: func() bool { return true }})
	if !reflect.DeepEqual(order, []string{"render", "publish"}) || !g.Pending(QueueWatch) {
		t.Fatalf("deferred flush must skip watches: %v", order)
	}
}

func TestWatchRewriteTriggersAnotherPass(t *testing.T) {
	g, o := setup()
	src := NewSignal(g, 0)
	mirror := NewSignal(g, 0)
	seen := -1
	NewEffect(o, KindRender, func() { seen = mirror.Get() })
	NewEffect(o, KindWatch, func() { mirror.Set(src.Get() * 2) })
	src.Set(4)
	if err := g.Flush(FlushHooks{}); err != nil {
		t.Fatal(err)
	}
	if seen != 8 {
		t.Fatalf("seen=%d", seen)
	}
}

func TestPendingCountsOnlyLiveEntries(t *testing.T) {
	g, o := setup()
	s := NewSignal(g, 0)
	e := NewEffect(o, KindRender, func() { s.Get() })
	s.Set(1)
	if !g.Pending(QueueRender) {
		t.Fatal()
	}
	e.Dispose()
	if g.Pending(QueueRender) {
		t.Fatal("a disposed queued effect must not count as pending")
	}
	if err := g.RunQueue(QueueRender); err != nil || g.Pending(QueueRender) {
		t.Fatal()
	}
	NewEffect(o, KindRender, func() { s.Get() })
	s.Set(2)
	if !g.Pending(QueueRender) {
		t.Fatal("popping a disposed entry must not skew the pending count")
	}
}

func TestManyKindsOfEffectsCoexist(t *testing.T) {
	g, o := setup()
	s := NewSignal(g, 0)
	var order []string
	for _, k := range []Kind{KindWatch, KindSink, KindRender} {
		k := k
		NewEffect(o, k, func() { s.Get(); order = append(order, fmt.Sprint(k)) })
	}
	order = nil
	s.Set(1)
	flush(t, g)
	if !reflect.DeepEqual(order, []string{"0", "1", "2"}) {
		t.Fatalf("%v", order)
	}
}

func TestComputedUnsubscribesWhenLastTargetGoes(t *testing.T) {
	g, root := setup()
	a := NewSignal(g, 1)
	evals := 0
	c := NewComputed(root, func() int { evals++; return a.Get() + 1 })
	unit := root.NewUnit()
	NewEffect(unit, KindRender, func() { c.Get() })
	if a.targets == nil {
		t.Fatal("computed with a target must be subscribed to its source")
	}
	unit.Dispose()
	if a.targets != nil {
		t.Fatal("computed kept its source subscription after its last target was disposed")
	}
	a.Set(5)
	if c.Get() != 6 {
		t.Fatal("computed must still be correct when read again")
	}
	if a.targets != nil {
		t.Fatal("an untracked read must not subscribe")
	}
}

func TestAbandonClearsQueuesAndReturnsOwners(t *testing.T) {
	g := NewGraph()
	root := g.NewRoot()
	src := NewSignal(g, 0)
	runs := 0
	NewEffect(root, KindRender, func() { src.Get(); runs++ })
	NewEffect(root, KindWatch, func() { src.Get(); runs++ })
	src.Set(1)
	if !g.Pending(QueueRender) || !g.Pending(QueueWatch) {
		t.Fatal("both queues should be pending")
	}

	owners := g.Abandon()

	if len(owners) != 1 || owners[0] != root {
		t.Fatalf("owners = %v", owners)
	}
	if g.Pending(QueueRender) || g.Pending(QueueWatch) {
		t.Fatal("queues should be empty")
	}
	before := runs
	if err := g.Flush(FlushHooks{}); err != nil || runs != before {
		t.Fatalf("abandoned effects ran: runs %d -> %d err %v", before, runs, err)
	}
	src.Set(2)
	if !g.Pending(QueueRender) {
		t.Fatal("abandoned effects must be schedulable again")
	}
}

func TestDropUnitDetachesAStaticUnit(t *testing.T) {
	g := NewGraph()
	root := g.NewRoot()
	static := root.NewUnit()
	static.Base, static.N = 5, 3
	live := root.NewUnit()
	live.OnCleanup(func() {})

	if !static.DropUnit() {
		t.Fatal("a unit without cells, computations or cleanups must drop even with an id block")
	}
	if live.DropUnit() {
		t.Fatal("a unit with a cleanup must not drop")
	}
	if root.units != live || live.nextU != nil {
		t.Fatalf("unit list not repaired: %v", root.units)
	}
}
