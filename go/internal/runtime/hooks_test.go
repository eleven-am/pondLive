package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestUseState(t *testing.T) {
	renderCount := 0
	var getCount func() int
	var setCount func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		getCount, setCount = UseState(ctx, 0)
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
	if getCount() != 0 {
		t.Errorf("expected initial value 0, got %d", getCount())
	}

	setCount(42)

	sess.mu.Lock()
	_, isDirty := sess.dirty[sess.root]
	sess.mu.Unlock()
	if !isDirty {
		t.Error("expected root to be marked dirty after state change")
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 2 {
		t.Errorf("expected 2 renders, got %d", renderCount)
	}
	if getCount() != 42 {
		t.Errorf("expected value 42, got %d", getCount())
	}
}

func TestUseStateEquality(t *testing.T) {
	renderCount := 0
	var setCount func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		_, setCount = UseState(ctx, 10)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	setCount(10)

	sess.mu.Lock()
	_, isDirty := sess.dirty[sess.root]
	sess.mu.Unlock()

	if isDirty {
		t.Error("expected root NOT to be marked dirty when value unchanged")
	}

	setCount(20)

	sess.mu.Lock()
	_, isDirty = sess.dirty[sess.root]
	sess.mu.Unlock()

	if !isDirty {
		t.Error("expected root to be marked dirty when value changed")
	}
}

func TestUseRef(t *testing.T) {
	renderCount := 0
	var ref *Ref[int]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		ref = UseRef(ctx, 0)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if ref.Cur != 0 {
		t.Errorf("expected initial value 0, got %d", ref.Cur)
	}

	ref.Cur = 42

	sess.mu.Lock()
	_, isDirty := sess.dirty[sess.root]
	sess.mu.Unlock()

	if isDirty {
		t.Error("expected root NOT to be marked dirty when ref mutated")
	}

	sess.markDirty(sess.root)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 2 {
		t.Errorf("expected 2 renders, got %d", renderCount)
	}
	if ref.Cur != 42 {
		t.Errorf("expected ref to maintain value 42, got %d", ref.Cur)
	}
}

func TestUseMemo(t *testing.T) {
	renderCount := 0
	computeCount := 0
	var setDep func(int)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		dep, set := UseState(ctx, 1)
		setDep = set

		result := UseMemo(ctx, func() int {
			computeCount++
			return dep() * 2
		}, []any{dep()})

		return &dom.StructuredNode{
			Tag:  "div",
			Text: string(rune(result)),
		}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 1 {
		t.Errorf("expected 1 render, got %d", renderCount)
	}
	if computeCount != 1 {
		t.Errorf("expected 1 compute, got %d", computeCount)
	}

	sess.markDirty(sess.root)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 2 {
		t.Errorf("expected 2 renders, got %d", renderCount)
	}
	if computeCount != 1 {
		t.Errorf("expected still 1 compute (memoized), got %d", computeCount)
	}

	setDep(5)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if renderCount != 3 {
		t.Errorf("expected 3 renders, got %d", renderCount)
	}
	if computeCount != 2 {
		t.Errorf("expected 2 computes (dep changed), got %d", computeCount)
	}
}

func TestHookMismatch(t *testing.T) {
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

	// Capture diagnostics
	var capturedDiag *Diagnostic
	sess.SetDiagnosticReporter(mockReporter{
		report: func(d Diagnostic) {
			capturedDiag = &d
		},
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	condition = false
	sess.markDirty(sess.root)
	sess.Flush()

	if capturedDiag == nil {
		t.Fatal("expected diagnostic to be reported for hook mismatch")
	}
}

type mockReporter struct {
	report func(Diagnostic)
}

func (m mockReporter) ReportDiagnostic(d Diagnostic) {
	if m.report != nil {
		m.report(d)
	}
}

func TestHookStability(t *testing.T) {
	var ref1, ref2 *Ref[int]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		ref1 = UseRef(ctx, 1)
		ref2 = UseRef(ctx, 2)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	firstRef1 := ref1
	firstRef2 := ref2

	sess.markDirty(sess.root)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if ref1 != firstRef1 {
		t.Error("expected ref1 to be stable across renders")
	}
	if ref2 != firstRef2 {
		t.Error("expected ref2 to be stable across renders")
	}
}

func TestMultipleStatesInComponent(t *testing.T) {
	var getA func() int
	var setA func(int)
	var getB func() string
	var setB func(string)

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		getA, setA = UseState(ctx, 0)
		getB, setB = UseState(ctx, "hello")
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if getA() != 0 {
		t.Errorf("expected A to be 0, got %d", getA())
	}
	if getB() != "hello" {
		t.Errorf("expected B to be 'hello', got %q", getB())
	}

	setA(42)
	setB("world")

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if getA() != 42 {
		t.Errorf("expected A to be 42, got %d", getA())
	}
	if getB() != "world" {
		t.Errorf("expected B to be 'world', got %q", getB())
	}
}
