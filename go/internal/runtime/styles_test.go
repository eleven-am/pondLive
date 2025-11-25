package runtime

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/work"
)

// TestUseStylesBasic verifies that UseStyles parses CSS and creates scoped rules.
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
	if styles.stylesheet == nil {
		t.Fatal("expected non-nil stylesheet")
	}
	if len(styles.stylesheet.Rules) != 2 {
		t.Errorf("expected 2 rules, got %d", len(styles.stylesheet.Rules))
	}
}

// TestUseStylesClassMethod verifies the Class method returns scoped names.
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

// TestUseStylesStyleTag verifies StyleTag returns a valid work.Element.
func TestUseStylesStyleTag(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles := UseStyles(ctx, `.card { background: #fff; }`)

	node := styles.StyleTag()
	elem, ok := node.(*work.Element)
	if !ok {
		t.Fatal("StyleTag did not return *work.Element")
	}
	if elem.Tag != "style" {
		t.Errorf("expected tag 'style', got %q", elem.Tag)
	}
	if elem.Stylesheet == nil {
		t.Error("expected non-nil Stylesheet on style tag")
	}
	if elem.Stylesheet.Hash != styles.hash {
		t.Errorf("hash mismatch: %q != %q", elem.Stylesheet.Hash, styles.hash)
	}
}

// TestUseStylesMediaQueries verifies media queries are parsed correctly.
func TestUseStylesMediaQueries(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles := UseStyles(ctx, `
		.container { width: 100%; }
		@media (min-width: 768px) {
			.container { width: 750px; }
		}
		@media (min-width: 1024px) {
			.container { width: 960px; }
			.sidebar { display: block; }
		}
	`)

	if len(styles.stylesheet.Rules) != 1 {
		t.Errorf("expected 1 top-level rule, got %d", len(styles.stylesheet.Rules))
	}
	if len(styles.stylesheet.MediaBlocks) != 2 {
		t.Errorf("expected 2 media blocks, got %d", len(styles.stylesheet.MediaBlocks))
	}

	media1 := styles.stylesheet.MediaBlocks[0]
	if media1.Query != "(min-width: 768px)" {
		t.Errorf("unexpected media query: %q", media1.Query)
	}
	if len(media1.Rules) != 1 {
		t.Errorf("expected 1 rule in first media block, got %d", len(media1.Rules))
	}

	media2 := styles.stylesheet.MediaBlocks[1]
	if len(media2.Rules) != 2 {
		t.Errorf("expected 2 rules in second media block, got %d", len(media2.Rules))
	}
}

// TestUseStylesStable verifies that UseStyles returns the same instance across renders.
func TestUseStylesStable(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}

	ctx1 := &Ctx{instance: inst, hookIndex: 0}
	styles1 := UseStyles(ctx1, `.card { background: #fff; }`)

	ctx2 := &Ctx{instance: inst, hookIndex: 0}
	styles2 := UseStyles(ctx2, `.card { background: #fff; }`)

	if styles1 != styles2 {
		t.Error("UseStyles should return same instance across renders")
	}
}

// TestUseStylesSelectorsScoped verifies selectors include the component hash.
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

	for _, rule := range styles.stylesheet.Rules {
		if !strings.Contains(rule.Selector, styles.hash) {
			t.Errorf("selector %q should contain hash %q", rule.Selector, styles.hash)
		}
	}
}

// TestUseStylesEmptyCSS verifies handling of empty CSS.
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
	if len(styles.stylesheet.Rules) != 0 {
		t.Errorf("expected 0 rules for empty CSS, got %d", len(styles.stylesheet.Rules))
	}
}

// TestUseStylesPanicOutsideRender verifies panic when called outside render.
func TestUseStylesPanicOutsideRender(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when called with nil ctx")
		}
	}()

	UseStyles(nil, ".foo { color: red; }")
}

// TestUseStylesPanicNilInstance verifies panic when instance is nil.
func TestUseStylesPanicNilInstance(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic when called with nil instance")
		}
	}()

	ctx := &Ctx{instance: nil}
	UseStyles(ctx, ".foo { color: red; }")
}

// TestUseStylesHookMismatch verifies panic on hook type mismatch.
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

// TestUseStylesMultipleHooks verifies multiple UseStyles calls in same component.
func TestUseStylesMultipleHooks(t *testing.T) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	ctx := &Ctx{instance: inst, hookIndex: 0}

	styles1 := UseStyles(ctx, `.card { background: white; }`)
	styles2 := UseStyles(ctx, `.btn { padding: 10px; }`)

	if styles1 == styles2 {
		t.Error("different UseStyles calls should return different instances")
	}

	if styles1.hash != styles2.hash {
		t.Error("styles in same component should have same hash")
	}
	if len(inst.HookFrame) != 2 {
		t.Errorf("expected 2 hook slots, got %d", len(inst.HookFrame))
	}
}

// TestUseStylesClassMethodEmptyHash verifies Class returns unmodified name when hash is empty.
func TestUseStylesClassMethodEmptyHash(t *testing.T) {
	styles := &Styles{hash: "", stylesheet: nil}

	className := styles.Class("card")
	if className != "card" {
		t.Errorf("expected 'card' when hash empty, got %q", className)
	}
}
