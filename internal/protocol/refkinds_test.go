package protocol

import "testing"

func TestRefAllowsOwnAndBaseNamesOnly(t *testing.T) {
	cases := []struct {
		kind string
		op   RefOp
		name string
		want bool
	}{
		{"MediaRef", RefMethod, "play", true},
		{"MediaRef", RefMethod, "focus", true},
		{"MediaRef", RefMethod, "submit", false},
		{"InputRef", RefQuery, "selectionStart", true},
		{"InputRef", RefSet, "disabled", false},
		{"InputRef", RefSet, "value", false},
		{"InputRef", RefSet, "innerHTML", false},
		{"ElementRef", RefMethod, "select", false},
		{"ElementRef", RefObserve, "rect", true},
		{"InputRef", RefObserve, "currentTime", false},
		{"Nope", RefMethod, "focus", false},
	}
	for _, c := range cases {
		if got := RefAllows(c.kind, c.op, c.name); got != c.want {
			t.Errorf("RefAllows(%s,%s,%s) = %v, want %v", c.kind, c.op, c.name, got, c.want)
		}
	}
}

func TestEventFieldsAreUniqueAndValidated(t *testing.T) {
	seen := map[string]bool{}
	for _, f := range EventFields {
		if seen[f] {
			t.Fatalf("duplicate field %q", f)
		}
		seen[f] = true
	}
	for _, f := range ComputedFields {
		if !seen[f] {
			t.Fatalf("computed field %q is not in the catalogue", f)
		}
	}
	if err := ValidateFields("clientX", "value"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateFields("clientX", "ownerDocument"); err == nil {
		t.Fatal("an unknown field must be rejected")
	}
}
