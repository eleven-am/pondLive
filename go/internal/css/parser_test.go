package css

import (
	"testing"
)

func TestParseSimpleRule(t *testing.T) {
	css := `.card { padding: 1rem; }`
	result := Parse(css)

	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}

	rule := result.Rules[0]
	if rule.Selector != ".card" {
		t.Errorf("expected selector '.card', got %q", rule.Selector)
	}

	if rule.Declarations != "padding: 1rem;" {
		t.Errorf("expected declarations 'padding: 1rem;', got %q", rule.Declarations)
	}
}

func TestParseMultipleRules(t *testing.T) {
	css := `
		.card { padding: 1rem; }
		.title { font-size: 2rem; }
		#header { color: blue; }
	`
	result := Parse(css)

	if len(result.Rules) != 3 {
		t.Fatalf("expected 3 rules, got %d", len(result.Rules))
	}

	selectors := []string{".card", ".title", "#header"}
	for i, expected := range selectors {
		if result.Rules[i].Selector != expected {
			t.Errorf("rule %d: expected selector %q, got %q", i, expected, result.Rules[i].Selector)
		}
	}
}

func TestParseCommaSeparatedSelectors(t *testing.T) {
	css := `.card, .panel { padding: 1rem; }`
	result := Parse(css)

	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}

	if result.Rules[0].Selector != ".card, .panel" {
		t.Errorf("expected selector '.card, .panel', got %q", result.Rules[0].Selector)
	}

	if len(result.Selectors) != 2 {
		t.Fatalf("expected 2 unique selectors, got %d", len(result.Selectors))
	}

	if result.Selectors[0] != ".card" || result.Selectors[1] != ".panel" {
		t.Errorf("expected [.card, .panel], got %v", result.Selectors)
	}
}

func TestParsePseudoClasses(t *testing.T) {
	css := `.card:hover { background: gray; }`
	result := Parse(css)

	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}

	if result.Rules[0].Selector != ".card:hover" {
		t.Errorf("expected selector '.card:hover', got %q", result.Rules[0].Selector)
	}
}

func TestParseDescendantSelectors(t *testing.T) {
	css := `.card .title { font-size: 1.5rem; }`
	result := Parse(css)

	if len(result.Rules) != 1 {
		t.Fatalf("expected 1 rule, got %d", len(result.Rules))
	}

	if result.Rules[0].Selector != ".card .title" {
		t.Errorf("expected selector '.card .title', got %q", result.Rules[0].Selector)
	}
}

func TestParseMediaQuery(t *testing.T) {
	css := `
		.card { padding: 1rem; }
		@media (max-width: 768px) {
			.card { padding: 0.5rem; }
			.title { font-size: 1rem; }
		}
	`
	result := Parse(css)

	if len(result.Rules) != 1 {
		t.Errorf("expected 1 regular rule, got %d", len(result.Rules))
	}

	if len(result.MediaRules) != 1 {
		t.Fatalf("expected 1 media rule, got %d", len(result.MediaRules))
	}

	media := result.MediaRules[0]
	if media.Query != "(max-width: 768px)" {
		t.Errorf("expected query '(max-width: 768px)', got %q", media.Query)
	}

	if len(media.Rules) != 2 {
		t.Fatalf("expected 2 rules inside media query, got %d", len(media.Rules))
	}
}

func TestParseComments(t *testing.T) {
	css := `
		/* This is a comment */
		.card { padding: 1rem; } /* inline comment */
		/* Another comment */
		.title { font-size: 2rem; }
	`
	result := Parse(css)

	if len(result.Rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(result.Rules))
	}
}

func TestExtractBaseSelector(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{".card", ".card"},
		{".card:hover", ".card"},
		{".card::before", ".card"},
		{".card.active", ".card"},
		{"#header", "#header"},
		{".parent .child", ".parent"},
		{".parent > .child", ".parent"},
		{".parent + .child", ".parent"},
		{".parent ~ .child", ".parent"},
	}

	for _, tt := range tests {
		result := ExtractBaseSelector(tt.input)
		if result != tt.expected {
			t.Errorf("ExtractBaseSelector(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
