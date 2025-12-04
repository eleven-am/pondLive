package route

import (
	"testing"
)

func TestNormalizeParts(t *testing.T) {
	parts := NormalizeParts("  /foo//bar?baz=1&baz=2#frag  ")
	if parts.Path != "/foo/bar" {
		t.Fatalf("expected canonical path /foo/bar, got %q", parts.Path)
	}
	if parts.RawQuery != "baz=1&baz=2" {
		t.Fatalf("expected raw query preserved, got %q", parts.RawQuery)
	}
	if parts.Hash != "frag" {
		t.Fatalf("expected hash fragment, got %q", parts.Hash)
	}
}

func TestNormalizeHash(t *testing.T) {
	if got := NormalizeHash("#frag"); got != "frag" {
		t.Fatalf("expected fragment normalized to 'frag', got %q", got)
	}
	if got := NormalizeHash("frag"); got != "frag" {
		t.Fatalf("expected fragment without prefix to remain, got %q", got)
	}
}
