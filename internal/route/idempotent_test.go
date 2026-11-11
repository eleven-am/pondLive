package route

import (
	"net/url"
	"reflect"
	"testing"
)

func canonicalOnce(loc Location) Location { return Canonicalize(loc) }

func TestCanonicalizeIsIdempotentForTrailingSpaceInSegment(t *testing.T) {
	once := canonicalOnce(Location{Path: "/a /"})
	twice := canonicalOnce(once)
	if once.Path != "/a" || twice.Path != once.Path {
		t.Fatalf("once = %q, twice = %q", once.Path, twice.Path)
	}
}

func TestCanonicalizeIsIdempotentForRepeatedHashMarks(t *testing.T) {
	once := canonicalOnce(Location{Path: "/", Hash: "##x"})
	twice := canonicalOnce(once)
	if once.Hash != "x" || twice.Hash != once.Hash {
		t.Fatalf("once = %q, twice = %q", once.Hash, twice.Hash)
	}
}

func TestCanonicalizeIsIdempotentForHashInsidePath(t *testing.T) {
	once := canonicalOnce(Location{Path: "/a#%23x"})
	twice := canonicalOnce(once)
	if !LocEqual(once, twice) {
		t.Fatalf("once = %+v, twice = %+v", once, twice)
	}
	once = canonicalOnce(Location{Path: "/a###x"})
	twice = canonicalOnce(once)
	if once.Hash != "x" || !LocEqual(once, twice) {
		t.Fatalf("once = %+v, twice = %+v", once, twice)
	}
}

func TestNormalizePartsIsIdempotentOnSpacedSegments(t *testing.T) {
	for _, in := range []string{"/a /", " / a / b ", "/ /", "a /b", "/a /b /"} {
		once := NormalizeParts(in).Path
		twice := NormalizeParts(once).Path
		if once != twice {
			t.Errorf("NormalizeParts(%q): once %q, twice %q", in, once, twice)
		}
	}
}

func FuzzCanonicalizeIsIdempotent(f *testing.F) {
	seeds := [][3]string{
		{"/a /", "", ""},
		{"/", "##x", ""},
		{"", "#", "a=1"},
		{"/a#b", "", "k= v &k=a"},
		{" ", " # ", "%zz"},
		{"/a?b=1#c", "#d", ""},
		{"//a//b//", "x", "z=1&a=2"},
	}
	for _, s := range seeds {
		f.Add(s[0], s[1], s[2])
	}
	f.Fuzz(func(t *testing.T, path, hash, rawQuery string) {
		query, _ := url.ParseQuery(rawQuery)
		once := Canonicalize(Location{Path: path, Hash: hash, Query: query})
		twice := Canonicalize(once)
		if once.Path != twice.Path || once.Hash != twice.Hash || !reflect.DeepEqual(once.Query, twice.Query) {
			t.Fatalf("not idempotent for %q %q %q: once %+v twice %+v", path, hash, rawQuery, once, twice)
		}
	})
}
