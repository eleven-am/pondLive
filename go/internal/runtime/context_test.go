package runtime

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/work"
)

// TestCreateContext verifies that CreateContext creates a context with unique ID and default value.
func TestCreateContext(t *testing.T) {
	ctx1 := CreateContext("default1")
	ctx2 := CreateContext("default2")

	if ctx1.id == ctx2.id {
		t.Error("expected different context IDs")
	}

	if ctx1.defaultValue != "default1" {
		t.Errorf("expected default value 'default1', got %q", ctx1.defaultValue)
	}
}

// TestUseContextReturnsDefault verifies that UseContext returns default when no provider exists.
func TestUseContextReturnsDefault(t *testing.T) {
	themeCtx := CreateContext("light")

	inst := &Instance{
		ID:        "test",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst}

	value, setter := themeCtx.UseContext(ctx)

	if value != "light" {
		t.Errorf("expected default 'light', got %q", value)
	}

	if setter != nil {
		t.Error("expected setter to be nil when no provider exists")
	}
}

// TestUseProviderStoresValue verifies that UseProvider stores value on instance.
func TestUseProviderStoresValue(t *testing.T) {
	themeCtx := CreateContext("light")

	inst := &Instance{
		ID:        "test",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{instance: inst, session: sess}

	value, setter := themeCtx.UseProvider(ctx, "dark")

	if value != "dark" {
		t.Errorf("expected 'dark', got %q", value)
	}

	if setter == nil {
		t.Error("expected setter to be non-nil")
	}

	if inst.Providers[themeCtx.id] != "dark" {
		t.Error("expected value to be stored on instance")
	}
}

// TestUseProviderReturnsSameValueOnRerender verifies that subsequent renders return stored value.
func TestUseProviderReturnsSameValueOnRerender(t *testing.T) {
	themeCtx := CreateContext("light")

	inst := &Instance{
		ID:        "test",
		HookFrame: []HookSlot{},
	}
	sess := &Session{}
	ctx := &Ctx{instance: inst, session: sess}

	value1, _ := themeCtx.UseProvider(ctx, "dark")

	value2, _ := themeCtx.UseProvider(ctx, "blue")

	if value1 != "dark" {
		t.Errorf("expected first render to return 'dark', got %q", value1)
	}

	if value2 != "dark" {
		t.Errorf("expected second render to return stored 'dark', got %q", value2)
	}
}

// TestSetterUpdatesValue verifies that the setter updates the context value.
func TestSetterUpdatesValue(t *testing.T) {
	themeCtx := CreateContext("light")

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}
	parent.Children = append(parent.Children, child)

	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{instance: parent, session: sess}

	_, setter := themeCtx.UseProvider(ctx, "dark")

	setter("blue")

	if parent.Providers[themeCtx.id] != "blue" {
		t.Error("expected value to be updated to 'blue'")
	}

	if len(sess.DirtyQueue) != 1 || sess.DirtyQueue[0] != child {
		t.Error("expected child to be marked dirty")
	}
}

// TestSetterNoOpWhenEqual verifies that setter doesn't trigger re-render when value unchanged.
func TestSetterNoOpWhenEqual(t *testing.T) {
	themeCtx := CreateContext("light")

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}
	parent.Children = append(parent.Children, child)

	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{instance: parent, session: sess}

	_, setter := themeCtx.UseProvider(ctx, "dark")

	setter("dark")

	if len(sess.DirtyQueue) != 0 {
		t.Error("expected no dirty children when value unchanged")
	}
}

// TestUseContextFindsAncestorProvider verifies that UseContext walks up the tree.
func TestUseContextFindsAncestorProvider(t *testing.T) {
	themeCtx := CreateContext("light")

	grandparent := &Instance{
		ID:        "grandparent",
		HookFrame: []HookSlot{},
		Providers: map[any]any{},
	}
	grandparent.Providers[themeCtx.id] = "dark"

	parent := &Instance{
		ID:        "parent",
		Parent:    grandparent,
		HookFrame: []HookSlot{},
	}

	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{instance: child}

	value, setter := themeCtx.UseContext(ctx)

	if value != "dark" {
		t.Errorf("expected to find ancestor value 'dark', got %q", value)
	}

	if setter == nil {
		t.Error("expected setter to be non-nil when provider exists")
	}
}

// TestNestedProviders verifies that closer provider shadows ancestor.
func TestNestedProviders(t *testing.T) {
	themeCtx := CreateContext("light")

	grandparent := &Instance{
		ID:        "grandparent",
		HookFrame: []HookSlot{},
		Providers: map[any]any{},
	}
	grandparent.Providers[themeCtx.id] = "dark"

	parent := &Instance{
		ID:        "parent",
		Parent:    grandparent,
		HookFrame: []HookSlot{},
		Providers: map[any]any{},
	}
	parent.Providers[themeCtx.id] = "blue"

	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{instance: child}

	value, _ := themeCtx.UseContext(ctx)

	if value != "blue" {
		t.Errorf("expected closer provider 'blue', got %q", value)
	}
}

// TestWithEqual verifies custom equality function is used.
func TestWithEqual(t *testing.T) {
	type Config struct {
		Theme string
		Debug bool
	}

	equalCalled := false
	configCtx := CreateContext(Config{Theme: "light"}).WithEqual(func(a, b Config) bool {
		equalCalled = true
		return a.Theme == b.Theme
	})

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}
	parent.Children = append(parent.Children, child)

	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{instance: parent, session: sess}

	_, setter := configCtx.UseProvider(ctx, Config{Theme: "dark", Debug: false})

	setter(Config{Theme: "dark", Debug: true})

	if !equalCalled {
		t.Error("expected custom equality function to be called")
	}

	if len(sess.DirtyQueue) != 0 {
		t.Error("expected no dirty children when custom equality returns true")
	}
}

// TestProviderCleanupOnUnmount verifies providers are cleared on unmount.
func TestProviderCleanupOnUnmount(t *testing.T) {
	themeCtx := CreateContext("light")

	inst := &Instance{
		ID:        "test",
		HookFrame: []HookSlot{},
		Providers: map[any]any{},
	}
	inst.Providers[themeCtx.id] = "dark"

	sess := &Session{}
	sess.cleanupInstance(inst)

	if inst.Providers != nil {
		t.Error("expected providers to be cleared on cleanup")
	}
}

// TestContextEpochPropagationNoExtraRenders verifies epoch update prevents extra renders.
func TestContextEpochPropagationNoExtraRenders(t *testing.T) {

	root := &Instance{
		ID:                   "root",
		Fn:                   func(*Ctx) work.Node { return nil },
		HookFrame:            []HookSlot{},
		Children:             []*Instance{},
		ContextEpoch:         0,
		CombinedContextEpoch: 0,
	}

	child := &Instance{
		ID:                   "child",
		Fn:                   func(*Ctx) work.Node { return nil },
		Parent:               root,
		HookFrame:            []HookSlot{},
		ParentContextEpoch:   0,
		ContextEpoch:         0,
		CombinedContextEpoch: 0,
	}
	root.Children = append(root.Children, child)

	sess := &Session{
		Root:       root,
		Components: map[string]*Instance{"root": root, "child": child},
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}

	flushRequests := 0
	sess.SetAutoFlush(func() {
		flushRequests++
	})

	root.NotifyContextChange(sess)

	if len(sess.DirtyQueue) != 1 {
		t.Fatalf("expected 1 dirty component, got %d", len(sess.DirtyQueue))
	}

	if flushRequests != 1 {
		t.Fatalf("expected 1 flush request, got %d", flushRequests)
	}

	child.ParentContextEpoch = root.CombinedContextEpoch

	sess.DirtyQueue = []*Instance{}
	sess.DirtySet = make(map[*Instance]struct{})
	sess.flushMu.Lock()
	sess.pendingFlush = false
	sess.flushMu.Unlock()

	root.NotifyContextChange(sess)

	if len(sess.DirtyQueue) != 1 {
		t.Fatalf("expected 1 dirty component after second change, got %d", len(sess.DirtyQueue))
	}

	if flushRequests != 2 {
		t.Fatalf("expected 2 flush requests, got %d", flushRequests)
	}
}

// TestContextEqualityPanicRecovery verifies panic in equality check is handled.
func TestContextEqualityPanicRecovery(t *testing.T) {
	type Uncomparable struct {
		Fn func()
	}

	ctx := CreateContext(Uncomparable{})

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Children:  []*Instance{},
	}
	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}
	parent.Children = append(parent.Children, child)

	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	pctx := &Ctx{instance: parent, session: sess}

	_, setter := ctx.UseProvider(pctx, Uncomparable{Fn: func() {}})

	defer func() {
		if r := recover(); r != nil {
			t.Errorf("setter panicked: %v", r)
		}
	}()

	setter(Uncomparable{Fn: func() {}})

	if len(sess.DirtyQueue) != 1 {
		t.Error("expected child to be marked dirty when equality panics")
	}
}

// TestUseContextReturnsValueAndSetter verifies UseContext returns both value and setter.
func TestUseContextReturnsValueAndSetter(t *testing.T) {
	themeCtx := CreateContext("light")

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Providers: map[any]any{},
		Children:  []*Instance{},
	}
	parent.Providers[themeCtx.id] = "dark"

	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}
	parent.Children = append(parent.Children, child)

	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{instance: child, session: sess}

	value, setter := themeCtx.UseContext(ctx)

	if value != "dark" {
		t.Errorf("expected 'dark', got %q", value)
	}

	if setter == nil {
		t.Fatal("expected setter to be non-nil")
	}

	setter("blue")

	if parent.Providers[themeCtx.id] != "blue" {
		t.Error("expected setter to update provider value to 'blue'")
	}
}

// TestUseContextValueReturnsOnlyValue verifies UseContextValue returns just the value.
func TestUseContextValueReturnsOnlyValue(t *testing.T) {
	themeCtx := CreateContext("light")

	parent := &Instance{
		ID:        "parent",
		HookFrame: []HookSlot{},
		Providers: map[any]any{},
	}
	parent.Providers[themeCtx.id] = "dark"

	child := &Instance{
		ID:        "child",
		Parent:    parent,
		HookFrame: []HookSlot{},
	}

	ctx := &Ctx{instance: child}

	value := themeCtx.UseContextValue(ctx)

	if value != "dark" {
		t.Errorf("expected 'dark', got %q", value)
	}
}

// TestTypeMismatchPanicsWithClearMessage verifies type mismatches produce clear panic messages.
func TestTypeMismatchPanicsWithClearMessage(t *testing.T) {
	themeCtx := CreateContext("light")

	inst := &Instance{
		ID:        "test",
		HookFrame: []HookSlot{},
		Providers: map[any]any{},
	}

	inst.Providers[themeCtx.id] = 123

	ctx := &Ctx{instance: inst}

	defer func() {
		r := recover()
		if r == nil {
			t.Fatal("expected panic on type mismatch")
		}
		msg, ok := r.(string)
		if !ok {
			t.Fatalf("expected string panic, got %T", r)
		}
		if !strings.Contains(msg, "type mismatch") {
			t.Errorf("expected panic message to contain 'type mismatch', got: %s", msg)
		}
	}()

	_, _ = themeCtx.UseContext(ctx)
}
