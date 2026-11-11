package metadata

import "testing"

func TestRatesMergeToTheFinestExplicitOne(t *testing.T) {
	cases := []struct {
		a, b, want int
	}{
		{RateDefault, 50, 50},
		{50, RateDefault, 50},
		{80, 50, 50},
		{RateFrame, 50, RateFrame},
		{RateOff, RateFrame, RateOff},
		{RateOff, 50, RateOff},
		{RateDefault, RateDefault, RateDefault},
	}
	for _, c := range cases {
		if got := Merge(EventOptions{Throttle: c.a}, EventOptions{Throttle: c.b}).Throttle; got != c.want {
			t.Errorf("Merge(%d, %d) = %d, want %d", c.a, c.b, got, c.want)
		}
	}
}

func TestHandlersWithDifferentRatesAreMarkedMixed(t *testing.T) {
	if MergeHandlers(EventOptions{Debounce: 100}, EventOptions{}).RateMixed != true {
		t.Fatal("a delayed and an immediate handler on one listener are mixed")
	}
	if MergeHandlers(EventOptions{Debounce: 100}, EventOptions{Debounce: 100}).RateMixed {
		t.Fatal("equal rates are not mixed")
	}
}

func TestKeyFiltersSurviveAHandlerMergeOnlyWhenEveryHandlerHasOne(t *testing.T) {
	both := MergeHandlers(EventOptions{Keys: []string{"Enter"}}, EventOptions{Keys: []string{"Escape", "Enter"}})
	if len(both.Keys) != 2 {
		t.Fatalf("keys = %v", both.Keys)
	}
	one := MergeHandlers(EventOptions{Keys: []string{"Enter"}}, EventOptions{})
	if len(one.Keys) != 0 {
		t.Fatalf("an unfiltered handler needs every key, got %v", one.Keys)
	}
	if got := Merge(EventOptions{Keys: []string{"a"}}, EventOptions{Keys: []string{"b"}}).Keys; len(got) != 2 {
		t.Fatalf("options of one handler accumulate: %v", got)
	}
}

func TestFieldsUnionKeepsOrderAndAllWins(t *testing.T) {
	got := Merge(EventOptions{Fields: []string{"a", "b", "c"}}, EventOptions{Fields: []string{"c", "d"}}).Fields
	if len(got) != 4 || got[0] != "a" || got[3] != "d" {
		t.Fatalf("fields = %v", got)
	}
	if got := Merge(EventOptions{Fields: []string{"a"}}, EventOptions{Fields: []string{FieldsAll}}).Fields; len(got) != 1 || got[0] != FieldsAll {
		t.Fatalf("fields = %v", got)
	}
}

func TestResolveAppliesNoisyAndPassiveDefaultsAndExplicitChoicesWin(t *testing.T) {
	cases := []struct {
		name  string
		event string
		o     EventOptions
		want  Resolved
	}{
		{"noisy default is a frame", "mousemove", EventOptions{}, Resolved{Rate: true}},
		{"scroll is noisy and passive", "scroll", EventOptions{}, Resolved{Rate: true, Passive: true}},
		{"click is neither", "click", EventOptions{}, Resolved{}},
		{"explicit ms replaces the default", "mousemove", EventOptions{Throttle: 100}, Resolved{Rate: true, Throttle: 100}},
		{"explicit frame on a quiet event", "click", EventOptions{Throttle: RateFrame}, Resolved{Rate: true}},
		{"immediate turns the default off", "mousemove", EventOptions{Throttle: RateOff}, Resolved{}},
		{"debounce wins over the default throttle", "wheel", EventOptions{Debounce: 50}, Resolved{Debounce: 50, Passive: true}},
		{"prevent beats the default passive", "touchmove", EventOptions{Prevent: true}, Resolved{Rate: true}},
		{"explicit passive stays", "click", EventOptions{Passive: true}, Resolved{Passive: true}},
	}
	for _, c := range cases {
		if got := Resolve(c.event, c.o); got != c.want {
			t.Errorf("%s: %+v, want %+v", c.name, got, c.want)
		}
	}
}
