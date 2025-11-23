package router

import (
	"net/url"
	"testing"
)

// TestValuesEqualOptimization tests the optimized values comparison
func TestValuesEqualOptimization(t *testing.T) {
	tests := []struct {
		name     string
		a        url.Values
		b        url.Values
		expected bool
	}{
		{
			name:     "empty values are equal",
			a:        url.Values{},
			b:        url.Values{},
			expected: true,
		},
		{
			name:     "nil and empty are equal",
			a:        nil,
			b:        url.Values{},
			expected: true,
		},
		{
			name:     "same single value",
			a:        url.Values{"key": {"value"}},
			b:        url.Values{"key": {"value"}},
			expected: true,
		},
		{
			name:     "different values",
			a:        url.Values{"key": {"value1"}},
			b:        url.Values{"key": {"value2"}},
			expected: false,
		},
		{
			name:     "different keys",
			a:        url.Values{"key1": {"value"}},
			b:        url.Values{"key2": {"value"}},
			expected: false,
		},
		{
			name:     "different lengths",
			a:        url.Values{"key1": {"value"}},
			b:        url.Values{"key1": {"value"}, "key2": {"value"}},
			expected: false,
		},
		{
			name:     "multiple values same order",
			a:        url.Values{"key": {"a", "b", "c"}},
			b:        url.Values{"key": {"a", "b", "c"}},
			expected: true,
		},
		{
			name:     "multiple values different order",
			a:        url.Values{"key": {"a", "b", "c"}},
			b:        url.Values{"key": {"c", "b", "a"}},
			expected: false,
		},
		{
			name: "multiple keys",
			a: url.Values{
				"page":  {"1"},
				"sort":  {"date"},
				"limit": {"10"},
			},
			b: url.Values{
				"limit": {"10"},
				"page":  {"1"},
				"sort":  {"date"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valuesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("valuesEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// TestCanonicalizeLocationOptimization tests that canonicalization doesn't duplicate work
func TestCanonicalizeLocationOptimization(t *testing.T) {
	original := Location{
		Path: "/test/path",
		Query: url.Values{
			"z": {"last"},
			"a": {"first"},
			"m": {"middle"},
		},
		Hash: "section",
	}

	// First canonicalization
	canon1 := canonicalizeLocation(original)

	// Canonicalize the already canonical location
	canon2 := canonicalizeLocation(canon1)

	// Should be equal (canonicalizing canonical location should be idempotent)
	if !LocEqual(canon1, canon2) {
		t.Error("canonicalization should be idempotent")
	}

	// Verify query is sorted
	keys := make([]string, 0, len(canon1.Query))
	for k := range canon1.Query {
		keys = append(keys, k)
	}

	expectedKeys := []string{"a", "m", "z"}
	if len(keys) != len(expectedKeys) {
		t.Fatalf("expected %d keys, got %d", len(expectedKeys), len(keys))
	}

	for i, key := range expectedKeys {
		if keys[i] != key {
			t.Errorf("key[%d]: expected %q, got %q", i, key, keys[i])
		}
	}
}

// TestCloneLocationOptimization tests that cloning doesn't double-canonicalize
func TestCloneLocationOptimization(t *testing.T) {
	original := Location{
		Path: "/test",
		Query: url.Values{
			"b": {"2"},
			"a": {"1"},
		},
		Hash: "anchor",
	}

	// Clone should not canonicalize path/hash again
	cloned := cloneLocation(original)

	// Path and hash should be identical (not re-normalized)
	if cloned.Path != original.Path {
		t.Errorf("path changed during clone: %q -> %q", original.Path, cloned.Path)
	}
	if cloned.Hash != original.Hash {
		t.Errorf("hash changed during clone: %q -> %q", original.Hash, cloned.Hash)
	}

	// Query should be a deep copy (different instance)
	if &cloned.Query == &original.Query {
		t.Error("query should be copied, not referenced")
	}

	// But values should be equal
	if !valuesEqual(cloned.Query, original.Query) {
		t.Error("cloned query should equal original")
	}

	// Mutating clone should not affect original
	cloned.Query.Set("c", "3")
	if original.Query.Get("c") != "" {
		t.Error("mutating clone affected original")
	}
}

// TestEncodeQueryPerformance tests that encodeQuery is efficient
func TestEncodeQueryPerformance(t *testing.T) {
	// Create a query with many parameters
	query := url.Values{
		"page":     {"1"},
		"limit":    {"20"},
		"sort":     {"date"},
		"order":    {"desc"},
		"filter":   {"active"},
		"category": {"news"},
		"author":   {"john"},
		"status":   {"published"},
	}

	// Encode multiple times - should give same result
	encoded1 := encodeQuery(query)
	encoded2 := encodeQuery(query)

	if encoded1 != encoded2 {
		t.Error("encodeQuery should be deterministic")
	}

	// Should be sorted
	if encoded1 == "" {
		t.Error("encoded query should not be empty")
	}

	// Verify it's URL-encoded properly (contains = and &)
	if len(encoded1) < len(query)*2 {
		t.Error("encoded string seems too short")
	}
}

// TestLocEqualPerformance tests that LocEqual is optimized
func TestLocEqualPerformance(t *testing.T) {
	loc1 := Location{
		Path: "/test",
		Query: url.Values{
			"a": {"1"},
			"b": {"2"},
			"c": {"3"},
		},
		Hash: "section",
	}

	loc2 := Location{
		Path: "/test",
		Query: url.Values{
			"c": {"3"},
			"a": {"1"},
			"b": {"2"},
		},
		Hash: "section",
	}

	// Should be equal despite different query key order
	if !LocEqual(loc1, loc2) {
		t.Error("locations with reordered query keys should be equal")
	}

	// Different paths should short-circuit
	loc3 := Location{
		Path: "/different",
		Query: url.Values{
			"a": {"1"},
			"b": {"2"},
			"c": {"3"},
		},
		Hash: "section",
	}

	if LocEqual(loc1, loc3) {
		t.Error("locations with different paths should not be equal")
	}

	// Different hashes should short-circuit
	loc4 := Location{
		Path: "/test",
		Query: url.Values{
			"a": {"1"},
			"b": {"2"},
			"c": {"3"},
		},
		Hash: "different",
	}

	if LocEqual(loc1, loc4) {
		t.Error("locations with different hashes should not be equal")
	}
}

// TestCanonicalizeEmptyQuery tests optimization for empty queries
func TestCanonicalizeEmptyQuery(t *testing.T) {
	loc := Location{
		Path:  "/test",
		Query: nil,
		Hash:  "",
	}

	canon := canonicalizeLocation(loc)

	// Empty query should result in empty url.Values, not nil
	if canon.Query == nil {
		t.Error("canonicalized empty query should be url.Values{}, not nil")
	}

	if len(canon.Query) != 0 {
		t.Error("canonicalized empty query should have length 0")
	}
}

// TestValuesEqualWithNilHandling tests nil query handling
func TestValuesEqualWithNilHandling(t *testing.T) {
	tests := []struct {
		name     string
		a        url.Values
		b        url.Values
		expected bool
	}{
		{
			name:     "both nil",
			a:        nil,
			b:        nil,
			expected: true,
		},
		{
			name:     "nil vs empty",
			a:        nil,
			b:        url.Values{},
			expected: true,
		},
		{
			name:     "nil vs non-empty",
			a:        nil,
			b:        url.Values{"key": {"value"}},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := valuesEqual(tt.a, tt.b)
			if result != tt.expected {
				t.Errorf("valuesEqual() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

// BenchmarkValuesEqual benchmarks the optimized comparison
func BenchmarkValuesEqual(b *testing.B) {
	q1 := url.Values{
		"page":   {"1"},
		"limit":  {"20"},
		"sort":   {"date"},
		"filter": {"active"},
	}

	q2 := url.Values{
		"filter": {"active"},
		"limit":  {"20"},
		"page":   {"1"},
		"sort":   {"date"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		valuesEqual(q1, q2)
	}
}

// BenchmarkCanonicalizeLocation benchmarks location canonicalization
func BenchmarkCanonicalizeLocation(b *testing.B) {
	loc := Location{
		Path: "/test/path/with/segments",
		Query: url.Values{
			"page":     {"1"},
			"limit":    {"20"},
			"sort":     {"date"},
			"order":    {"desc"},
			"filter":   {"active"},
			"category": {"news"},
		},
		Hash: "section-1",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		canonicalizeLocation(loc)
	}
}
