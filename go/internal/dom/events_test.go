package dom

import (
	"reflect"
	"testing"
)

type fakeUpdate struct{ id string }

func (fakeUpdate) isUpdates() {}

func TestMergeEventOptionsPrioritizesExtraValues(t *testing.T) {
	base := EventOptions{
		Key:    "base",
		Listen: []string{"focus", "blur"},
		Props:  []string{"target.value"},
	}
	extra := EventOptions{
		Key:    "extra",
		Listen: []string{"submit", "blur"},
		Props:  []string{"target.checked"},
	}

	merged := MergeEventOptions(base, extra)

	if merged.Key != "extra" {
		t.Fatalf("expected extra key to override base, got %q", merged.Key)
	}
	expectedListen := []string{"focus", "blur", "submit", "blur"}
	if !reflect.DeepEqual(merged.Listen, expectedListen) {
		t.Fatalf("unexpected listen list: %#v", merged.Listen)
	}
	expectedProps := []string{"target.value", "target.checked"}
	if !reflect.DeepEqual(merged.Props, expectedProps) {
		t.Fatalf("unexpected props list: %#v", merged.Props)
	}
}

func TestMergeEventOptionsFallsBackToBaseKey(t *testing.T) {
	base := EventOptions{Key: "base"}
	extra := EventOptions{Listen: []string{"focus"}}

	merged := MergeEventOptions(base, extra)

	if merged.Key != "base" {
		t.Fatalf("expected base key to persist, got %q", merged.Key)
	}
}

func TestDefaultEventOptionsCaseInsensitiveClone(t *testing.T) {
	opts := DefaultEventOptions("InPut")
	if !reflect.DeepEqual(opts.Listen, []string{"change"}) {
		t.Fatalf("expected input preset listeners, got %#v", opts.Listen)
	}
	if !reflect.DeepEqual(opts.Props, []string{"target.value"}) {
		t.Fatalf("expected input props, got %#v", opts.Props)
	}

	// Mutating the returned slices should not affect subsequent calls.
	opts.Listen[0] = "mutated"
	opts.Props[0] = "modified"

	fresh := DefaultEventOptions("input")
	if !reflect.DeepEqual(fresh.Listen, []string{"change"}) {
		t.Fatalf("expected fresh listeners, got %#v", fresh.Listen)
	}
	if !reflect.DeepEqual(fresh.Props, []string{"target.value"}) {
		t.Fatalf("expected fresh props, got %#v", fresh.Props)
	}
}

func TestMergeEventBindingChainsHandlersAndMetadata(t *testing.T) {
	var calls []string

	first := EventBinding{
		Handler: func(ev Event) Updates {
			calls = append(calls, "first:"+ev.Name)
			return fakeUpdate{id: "first"}
		},
		Listen: []string{"change"},
		Props:  []string{"target.value"},
		Key:    "alpha",
	}
	second := EventBinding{
		Handler: func(ev Event) Updates {
			calls = append(calls, "second:"+ev.Name)
			return fakeUpdate{id: "second"}
		},
		Listen: []string{"Change", "submit"},
		Props:  []string{"target.value", "target.checked"},
	}

	merged := MergeEventBinding(first, second)

	if merged.Key != "alpha" {
		t.Fatalf("expected existing key to remain, got %q", merged.Key)
	}
	if !reflect.DeepEqual(merged.Listen, []string{"change", "submit"}) {
		t.Fatalf("expected deduped listeners, got %#v", merged.Listen)
	}
	if !reflect.DeepEqual(merged.Props, []string{"target.value", "target.checked"}) {
		t.Fatalf("expected deduped props, got %#v", merged.Props)
	}

	result := merged.Handler(Event{Name: "input"})
	if !reflect.DeepEqual(calls, []string{"first:input", "second:input"}) {
		t.Fatalf("expected chained handler execution, got %#v", calls)
	}
	update, ok := result.(fakeUpdate)
	if !ok || update.id != "second" {
		t.Fatalf("expected second handler update, got %#v", result)
	}
}

func TestMergeEventBindingRetainsExistingWhenAdditionNil(t *testing.T) {
	invoked := false
	existing := EventBinding{
		Handler: func(ev Event) Updates {
			invoked = true
			return nil
		},
		Listen: []string{"focus"},
		Props:  []string{"target.value"},
		Key:    "persist",
	}

	merged := MergeEventBinding(existing, EventBinding{})

	if merged.Key != "persist" {
		t.Fatalf("expected key to persist, got %q", merged.Key)
	}
	if !reflect.DeepEqual(merged.Listen, []string{"focus"}) {
		t.Fatalf("expected listen list to remain, got %#v", merged.Listen)
	}
	if !reflect.DeepEqual(merged.Props, []string{"target.value"}) {
		t.Fatalf("expected props to remain, got %#v", merged.Props)
	}

	merged.Handler(Event{Name: "focus"})
	if !invoked {
		t.Fatal("expected original handler to be invoked")
	}
}

func TestSanitizeEventListFiltersDuplicatesAndPrimary(t *testing.T) {
	cleaned := sanitizeEventList("click", []string{" click ", "", "CLICK", "change", "focus", "change"})
	if !reflect.DeepEqual(cleaned, []string{"change", "focus"}) {
		t.Fatalf("unexpected sanitized list: %#v", cleaned)
	}

	if sanitized := sanitizeEventList("click", []string{"", " \t ", "CLICK"}); sanitized != nil {
		t.Fatalf("expected nil result for empty payload, got %#v", sanitized)
	}
}

func TestSanitizeSelectorListRemovesEmptiesAndDuplicates(t *testing.T) {
	cleaned := sanitizeSelectorList([]string{" target.value ", "", "target.value", "target.checked"})
	if !reflect.DeepEqual(cleaned, []string{"target.value", "target.checked"}) {
		t.Fatalf("unexpected selector list: %#v", cleaned)
	}

	if sanitized := sanitizeSelectorList([]string{" ", ""}); sanitized != nil {
		t.Fatalf("expected nil selector list, got %#v", sanitized)
	}
}

func TestMergeStringSetHandlesCaseFolding(t *testing.T) {
	merged := mergeStringSet([]string{"Focus", ""}, []string{"focus", "blur", "BLUR"}, true)
	if !reflect.DeepEqual(merged, []string{"Focus", "blur"}) {
		t.Fatalf("unexpected merged string set: %#v", merged)
	}

	if result := mergeStringSet(nil, nil, false); result != nil {
		t.Fatalf("expected nil result for empty inputs, got %#v", result)
	}
}

func TestEventBindingWithOptionsSanitizesMetadata(t *testing.T) {
	binding := EventBinding{}
	opts := EventOptions{
		Key:    "assigned",
		Listen: []string{" change ", "blur", "Change"},
		Props:  []string{" target.value ", "", "target.value"},
	}

	configured := binding.WithOptions(opts, "change")

	if configured.Key != "assigned" {
		t.Fatalf("expected key override, got %q", configured.Key)
	}
	if !reflect.DeepEqual(configured.Listen, []string{"blur"}) {
		t.Fatalf("expected sanitized listeners, got %#v", configured.Listen)
	}
	if !reflect.DeepEqual(configured.Props, []string{"target.value"}) {
		t.Fatalf("expected sanitized props, got %#v", configured.Props)
	}
}

func TestMergeEventBindingAdoptsAdditionWhenExistingEmpty(t *testing.T) {
	addition := EventBinding{
		Handler: func(ev Event) Updates {
			if ev.Name != "click" {
				t.Fatalf("unexpected event %q", ev.Name)
			}
			return fakeUpdate{id: "extra"}
		},
		Listen: []string{"click"},
		Props:  []string{"target.id"},
		Key:    "new",
	}

	merged := MergeEventBinding(EventBinding{}, addition)

	if merged.Key != "new" {
		t.Fatalf("expected key adoption, got %q", merged.Key)
	}
	if !reflect.DeepEqual(merged.Listen, []string{"click"}) {
		t.Fatalf("expected listener adoption, got %#v", merged.Listen)
	}
	if !reflect.DeepEqual(merged.Props, []string{"target.id"}) {
		t.Fatalf("expected prop adoption, got %#v", merged.Props)
	}
	if merged.Handler == nil {
		t.Fatal("expected merged handler to be non-nil")
	}

	if result := merged.Handler(Event{Name: "click"}); result.(fakeUpdate).id != "extra" {
		t.Fatalf("expected extra handler result, got %#v", result)
	}
}

func TestMergeEventBindingNilHandlersYieldNil(t *testing.T) {
	existing := EventBinding{Listen: []string{"input"}}
	merged := MergeEventBinding(existing, EventBinding{})

	if merged.Handler != nil {
		t.Fatal("expected nil handler when neither binding provides one")
	}
	if !reflect.DeepEqual(merged.Listen, []string{"input"}) {
		t.Fatalf("expected existing listeners to remain, got %#v", merged.Listen)
	}
	if merged.Props != nil {
		t.Fatalf("expected nil props when merging nil sets, got %#v", merged.Props)
	}

	merged.Listen[0] = "changed"
	if existing.Listen[0] != "input" {
		t.Fatalf("expected original slice to remain unchanged, got %#v", existing.Listen)
	}
}

func TestDefaultEventOptionsUnknownEvent(t *testing.T) {
	opts := DefaultEventOptions("does-not-exist")
	if opts.Key != "" || opts.Listen != nil || opts.Props != nil {
		t.Fatalf("expected empty options for unknown event, got %#v", opts)
	}
}

func TestMergeEventOptionsProducesIndependentSlices(t *testing.T) {
	base := EventOptions{Listen: []string{"focus"}, Props: []string{"target.value"}}
	extra := EventOptions{Listen: []string{"blur"}, Props: []string{"target.checked"}}

	merged := MergeEventOptions(base, extra)

	base.Listen[0] = "mutated"
	extra.Props[0] = "altered"

	if !reflect.DeepEqual(merged.Listen, []string{"focus", "blur"}) {
		t.Fatalf("expected merged listeners to remain intact, got %#v", merged.Listen)
	}
	if !reflect.DeepEqual(merged.Props, []string{"target.value", "target.checked"}) {
		t.Fatalf("expected merged props to remain intact, got %#v", merged.Props)
	}
}
