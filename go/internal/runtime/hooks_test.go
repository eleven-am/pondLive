package runtime

import "testing"

func TestUseState(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	val, set := UseState[int](ctx, 42)
	if val != 42 {
		t.Errorf("expected initial value 42, got %d", val)
	}

	set(100)
	if len(sess.DirtyQueue) != 1 {
		t.Errorf("expected component to be marked dirty")
	}

	ctx.hookIndex = 0
	val2, _ := UseState[int](ctx, 42)
	if val2 != 100 {
		t.Errorf("expected persisted value 100, got %d", val2)
	}
}

func TestUseStateMultiple(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	count, setCount := UseState[int](ctx, 0)
	name, setName := UseState[string](ctx, "alice")

	if count != 0 {
		t.Errorf("expected count 0, got %d", count)
	}
	if name != "alice" {
		t.Errorf("expected name 'alice', got %s", name)
	}

	setCount(5)
	setName("bob")

	ctx.hookIndex = 0
	count2, _ := UseState[int](ctx, 0)
	name2, _ := UseState[string](ctx, "alice")

	if count2 != 5 {
		t.Errorf("expected count 5, got %d", count2)
	}
	if name2 != "bob" {
		t.Errorf("expected name 'bob', got %s", name2)
	}
}

func TestUseRef(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	ref := UseRef[int](ctx, 10)
	if ref.Current != 10 {
		t.Errorf("expected initial value 10, got %d", ref.Current)
	}

	ref.Current = 20

	ctx.hookIndex = 0
	ref2 := UseRef[int](ctx, 10)
	if ref2.Current != 20 {
		t.Errorf("expected persisted value 20, got %d", ref2.Current)
	}

	if ref != ref2 {
		t.Error("expected same ref pointer across renders")
	}
}

func TestUseMemo(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	callCount := 0
	compute := func() int {
		callCount++
		return 42
	}

	val := UseMemo[int](ctx, compute, "dep1")
	if val != 42 {
		t.Errorf("expected value 42, got %d", val)
	}
	if callCount != 1 {
		t.Errorf("expected compute called once, got %d", callCount)
	}

	ctx.hookIndex = 0
	val2 := UseMemo[int](ctx, compute, "dep1")
	if val2 != 42 {
		t.Errorf("expected value 42, got %d", val2)
	}
	if callCount != 1 {
		t.Errorf("expected compute still called once (cached), got %d", callCount)
	}

	ctx.hookIndex = 0
	val3 := UseMemo[int](ctx, compute, "dep2")
	if val3 != 42 {
		t.Errorf("expected value 42, got %d", val3)
	}
	if callCount != 2 {
		t.Errorf("expected compute called twice (deps changed), got %d", callCount)
	}
}

func TestUseEffect(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	setupCount := 0
	cleanupCount := 0

	effect := func() func() {
		setupCount++
		return func() {
			cleanupCount++
		}
	}

	UseEffect(ctx, effect, "dep1")
	if setupCount != 0 {
		t.Errorf("expected effect not run yet (deferred), got %d", setupCount)
	}
	if len(sess.PendingEffects) != 1 {
		t.Errorf("expected 1 pending effect, got %d", len(sess.PendingEffects))
	}

	sess.runPendingEffects()
	if setupCount != 1 {
		t.Errorf("expected effect setup once after flush, got %d", setupCount)
	}
	if cleanupCount != 0 {
		t.Errorf("expected no cleanup yet, got %d", cleanupCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect, "dep1")
	if len(sess.PendingEffects) != 0 {
		t.Errorf("expected no pending effects (deps unchanged), got %d", len(sess.PendingEffects))
	}
	sess.runPendingEffects()
	if setupCount != 1 {
		t.Errorf("expected effect still setup once (cached), got %d", setupCount)
	}
	if cleanupCount != 0 {
		t.Errorf("expected no cleanup (deps unchanged), got %d", cleanupCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect, "dep2")
	if len(sess.PendingCleanups) != 1 {
		t.Errorf("expected 1 pending cleanup (deps changed), got %d", len(sess.PendingCleanups))
	}
	if len(sess.PendingEffects) != 1 {
		t.Errorf("expected 1 pending effect (deps changed), got %d", len(sess.PendingEffects))
	}
	sess.runPendingEffects()
	if setupCount != 2 {
		t.Errorf("expected effect setup twice (deps changed), got %d", setupCount)
	}
	if cleanupCount != 1 {
		t.Errorf("expected cleanup called once (deps changed), got %d", cleanupCount)
	}
}

func TestUseEffectNoDeps(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	runCount := 0
	effect := func() func() {
		runCount++
		return nil
	}

	UseEffect(ctx, effect)
	if len(sess.PendingEffects) != 1 {
		t.Errorf("expected 1 pending effect, got %d", len(sess.PendingEffects))
	}
	sess.runPendingEffects()
	if runCount != 1 {
		t.Errorf("expected effect to run once, got %d", runCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect)
	if len(sess.PendingEffects) != 1 {
		t.Errorf("expected 1 pending effect (no deps = run every render), got %d", len(sess.PendingEffects))
	}
	sess.runPendingEffects()
	if runCount != 2 {
		t.Errorf("expected effect to run twice (no deps), got %d", runCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect)
	if len(sess.PendingEffects) != 1 {
		t.Errorf("expected 1 pending effect (no deps = run every render), got %d", len(sess.PendingEffects))
	}
	sess.runPendingEffects()
	if runCount != 3 {
		t.Errorf("expected effect to run three times (no deps), got %d", runCount)
	}
}

func TestUseEffectNoDepsWithCleanup(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	setupCount := 0
	cleanupCount := 0
	effect := func() func() {
		setupCount++
		return func() {
			cleanupCount++
		}
	}

	UseEffect(ctx, effect)
	sess.runPendingEffects()
	if setupCount != 1 {
		t.Errorf("expected 1 setup, got %d", setupCount)
	}
	if cleanupCount != 0 {
		t.Errorf("expected 0 cleanup, got %d", cleanupCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect)
	if len(sess.PendingCleanups) != 1 {
		t.Errorf("expected 1 pending cleanup (no deps = cleanup old), got %d", len(sess.PendingCleanups))
	}
	sess.runPendingEffects()
	if setupCount != 2 {
		t.Errorf("expected 2 setups, got %d", setupCount)
	}
	if cleanupCount != 1 {
		t.Errorf("expected 1 cleanup (old effect), got %d", cleanupCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect)
	if len(sess.PendingCleanups) != 1 {
		t.Errorf("expected 1 pending cleanup, got %d", len(sess.PendingCleanups))
	}
	sess.runPendingEffects()
	if setupCount != 3 {
		t.Errorf("expected 3 setups, got %d", setupCount)
	}
	if cleanupCount != 2 {
		t.Errorf("expected 2 cleanups, got %d", cleanupCount)
	}
}

func TestDepsEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []any
		b        []any
		expected bool
	}{
		{
			name:     "empty slices",
			a:        []any{},
			b:        []any{},
			expected: true,
		},
		{
			name:     "same primitives",
			a:        []any{1, "hello", true},
			b:        []any{1, "hello", true},
			expected: true,
		},
		{
			name:     "different primitives",
			a:        []any{1, "hello"},
			b:        []any{2, "hello"},
			expected: false,
		},
		{
			name:     "different lengths",
			a:        []any{1, 2},
			b:        []any{1},
			expected: false,
		},
		{
			name:     "same function pointer",
			a:        []any{TestDepsEqual},
			b:        []any{TestDepsEqual},
			expected: true,
		},
		{
			name:     "different function pointers",
			a:        []any{TestDepsEqual},
			b:        []any{TestUseState},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := depsEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestDepsEqualMaps(t *testing.T) {
	m1 := map[string]int{"a": 1}
	m2 := map[string]int{"a": 1}

	if !depsEqual([]any{m1}, []any{m1}) {
		t.Error("expected same map instance to be equal")
	}

	if depsEqual([]any{m1}, []any{m2}) {
		t.Error("expected different map instances to not be equal")
	}
}

func TestCloneDeps(t *testing.T) {
	original := []any{1, "hello", true}
	cloned := cloneDeps(original)

	if len(cloned) != len(original) {
		t.Errorf("expected same length, got %d", len(cloned))
	}

	for i := range original {
		if cloned[i] != original[i] {
			t.Errorf("expected same value at index %d", i)
		}
	}

	cloned[0] = 999
	if original[0] == 999 {
		t.Error("mutating clone affected original")
	}
}

func TestHookMismatch(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	UseState[int](ctx, 0)

	ctx.hookIndex = 0
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for hook mismatch")
		}
	}()
	UseRef[int](ctx, 0)
}

func TestUseElement(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	ref := UseElement(ctx)
	if ref == nil {
		t.Fatal("expected ref to be non-nil")
	}
	if ref.RefID() != "test-comp:r0" {
		t.Errorf("expected ref ID 'test-comp:r0', got %s", ref.RefID())
	}

	ctx.hookIndex = 0
	ref2 := UseElement(ctx)
	if ref2.RefID() != "test-comp:r0" {
		t.Errorf("expected same ref ID 'test-comp:r0', got %s", ref2.RefID())
	}

	if ref != ref2 {
		t.Error("expected same ref pointer across renders")
	}
}

func TestUseElementMultiple(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	ref1 := UseElement(ctx)
	ref2 := UseElement(ctx)
	ref3 := UseElement(ctx)

	if ref1.RefID() != "test-comp:r0" {
		t.Errorf("expected ref1 ID 'test-comp:r0', got %s", ref1.RefID())
	}
	if ref2.RefID() != "test-comp:r1" {
		t.Errorf("expected ref2 ID 'test-comp:r1', got %s", ref2.RefID())
	}
	if ref3.RefID() != "test-comp:r2" {
		t.Errorf("expected ref3 ID 'test-comp:r2', got %s", ref3.RefID())
	}

	ctx.hookIndex = 0
	ref1b := UseElement(ctx)
	ref2b := UseElement(ctx)
	ref3b := UseElement(ctx)

	if ref1 != ref1b || ref2 != ref2b || ref3 != ref3b {
		t.Error("expected same ref pointers across renders")
	}
}

func TestUseElementWithoutSessionDoesNotPanic(t *testing.T) {
	inst := &Instance{ID: "stateless"}
	ctx := &Ctx{instance: inst}

	ref := UseElement(ctx)
	if ref == nil {
		t.Fatal("expected ref even without session")
	}
	if ref.RefID() == "" {
		t.Fatalf("expected non-empty fallback ref ID")
	}

	ctx.hookIndex = 0
	refAgain := UseElement(ctx)
	if refAgain.RefID() != ref.RefID() {
		t.Fatalf("expected stable ref ID, got %q vs %q", refAgain.RefID(), ref.RefID())
	}
}

func TestElementRefNilSafe(t *testing.T) {
	var ref *ElementRef

	if ref.RefID() != "" {
		t.Errorf("expected empty string for nil ref, got %s", ref.RefID())
	}
}

func TestUseStateEquality(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	val, set := UseState[int](ctx, 42)
	if val != 42 {
		t.Errorf("expected initial value 42, got %d", val)
	}

	set(42)
	if len(sess.DirtyQueue) != 0 {
		t.Errorf("expected no dirty marking for same value, got %d dirty", len(sess.DirtyQueue))
	}

	set(100)
	if len(sess.DirtyQueue) != 1 {
		t.Errorf("expected 1 dirty component for different value, got %d", len(sess.DirtyQueue))
	}

	sess.DirtyQueue = sess.DirtyQueue[:0]
	sess.DirtySet = make(map[*Instance]struct{})

	set(100)
	if len(sess.DirtyQueue) != 0 {
		t.Errorf("expected no dirty marking for same value (100), got %d dirty", len(sess.DirtyQueue))
	}
}

func TestUseStateWithCustomEquality(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	sameSign := func(a, b int) bool {
		return (a >= 0 && b >= 0) || (a < 0 && b < 0)
	}

	val, set := UseState[int](ctx, 5, WithEqual(sameSign))
	if val != 5 {
		t.Errorf("expected initial value 5, got %d", val)
	}

	set(10)
	if len(sess.DirtyQueue) != 0 {
		t.Errorf("expected no dirty (same sign positive), got %d dirty", len(sess.DirtyQueue))
	}

	set(-5)
	if len(sess.DirtyQueue) != 1 {
		t.Errorf("expected 1 dirty (different sign), got %d", len(sess.DirtyQueue))
	}

	sess.DirtyQueue = sess.DirtyQueue[:0]
	sess.DirtySet = make(map[*Instance]struct{})

	set(-10)
	if len(sess.DirtyQueue) != 0 {
		t.Errorf("expected no dirty (same sign negative), got %d dirty", len(sess.DirtyQueue))
	}
}

func TestUseStateStructEquality(t *testing.T) {
	type User struct {
		ID   int
		Name string
	}

	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	user, setUser := UseState[User](ctx, User{ID: 1, Name: "Alice"})
	if user.Name != "Alice" {
		t.Errorf("expected Alice, got %s", user.Name)
	}

	setUser(User{ID: 1, Name: "Alice"})
	if len(sess.DirtyQueue) != 0 {
		t.Errorf("expected no dirty for equal struct, got %d dirty", len(sess.DirtyQueue))
	}

	setUser(User{ID: 1, Name: "Bob"})
	if len(sess.DirtyQueue) != 1 {
		t.Errorf("expected 1 dirty for different struct, got %d", len(sess.DirtyQueue))
	}
}

func TestDefaultEqual(t *testing.T) {
	eq := defaultEqual[int]()

	if !eq(5, 5) {
		t.Error("expected 5 == 5")
	}
	if eq(5, 6) {
		t.Error("expected 5 != 6")
	}

	eqStr := defaultEqual[string]()
	if !eqStr("hello", "hello") {
		t.Error("expected 'hello' == 'hello'")
	}
	if eqStr("hello", "world") {
		t.Error("expected 'hello' != 'world'")
	}

	type Data struct {
		Values []int
	}
	eqData := defaultEqual[Data]()
	if !eqData(Data{Values: []int{1, 2, 3}}, Data{Values: []int{1, 2, 3}}) {
		t.Error("expected equal slices to be equal")
	}
	if eqData(Data{Values: []int{1, 2, 3}}, Data{Values: []int{1, 2, 4}}) {
		t.Error("expected different slices to not be equal")
	}
}

func TestUseEffectEmptyDepsRunsOnce(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	runCount := 0
	effect := func() func() {
		runCount++
		return nil
	}

	UseEffect(ctx, effect, []any{}...)
	sess.runPendingEffects()
	if runCount != 1 {
		t.Errorf("expected effect to run once on mount, got %d", runCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect, []any{}...)
	if len(sess.PendingEffects) != 0 {
		t.Errorf("expected no pending effects on re-render with empty deps, got %d", len(sess.PendingEffects))
	}
	sess.runPendingEffects()
	if runCount != 1 {
		t.Errorf("expected effect to still be 1 (empty deps = run once), got %d", runCount)
	}

	ctx.hookIndex = 0
	UseEffect(ctx, effect, []any{}...)
	sess.runPendingEffects()
	if runCount != 1 {
		t.Errorf("expected effect to still be 1 after third render, got %d", runCount)
	}
}

func TestUseEffectNilVsEmptyDepsDistinction(t *testing.T) {
	instNil := &Instance{
		ID:        "test-nil-deps",
		HookFrame: []HookSlot{},
	}
	sessNil := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctxNil := &Ctx{
		instance:  instNil,
		session:   sessNil,
		hookIndex: 0,
	}

	nilDepsRunCount := 0
	nilDepsEffect := func() func() {
		nilDepsRunCount++
		return nil
	}

	UseEffect(ctxNil, nilDepsEffect)
	sessNil.runPendingEffects()
	if nilDepsRunCount != 1 {
		t.Errorf("expected nil deps effect to run once on mount, got %d", nilDepsRunCount)
	}

	ctxNil.hookIndex = 0
	UseEffect(ctxNil, nilDepsEffect)
	sessNil.runPendingEffects()
	if nilDepsRunCount != 2 {
		t.Errorf("expected nil deps effect to run every render, got %d", nilDepsRunCount)
	}

	ctxNil.hookIndex = 0
	UseEffect(ctxNil, nilDepsEffect)
	sessNil.runPendingEffects()
	if nilDepsRunCount != 3 {
		t.Errorf("expected nil deps effect to run on third render, got %d", nilDepsRunCount)
	}

	instEmpty := &Instance{
		ID:        "test-empty-deps",
		HookFrame: []HookSlot{},
	}
	sessEmpty := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctxEmpty := &Ctx{
		instance:  instEmpty,
		session:   sessEmpty,
		hookIndex: 0,
	}

	emptyDepsRunCount := 0
	emptyDepsEffect := func() func() {
		emptyDepsRunCount++
		return nil
	}

	UseEffect(ctxEmpty, emptyDepsEffect, []any{}...)
	sessEmpty.runPendingEffects()
	if emptyDepsRunCount != 1 {
		t.Errorf("expected empty deps effect to run once on mount, got %d", emptyDepsRunCount)
	}

	ctxEmpty.hookIndex = 0
	UseEffect(ctxEmpty, emptyDepsEffect, []any{}...)
	sessEmpty.runPendingEffects()
	if emptyDepsRunCount != 1 {
		t.Errorf("expected empty deps effect to NOT run on re-render, got %d", emptyDepsRunCount)
	}

	ctxEmpty.hookIndex = 0
	UseEffect(ctxEmpty, emptyDepsEffect, []any{}...)
	sessEmpty.runPendingEffects()
	if emptyDepsRunCount != 1 {
		t.Errorf("expected empty deps effect to NOT run on third render, got %d", emptyDepsRunCount)
	}
}

func TestUseEffectEmptyDepsHasDepsTrue(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	effect := func() func() { return nil }
	UseEffect(ctx, effect, []any{}...)

	if len(inst.HookFrame) != 1 {
		t.Fatalf("expected 1 hook slot, got %d", len(inst.HookFrame))
	}

	cell, ok := inst.HookFrame[0].Value.(*effectCell)
	if !ok {
		t.Fatal("expected effectCell")
	}

	if !cell.hasDeps {
		t.Error("expected hasDeps to be true for empty deps array")
	}

	if len(cell.deps) != 0 {
		t.Errorf("expected empty deps slice, got %d", len(cell.deps))
	}
}

func TestUseEffectNilDepsHasDepsFalse(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		PendingEffects:  []effectTask{},
		PendingCleanups: []cleanupTask{},
	}
	ctx := &Ctx{
		instance:  inst,
		session:   sess,
		hookIndex: 0,
	}

	effect := func() func() { return nil }
	UseEffect(ctx, effect)

	if len(inst.HookFrame) != 1 {
		t.Fatalf("expected 1 hook slot, got %d", len(inst.HookFrame))
	}

	cell, ok := inst.HookFrame[0].Value.(*effectCell)
	if !ok {
		t.Fatal("expected effectCell")
	}

	if cell.hasDeps {
		t.Error("expected hasDeps to be false for nil deps (no deps passed)")
	}
}
