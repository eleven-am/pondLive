package css

import (
	"strings"
	"testing"
)

func TestScopeSimpleClass(t *testing.T) {
	css := `.card { padding: 1rem; }`
	result := Scope(css, "comp-123")

	if !strings.Contains(result.CSS, ".card-") {
		t.Errorf("expected scoped class name, got: %s", result.CSS)
	}

	scopedClass, ok := result.SelectorMap[".card"]
	if !ok {
		t.Fatal("expected .card in selector map")
	}

	if !strings.HasPrefix(scopedClass, ".card-") {
		t.Errorf("expected scoped class to start with '.card-', got %q", scopedClass)
	}
}

func TestScopeSimpleID(t *testing.T) {
	css := `#header { color: blue; }`
	result := Scope(css, "comp-123")

	if !strings.Contains(result.CSS, "#header-") {
		t.Errorf("expected scoped ID, got: %s", result.CSS)
	}

	scopedID, ok := result.SelectorMap["#header"]
	if !ok {
		t.Fatal("expected #header in selector map")
	}

	if !strings.HasPrefix(scopedID, "#header-") {
		t.Errorf("expected scoped ID to start with '#header-', got %q", scopedID)
	}
}

func TestScopeMultipleSelectors(t *testing.T) {
	css := `
		.card { padding: 1rem; }
		.title { font-size: 2rem; }
		#header { color: blue; }
	`
	result := Scope(css, "comp-123")

	expectedKeys := []string{".card", ".title", "#header"}
	for _, key := range expectedKeys {
		if _, ok := result.SelectorMap[key]; !ok {
			t.Errorf("expected %q in selector map", key)
		}
	}
}

func TestScopePseudoClass(t *testing.T) {
	css := `.card:hover { background: gray; }`
	result := Scope(css, "comp-123")

	if !strings.Contains(result.CSS, ".card-") {
		t.Errorf("expected scoped class in pseudo-class selector, got: %s", result.CSS)
	}

	if !strings.Contains(result.CSS, ":hover") {
		t.Errorf("expected :hover preserved, got: %s", result.CSS)
	}
}

func TestScopeDescendantSelector(t *testing.T) {
	css := `.card .title { font-size: 1.5rem; }`
	result := Scope(css, "comp-123")

	if !strings.Contains(result.CSS, ".card-") {
		t.Errorf("expected scoped .card, got: %s", result.CSS)
	}

	if !strings.Contains(result.CSS, " .title") {
		t.Errorf("expected descendant selector preserved, got: %s", result.CSS)
	}
}

func TestScopeMediaQuery(t *testing.T) {
	css := `
		.card { padding: 1rem; }
		@media (max-width: 768px) {
			.card { padding: 0.5rem; }
		}
	`
	result := Scope(css, "comp-123")

	if !strings.Contains(result.CSS, "@media") {
		t.Error("expected @media query preserved")
	}

	parts := strings.Split(result.CSS, "\n")
	scopedCount := 0
	for _, part := range parts {
		if strings.Contains(part, ".card-") {
			scopedCount++
		}
	}

	if scopedCount < 2 {
		t.Errorf("expected .card scoped in both regular and media rules, got %d occurrences", scopedCount)
	}
}

func TestScopeCommaSeparatedSelectors(t *testing.T) {
	css := `.card, .panel { padding: 1rem; }`
	result := Scope(css, "comp-123")

	if !strings.Contains(result.CSS, ".card-") {
		t.Error("expected .card to be scoped")
	}

	if !strings.Contains(result.CSS, ".panel-") {
		t.Error("expected .panel to be scoped")
	}

	if _, ok := result.SelectorMap[".card"]; !ok {
		t.Error("expected .card in selector map")
	}

	if _, ok := result.SelectorMap[".panel"]; !ok {
		t.Error("expected .panel in selector map")
	}
}

func TestStyleLookupGet(t *testing.T) {
	selectorMap := map[string]string{
		".card":   ".card-abc123",
		"#header": "#header-abc123",
	}

	lookup := NewStyleLookup(selectorMap)

	if got := lookup.Get(".card"); got != "card-abc123" {
		t.Errorf("Get(.card) = %q, expected %q", got, "card-abc123")
	}

	if got := lookup.Get("#header"); got != "header-abc123" {
		t.Errorf("Get(#header) = %q, expected %q", got, "header-abc123")
	}
}

func TestStyleLookupClass(t *testing.T) {
	selectorMap := map[string]string{
		".card": ".card-abc123",
	}

	lookup := NewStyleLookup(selectorMap)

	if got := lookup.Class("card"); got != "card-abc123" {
		t.Errorf("Class(card) = %q, expected %q", got, "card-abc123")
	}

	if got := lookup.Class(".card"); got != "card-abc123" {
		t.Errorf("Class(.card) = %q, expected %q", got, "card-abc123")
	}
}

func TestStyleLookupID(t *testing.T) {
	selectorMap := map[string]string{
		"#header": "#header-abc123",
	}

	lookup := NewStyleLookup(selectorMap)

	if got := lookup.ID("header"); got != "header-abc123" {
		t.Errorf("ID(header) = %q, expected %q", got, "header-abc123")
	}

	if got := lookup.ID("#header"); got != "header-abc123" {
		t.Errorf("ID(#header) = %q, expected %q", got, "header-abc123")
	}
}

func TestStyleLookupCall(t *testing.T) {
	selectorMap := map[string]string{
		".card": ".card-abc123",
	}

	lookup := NewStyleLookup(selectorMap)

	if got := lookup.Call(".card"); got != "card-abc123" {
		t.Errorf("Call(.card) = %q, expected %q", got, "card-abc123")
	}
}

func TestSameComponentIDProducesSameHash(t *testing.T) {
	css := `.card { padding: 1rem; }`

	result1 := Scope(css, "comp-123")
	result2 := Scope(css, "comp-123")

	if result1.ComponentHash != result2.ComponentHash {
		t.Errorf("same component ID should produce same hash: %q vs %q", result1.ComponentHash, result2.ComponentHash)
	}

	if result1.SelectorMap[".card"] != result2.SelectorMap[".card"] {
		t.Errorf("same component ID should produce same scoped selectors")
	}
}

func TestDifferentComponentIDsProduceDifferentHashes(t *testing.T) {
	css := `.card { padding: 1rem; }`

	result1 := Scope(css, "comp-123")
	result2 := Scope(css, "comp-456")

	if result1.ComponentHash == result2.ComponentHash {
		t.Error("different component IDs should produce different hashes")
	}

	if result1.SelectorMap[".card"] == result2.SelectorMap[".card"] {
		t.Error("different component IDs should produce different scoped selectors")
	}
}
