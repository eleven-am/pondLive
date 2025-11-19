package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom2/diff"
)

func TestUseEffectBasic(t *testing.T) {
	effectRan := false

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		UseEffect(ctx, func() Cleanup {
			effectRan = true
			return nil
		})
		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if !effectRan {
		t.Error("expected effect to run after flush")
	}
}

func TestUseEffectCleanup(t *testing.T) {
	effectCount := 0
	cleanupCount := 0

	var setDep func(int)
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		dep, set := UseState(ctx, 1)
		setDep = set

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return func() {
				cleanupCount++
			}
		}, dep())

		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 1 {
		t.Errorf("expected effect to run once, got %d", effectCount)
	}
	if cleanupCount != 0 {
		t.Errorf("expected no cleanup yet, got %d", cleanupCount)
	}

	setDep(2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 2 {
		t.Errorf("expected effect to run twice, got %d", effectCount)
	}
	if cleanupCount != 1 {
		t.Errorf("expected cleanup to run once, got %d", cleanupCount)
	}

	setDep(3)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 3 {
		t.Errorf("expected effect to run 3 times, got %d", effectCount)
	}
	if cleanupCount != 2 {
		t.Errorf("expected cleanup to run twice, got %d", cleanupCount)
	}
}

func TestUseEffectSameDeps(t *testing.T) {
	effectCount := 0

	var triggerRender func()
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		count, setCount := UseState(ctx, 0)
		triggerRender = func() { setCount(count() + 1) }

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return nil
		}, 42)

		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 1 {
		t.Errorf("expected effect to run once, got %d", effectCount)
	}

	triggerRender()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 1 {
		t.Errorf("expected effect to still be 1 (not re-run), got %d", effectCount)
	}
}

func TestUseEffectMultipleDeps(t *testing.T) {
	effectCount := 0

	var setA func(int)
	var setB func(string)
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		a, setALocal := UseState(ctx, 1)
		b, setBLocal := UseState(ctx, "hello")
		setA = setALocal
		setB = setBLocal

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return nil
		}, a(), b())

		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 1 {
		t.Errorf("expected 1 effect run, got %d", effectCount)
	}

	setA(2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 2 {
		t.Errorf("expected 2 effect runs, got %d", effectCount)
	}

	setB("world")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 3 {
		t.Errorf("expected 3 effect runs, got %d", effectCount)
	}

	setA(3)
	setB("!")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 4 {
		t.Errorf("expected 4 effect runs, got %d", effectCount)
	}
}

func TestUseEffectExecutionOrder(t *testing.T) {
	var order []string

	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		UseEffect(ctx, func() Cleanup {
			order = append(order, "effect1")
			return func() {
				order = append(order, "cleanup1")
			}
		}, 1)

		UseEffect(ctx, func() Cleanup {
			order = append(order, "effect2")
			return func() {
				order = append(order, "cleanup2")
			}
		}, 2)

		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	expected := []string{"effect1", "effect2"}
	if len(order) != len(expected) {
		t.Fatalf("expected order %v, got %v", expected, order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("at index %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestUseEffectWithStateUpdate(t *testing.T) {
	effectRan := false
	var countInEffect int

	var setCount func(int)
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		count, set := UseState(ctx, 0)
		setCount = set

		UseEffect(ctx, func() Cleanup {
			effectRan = true
			countInEffect = count()
			return nil
		}, count())

		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if !effectRan {
		t.Error("expected effect to run")
	}
	if countInEffect != 0 {
		t.Errorf("expected count in effect to be 0, got %d", countInEffect)
	}

	setCount(42)
	effectRan = false
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if !effectRan {
		t.Error("expected effect to run again")
	}
	if countInEffect != 42 {
		t.Errorf("expected count in effect to be 42, got %d", countInEffect)
	}
}

func TestUseEffectNoDeps(t *testing.T) {
	effectCount := 0

	var triggerRender func()
	comp := func(ctx Ctx, props struct{}) *dom2.StructuredNode {
		count, setCount := UseState(ctx, 0)
		triggerRender = func() { setCount(count() + 1) }

		UseEffect(ctx, func() Cleanup {
			effectCount++
			return nil
		})

		return &dom2.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 1 {
		t.Errorf("expected 1 effect run, got %d", effectCount)
	}

	triggerRender()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 2 {
		t.Errorf("expected 2 effect runs, got %d", effectCount)
	}

	triggerRender()
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
	if effectCount != 3 {
		t.Errorf("expected 3 effect runs, got %d", effectCount)
	}
}
