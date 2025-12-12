package route

import (
	"net/url"
	"testing"
)

func TestParseHref(t *testing.T) {
	tests := []struct {
		name     string
		href     string
		wantPath string
		wantHash string
		wantKeys []string
	}{
		{
			name:     "simple path",
			href:     "/users",
			wantPath: "/users",
			wantHash: "",
			wantKeys: nil,
		},
		{
			name:     "path with query",
			href:     "/search?q=test&page=1",
			wantPath: "/search",
			wantHash: "",
			wantKeys: []string{"q", "page"},
		},
		{
			name:     "path with hash",
			href:     "/docs#section",
			wantPath: "/docs",
			wantHash: "section",
			wantKeys: nil,
		},
		{
			name:     "full URL",
			href:     "/page?foo=bar#anchor",
			wantPath: "/page",
			wantHash: "anchor",
			wantKeys: []string{"foo"},
		},
		{
			name:     "invalid URL",
			href:     "://invalid",
			wantPath: "/",
			wantHash: "",
			wantKeys: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			loc := ParseHref(tt.href)
			if loc.Path != tt.wantPath {
				t.Errorf("ParseHref(%q).Path = %q, want %q", tt.href, loc.Path, tt.wantPath)
			}
			if loc.Hash != tt.wantHash {
				t.Errorf("ParseHref(%q).Hash = %q, want %q", tt.href, loc.Hash, tt.wantHash)
			}
			for _, key := range tt.wantKeys {
				if loc.Query.Get(key) == "" {
					t.Errorf("ParseHref(%q).Query missing key %q", tt.href, key)
				}
			}
		})
	}
}

func TestBuildHref(t *testing.T) {
	tests := []struct {
		name string
		loc  Location
		want string
	}{
		{
			name: "simple path",
			loc:  Location{Path: "/users"},
			want: "/users",
		},
		{
			name: "empty path",
			loc:  Location{Path: ""},
			want: "/",
		},
		{
			name: "path with query",
			loc: Location{
				Path:  "/search",
				Query: url.Values{"q": {"test"}},
			},
			want: "/search?q=test",
		},
		{
			name: "path with hash without prefix",
			loc: Location{
				Path: "/docs",
				Hash: "section",
			},
			want: "/docs#section",
		},
		{
			name: "path with hash with prefix",
			loc: Location{
				Path: "/docs",
				Hash: "#section",
			},
			want: "/docs#section",
		},
		{
			name: "full location",
			loc: Location{
				Path:  "/page",
				Query: url.Values{"foo": {"bar"}},
				Hash:  "anchor",
			},
			want: "/page?foo=bar#anchor",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := BuildHref(tt.loc)
			if got != tt.want {
				t.Errorf("BuildHref() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSetSearch(t *testing.T) {
	loc := Location{Path: "/test", Query: url.Values{"old": {"value"}}}
	newValues := url.Values{"new": {"value"}}

	result := SetSearch(loc, newValues)

	if result.Query.Get("old") != "" {
		t.Error("SetSearch should replace old query values")
	}
	if result.Query.Get("new") != "value" {
		t.Error("SetSearch should set new query values")
	}
	if result.Path != "/test" {
		t.Error("SetSearch should preserve path")
	}
}

func TestAddSearch(t *testing.T) {
	t.Run("nil query", func(t *testing.T) {
		loc := Location{Path: "/test"}
		result := AddSearch(loc, "key", "val1", "val2")

		vals := result.Query["key"]
		if len(vals) != 2 || vals[0] != "val1" || vals[1] != "val2" {
			t.Errorf("AddSearch with nil query failed, got %v", vals)
		}
	})

	t.Run("existing query", func(t *testing.T) {
		loc := Location{Path: "/test", Query: url.Values{"existing": {"value"}}}
		result := AddSearch(loc, "new", "val")

		if result.Query.Get("existing") != "value" {
			t.Error("AddSearch should preserve existing values")
		}
		if result.Query.Get("new") != "val" {
			t.Error("AddSearch should add new values")
		}
	})
}

func TestDelSearch(t *testing.T) {
	t.Run("nil query", func(t *testing.T) {
		loc := Location{Path: "/test"}
		result := DelSearch(loc, "key")
		if result.Path != "/test" {
			t.Error("DelSearch with nil query should return unchanged location")
		}
	})

	t.Run("existing key", func(t *testing.T) {
		loc := Location{Path: "/test", Query: url.Values{"key": {"value"}, "other": {"val"}}}
		result := DelSearch(loc, "key")

		if result.Query.Get("key") != "" {
			t.Error("DelSearch should remove key")
		}
		if result.Query.Get("other") != "val" {
			t.Error("DelSearch should preserve other keys")
		}
	})
}

func TestMergeSearch(t *testing.T) {
	t.Run("nil query", func(t *testing.T) {
		loc := Location{Path: "/test"}
		result := MergeSearch(loc, url.Values{"key": {"val"}})

		if result.Query.Get("key") != "val" {
			t.Error("MergeSearch with nil query should add values")
		}
	})

	t.Run("merge values", func(t *testing.T) {
		loc := Location{Path: "/test", Query: url.Values{"a": {"1"}}}
		result := MergeSearch(loc, url.Values{"b": {"2"}, "c": {"3", "4"}})

		if result.Query.Get("a") != "1" {
			t.Error("MergeSearch should preserve existing values")
		}
		if result.Query.Get("b") != "2" {
			t.Error("MergeSearch should add new values")
		}
		cVals := result.Query["c"]
		if len(cVals) < 2 {
			t.Error("MergeSearch should add multiple values")
		}
	})
}

func TestClearSearch(t *testing.T) {
	loc := Location{Path: "/test", Query: url.Values{"key": {"val"}}, Hash: "anchor"}
	result := ClearSearch(loc)

	if len(result.Query) != 0 {
		t.Error("ClearSearch should clear query")
	}
	if result.Path != "/test" {
		t.Error("ClearSearch should preserve path")
	}
	if result.Hash != "anchor" {
		t.Error("ClearSearch should preserve hash")
	}
}

func TestLocationClone(t *testing.T) {
	original := Location{
		Path:  "/test",
		Query: url.Values{"key": {"val"}},
		Hash:  "anchor",
	}

	clone := original.Clone()

	if clone.Path != original.Path {
		t.Error("Clone should copy path")
	}
	if clone.Hash != original.Hash {
		t.Error("Clone should copy hash")
	}
	if clone.Query.Get("key") != "val" {
		t.Error("Clone should copy query")
	}

	clone.Query.Set("key", "modified")
	if original.Query.Get("key") == "modified" {
		t.Error("Clone should create independent query copy")
	}
}

func TestLocEqual(t *testing.T) {
	tests := []struct {
		name string
		a    Location
		b    Location
		want bool
	}{
		{
			name: "equal simple",
			a:    Location{Path: "/test"},
			b:    Location{Path: "/test"},
			want: true,
		},
		{
			name: "different path",
			a:    Location{Path: "/test1"},
			b:    Location{Path: "/test2"},
			want: false,
		},
		{
			name: "different hash",
			a:    Location{Path: "/test", Hash: "a"},
			b:    Location{Path: "/test", Hash: "b"},
			want: false,
		},
		{
			name: "equal with query",
			a:    Location{Path: "/test", Query: url.Values{"k": {"v"}}},
			b:    Location{Path: "/test", Query: url.Values{"k": {"v"}}},
			want: true,
		},
		{
			name: "different query length",
			a:    Location{Path: "/test", Query: url.Values{"a": {"1"}}},
			b:    Location{Path: "/test", Query: url.Values{"a": {"1"}, "b": {"2"}}},
			want: false,
		},
		{
			name: "different query keys",
			a:    Location{Path: "/test", Query: url.Values{"a": {"1"}}},
			b:    Location{Path: "/test", Query: url.Values{"b": {"1"}}},
			want: false,
		},
		{
			name: "different query value count",
			a:    Location{Path: "/test", Query: url.Values{"a": {"1"}}},
			b:    Location{Path: "/test", Query: url.Values{"a": {"1", "2"}}},
			want: false,
		},
		{
			name: "different query values",
			a:    Location{Path: "/test", Query: url.Values{"a": {"1", "2"}}},
			b:    Location{Path: "/test", Query: url.Values{"a": {"1", "3"}}},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := LocEqual(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("LocEqual() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCloneValues(t *testing.T) {
	t.Run("nil values", func(t *testing.T) {
		result := cloneValues(nil)
		if result == nil {
			t.Error("cloneValues(nil) should return empty Values, not nil")
		}
	})

	t.Run("copies values", func(t *testing.T) {
		original := url.Values{"key": {"val1", "val2"}}
		clone := cloneValues(original)

		clone["key"][0] = "modified"
		if original["key"][0] == "modified" {
			t.Error("cloneValues should create independent copy")
		}
	})
}
