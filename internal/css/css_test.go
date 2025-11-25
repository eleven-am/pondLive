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
	if !strings.Contains(out, "color: blue; z-index: 1") {
		t.Fatalf("expected properties sorted in serialization, got %q", out)
	}
}
