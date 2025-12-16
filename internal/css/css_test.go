package css

import (
	"strings"
	"testing"
)

func getDeclValue(decls []Declaration, prop string) string {
	for _, d := range decls {
		if d.Property == prop {
			return d.Value
		}
	}
	return ""
}

func TestParseAndScopeSimple(t *testing.T) {
	ss := ParseAndScope(`.btn { color: red; padding: 4px; }`, "component-1")
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ss.Rules))
	}
	selector := ss.Rules[0].Selector
	if selector == ".btn" || selector == "" {
		t.Fatalf("expected scoped selector, got %q", selector)
	}
	decls := ss.Rules[0].Decls
	if getDeclValue(decls, "color") != "red" || getDeclValue(decls, "padding") != "4px" {
		t.Fatalf("unexpected decls: %#v", decls)
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
	if getDeclValue(mr.Rules[0].Decls, "margin") != "0" {
		t.Fatalf("unexpected decls in media rule: %#v", mr.Rules[0].Decls)
	}
}

func TestParseAndScopeBasicRule(t *testing.T) {
	ss := ParseAndScope(`.foo { color: blue; }`, "component-3")
	if getDeclValue(ss.Rules[0].Decls, "color") != "blue" {
		t.Fatalf("decl not preserved")
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
		if getDeclValue(rule.Decls, "color") != "green" {
			t.Fatalf("expected color decl, got %#v", rule.Decls)
		}
		if getDeclValue(rule.Decls, "invalid") != "" {
			t.Fatalf("invalid declaration should be skipped: %#v", rule.Decls)
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
		if getDeclValue(rule.Decls, "padding") != "8px" {
			t.Fatalf("missing padding in media rule %#v", rule.Decls)
		}
		if !strings.Contains(rule.Selector, hash) {
			t.Fatalf("selector %q missing hash %q", rule.Selector, hash)
		}
	}
}

func TestDeclarationOrderPreserved(t *testing.T) {
	ss := ParseAndScope(`.foo { z-index: 1; color: blue; }`, "order-comp")
	if len(ss.Rules[0].Decls) != 2 {
		t.Fatalf("expected 2 declarations, got %d", len(ss.Rules[0].Decls))
	}
	if ss.Rules[0].Decls[0].Property != "z-index" || ss.Rules[0].Decls[1].Property != "color" {
		t.Fatalf("expected properties in declaration order, got %#v", ss.Rules[0].Decls)
	}
}

func TestDeclarationDuplicatesPreserved(t *testing.T) {
	ss := ParseAndScope(`.foo { background: red; background: blue; color: green; }`, "order-dup")
	decls := ss.Rules[0].Decls
	if len(decls) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(decls))
	}
	if decls[0].Property != "background" || decls[0].Value != "red" {
		t.Fatalf("expected first background:red, got %#v", decls[0])
	}
	if decls[1].Property != "background" || decls[1].Value != "blue" {
		t.Fatalf("expected second background:blue, got %#v", decls[1])
	}
	if decls[2].Property != "color" || decls[2].Value != "green" {
		t.Fatalf("expected color:green, got %#v", decls[2])
	}
}

func TestParseDeclarationsWithSemicolonsInValues(t *testing.T) {
	ss := ParseAndScope(`.foo { background: url(data:image/svg+xml;utf8,<svg></svg>); color: black; }`, "semicolon")
	if got := getDeclValue(ss.Rules[0].Decls, "background"); !strings.Contains(got, ";utf8,") {
		t.Fatalf("expected semicolon inside value preserved, got %q", got)
	}
}

func TestParsePreservesUnknownAtRules(t *testing.T) {
	ss := ParseAndScope(`@supports (display: grid) { .a { color: red; } } .b { color: blue; }`, "atrule")
	if len(ss.OtherBlocks) != 1 {
		t.Fatalf("expected 1 other block, got %d", len(ss.OtherBlocks))
	}
	if !strings.Contains(ss.OtherBlocks[0], "@supports (display: grid)") {
		t.Fatalf("expected @supports block preserved, got %q", ss.OtherBlocks[0])
	}
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 regular rule, got %d", len(ss.Rules))
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

func TestParseKeyframes(t *testing.T) {
	cssInput := `@keyframes fadeIn {
		0% { opacity: 0; }
		100% { opacity: 1; }
	}`
	ss := ParseAndScope(cssInput, "kf-comp")
	if len(ss.Keyframes) != 1 {
		t.Fatalf("expected 1 keyframes block, got %d", len(ss.Keyframes))
	}
	kf := ss.Keyframes[0]
	hash := hashComponent("kf-comp")
	if !strings.Contains(kf.Name, "fadeIn-"+hash) {
		t.Fatalf("expected keyframes name to be scoped, got %q", kf.Name)
	}
	if len(kf.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(kf.Steps))
	}
	if kf.Steps[0].Selector != "0%" || kf.Steps[1].Selector != "100%" {
		t.Fatalf("unexpected step selectors: %q, %q", kf.Steps[0].Selector, kf.Steps[1].Selector)
	}
}

func TestParseKeyframesEmptyComponentID(t *testing.T) {
	cssInput := `@keyframes slideIn { from { transform: translateX(-100%); } to { transform: translateX(0); } }`
	ss := ParseAndScope(cssInput, "")
	if len(ss.Keyframes) != 1 {
		t.Fatalf("expected 1 keyframes block, got %d", len(ss.Keyframes))
	}
	if ss.Keyframes[0].Name != "slideIn" {
		t.Fatalf("expected keyframes name unchanged with empty componentID, got %q", ss.Keyframes[0].Name)
	}
}

func TestKeyframesStructure(t *testing.T) {
	cssInput := `@keyframes bounce { 0% { transform: scale(1); } 50% { transform: scale(1.2); } 100% { transform: scale(1); } }`
	ss := ParseAndScope(cssInput, "bounce-comp")
	hash := hashComponent("bounce-comp")
	if len(ss.Keyframes) != 1 {
		t.Fatalf("expected 1 keyframes block, got %d", len(ss.Keyframes))
	}
	kf := ss.Keyframes[0]
	if !strings.Contains(kf.Name, "bounce-"+hash) {
		t.Fatalf("expected keyframes name scoped, got %q", kf.Name)
	}
	if len(kf.Steps) != 3 {
		t.Fatalf("expected 3 steps, got %d", len(kf.Steps))
	}
	if kf.Steps[0].Selector != "0%" || kf.Steps[1].Selector != "50%" || kf.Steps[2].Selector != "100%" {
		t.Fatalf("unexpected step selectors")
	}
	if getDeclValue(kf.Steps[0].Decls, "transform") != "scale(1)" {
		t.Fatalf("expected transform in first step")
	}
}

func TestParseMultipleKeyframes(t *testing.T) {
	cssInput := `
		@keyframes fadeIn { from { opacity: 0; } to { opacity: 1; } }
		@keyframes fadeOut { from { opacity: 1; } to { opacity: 0; } }
		.btn { color: red; }
	`
	ss := ParseAndScope(cssInput, "multi-kf")
	if len(ss.Keyframes) != 2 {
		t.Fatalf("expected 2 keyframes blocks, got %d", len(ss.Keyframes))
	}
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 regular rule, got %d", len(ss.Rules))
	}
}

func TestKeyframesWithMediaQuery(t *testing.T) {
	cssInput := `
		@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }
		@media screen { .card { margin: 0; } }
	`
	ss := ParseAndScope(cssInput, "mixed")
	if len(ss.Keyframes) != 1 {
		t.Fatalf("expected 1 keyframes block, got %d", len(ss.Keyframes))
	}
	if len(ss.MediaRules) != 1 {
		t.Fatalf("expected 1 media rule, got %d", len(ss.MediaRules))
	}
}

func TestMediaRuleStructure(t *testing.T) {
	cssInput := `@media (min-width: 768px) { .container { max-width: 720px; padding: 1rem; } }`
	ss := ParseAndScope(cssInput, "media-ser")
	if len(ss.MediaRules) != 1 {
		t.Fatalf("expected 1 media rule, got %d", len(ss.MediaRules))
	}
	mr := ss.MediaRules[0]
	if mr.Query != "(min-width: 768px)" {
		t.Fatalf("expected media query preserved, got %q", mr.Query)
	}
	if len(mr.Rules) != 1 {
		t.Fatalf("expected 1 rule in media, got %d", len(mr.Rules))
	}
	if getDeclValue(mr.Rules[0].Decls, "max-width") != "720px" {
		t.Fatalf("expected max-width in media rule")
	}
	if getDeclValue(mr.Rules[0].Decls, "padding") != "1rem" {
		t.Fatalf("expected padding in media rule")
	}
}

func TestMultipleMediaRules(t *testing.T) {
	cssInput := `
		@media screen { .a { color: red; } }
		@media print { .b { color: black; } }
	`
	ss := ParseAndScope(cssInput, "multi-media")
	if len(ss.MediaRules) != 2 {
		t.Fatalf("expected 2 media rules, got %d", len(ss.MediaRules))
	}
	if ss.MediaRules[0].Query != "screen" {
		t.Fatalf("expected @media screen, got %q", ss.MediaRules[0].Query)
	}
	if ss.MediaRules[1].Query != "print" {
		t.Fatalf("expected @media print, got %q", ss.MediaRules[1].Query)
	}
}

func TestCompleteStylesheetStructure(t *testing.T) {
	cssInput := `
		.btn { color: blue; background: white; }
		@media screen { .card { margin: 1rem; } }
		@keyframes fadeIn { 0% { opacity: 0; } 100% { opacity: 1; } }
		@supports (display: grid) { .grid { display: grid; } }
	`
	ss := ParseAndScope(cssInput, "complete")
	hash := hashComponent("complete")
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ss.Rules))
	}
	if getDeclValue(ss.Rules[0].Decls, "color") != "blue" {
		t.Fatalf("expected color in rule")
	}
	if len(ss.MediaRules) != 1 {
		t.Fatalf("expected 1 media rule, got %d", len(ss.MediaRules))
	}
	if ss.MediaRules[0].Query != "screen" {
		t.Fatalf("expected @media screen, got %q", ss.MediaRules[0].Query)
	}
	if len(ss.Keyframes) != 1 {
		t.Fatalf("expected 1 keyframes, got %d", len(ss.Keyframes))
	}
	if !strings.Contains(ss.Keyframes[0].Name, "fadeIn-"+hash) {
		t.Fatalf("expected keyframes name scoped, got %q", ss.Keyframes[0].Name)
	}
	if len(ss.OtherBlocks) != 1 {
		t.Fatalf("expected 1 other block, got %d", len(ss.OtherBlocks))
	}
	if !strings.Contains(ss.OtherBlocks[0], "@supports") {
		t.Fatalf("expected @supports block, got %q", ss.OtherBlocks[0])
	}
}

func TestOtherBlocksPreserved(t *testing.T) {
	cssInput := `@supports (display: flex) { .flex { display: flex; } }`
	ss := ParseAndScope(cssInput, "other")
	if len(ss.OtherBlocks) != 1 {
		t.Fatalf("expected 1 other block, got %d", len(ss.OtherBlocks))
	}
	if !strings.Contains(ss.OtherBlocks[0], "@supports (display: flex)") {
		t.Fatalf("expected @supports preserved, got %q", ss.OtherBlocks[0])
	}
}

func TestDeclsPreserveOrderAndDuplicates(t *testing.T) {
	cssInput := `.test { background: red; background: blue; color: green; }`
	ss := ParseAndScope(cssInput, "decl-test")
	if len(ss.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(ss.Rules))
	}
	decls := ss.Rules[0].Decls
	if len(decls) != 3 {
		t.Fatalf("expected 3 declarations, got %d", len(decls))
	}
	if decls[0].Property != "background" || decls[0].Value != "red" {
		t.Fatalf("expected first decl background:red, got %#v", decls[0])
	}
	if decls[1].Property != "background" || decls[1].Value != "blue" {
		t.Fatalf("expected second decl background:blue, got %#v", decls[1])
	}
	if decls[2].Property != "color" || decls[2].Value != "green" {
		t.Fatalf("expected third decl color:green, got %#v", decls[2])
	}
}

func TestKeyframeStepsUseDecls(t *testing.T) {
	cssInput := `@keyframes test { 0% { opacity: 0; transform: scale(0); } 100% { opacity: 1; transform: scale(1); } }`
	ss := ParseAndScope(cssInput, "kf-decls")
	if len(ss.Keyframes) != 1 {
		t.Fatalf("expected 1 keyframes, got %d", len(ss.Keyframes))
	}
	kf := ss.Keyframes[0]
	if len(kf.Steps) != 2 {
		t.Fatalf("expected 2 steps, got %d", len(kf.Steps))
	}
	if len(kf.Steps[0].Decls) != 2 {
		t.Fatalf("expected 2 declarations in first step, got %d", len(kf.Steps[0].Decls))
	}
}
