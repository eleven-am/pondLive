package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

// TestIntegration_StateEffectMemo tests State + Effect + Memo interaction
func TestIntegration_StateEffectMemo(t *testing.T) {
	effectRuns := 0
	memoRuns := 0
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, set := UseState(ctx, 0)
		setState = set

		doubled := UseMemo(ctx, func() int {
			memoRuns++
			return state() * 2
		}, state())

		UseEffect(ctx, func() Cleanup {
			effectRuns++
			_ = doubled
			return nil
		}, state(), doubled)

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if memoRuns != 1 {
		t.Errorf("expected 1 memo run, got %d", memoRuns)
	}
	if effectRuns != 1 {
		t.Errorf("expected 1 effect run, got %d", effectRuns)
	}

	setState(5)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if memoRuns != 2 {
		t.Errorf("expected 2 memo runs, got %d", memoRuns)
	}
	if effectRuns != 2 {
		t.Errorf("expected 2 effect runs, got %d", effectRuns)
	}
}

// TestIntegration_EffectTriggersState tests effect that updates state
func TestIntegration_EffectTriggersState(t *testing.T) {
	renderCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		count, setCount := UseState(ctx, 0)
		initialized, setInitialized := UseState(ctx, false)

		UseEffect(ctx, func() Cleanup {
			if !initialized() {
				setInitialized(true)
				setCount(42)
			}
			return nil
		})

		return &dom.StructuredNode{
			Tag:  "div",
			Text: string(rune('0' + count())),
		}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 1 {
		t.Errorf("expected 1 render after first flush, got %d", renderCount)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	if renderCount < 2 {
		t.Errorf("expected at least 2 renders after second flush, got %d", renderCount)
	}
}

// TestIntegration_RefInEffect tests using ref in effect
func TestIntegration_RefInEffect(t *testing.T) {
	effectRuns := 0
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, set := UseState(ctx, 0)
		setState = set
		prevState := UseRef(ctx, 0)

		UseEffect(ctx, func() Cleanup {
			effectRuns++
			prev := prevState.Cur
			current := state()

			prevState.Cur = current

			_ = prev
			return nil
		}, state())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectRuns != 1 {
		t.Errorf("expected 1 effect run, got %d", effectRuns)
	}

	setState(5)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectRuns != 2 {
		t.Errorf("expected 2 effect runs, got %d", effectRuns)
	}
}

// TestIntegration_MemoWithEffectDeps tests memo result used in effect deps
func TestIntegration_MemoWithEffectDeps(t *testing.T) {
	memoRuns := 0
	effectRuns := 0
	var setA func(int)
	var setB func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		a, setALocal := UseState(ctx, 1)
		_, setBLocal := UseState(ctx, 10)
		setA = setALocal
		setB = setBLocal

		sum := UseMemo(ctx, func() int {
			memoRuns++
			return a() + 100
		}, a())

		UseEffect(ctx, func() Cleanup {
			effectRuns++
			_ = sum
			return nil
		}, sum)

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	initialMemo := memoRuns
	initialEffect := effectRuns

	setB(20)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if memoRuns != initialMemo {
		t.Errorf("expected memo not to recompute (b changed), got %d runs", memoRuns)
	}
	if effectRuns != initialEffect {
		t.Errorf("expected effect not to re-run (memo unchanged), got %d runs", effectRuns)
	}

	setA(2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if memoRuns != initialMemo+1 {
		t.Errorf("expected memo to recompute (a changed), got %d runs", memoRuns)
	}
	if effectRuns != initialEffect+1 {
		t.Errorf("expected effect to re-run (memo changed), got %d runs", effectRuns)
	}
}

// TestIntegration_MultipleStateUpdatesInEffect tests multiple state updates in single effect
func TestIntegration_MultipleStateUpdatesInEffect(t *testing.T) {
	renderCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		a, setA := UseState(ctx, 0)
		b, setB := UseState(ctx, 0)
		trigger, setTrigger := UseState(ctx, false)

		UseEffect(ctx, func() Cleanup {
			if trigger() {
				setA(10)
				setB(20)
			}
			return nil
		}, trigger())

		_ = a
		_ = b

		if renderCount == 1 {

			setTrigger(true)
		}

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("third flush failed: %v", err)
	}

	if renderCount < 2 {
		t.Errorf("expected at least 2 renders, got %d", renderCount)
	}
}

// TestIntegration_ContextWithHooks tests context + hooks interaction
func TestIntegration_ContextWithHooks(t *testing.T) {
	type ConfigContext struct {
		Value int
	}

	ctxKey := CreateContext(ConfigContext{Value: 0})
	effectRuns := 0

	child := func(childCtx Ctx, childProps struct{}) *dom.StructuredNode {
		cfg := ctxKey.Use(childCtx)

		UseEffect(childCtx, func() Cleanup {
			effectRuns++
			_ = cfg.Value
			return nil
		}, cfg)

		return &dom.StructuredNode{Tag: "span"}
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		config, setConfig := UseState(ctx, ConfigContext{Value: 42})

		UseEffect(ctx, func() Cleanup {

			setConfig(ConfigContext{Value: 100})
			return nil
		})

		return ctxKey.Provide(ctx, config(), func(pctx Ctx) *dom.StructuredNode {
			childNode := Render(pctx, child, struct{}{})
			return &dom.StructuredNode{
				Tag:      "div",
				Children: []*dom.StructuredNode{childNode},
			}
		})
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectRuns < 1 {
		t.Errorf("expected at least 1 effect run, got %d", effectRuns)
	}
}

// TestIntegration_ElementRefWithEffect tests UseElement + UseEffect
func TestIntegration_ElementRefWithEffect(t *testing.T) {
	effectRuns := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		ref := UseRef[int](ctx, 0)

		UseEffect(ctx, func() Cleanup {
			effectRuns++

			_ = ref
			return nil
		})

		return &dom.StructuredNode{
			Tag: "div",
		}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectRuns != 1 {
		t.Errorf("expected 1 effect run, got %d", effectRuns)
	}
}

// TestIntegration_ComplexComponentTree tests hooks in component hierarchy
func TestIntegration_ComplexComponentTree(t *testing.T) {
	parentRenders := 0
	childRenders := 0
	var setParentState func(int)

	child := func(ctx Ctx, props struct{ Value int }) *dom.StructuredNode {
		childRenders++
		localState, _ := UseState(ctx, 0)

		UseEffect(ctx, func() Cleanup {
			_ = localState
			return nil
		}, props.Value)

		return &dom.StructuredNode{
			Tag:  "span",
			Text: string(rune('0' + props.Value)),
		}
	}

	parent := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		parentRenders++
		parentState, set := UseState(ctx, 0)
		setParentState = set

		childComp := Render(ctx, child, struct{ Value int }{Value: parentState()})

		return &dom.StructuredNode{
			Tag:      "div",
			Children: []*dom.StructuredNode{childComp},
		}
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if parentRenders != 1 {
		t.Errorf("expected 1 parent render, got %d", parentRenders)
	}
	if childRenders != 1 {
		t.Errorf("expected 1 child render, got %d", childRenders)
	}

	setParentState(5)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if parentRenders != 2 {
		t.Errorf("expected 2 parent renders, got %d", parentRenders)
	}
	if childRenders != 2 {
		t.Errorf("expected 2 child renders, got %d", childRenders)
	}
}

// TestIntegration_EffectCleanupChain tests cleanup chain across updates
func TestIntegration_EffectCleanupChain(t *testing.T) {
	var events []string
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, set := UseState(ctx, 0)
		setState = set

		UseEffect(ctx, func() Cleanup {
			events = append(events, "effect1-setup")
			return func() {
				events = append(events, "effect1-cleanup")
			}
		}, state())

		UseEffect(ctx, func() Cleanup {
			events = append(events, "effect2-setup")
			return func() {
				events = append(events, "effect2-cleanup")
			}
		}, state())

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expected := []string{"effect1-setup", "effect2-setup"}
	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(events), events)
	}

	events = nil
	setState(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expected = []string{"effect1-cleanup", "effect2-cleanup", "effect1-setup", "effect2-setup"}
	if len(events) != len(expected) {
		t.Fatalf("expected %d events, got %d: %v", len(expected), len(events), events)
	}

	for i, exp := range expected {
		if events[i] != exp {
			t.Errorf("at index %d: expected %s, got %s", i, exp, events[i])
		}
	}
}

// TestIntegration_MemoWithCallback tests memoized callback
func TestIntegration_MemoWithCallback(t *testing.T) {
	memoRuns := 0
	callbackRuns := 0
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		state, set := UseState(ctx, 0)
		setState = set

		callback := UseMemo(ctx, func() func() int {
			memoRuns++
			return func() int {
				callbackRuns++
				return state() * 2
			}
		}, state())

		_ = callback()

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if memoRuns != 1 {
		t.Errorf("expected 1 memo run, got %d", memoRuns)
	}
	if callbackRuns != 1 {
		t.Errorf("expected 1 callback run, got %d", callbackRuns)
	}

	setState(5)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if memoRuns != 2 {
		t.Errorf("expected 2 memo runs, got %d", memoRuns)
	}
	if callbackRuns != 2 {
		t.Errorf("expected 2 callback runs, got %d", callbackRuns)
	}
}

// TestIntegration_StateRefMemoEffect tests all hooks together
func TestIntegration_StateRefMemoEffect(t *testing.T) {
	renderCount := 0
	memoCount := 0
	effectCount := 0
	var setState func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++

		state, set := UseState(ctx, 0)
		setState = set

		prevState := UseRef(ctx, 0)

		doubled := UseMemo(ctx, func() int {
			memoCount++
			return state() * 2
		}, state())

		UseEffect(ctx, func() Cleanup {
			effectCount++
			prev := prevState.Cur
			prevState.Cur = state()

			_ = prev
			_ = doubled
			_ = state

			return func() {

				_ = state
			}
		}, state(), doubled)

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 1 {
		t.Errorf("expected 1 render, got %d", renderCount)
	}
	if memoCount != 1 {
		t.Errorf("expected 1 memo, got %d", memoCount)
	}
	if effectCount != 1 {
		t.Errorf("expected 1 effect, got %d", effectCount)
	}

	setState(5)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 2 {
		t.Errorf("expected 2 renders, got %d", renderCount)
	}
	if memoCount != 2 {
		t.Errorf("expected 2 memos, got %d", memoCount)
	}
	if effectCount != 2 {
		t.Errorf("expected 2 effects, got %d", effectCount)
	}
}

// TestIntegration_ConditionalHooks tests hook usage patterns (should panic)
func TestIntegration_ConditionalHooks(t *testing.T) {
	condition := true

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		if condition {
			UseState(ctx, 0)
		} else {
			UseRef(ctx, 0)
		}
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	var capturedDiag *Diagnostic
	sess.SetDiagnosticReporter(mockReporter{
		report: func(d Diagnostic) {
			capturedDiag = &d
		},
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	condition = false
	sess.markDirty(sess.root)
	sess.Flush()

	if capturedDiag == nil {
		t.Error("expected diagnostic for hook mismatch")
	}
}

// TestIntegration_EffectWithRefDeps tests effect depending on ref (anti-pattern)
func TestIntegration_EffectWithRefDeps(t *testing.T) {
	effectRuns := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		ref := UseRef(ctx, 0)

		UseEffect(ctx, func() Cleanup {
			effectRuns++
			_ = ref.Cur
			return nil
		}, ref)

		ref.Cur++

		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectRuns != 1 {
		t.Errorf("expected 1 effect run, got %d", effectRuns)
	}

	sess.markDirty(sess.root)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if effectRuns != 1 {
		t.Errorf("expected still 1 effect run (ref unchanged), got %d", effectRuns)
	}
}
