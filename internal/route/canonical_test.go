package route

import (
	"net/url"
	"testing"
)

func TestCanonicalizeValues(t *testing.T) {
	input := url.Values{
		"b": {"2"},
		"a": {"1"},
		"c": {"3", "4"},
	}

	result := canonicalValues(input)

	if result.Get("a") != "1" {
		t.Error("expected a=1")
	}
	if result.Get("b") != "2" {
		t.Error("expected b=2")
	}
	if vals := result["c"]; len(vals) != 2 || vals[0] != "3" || vals[1] != "4" {
		t.Error("expected c=[3,4]")
	}
}

func TestCanonicalizeValues_Nil(t *testing.T) {
	result := canonicalValues(nil)
	if result == nil {
		t.Error("expected non-nil result for nil input")
	}
}

func TestCanonicalizeList(t *testing.T) {
	tests := []struct {
		input    []string
		expected []string
	}{
		{nil, []string{}},
		{[]string{}, []string{}},
		{[]string{" b ", "a"}, []string{"a", "b"}},
		{[]string{"c", "a", "b"}, []string{"a", "b", "c"}},
	}

	for _, tc := range tests {
		result := canonicalList(tc.input)
		if len(result) != len(tc.expected) {
			t.Errorf("canonicalList(%v) length = %d, want %d", tc.input, len(result), len(tc.expected))
			continue
		}
		for i := range result {
			if result[i] != tc.expected[i] {
				t.Errorf("canonicalList(%v)[%d] = %q, want %q", tc.input, i, result[i], tc.expected[i])
			}
		}
	}
}

func TestFromNavigationKeepsPartialQueryOnMalformedInput(t *testing.T) {
	loc := FromNavigation("", "a=1&%zz", "#h")
	if loc.Path != "/" || loc.Hash != "h" || len(loc.Query["a"]) != 1 {
		t.Fatalf("loc = %+v", loc)
	}
}
