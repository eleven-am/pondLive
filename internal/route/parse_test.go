package route

import "testing"

func TestParseStatic(t *testing.T) {
	match, err := Parse("/about", "/about", "")
	if err != nil {
		t.Fatalf("expected static match: %v", err)
	}
	if match.Pattern != "/about" {
		t.Fatalf("unexpected pattern: %q", match.Pattern)
	}
	if len(match.Params) != 0 {
		t.Fatalf("expected no params, got %#v", match.Params)
	}
}

func TestParseDynamic(t *testing.T) {
	match, err := Parse("/users/:id", "/users/123", "")
	if err != nil {
		t.Fatalf("expected dynamic match: %v", err)
	}
	if match.Params["id"] != "123" {
		t.Fatalf("expected id=123, got %#v", match.Params)
	}
}

func TestParseWildcard(t *testing.T) {
	match, err := Parse("/files/*rest", "/files/a/b/c", "")
	if err != nil {
		t.Fatalf("expected wildcard match: %v", err)
	}
	if match.Params["rest"] != "a/b/c" {
		t.Fatalf("expected rest=a/b/c, got %#v", match.Params)
	}
}

func TestParseOptional(t *testing.T) {
	match, err := Parse("/blog/:slug?", "/blog", "")
	if err != nil {
		t.Fatalf("expected optional match: %v", err)
	}
	if len(match.Params) != 0 {
		t.Fatalf("expected empty params, got %#v", match.Params)
	}
}

func TestParseQueryValues(t *testing.T) {
	match, err := Parse("/search", "/search", "q=golang&tag=ui&tag=go")
	if err != nil {
		t.Fatalf("expected query parse: %v", err)
	}
	if match.Query.Get("q") != "golang" {
		t.Fatalf("expected q parameter, got %#v", match.Query)
	}
	values := match.Query["tag"]
	if len(values) != 2 {
		t.Fatalf("expected two tag values, got %#v", values)
	}
}

func TestParseDerivesQueryAndPathFromHref(t *testing.T) {
	match, err := Parse("/search", "/search?q=golang#results", "")
	if err != nil {
		t.Fatalf("expected href style parse: %v", err)
	}
	if match.Path != "/search" {
		t.Fatalf("expected normalized path, got %q", match.Path)
	}
	if match.Query.Get("q") != "golang" {
		t.Fatalf("expected derived query param, got %#v", match.Query)
	}
	if match.RawQuery != "q=golang" {
		t.Fatalf("expected raw query to match derived value, got %q", match.RawQuery)
	}
}

func TestBestMatchPrefersSpecific(t *testing.T) {
	patterns := []string{"/users/:id", "/users/:id/settings"}
	match, idx, ok := BestMatch("/users/1/settings", "", patterns)
	if !ok {
		t.Fatal("expected best match")
	}
	if match.Pattern != "/users/:id/settings" {
		t.Fatalf("expected specific pattern, got %q", match.Pattern)
	}
	if idx != 1 {
		t.Fatalf("expected index 1, got %d", idx)
	}
}

func TestBestMatchNoMatch(t *testing.T) {
	patterns := []string{"/users/:id"}
	_, _, ok := BestMatch("/about", "", patterns)
	if ok {
		t.Fatal("expected no match")
	}
}

func TestParseInvalidQuery(t *testing.T) {
	_, err := Parse("/search", "/search", "%zz")
	if err == nil {
		t.Fatal("expected invalid query to error")
	}
}

func TestParseNormalizesInput(t *testing.T) {
	match, err := Parse("users/:id", " users/42 ", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match.Path != "/users/42" {
		t.Fatalf("expected normalized path, got %q", match.Path)
	}
	if match.Pattern != "/users/:id" {
		t.Fatalf("expected normalized pattern, got %q", match.Pattern)
	}
	if got := match.Params["id"]; got != "42" {
		t.Fatalf("expected id=42, got %q", got)
	}
}

func TestParseRejectsExtraSegments(t *testing.T) {
	_, err := Parse("/blog/:slug?", "/blog/foo/bar", "")
	if err == nil {
		t.Fatal("expected error for unmatched remainder")
	}
}

func TestParseWildcardWithoutName(t *testing.T) {
	match, err := Parse("/files/*", "/files/a/b", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := match.Params["*"]; got != "a/b" {
		t.Fatalf("expected wildcard remainder, got %q", got)
	}
	if match.Rest != "/a/b" {
		t.Fatalf("expected rest=/a/b, got %q", match.Rest)
	}
}

func TestPreferUsesScoreAndRestLength(t *testing.T) {
	higherScore := Match{Pattern: "/preferred", Score: 5, Rest: "/rest"}
	lowerScore := Match{Pattern: "/other", Score: 4, Rest: ""}
	if !Prefer(higherScore, lowerScore) {
		t.Fatal("expected higher score to win")
	}

	shorterRest := Match{Pattern: "/short", Score: 3, Rest: "/a"}
	longerRest := Match{Pattern: "/long", Score: 3, Rest: "/a/b"}
	if !Prefer(shorterRest, longerRest) {
		t.Fatal("expected shorter rest to win on tie")
	}

	if Prefer(longerRest, shorterRest) {
		t.Fatal("expected longer rest to lose on tie")
	}
}

func TestBestMatchSkipsPatternsWithErrors(t *testing.T) {
	patterns := []string{"/ok"}
	if _, _, ok := BestMatch("/ok", "%zz", patterns); ok {
		t.Fatal("expected invalid query to prevent match")
	}
}

func TestNormalizePathEnsuresLeadingSlash(t *testing.T) {
	if got := NormalizePath("users/:id"); got != "/users/:id" {
		t.Fatalf("expected normalized pattern with slash, got %q", got)
	}
	if got := NormalizePath(""); got != "/" {
		t.Fatalf("expected empty pattern to normalize to root, got %q", got)
	}
}

func TestNormalizePathCollapsesSeparatorsAndWhitespace(t *testing.T) {
	if got := NormalizePath("  //users//:id  "); got != "/users/:id" {
		t.Fatalf("expected collapsed pattern, got %q", got)
	}
}

func TestNormalizePathTrimsWhitespace(t *testing.T) {
	if got := NormalizePath("  /users/:id  "); got != "/users/:id" {
		t.Fatalf("expected whitespace trimmed, got %q", got)
	}
	if got := NormalizePath("   "); got != "/" {
		t.Fatalf("expected whitespace-only to normalize to root, got %q", got)
	}
}

func TestNormalizePathCollapsesDuplicateSlashes(t *testing.T) {
	if got := NormalizePath("//users//:id"); got != "/users/:id" {
		t.Fatalf("expected duplicate slashes collapsed, got %q", got)
	}
	if got := NormalizePath("/users///profile"); got != "/users/profile" {
		t.Fatalf("expected triple slashes collapsed, got %q", got)
	}
	if got := NormalizePath("///"); got != "/" {
		t.Fatalf("expected only slashes to normalize to root, got %q", got)
	}
}

func TestNormalizePathHandlesMixedIssues(t *testing.T) {
	if got := NormalizePath(" //users//:id "); got != "/users/:id" {
		t.Fatalf("expected mixed issues normalized, got %q", got)
	}
}

func TestParseMatchesNormalizedPatternToPath(t *testing.T) {
	match, err := Parse("  //users//:id  ", "/users/42", "")
	if err != nil {
		t.Fatalf("expected match after normalization: %v", err)
	}
	if match.Pattern != "/users/:id" {
		t.Fatalf("expected normalized pattern, got %q", match.Pattern)
	}
	if match.Params["id"] != "42" {
		t.Fatalf("expected id=42, got %q", match.Params["id"])
	}
}

func TestParseTreatsBlankPathAsRoot(t *testing.T) {
	match, err := Parse("", "", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match.Path != "/" || match.Pattern != "/" {
		t.Fatalf("expected root normalization, got %#v", match)
	}
	if len(match.Params) != 0 || len(match.Query) != 0 {
		t.Fatalf("expected empty params and query, got %#v", match)
	}
}

func TestParseHandlesStackedOptionalSegments(t *testing.T) {
	match, err := Parse("/posts/:year?/:month?", "/posts/2024", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if match.Params["year"] != "2024" {
		t.Fatalf("expected year parameter, got %#v", match.Params)
	}
	if _, ok := match.Params["month"]; ok {
		t.Fatalf("did not expect month parameter, got %#v", match.Params)
	}

	match, err = Parse("/posts/:year?/:month?", "/posts", "")
	if err != nil {
		t.Fatalf("unexpected error for missing optionals: %v", err)
	}
	if len(match.Params) != 0 {
		t.Fatalf("expected empty params when skipping optionals, got %#v", match.Params)
	}
}

func TestParseWildcardWithoutRemainder(t *testing.T) {
	match, err := Parse("/files/*rest", "/files", "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val, ok := match.Params["rest"]; !ok || val != "" {
		t.Fatalf("expected empty rest capture, got %#v", match.Params)
	}
	if match.Rest != "" {
		t.Fatalf("expected empty rest path, got %q", match.Rest)
	}
}

func TestBestMatchKeepsFirstOnExactTie(t *testing.T) {
	patterns := []string{"/a/:id", "/a/:name"}
	match, idx, ok := BestMatch("/a/1", "", patterns)
	if !ok {
		t.Fatal("expected match on tie")
	}
	if match.Pattern != "/a/:id" || idx != 0 {
		t.Fatalf("expected first pattern to win tie, got pattern %q idx %d", match.Pattern, idx)
	}
}

func TestBestMatchPrefersTheMostSpecificPatternRegardlessOfOrder(t *testing.T) {
	cases := []struct {
		path     string
		patterns []string
		want     string
	}{
		{"/x", []string{"/x/*", "/x"}, "/x"},
		{"/x", []string{"/x", "/x/*"}, "/x"},
		{"/", []string{"*", "/"}, "/"},
		{"/", []string{"/", "*"}, "/"},
		{"/a/new", []string{"/a/:id", "/a/new"}, "/a/new"},
		{"/a/new", []string{"/a/new", "/a/:id"}, "/a/new"},
		{"/a/b/c", []string{"*", "/a/*", "/a/b/*"}, "/a/b/*"},
		{"/a/b/c", []string{"/a/b/*", "/a/*", "*"}, "/a/b/*"},
		{"/zzz", []string{"/a/*", "*"}, "/*"},
	}
	for _, c := range cases {
		m, _, ok := BestMatch(c.path, "", c.patterns)
		if !ok || m.Pattern != c.want {
			t.Errorf("%s in %v: got %q (ok=%v), want %q", c.path, c.patterns, m.Pattern, ok, c.want)
		}
	}
}

func TestAWildcardThatConsumesSomethingBeatsAnOptionalParamPatternOfTheSameBaseScore(t *testing.T) {
	for _, patterns := range [][]string{{"/x/*", "/:a?/:b"}, {"/:a?/:b", "/x/*"}} {
		match, idx, ok := BestMatch("/x/y", "", patterns)
		if !ok || match.Pattern != "/x/*" || patterns[idx] != "/x/*" {
			t.Fatalf("patterns %v: picked %q (index %d, ok %v), want /x/*", patterns, match.Pattern, idx, ok)
		}
	}
}

func TestAWildcardThatConsumesNothingLosesToTheExactPatternOfTheSameScore(t *testing.T) {
	match, _, ok := BestMatch("/x", "", []string{"/x/*", "/x"})
	if !ok || match.Pattern != "/x" {
		t.Fatalf("picked %q, want /x", match.Pattern)
	}
}
