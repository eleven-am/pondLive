package runtime

import (
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

// TestUseEffect_RapidStateChanges tests cleanup timing with rapid state updates
func TestUseEffect_RapidStateChanges(t *testing.T) {
	effectCount := 0
	cleanupCount := 0
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, set := UseState(ctx, 0)
		setState = set

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return func() {
				cleanupCount++
			}
		}, state())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for i := 1; i <= 10; i++ {
		setState(i)
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}

	if effectCount != 11 {
		t.Errorf("expected 11 effect runs, got %d", effectCount)
	}

	if cleanupCount != 10 {
		t.Errorf("expected 10 cleanups, got %d", cleanupCount)
	}
}

// TestUseEffect_MultipleEffectsInterdependent tests effects that depend on each other
func TestUseEffect_MultipleEffectsInterdependent(t *testing.T) {
	var order []string
	var setA func(int)
	var setB func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		a, setALocal := UseState(ctx, 0)
		b, setBLocal := UseState(ctx, 0)
		setA = setALocal
		setB = setBLocal

		UseEffect(ctx, func() Cleanup {
			order = append(order, "effect1-run")
			return func() {
				order = append(order, "effect1-cleanup")
			}
		}, a())

		UseEffect(ctx, func() Cleanup {
			order = append(order, "effect2-run")
			return func() {
				order = append(order, "effect2-cleanup")
			}
		}, b())

		UseEffect(ctx, func() Cleanup {
			order = append(order, "effect3-run")
			return func() {
				order = append(order, "effect3-cleanup")
			}
		}, a(), b())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expected := []string{"effect1-run", "effect2-run", "effect3-run"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(order), order)
	}

	order = nil
	setA(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expected = []string{"effect1-cleanup", "effect3-cleanup", "effect1-run", "effect3-run"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(order), order)
	}
	for i, exp := range expected {
		if order[i] != exp {
			t.Errorf("at index %d: expected %s, got %s", i, exp, order[i])
		}
	}

	order = nil
	setB(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expected = []string{"effect2-cleanup", "effect3-cleanup", "effect2-run", "effect3-run"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(order), order)
	}
}

// TestUseEffect_NoOpEffect tests effects that return nil cleanup
func TestUseEffect_NoOpEffect(t *testing.T) {
	effectRuns := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		UseEffect(ctx, func() Cleanup {
			effectRuns++
			return nil
		})

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	for i := 0; i < 5; i++ {
		sess.markDirty(sess.root)
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}

	if effectRuns != 5 {
		t.Errorf("expected 5 effect runs, got %d", effectRuns)
	}
}

// TestUseEffect_StateUpdateInEffect tests effects that update state
func TestUseEffect_StateUpdateInEffect(t *testing.T) {
	renderCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		count, set := UseState(ctx, 0)

		UseEffect(ctx, func() Cleanup {
			if count() == 0 {

				set(1)
			}
			return nil
		})

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 1 {
		t.Errorf("expected 1 initial render, got %d", renderCount)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	if renderCount != 2 {
		t.Errorf("expected 2 renders after second flush, got %d", renderCount)
	}
}

// TestUseEffect_CleanupBeforeReRun tests cleanup runs before effect re-runs
func TestUseEffect_CleanupBeforeReRun(t *testing.T) {
	var events []string
	var setDep func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		dep, set := UseState(ctx, 0)
		setDep = set

		UseEffect(ctx, func() Cleanup {
			events = append(events, "setup")
			return func() {
				events = append(events, "cleanup")
			}
		}, dep())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if len(events) != 1 || events[0] != "setup" {
		t.Errorf("expected initial setup, got %v", events)
	}

	events = nil
	setDep(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expected := []string{"cleanup", "setup"}
	if len(events) != len(expected) {
		t.Fatalf("expected %v, got %v", expected, events)
	}
	for i, exp := range expected {
		if events[i] != exp {
			t.Errorf("at index %d: expected %s, got %s", i, exp, events[i])
		}
	}
}

// TestUseEffect_FunctionDependency tests effect with function dependency
func TestUseEffect_FunctionDependency(t *testing.T) {
	effectCount := 0

	fn1 := func() int { return 1 }
	fn2 := func() int { return 2 }

	var setFn func(func() int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		fn, set := UseState(ctx, fn1)
		setFn = set

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return nil
		}, fn())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected 1 effect run, got %d", effectCount)
	}

	setFn(fn1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected still 1 effect run (same function), got %d", effectCount)
	}

	setFn(fn2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 2 {
		t.Errorf("expected 2 effect runs (different function), got %d", effectCount)
	}
}

// TestUseEffect_ChannelDependency tests effect with channel dependency
func TestUseEffect_ChannelDependency(t *testing.T) {
	effectCount := 0

	ch1 := make(chan int)
	ch2 := make(chan int)

	var setCh func(chan int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		ch, set := UseState(ctx, ch1)
		setCh = set

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return nil
		}, ch())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected 1 effect run, got %d", effectCount)
	}

	setCh(ch1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected still 1 effect run (same channel), got %d", effectCount)
	}

	setCh(ch2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 2 {
		t.Errorf("expected 2 effect runs (different channel), got %d", effectCount)
	}
}

// TestUseEffect_MapDependency tests effect with map dependency
func TestUseEffect_MapDependency(t *testing.T) {
	effectCount := 0

	map1 := map[string]int{"a": 1}
	map2 := map[string]int{"a": 1}
	map3 := map[string]int{"a": 2, "b": 3}

	var setMap func(map[string]int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		m, set := UseState(ctx, map1)
		setMap = set

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return nil
		}, m())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected 1 effect run, got %d", effectCount)
	}

	setMap(map1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected still 1 effect run (same map), got %d", effectCount)
	}

	setMap(map2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected still 1 effect run (same contents), got %d", effectCount)
	}

	setMap(map3)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 2 {
		t.Errorf("expected 2 effect runs (different contents), got %d", effectCount)
	}
}

// TestUseMemo_FunctionDependency tests memo with function dependency
func TestUseMemo_FunctionDependency(t *testing.T) {
	computeCount := 0

	fn1 := func() int { return 1 }
	fn2 := func() int { return 2 }

	var setFn func(func() int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		fn, set := UseState(ctx, fn1)
		setFn = set

		_ = UseMemo(ctx, func() int {
			computeCount++
			return fn()()
		}, fn())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if computeCount != 1 {
		t.Errorf("expected 1 compute, got %d", computeCount)
	}

	setFn(fn1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if computeCount != 1 {
		t.Errorf("expected still 1 compute (same function), got %d", computeCount)
	}

	setFn(fn2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if computeCount != 2 {
		t.Errorf("expected 2 computes (different function), got %d", computeCount)
	}
}

// TestUseMemo_ExpensiveComputation tests memoization of expensive computation
func TestUseMemo_ExpensiveComputation(t *testing.T) {
	computeCount := 0
	var setA func(int)
	var setB func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		a, setALocal := UseState(ctx, 1)
		_, setBLocal := UseState(ctx, 100)
		setA = setALocal
		setB = setBLocal

		result := UseMemo(ctx, func() int {
			computeCount++
			sum := 0
			for i := 0; i < 1000; i++ {
				sum += a()
			}
			return sum
		}, a())

		_ = result

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if computeCount != 1 {
		t.Errorf("expected 1 compute, got %d", computeCount)
	}

	setB(200)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if computeCount != 1 {
		t.Errorf("expected still 1 compute (b changed but not in deps), got %d", computeCount)
	}

	setA(2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if computeCount != 2 {
		t.Errorf("expected 2 computes (a changed), got %d", computeCount)
	}
}

// TestUseMemo_NoDeps tests memo without dependencies (computes once, caches forever)
func TestUseMemo_NoDeps(t *testing.T) {
	computeCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		_ = UseMemo(ctx, func() int {
			computeCount++
			return 42
		})

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	for i := 0; i < 5; i++ {
		sess.markDirty(sess.root)
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}

	if computeCount != 1 {
		t.Errorf("expected 1 compute (cached), got %d", computeCount)
	}
}

// TestUseEffect_AsyncCleanup tests that cleanup doesn't interfere with async
func TestUseEffect_AsyncCleanup(t *testing.T) {
	var cleanupWg sync.WaitGroup
	cleanupRan := false
	var setDep func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		dep, set := UseState(ctx, 0)
		setDep = set

		UseEffect(ctx, func() Cleanup {
			return func() {
				cleanupWg.Add(1)
				go func() {
					time.Sleep(10 * time.Millisecond)
					cleanupRan = true
					cleanupWg.Done()
				}()
			}
		}, dep())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	setDep(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	cleanupWg.Wait()

	if !cleanupRan {
		t.Error("expected async cleanup to run")
	}
}

// TestUseEffect_ZeroDepsArray tests empty dependency array (run once)
func TestUseEffect_ZeroDepsArray(t *testing.T) {
	effectCount := 0
	var triggerRender func()

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		count, setCount := UseState(ctx, 0)
		triggerRender = func() { setCount(count() + 1) }

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return nil
		}, 42)

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectCount != 1 {
		t.Errorf("expected 1 effect run, got %d", effectCount)
	}

	for i := 0; i < 5; i++ {
		triggerRender()
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}

	if effectCount != 1 {
		t.Errorf("expected still 1 effect run (deps never changed), got %d", effectCount)
	}
}
