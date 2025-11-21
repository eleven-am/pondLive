package session

import (
	"net/url"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func TestInitialLocation(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	loc := sess.InitialLocation()
	if loc.Path != "" {
		t.Errorf("expected empty path before MergeRequest, got %q", loc.Path)
	}

	req := newRequest("/users/123?tab=profile&sort=name")
	sess.MergeRequest(req)

	loc = sess.InitialLocation()
	if loc.Path != "/users/123" {
		t.Errorf("expected path '/users/123', got %q", loc.Path)
	}

	if loc.Query.Get("tab") != "profile" {
		t.Errorf("expected tab=profile, got %q", loc.Query.Get("tab"))
	}

	if loc.Query.Get("sort") != "name" {
		t.Errorf("expected sort=name, got %q", loc.Query.Get("sort"))
	}
}

func TestInitialLocationClone(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	req := newRequest("/test?foo=bar")
	sess.MergeRequest(req)

	loc1 := sess.InitialLocation()
	loc1.Query.Set("foo", "modified")

	loc2 := sess.InitialLocation()
	if loc2.Query.Get("foo") != "bar" {
		t.Errorf("expected query to be cloned, got %q", loc2.Query.Get("foo"))
	}
}

func TestCloneQuery(t *testing.T) {
	original := url.Values{
		"foo": []string{"bar", "baz"},
		"qux": []string{"quux"},
	}

	cloned := cloneQuery(original)

	cloned.Set("foo", "modified")
	cloned.Del("qux")

	if len(original["foo"]) != 2 || original["foo"][0] != "bar" {
		t.Errorf("original was modified")
	}
	if _, ok := original["qux"]; !ok {
		t.Errorf("qux was deleted from original")
	}
}

func TestCloneQueryNil(t *testing.T) {
	cloned := cloneQuery(nil)
	if cloned == nil {
		t.Error("expected non-nil url.Values for nil input")
	}
	if len(cloned) != 0 {
		t.Errorf("expected empty url.Values, got %v", cloned)
	}
}

func TestInitialLocationDefaultsToEmpty(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	loc := sess.InitialLocation()
	if loc.Path != "" {
		t.Errorf("expected empty path, got %q", loc.Path)
	}
	if len(loc.Query) != 0 {
		t.Errorf("expected empty query, got %v", loc.Query)
	}
	if loc.Hash != "" {
		t.Errorf("expected empty hash, got %q", loc.Hash)
	}
}

func TestInitialLocationNilSession(t *testing.T) {
	var sess *LiveSession
	loc := sess.InitialLocation()
	if loc.Path != "/" {
		t.Errorf("expected default path '/', got %q", loc.Path)
	}
}

func TestMergeRequestMultipleQueryValues(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	req := newRequest("/search?tag=go&tag=rust&tag=python&lang=en")
	sess.MergeRequest(req)

	loc := sess.InitialLocation()
	tags := loc.Query["tag"]
	if len(tags) != 3 {
		t.Errorf("expected 3 tag values, got %d", len(tags))
	}
	if tags[0] != "go" || tags[1] != "rust" || tags[2] != "python" {
		t.Errorf("unexpected tag values: %v", tags)
	}
	if loc.Query.Get("lang") != "en" {
		t.Errorf("expected lang=en, got %q", loc.Query.Get("lang"))
	}
}

func TestMergeRequestEncodedQuery(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	req := newRequest("/search?q=hello+world&filter=type%3Apost")
	sess.MergeRequest(req)

	loc := sess.InitialLocation()
	if loc.Query.Get("q") != "hello world" {
		t.Errorf("expected decoded 'hello world', got %q", loc.Query.Get("q"))
	}
	if loc.Query.Get("filter") != "type:post" {
		t.Errorf("expected decoded 'type:post', got %q", loc.Query.Get("filter"))
	}
}

func TestMergeRequestRootPath(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	req := newRequest("/")
	sess.MergeRequest(req)

	loc := sess.InitialLocation()
	if loc.Path != "/" {
		t.Errorf("expected path '/', got %q", loc.Path)
	}
}

func TestMergeRequestPreservesHash(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	req := newRequest("/docs/intro")
	sess.MergeRequest(req)

	loc := sess.InitialLocation()

	loc.Hash = "section-1"
	if loc.Hash != "section-1" {
		t.Errorf("expected hash 'section-1', got %q", loc.Hash)
	}
}

func TestMergeRequestOverwrites(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	req1 := newRequest("/first?foo=bar")
	sess.MergeRequest(req1)

	loc1 := sess.InitialLocation()
	if loc1.Path != "/first" {
		t.Errorf("expected path '/first', got %q", loc1.Path)
	}

	req2 := newRequest("/second?baz=qux")
	sess.MergeRequest(req2)

	loc2 := sess.InitialLocation()
	if loc2.Path != "/second" {
		t.Errorf("expected path '/second' after overwrite, got %q", loc2.Path)
	}
	if loc2.Query.Get("foo") != "" {
		t.Errorf("expected old query param to be cleared, got %q", loc2.Query.Get("foo"))
	}
	if loc2.Query.Get("baz") != "qux" {
		t.Errorf("expected new query param 'baz=qux', got %q", loc2.Query.Get("baz"))
	}
}

func TestMergeRequestNilRequest(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	sess.MergeRequest(nil)

	loc := sess.InitialLocation()
	if loc.Path != "" {
		t.Errorf("expected empty path after nil request, got %q", loc.Path)
	}
}

func TestLocationStructure(t *testing.T) {
	loc := Location{
		Path:  "/users/123",
		Query: url.Values{"tab": []string{"profile"}},
		Hash:  "section-bio",
	}

	if loc.Path != "/users/123" {
		t.Errorf("expected path '/users/123', got %q", loc.Path)
	}
	if loc.Query.Get("tab") != "profile" {
		t.Errorf("expected tab=profile, got %q", loc.Query.Get("tab"))
	}
	if loc.Hash != "section-bio" {
		t.Errorf("expected hash 'section-bio', got %q", loc.Hash)
	}
}

func TestCloneQueryPreservesMultipleValues(t *testing.T) {
	original := url.Values{
		"colors": []string{"red", "green", "blue"},
	}

	cloned := cloneQuery(original)

	if len(cloned["colors"]) != 3 {
		t.Errorf("expected 3 color values, got %d", len(cloned["colors"]))
	}

	cloned["colors"] = append(cloned["colors"], "yellow")

	if len(original["colors"]) != 3 {
		t.Errorf("original was modified, expected 3 values, got %d", len(original["colors"]))
	}
}

func TestCloneQueryEmpty(t *testing.T) {
	original := url.Values{}
	cloned := cloneQuery(original)

	if cloned == nil {
		t.Error("expected non-nil cloned Values")
	}
	if len(cloned) != 0 {
		t.Errorf("expected empty cloned Values, got %v", cloned)
	}
}

func TestMergeRequestWithComplexPath(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		return dom.ElementNode("div")
	}

	sess := New(SessionID("test"), 1, app, nil)

	req := newRequest("/api/v2/users/123/posts/456/comments?sort=date&order=desc")
	sess.MergeRequest(req)

	loc := sess.InitialLocation()
	if loc.Path != "/api/v2/users/123/posts/456/comments" {
		t.Errorf("unexpected path: %q", loc.Path)
	}
	if loc.Query.Get("sort") != "date" {
		t.Errorf("expected sort=date, got %q", loc.Query.Get("sort"))
	}
	if loc.Query.Get("order") != "desc" {
		t.Errorf("expected order=desc, got %q", loc.Query.Get("order"))
	}
}
