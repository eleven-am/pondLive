package runtime

import (
	"fmt"
	"runtime"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

// TestStress_ManyHooksInComponent tests component with many hooks
func TestStress_ManyHooksInComponent(t *testing.T) {
	hookCount := 100
	var setters []func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {

		for i := 0; i < hookCount; i++ {
			_, set := UseState(ctx, i)
			if len(setters) < hookCount {
				setters = append(setters, set)
			}
		}

		for i := 0; i < hookCount; i++ {
			UseRef(ctx, i)
		}

		for i := 0; i < hookCount; i++ {
			UseEffect(ctx, func() Cleanup {
				return nil
			}, i)
		}

		for i := 0; i < hookCount; i++ {
			UseMemo(ctx, func() int {
				return i * 2
			}, i)
		}

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	start := time.Now()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Initial render with %d hooks took %v", hookCount*4, elapsed)

	if len(setters) > 0 {
		setters[0](999)
		start = time.Now()
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush failed: %v", err)
		}
		elapsed = time.Since(start)
		t.Logf("Re-render with state update took %v", elapsed)
	}
}

// TestStress_DeepEffectChains tests effects that trigger other effects
func TestStress_DeepEffectChains(t *testing.T) {
	depth := 50
	effectCounts := make([]int, depth)
	var setters []func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {

		for i := 0; i < depth; i++ {
			val, set := UseState(ctx, 0)
			if len(setters) < depth {
				setters = append(setters, set)
			}

			idx := i
			UseEffect(ctx, func() Cleanup {
				effectCounts[idx]++
				return nil
			}, val())
		}

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for i := 0; i < depth; i++ {
		if i < len(setters) {
			setters[i](i + 1)
		}
	}

	start := time.Now()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Effect chain with depth %d took %v", depth, elapsed)

	for i, count := range effectCounts {
		if count < 2 {
			t.Errorf("effect %d: expected at least 2 runs, got %d", i, count)
		}
	}
}

// TestStress_LargeDependencyArrays tests hooks with many dependencies
func TestStress_LargeDependencyArrays(t *testing.T) {
	depCount := 100
	effectRuns := 0
	memoRuns := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {

		var deps []any
		for i := 0; i < depCount; i++ {
			val, _ := UseState(ctx, i)
			deps = append(deps, val())
		}

		UseEffect(ctx, func() Cleanup {
			effectRuns++
			return nil
		}, deps...)

		_ = UseMemo(ctx, func() int {
			memoRuns++
			return len(deps)
		}, deps...)

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	start := time.Now()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Render with %d dependencies took %v", depCount, elapsed)

	if effectRuns != 1 {
		t.Errorf("expected 1 effect run, got %d", effectRuns)
	}

	if memoRuns != 1 {
		t.Errorf("expected 1 memo run, got %d", memoRuns)
	}
}

// TestStress_RapidReRenders tests many rapid re-renders
func TestStress_RapidReRenders(t *testing.T) {
	renders := 1000
	var setState func(int)
	renderCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		_, set := UseState(ctx, 0)
		setState = set
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	start := time.Now()
	for i := 0; i < renders; i++ {
		setState(i)
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	t.Logf("%d rapid re-renders took %v (avg: %v)", renders, elapsed, elapsed/time.Duration(renders))

	if renderCount != renders {
		t.Errorf("expected %d renders, got %d", renders, renderCount)
	}
}

// TestStress_MemoryUsage tests memory with many components
func TestStress_MemoryUsage(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping memory test in short mode")
	}

	sessionCount := 1000

	runtime.GC()
	var m1 runtime.MemStats
	runtime.ReadMemStats(&m1)

	sessions := make([]*ComponentSession, sessionCount)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		UseState(ctx, 0)
		UseRef(ctx, 0)
		UseEffect(ctx, func() Cleanup { return nil }, 1)
		UseMemo(ctx, func() int { return 42 }, 1)
		return &dom.StructuredNode{Tag: "div"}
	}

	for i := 0; i < sessionCount; i++ {
		sessions[i] = NewSession(comp, struct{}{})
		sessions[i].SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
		if err := sessions[i].Flush(); err != nil {
			t.Fatalf("session %d flush failed: %v", i, err)
		}
	}

	runtime.GC()
	var m2 runtime.MemStats
	runtime.ReadMemStats(&m2)

	allocated := m2.Alloc - m1.Alloc
	perSession := allocated / uint64(sessionCount)

	t.Logf("Created %d sessions", sessionCount)
	t.Logf("Total allocated: %d bytes", allocated)
	t.Logf("Per session: %d bytes", perSession)

	sessions = nil
	runtime.GC()

	var m3 runtime.MemStats
	runtime.ReadMemStats(&m3)

	freed := m2.Alloc - m3.Alloc
	t.Logf("Freed after GC: %d bytes", freed)
}

// TestStress_EffectCleanupLoad tests many effect cleanups
func TestStress_EffectCleanupLoad(t *testing.T) {

	const maxEffects = 64
	effectCount := 500
	cleanupCount := 0
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, set := UseState(ctx, 0)
		setState = set

		for i := 0; i < effectCount; i++ {
			UseEffect(ctx, func() Cleanup {
				return func() {
					cleanupCount++
				}
			}, state())
		}

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	setState(1)

	start := time.Now()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Running %d effect cleanups took %v", cleanupCount, elapsed)

	if cleanupCount != maxEffects {
		t.Errorf("expected %d cleanups (max effects limit), got %d", maxEffects, cleanupCount)
	}
}

// TestStress_MemoRecomputationLoad tests expensive memo recomputation
func TestStress_MemoRecomputationLoad(t *testing.T) {
	iterations := 100
	computeCount := 0
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, set := UseState(ctx, 0)
		setState = set

		_ = UseMemo(ctx, func() int {
			computeCount++
			sum := 0
			for i := 0; i < 10000; i++ {
				sum += i
			}
			return sum + state()
		}, state())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	start := time.Now()
	for i := 0; i < iterations; i++ {
		setState(i)
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}
	elapsed := time.Since(start)

	t.Logf("%d memo recomputations took %v (avg: %v)", iterations, elapsed, elapsed/time.Duration(iterations))

	if computeCount != iterations {
		t.Errorf("expected %d computes, got %d", iterations, computeCount)
	}
}

// TestStress_ComplexDependencyPatterns tests complex dep patterns
func TestStress_ComplexDependencyPatterns(t *testing.T) {
	fn1 := func() int { return 1 }
	fn2 := func() int { return 2 }
	ch1 := make(chan int)
	ch2 := make(chan int)
	map1 := map[string]int{"a": 1}
	map2 := map[string]int{"a": 2}

	effectRuns := 0
	var setWhich func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		which, set := UseState(ctx, 0)
		setWhich = set

		switch which() {
		case 0:
			UseEffect(ctx, func() Cleanup {
				effectRuns++
				return nil
			}, fn1, ch1, map1)
		case 1:
			UseEffect(ctx, func() Cleanup {
				effectRuns++
				return nil
			}, fn2, ch1, map1)
		case 2:
			UseEffect(ctx, func() Cleanup {
				effectRuns++
				return nil
			}, fn1, ch2, map1)
		case 3:
			UseEffect(ctx, func() Cleanup {
				effectRuns++
				return nil
			}, fn1, ch1, map2)
		}

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	for i := 0; i < 4; i++ {
		if i > 0 {
			setWhich(i)
		}
		if err := sess.Flush(); err != nil {
			t.Fatalf("flush %d failed: %v", i, err)
		}
	}

	if effectRuns != 4 {
		t.Errorf("expected 4 effect runs, got %d", effectRuns)
	}
}

// TestStress_StateUpdateBatching tests batching of state updates
func TestStress_StateUpdateBatching(t *testing.T) {
	updates := 1000
	var setState func(int)
	renderCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		_, set := UseState(ctx, 0)
		setState = set
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for i := 0; i < updates; i++ {
		setState(i)
	}

	start := time.Now()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Batched %d updates in single flush: %v", updates, elapsed)

	if renderCount != 2 {
		t.Errorf("expected 2 renders (batched), got %d", renderCount)
	}
}

// TestStress_NestedComponents tests deep component hierarchy
func TestStress_NestedComponents(t *testing.T) {
	depth := 20
	renderCounts := make([]int, depth)

	var makeComp func(int) func(Ctx, struct{}) *dom.StructuredNode
	makeComp = func(level int) func(Ctx, struct{}) *dom.StructuredNode {
		return func(ctx Ctx, props struct{}) *dom.StructuredNode {
			renderCounts[level]++
			UseState(ctx, 0)
			UseEffect(ctx, func() Cleanup { return nil }, 1)

			children := []*dom.StructuredNode{}
			if level < depth-1 {

				childComp := Render(ctx, makeComp(level+1), struct{}{})
				children = append(children, &dom.StructuredNode{
					Tag:      "component",
					Children: []*dom.StructuredNode{childComp},
				})
			}

			return &dom.StructuredNode{
				Tag:      "div",
				Children: children,
			}
		}
	}

	sess := NewSession(makeComp(0), struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	start := time.Now()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Nested component tree (depth %d) rendered in %v", depth, elapsed)

	for i, count := range renderCounts {
		if count < 1 {
			t.Errorf("level %d: expected at least 1 render, got %d", i, count)
		}
	}
}

// TestStress_PatchGenerationLoad tests many patches
func TestStress_PatchGenerationLoad(t *testing.T) {
	elements := 100
	var setters []func(string)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		children := make([]*dom.StructuredNode, elements)
		for i := 0; i < elements; i++ {
			text, set := UseState(ctx, fmt.Sprintf("item-%d", i))
			if len(setters) < elements {
				setters = append(setters, set)
			}
			children[i] = &dom.StructuredNode{
				Tag:  "span",
				Text: text(),
			}
		}

		return &dom.StructuredNode{
			Tag:      "div",
			Children: children,
		}
	}

	patchCount := 0
	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error {
		patchCount += len(patches)
		return nil
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for i, set := range setters {
		set(fmt.Sprintf("updated-%d", i))
	}

	start := time.Now()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Generated %d patches for %d elements in %v", patchCount, elements, elapsed)

	if patchCount < elements {
		t.Errorf("expected at least %d patches, got %d", elements, patchCount)
	}
}

// TestStress_EffectExecutionOrder tests effect ordering under load
func TestStress_EffectExecutionOrder(t *testing.T) {
	effectCount := 100
	var order []int

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, _ := UseState(ctx, 0)

		for i := 0; i < effectCount; i++ {
			idx := i
			UseEffect(ctx, func() Cleanup {
				order = append(order, idx)
				return nil
			}, state())
		}

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	const maxEffects = 64
	if len(order) != maxEffects {
		t.Fatalf("expected %d effects (max limit), got %d", maxEffects, len(order))
	}

	for i := 0; i < maxEffects; i++ {
		if order[i] != i {
			t.Errorf("at index %d: expected %d, got %d", i, i, order[i])
		}
	}

}
