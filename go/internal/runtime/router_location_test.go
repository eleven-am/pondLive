package runtime

import "testing"

func TestCanonicalizeLocationUsesNormalizedParts(t *testing.T) {
	loc := canonicalizeLocation(Location{Path: " /docs//intro?page=1#overview "})
	if loc.Path != "/docs/intro" {
		t.Fatalf("expected normalized path, got %q", loc.Path)
	}
	if len(loc.Query) != 0 {
		t.Fatalf("expected empty query, got %#v", loc.Query)
	}
	if loc.Hash != "overview" {
		t.Fatalf("expected hash fragment from path, got %q", loc.Hash)
	}
}

func TestCanonicalizeLocationPrefersExplicitHash(t *testing.T) {
	loc := canonicalizeLocation(Location{Path: "/docs#ignored", Hash: "#kept"})
	if loc.Hash != "kept" {
		t.Fatalf("expected explicit hash to be honored, got %q", loc.Hash)
	}
}
