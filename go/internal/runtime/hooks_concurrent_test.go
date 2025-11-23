package runtime

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

// TestConcurrent_MultipleStateUpdates tests concurrent state updates
func TestConcurrent_MultipleStateUpdates(t *testing.T) {
	var setState func(int)
	finalValue := int32(0)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		count, set := UseState(ctx, 0)
		setState = set
		atomic.StoreInt32(&finalValue, int32(count()))
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 100
	updatesPerGoroutine := 10

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < updatesPerGoroutine; j++ {
				setState(id*updatesPerGoroutine + j)
				time.Sleep(time.Microsecond)
			}
		}(i)
	}

	wg.Wait()

	if err := sess.Flush(); err != nil {
		t.Fatalf("final flush failed: %v", err)
	}

	if atomic.LoadInt32(&finalValue) == 0 {
		t.Error("expected state to be updated")
	}
}

// TestConcurrent_StateUpdateAndFlush tests concurrent state updates and flushes
func TestConcurrent_StateUpdateAndFlush(t *testing.T) {
	var setState func(int)
	renderCount := int32(0)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		atomic.AddInt32(&renderCount, 1)
		_, set := UseState(ctx, 0)
		setState = set
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			setState(i)
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			sess.Flush()
			time.Sleep(time.Millisecond)
		}
	}()

	wg.Wait()

	if atomic.LoadInt32(&renderCount) < 2 {
		t.Errorf("expected multiple renders, got %d", renderCount)
	}
}

// TestConcurrent_EffectExecution tests concurrent effect execution
func TestConcurrent_EffectExecution(t *testing.T) {
	effectCount := int32(0)
	cleanupCount := int32(0)
	var setDep func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		dep, set := UseState(ctx, 0)
		setDep = set

		UseEffect(ctx, func() Cleanup {
			atomic.AddInt32(&effectCount, 1)
			return func() {
				atomic.AddInt32(&cleanupCount, 1)
			}
		}, dep())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 50

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			setDep(id)
			sess.Flush()
		}(i)
	}

	wg.Wait()

	effects := atomic.LoadInt32(&effectCount)
	cleanups := atomic.LoadInt32(&cleanupCount)

	if effects < 2 {
		t.Errorf("expected at least 2 effect runs, got %d", effects)
	}

	if cleanups >= effects {
		t.Errorf("expected cleanups (%d) to be less than effects (%d)", cleanups, effects)
	}
}

// TestConcurrent_MultipleHooksInComponent tests concurrent access to multiple hooks
func TestConcurrent_MultipleHooksInComponent(t *testing.T) {
	var setA func(int)
	var setB func(string)
	var setC func(bool)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		_, setALocal := UseState(ctx, 0)
		_, setBLocal := UseState(ctx, "")
		_, setCLocal := UseState(ctx, false)

		setA = setALocal
		setB = setBLocal
		setC = setCLocal

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup

	wg.Add(3)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			setA(i)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			setB(string(rune('a' + (i % 26))))
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			setC(i%2 == 0)
		}
	}()

	wg.Wait()

	if err := sess.Flush(); err != nil {
		t.Fatalf("final flush failed: %v", err)
	}
}

// TestConcurrent_UseMemoWithChangingDeps tests concurrent memo recomputation
func TestConcurrent_UseMemoWithChangingDeps(t *testing.T) {
	computeCount := int32(0)
	var setDep func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		dep, set := UseState(ctx, 0)
		setDep = set

		_ = UseMemo(ctx, func() int {
			atomic.AddInt32(&computeCount, 1)
			return dep() * 2
		}, dep())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 50

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			setDep(id)
			sess.Flush()
		}(i)
	}

	wg.Wait()

	computes := atomic.LoadInt32(&computeCount)
	if computes < 2 {
		t.Errorf("expected at least 2 computes, got %d", computes)
	}
}

// TestConcurrent_RefMutations tests concurrent ref mutations (should be safe)
func TestConcurrent_RefMutations(t *testing.T) {
	var ref *Ref[int]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		ref = UseRef(ctx, 0)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				ref.Cur = id*100 + j
			}
		}(i)
	}

	wg.Wait()

}

// TestConcurrent_MarkDirtyFromMultipleGoroutines tests concurrent markDirty calls
func TestConcurrent_MarkDirtyFromMultipleGoroutines(t *testing.T) {
	renderCount := int32(0)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		atomic.AddInt32(&renderCount, 1)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 50

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				sess.markDirty(sess.root)
			}
		}()
	}

	wg.Wait()

	if err := sess.Flush(); err != nil {
		t.Fatalf("final flush failed: %v", err)
	}

	if atomic.LoadInt32(&renderCount) < 2 {
		t.Errorf("expected at least 2 renders, got %d", renderCount)
	}
}

// TestConcurrent_EffectBatchingUnderLoad tests effect batching with concurrent updates
func TestConcurrent_EffectBatchingUnderLoad(t *testing.T) {
	effectRuns := int32(0)
	var setDep func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		dep, set := UseState(ctx, 0)
		setDep = set

		UseEffect(ctx, func() Cleanup {
			atomic.AddInt32(&effectRuns, 1)
			return nil
		}, dep())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	initialRuns := atomic.LoadInt32(&effectRuns)

	var wg sync.WaitGroup
	updates := 1000

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < updates; i++ {
			setDep(i)
		}
	}()

	wg.Wait()

	if err := sess.Flush(); err != nil {
		t.Fatalf("final flush failed: %v", err)
	}

	finalRuns := atomic.LoadInt32(&effectRuns)

	if finalRuns-initialRuns > int32(updates/10) {
		t.Logf("warning: effects might not be batching efficiently (expected < %d runs, got %d)", updates/10, finalRuns-initialRuns)
	}
}

// TestConcurrent_SessionIsolation tests that concurrent sessions don't interfere
func TestConcurrent_SessionIsolation(t *testing.T) {
	renderCounts := make([]int32, 10)

	comp := func(idx int) func(Ctx, struct{}) *dom.StructuredNode {
		return func(ctx Ctx, props struct{}) *dom.StructuredNode {
			atomic.AddInt32(&renderCounts[idx], 1)
			return &dom.StructuredNode{Tag: "div"}
		}
	}

	sessions := make([]*ComponentSession, 10)
	for i := range sessions {
		sessions[i] = NewSession(comp(i), struct{}{})
		sessions[i].SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	}

	var wg sync.WaitGroup
	wg.Add(len(sessions))

	for i, sess := range sessions {
		go func(idx int, s *ComponentSession) {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				s.markDirty(s.root)
				if err := s.Flush(); err != nil {
					t.Errorf("session %d flush failed: %v", idx, err)
				}
			}
		}(i, sess)
	}

	wg.Wait()

	for i, count := range renderCounts {
		if count < 10 {
			t.Errorf("session %d: expected at least 10 renders, got %d", i, count)
		}
	}
}

// TestConcurrent_StateEqualityCheck tests concurrent state updates with equality
func TestConcurrent_StateEqualityCheck(t *testing.T) {
	renderCount := int32(0)
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		atomic.AddInt32(&renderCount, 1)
		_, set := UseState(ctx, 42)
		setState = set
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	initialRenders := atomic.LoadInt32(&renderCount)

	var wg sync.WaitGroup
	goroutines := 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < 10; j++ {
				setState(42)
			}
		}()
	}

	wg.Wait()

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	finalRenders := atomic.LoadInt32(&renderCount)

	if finalRenders != initialRenders {
		t.Errorf("expected %d renders (no change), got %d", initialRenders, finalRenders)
	}
}

// TestConcurrent_PatchSenderCalls tests concurrent patch generation
func TestConcurrent_PatchSenderCalls(t *testing.T) {
	patchCount := int32(0)
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		count, set := UseState(ctx, 0)
		setState = set
		return &dom.StructuredNode{
			Tag:  "div",
			Text: string(rune('0' + count())),
		}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error {
		atomic.AddInt32(&patchCount, 1)
		return nil
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup
	goroutines := 50

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			setState(id % 10)
			sess.Flush()
		}(i)
	}

	wg.Wait()

	patches := atomic.LoadInt32(&patchCount)
	if patches < 2 {
		t.Errorf("expected at least 2 patch sender calls, got %d", patches)
	}
}

// TestConcurrent_ComplexHookInteractions tests concurrent complex scenarios
func TestConcurrent_ComplexHookInteractions(t *testing.T) {
	var setState func(int)
	var setMap func(map[string]int)
	effectCount := int32(0)
	memoCount := int32(0)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, setStateLocal := UseState(ctx, 0)
		m, setMapLocal := UseState(ctx, map[string]int{"count": 0})
		setState = setStateLocal
		setMap = setMapLocal

		UseEffect(ctx, func() Cleanup {
			atomic.AddInt32(&effectCount, 1)
			return nil
		}, state())

		_ = UseMemo(ctx, func() int {
			atomic.AddInt32(&memoCount, 1)
			return state() * 2
		}, state())

		_ = m

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	var wg sync.WaitGroup

	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			setState(i)
			time.Sleep(time.Microsecond * 10)
		}
	}()

	go func() {
		defer wg.Done()
		for i := 0; i < 50; i++ {
			newMap := map[string]int{"count": i}
			setMap(newMap)
			time.Sleep(time.Microsecond * 10)
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < 100; i++ {
			sess.Flush()
			time.Sleep(time.Microsecond * 5)
		}
	}()

	wg.Wait()

	if err := sess.Flush(); err != nil {
		t.Fatalf("final flush failed: %v", err)
	}

	if atomic.LoadInt32(&effectCount) < 2 {
		t.Errorf("expected at least 2 effect runs, got %d", atomic.LoadInt32(&effectCount))
	}

	if atomic.LoadInt32(&memoCount) < 2 {
		t.Errorf("expected at least 2 memo computes, got %d", atomic.LoadInt32(&memoCount))
	}
}
