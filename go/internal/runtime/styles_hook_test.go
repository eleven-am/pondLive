package runtime

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestUseStylesBasic(t *testing.T) {
	var styles *Styles

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		styles = UseStyles(ctx, `
			.card {
				background: #fff;
				color: #333;
			}
			.btn {
				padding: 10px;
			}
		`)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

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

func TestUseStylesClassMethod(t *testing.T) {
	var styles *Styles

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		styles = UseStyles(ctx, `.card { background: #fff; }`)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	className := styles.Class("card")
	if !strings.HasPrefix(className, "card-") {
		t.Errorf("expected class to start with 'card-', got %q", className)
	}
	if className == "card" {
		t.Error("expected scoped class name, got unscoped")
	}
}

func TestUseStylesStyleTag(t *testing.T) {
	var styles *Styles

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		styles = UseStyles(ctx, `.card { background: #fff; }`)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	item := styles.StyleTag()
	node, ok := item.(*dom.StructuredNode)
	if !ok {
		t.Fatal("StyleTag did not return *dom.StructuredNode")
	}
	if node.Tag != "style" {
		t.Errorf("expected tag 'style', got %q", node.Tag)
	}
	if node.Stylesheet == nil {
		t.Error("expected non-nil Stylesheet on style tag")
	}
	if node.Stylesheet.Hash != styles.hash {
		t.Errorf("hash mismatch: %q != %q", node.Stylesheet.Hash, styles.hash)
	}
}

func TestUseStylesMediaQueries(t *testing.T) {
	var styles *Styles

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		styles = UseStyles(ctx, `
			.container { width: 100%; }
			@media (min-width: 768px) {
				.container { width: 750px; }
			}
			@media (min-width: 1024px) {
				.container { width: 960px; }
				.sidebar { display: block; }
			}
		`)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

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

func TestUseStylesStable(t *testing.T) {
	var styles1, styles2 *Styles
	renderCount := 0

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		renderCount++
		s := UseStyles(ctx, `.card { background: #fff; }`)
		if renderCount == 1 {
			styles1 = s
		} else {
			styles2 = s
		}
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	sess.markDirty(sess.root)
	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	if styles1 != styles2 {
		t.Error("UseStyles should return same instance across renders")
	}
}

func TestUseStylesSelectorsScoped(t *testing.T) {
	var styles *Styles

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		styles = UseStyles(ctx, `
			.card { background: #fff; }
			#main { color: red; }
		`)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	for _, rule := range styles.stylesheet.Rules {
		if !strings.Contains(rule.Selector, styles.hash) {
			t.Errorf("selector %q should contain hash %q", rule.Selector, styles.hash)
		}
	}
}

func TestUseStylesEmptyCSS(t *testing.T) {
	var styles *Styles

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		styles = UseStyles(ctx, "")
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if styles == nil {
		t.Fatal("UseStyles returned nil for empty CSS")
	}
	if len(styles.stylesheet.Rules) != 0 {
		t.Errorf("expected 0 rules for empty CSS, got %d", len(styles.stylesheet.Rules))
	}
}
