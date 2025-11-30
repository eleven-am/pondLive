package css

import (
	"strings"
	"testing"
)

func TestParseAndScopeSimple(t *testing.T) {
	ss := ParseAndScope(`.btn { color: red; padding: 4px; }`, "component-1")
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ss.Rules))
	}
	selector := ss.Rules[0].Selector
	if selector == ".btn" || selector == "" {
		t.Fatalf("expected scoped selector, got %q", selector)
	}
	props := ss.Rules[0].Props
	if props["color"] != "red" || props["padding"] != "4px" {
		t.Fatalf("unexpected props: %#v", props)
	}
}

func TestParseAndScopeMedia(t *testing.T) {
	cssInput := `@media screen { .card { margin: 0; } }`
	ss := ParseAndScope(cssInput, "component-2")
	if len(ss.MediaRules) != 1 {
		t.Fatalf("expected 1 media rule, got %d", len(ss.MediaRules))
	}
	mr := ss.MediaRules[0]
	if mr.Query != "screen" {
		t.Fatalf("unexpected media query %q", mr.Query)
	}
	if len(mr.Rules) != 1 {
		t.Fatalf("expected 1 rule inside media, got %d", len(mr.Rules))
	}
	if mr.Rules[0].Props["margin"] != "0" {
		t.Fatalf("unexpected props in media rule: %#v", mr.Rules[0].Props)
	}
}

func TestSerialize(t *testing.T) {
	ss := ParseAndScope(`.foo { color: blue; }`, "component-3")
	out := ss.Serialize()
	if out == "" {
		t.Fatalf("expected non-empty serialization")
	}
	if ss.Rules[0].Props["color"] != "blue" {
		t.Fatalf("prop not preserved")
	}
	if ss.Rules[0].Selector == ".foo" {
		t.Fatalf("selector not scoped: %q", ss.Rules[0].Selector)
	}
}

func TestScopeIgnoresRootSelectors(t *testing.T) {
	ss := ParseAndScope(`:root { --color: black; } body { margin: 0; } .wrapper { padding: 1rem; }`, "root-comp")
	if len(ss.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(ss.Rules))
	}
	root := ss.Rules[0]
	if root.Selector != ":root" {
		t.Fatalf("expected :root to remain unchanged, got %q", root.Selector)
	}
	body := ss.Rules[1]
	if body.Selector != "body" {
		t.Fatalf("expected body to remain unchanged, got %q", body.Selector)
	}
	hash := hashComponent("root-comp")
	wrapper := ss.Rules[2]
	if !strings.Contains(wrapper.Selector, hash) {
		t.Fatalf("expected wrapper selector to include hash %q, got %q", hash, wrapper.Selector)
	}
}

func TestScopeComplexSelectors(t *testing.T) {
	cssInput := `.nav > li.active:hover::before { content: ""; }`
	ss := ParseAndScope(cssInput, "complex-comp")
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ss.Rules))
	}
	selector := ss.Rules[0].Selector
	hash := hashComponent("complex-comp")
	if !strings.Contains(selector, hash) {
		t.Fatalf("expected selector to contain hash %q, got %q", hash, selector)
	}
	if !strings.Contains(selector, ":hover::before") {
		t.Fatalf("expected pseudo selectors preserved, got %q", selector)
	}
}

func TestScopeMultipleSelectorsAndInvalidDeclarations(t *testing.T) {
	cssInput := `.a, .b { color: green; invalid }`
	ss := ParseAndScope(cssInput, "multi-comp")
	if len(ss.Rules) != 2 {
		t.Fatalf("expected 2 scoped selectors, got %d", len(ss.Rules))
	}
	for _, rule := range ss.Rules {
		if rule.Props["color"] != "green" {
			t.Fatalf("expected color prop, got %#v", rule.Props)
		}
		if _, bad := rule.Props["invalid"]; bad {
			t.Fatalf("invalid declaration should be skipped: %#v", rule.Props)
		}
	}
}

func TestScopeMediaRuleMultipleSelectors(t *testing.T) {
	cssInput := `@media (min-width: 600px) { .x, .y:hover { padding: 8px; } }`
	ss := ParseAndScope(cssInput, "media-comp")
	if len(ss.MediaRules) != 1 {
		t.Fatalf("expected 1 media rule, got %d", len(ss.MediaRules))
	}
	mr := ss.MediaRules[0]
	if mr.Query != "(min-width: 600px)" {
		t.Fatalf("unexpected media query %q", mr.Query)
	}
	if len(mr.Rules) != 2 {
		t.Fatalf("expected 2 selectors inside media, got %d", len(mr.Rules))
	}
	hash := hashComponent("media-comp")
	for _, rule := range mr.Rules {
		if rule.Props["padding"] != "8px" {
			t.Fatalf("missing padding in media rule %#v", rule.Props)
		}
		if !strings.Contains(rule.Selector, hash) {
			t.Fatalf("selector %q missing hash %q", rule.Selector, hash)
		}
	}
}

func TestSerializeOrdersProperties(t *testing.T) {
	ss := ParseAndScope(`.foo { z-index: 1; color: blue; }`, "order-comp")
	out := ss.Serialize()
	if !strings.Contains(out, "z-index: 1; color: blue") {
		t.Fatalf("expected properties in declaration order, got %q", out)
	}
}

func TestSerializePreservesDeclarationOrderAndDuplicates(t *testing.T) {
	ss := ParseAndScope(`.foo { background: red; background: blue; color: green; }`, "order-dup")
	out := ss.Serialize()
	if !strings.Contains(out, "background: red; background: blue; color: green") {
		t.Fatalf("expected declaration order and duplicates preserved, got %q", out)
	}
}

func TestParseDeclarationsWithSemicolonsInValues(t *testing.T) {
	ss := ParseAndScope(`.foo { background: url(data:image/svg+xml;utf8,<svg></svg>); color: black; }`, "semicolon")
	if got := ss.Rules[0].Props["background"]; !strings.Contains(got, ";utf8,") {
		t.Fatalf("expected semicolon inside value preserved, got %q", got)
	}
}

func TestScopeTagClassSelector(t *testing.T) {
	ss := ParseAndScope(`button.btn { color: red; }`, "comp")
	hash := hashComponent("comp")
	if !strings.Contains(ss.Rules[0].Selector, ".btn-"+hash) {
		t.Fatalf("expected .btn to be scoped, got %q", ss.Rules[0].Selector)
	}
}

func TestScopeTagIdSelector(t *testing.T) {
	ss := ParseAndScope(`div#main { color: red; }`, "comp")
	hash := hashComponent("comp")
	if !strings.Contains(ss.Rules[0].Selector, "#main-"+hash) {
		t.Fatalf("expected #main to be scoped, got %q", ss.Rules[0].Selector)
	}
}

func TestScopePseudoFunctionSelectors(t *testing.T) {
	ss := ParseAndScope(`:is(.foo, .bar) { color: red; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".foo-"+hash) || !strings.Contains(sel, ".bar-"+hash) {
		t.Fatalf("expected classes inside :is() to be scoped, got %q", sel)
	}
}

func TestScopeWherePseudo(t *testing.T) {
	ss := ParseAndScope(`:where(.card) { padding: 1rem; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".card-"+hash) {
		t.Fatalf("expected .card inside :where() to be scoped, got %q", sel)
	}
}

func TestScopeNotPseudo(t *testing.T) {
	ss := ParseAndScope(`.card:not(.active) { opacity: 0.5; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".card-"+hash) || !strings.Contains(sel, ".active-"+hash) {
		t.Fatalf("expected both classes scoped, got %q", sel)
	}
}

func TestMultilineComment(t *testing.T) {
	css := "/* multi\nline\ncomment */ .foo { color: red; }"
	ss := ParseAndScope(css, "comp")
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 rule after stripping multiline comment, got %d", len(ss.Rules))
	}
	hash := hashComponent("comp")
	if !strings.Contains(ss.Rules[0].Selector, ".foo-"+hash) {
		t.Fatalf("expected .foo to be scoped, got %q", ss.Rules[0].Selector)
	}
}

func TestScopeClassChain(t *testing.T) {
	ss := ParseAndScope(`.foo.bar { color: red; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".foo-"+hash) {
		t.Fatalf("expected .foo to be scoped, got %q", sel)
	}
	if !strings.Contains(sel, ".bar-"+hash) {
		t.Fatalf("expected .bar to be scoped, got %q", sel)
	}
}

func TestScopeTripleClassChain(t *testing.T) {
	ss := ParseAndScope(`.a.b.c { color: red; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".a-"+hash) || !strings.Contains(sel, ".b-"+hash) || !strings.Contains(sel, ".c-"+hash) {
		t.Fatalf("expected all classes scoped, got %q", sel)
	}
}

func TestScopeAttributeWithClass(t *testing.T) {
	ss := ParseAndScope(`[type=text].btn { color: red; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".btn-"+hash) {
		t.Fatalf("expected .btn to be scoped, got %q", sel)
	}
	if !strings.Contains(sel, "[type=text]") {
		t.Fatalf("expected attribute preserved, got %q", sel)
	}
}

func TestScopeClassWithAttribute(t *testing.T) {
	ss := ParseAndScope(`.input[type=text] { color: red; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".input-"+hash) {
		t.Fatalf("expected .input to be scoped, got %q", sel)
	}
	if !strings.Contains(sel, "[type=text]") {
		t.Fatalf("expected attribute preserved, got %q", sel)
	}
}

func TestScopeMultipleClasses(t *testing.T) {
	ss := ParseAndScope(`.a .b { color: red; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".a-"+hash) || !strings.Contains(sel, ".b-"+hash) {
		t.Fatalf("expected both classes scoped, got %q", sel)
	}
}

func TestScopeDescendantCombinator(t *testing.T) {
	ss := ParseAndScope(`.parent > .child { margin: 0; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".parent-"+hash) || !strings.Contains(sel, ".child-"+hash) {
		t.Fatalf("expected both classes scoped, got %q", sel)
	}
}

func TestScopeSiblingCombinator(t *testing.T) {
	ss := ParseAndScope(`.first + .second { padding: 0; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".first-"+hash) || !strings.Contains(sel, ".second-"+hash) {
		t.Fatalf("expected both classes scoped, got %q", sel)
	}
}

func TestScopeSlottedPseudo(t *testing.T) {
	ss := ParseAndScope(`::slotted(.btn) { color: red; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".btn-"+hash) {
		t.Fatalf("expected .btn inside ::slotted() to be scoped, got %q", sel)
	}
}

func TestScopeCuePseudo(t *testing.T) {
	ss := ParseAndScope(`::cue(.caption) { color: white; }`, "comp")
	hash := hashComponent("comp")
	sel := ss.Rules[0].Selector
	if !strings.Contains(sel, ".caption-"+hash) {
		t.Fatalf("expected .caption inside ::cue() to be scoped, got %q", sel)
	}
}

func TestEmptyComponentID(t *testing.T) {
	ss := ParseAndScope(`.btn { color: red; }`, "")
	sel := ss.Rules[0].Selector
	if sel != ".btn" {
		t.Fatalf("expected .btn unchanged with empty componentID, got %q", sel)
	}
}

func TestEmptyComponentIDNoTrailingDash(t *testing.T) {
	ss := ParseAndScope(`.foo.bar { color: red; }`, "")
	sel := ss.Rules[0].Selector
	if strings.Contains(sel, "-") {
		t.Fatalf("selector should not have any dashes with empty componentID, got %q", sel)
	}
}
