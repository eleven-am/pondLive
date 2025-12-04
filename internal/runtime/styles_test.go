package runtime

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/metadata"
)

func TestUseStylesBasic(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles := UseStyles(ctx, `
		.card {
			background: #fff;
			color: #333;
		}
		.btn {
			padding: 10px;
		}
	`)

	if styles == nil {
		t.Fatal("UseStyles returned nil")
	}
	if styles.hash == "" {
		t.Error("expected non-empty hash")
	}

	cell, ok := inst.HookFrame[0].Value.(*stylesCell)
	if !ok {
		t.Fatal("expected stylesCell in hook frame")
	}
	if cell.stylesheet == nil {
		t.Fatal("expected non-nil stylesheet in cell")
	}
	if len(cell.stylesheet.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(cell.stylesheet.Rules))
	}
}

func TestUseStylesClassMethod(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles := UseStyles(ctx, `.card { background: #fff; }`)

	className := styles.Class("card")
	if !strings.HasPrefix(className, "card-") {
		t.Errorf("expected class to start with 'card-', got %q", className)
	}
	if className == "card" {
		t.Error("expected scoped class name, got unscoped")
	}
}

func TestUseStylesAppendFnCalled(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	var capturedStylesheet *metadata.Stylesheet
	appendFn := func(_ *Ctx, s *metadata.Stylesheet) {
		capturedStylesheet = s
	}

	styles := UseStyles(ctx, `.card { background: #fff; }`, appendFn)

	if capturedStylesheet == nil {
		t.Fatal("append function was not called")
	}
	if capturedStylesheet.Hash != styles.hash {
		t.Errorf("hash mismatch: %q != %q", capturedStylesheet.Hash, styles.hash)
	}
	if len(capturedStylesheet.Rules) != 1 {
		t.Errorf("expected 1 rule, got %d", len(capturedStylesheet.Rules))
	}
}

func TestUseStylesAppendFnNotCalledOnRerender(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}

	callCount := 0
	appendFn := func(_ *Ctx, _ *metadata.Stylesheet) {
		callCount++
	}

	ctx1 := &Ctx{instance: inst, hookIndex: 0}
	UseStyles(ctx1, `.card { background: #fff; }`, appendFn)

	ctx2 := &Ctx{instance: inst, hookIndex: 0}
	UseStyles(ctx2, `.card { background: #fff; }`, appendFn)

	if callCount != 1 {
		t.Errorf("expected append function called once, got %d", callCount)
	}
}

func TestUseStylesMediaQueries(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	UseStyles(ctx, `
		.container { width: 100%; }
		@media (min-width: 768px) {
			.container { width: 750px; }
		}
		@media (min-width: 1024px) {
			.container { width: 960px; }
			.sidebar { display: block; }
		}
	`)

	cell, ok := inst.HookFrame[0].Value.(*stylesCell)
	if !ok {
		t.Fatal("expected stylesCell in hook frame")
	}

	if len(cell.stylesheet.Rules) != 1 {
		t.Errorf("expected 1 top-level rule, got %d", len(cell.stylesheet.Rules))
	}
	if len(cell.stylesheet.MediaBlocks) != 2 {
		t.Errorf("expected 2 media blocks, got %d", len(cell.stylesheet.MediaBlocks))
	}

	media1 := cell.stylesheet.MediaBlocks[0]
	if media1.Query != "(min-width: 768px)" {
		t.Errorf("unexpected media query: %q", media1.Query)
	}
	if len(media1.Rules) != 1 {
		t.Errorf("expected 1 rule in first media block, got %d", len(media1.Rules))
	}

	media2 := cell.stylesheet.MediaBlocks[1]
	if len(media2.Rules) != 2 {
		t.Errorf("expected 2 rules in second media block, got %d", len(media2.Rules))
	}
}

func TestUseStylesStable(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}

	ctx1 := &Ctx{instance: inst, hookIndex: 0}
	styles1 := UseStyles(ctx1, `.card { background: #fff; }`)

	ctx2 := &Ctx{instance: inst, hookIndex: 0}
	styles2 := UseStyles(ctx2, `.card { background: #fff; }`)

	if styles1.hash != styles2.hash {
		t.Error("UseStyles should return same hash across renders")
	}
}

func TestUseStylesSelectorsScoped(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles := UseStyles(ctx, `
		.card { background: #fff; }
		#main { color: red; }
	`)

	cell, ok := inst.HookFrame[0].Value.(*stylesCell)
	if !ok {
		t.Fatal("expected stylesCell in hook frame")
	}

	for _, rule := range cell.stylesheet.Rules {
		if !strings.Contains(rule.Selector, styles.hash) {
			t.Errorf("selector %q should contain hash %q", rule.Selector, styles.hash)
		}
	}
}

func TestUseStylesEmptyCSS(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles := UseStyles(ctx, "")

	if styles == nil {
		t.Fatal("UseStyles returned nil for empty CSS")
	}

	cell, ok := inst.HookFrame[0].Value.(*stylesCell)
	if !ok {
		t.Fatal("expected stylesCell in hook frame")
	}
	if len(cell.stylesheet.Rules) != 0 {
		t.Errorf("expected 0 rules for empty CSS, got %d", len(cell.stylesheet.Rules))
	}
}

func TestUseStylesPanicOutsideRender(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when called with nil ctx")
		}
	}()

	UseStyles(nil, ".foo { color: red; }")
}

func TestUseStylesPanicNilInstance(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when called with nil instance")
		}
	}()

	ctx := &Ctx{instance: nil}
	UseStyles(ctx, ".foo { color: red; }")
}

func TestUseStylesHookMismatch(t *testing.T) {
	inst := &Instance{
		ID: "test-comp",
		HookFrame: []HookSlot{
			{Type: HookTypeState, Value: &stateCell[int]{}},
		},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic on hook mismatch")
		}
	}()

	UseStyles(ctx, ".foo { color: red; }")
}

func TestUseStylesMultipleHooks(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles1 := UseStyles(ctx, `.card { background: white; }`)
	styles2 := UseStyles(ctx, `.btn { padding: 10px; }`)

	if styles1.hash == styles2.hash && styles1 == styles2 {
		t.Error("different UseStyles calls should return different instances")
	}

	if styles1.hash != styles2.hash {
		t.Error("styles in same component should have same hash")
	}
	if len(inst.HookFrame) != 2 {
		t.Errorf("expected 2 hook slots, got %d", len(inst.HookFrame))
	}
}

func TestUseStylesClassMethodEmptyHash(t *testing.T) {
	styles := &Styles{hash: ""}

	className := styles.Class("card")
	if className != "card" {
		t.Errorf("expected 'card' when hash empty, got %q", className)
	}
}
