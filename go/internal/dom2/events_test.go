package dom2

import (
	"reflect"
	"testing"
)

type testUpdates string

func (testUpdates) isUpdates() {}

func TestMergeEventBindingCombinesHandlersAndMetadata(t *testing.T) {
	var calls []string
	base := EventBinding{
		Handler: func(Event) Updates {
			calls = append(calls, "base")
			return nil
		},
		Listen: []string{"change"},
		Props:  []string{"target.value"},
	}
	addition := EventBinding{
		Handler: func(Event) Updates {
			calls = append(calls, "addition")
			return testUpdates("addition")
		},
		Listen: []string{"Change", "focus"},
		Props:  []string{"target.value", "target.checked"},
		Key:    "override",
	}

	merged := MergeEventBinding(base, addition)

	if merged.Key != "override" {
		t.Fatalf("expected override key, got %q", merged.Key)
	}

	if got := merged.Handler(Event{}); got != testUpdates("addition") {
		t.Fatalf("last handler result not returned, got %v", got)
	}

	wantCalls := []string{"base", "addition"}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("handlers executed in wrong order: got %v want %v", calls, wantCalls)
	}

	if want := []string{"change", "focus"}; !reflect.DeepEqual(merged.Listen, want) {
		t.Fatalf("listen metadata mismatch: got %v want %v", merged.Listen, want)
	}

	if want := []string{"target.value", "target.checked"}; !reflect.DeepEqual(merged.Props, want) {
		t.Fatalf("props metadata mismatch: got %v want %v", merged.Props, want)
	}
}

func TestMergeEventOptionsOverrides(t *testing.T) {
	base := EventOptions{
		Key:    "base",
		Listen: []string{"change"},
		Props:  []string{"target.value"},
	}

	extra := EventOptions{
		Listen: []string{"input"},
		Props:  []string{"target.checked"},
	}

	t.Run("preserves base key when extra empty", func(t *testing.T) {
		merged := MergeEventOptions(base, extra)
		if merged.Key != "base" {
			t.Fatalf("expected base key to win, got %q", merged.Key)
		}
		if want := []string{"change", "input"}; !reflect.DeepEqual(merged.Listen, want) {
			t.Fatalf("listen mismatch: got %v want %v", merged.Listen, want)
		}
		if want := []string{"target.value", "target.checked"}; !reflect.DeepEqual(merged.Props, want) {
			t.Fatalf("props mismatch: got %v want %v", merged.Props, want)
		}
	})

	t.Run("extra key overrides base", func(t *testing.T) {
		extra.Key = "override"
		merged := MergeEventOptions(base, extra)
		if merged.Key != "override" {
			t.Fatalf("expected override key, got %q", merged.Key)
		}
	})
}

func TestDefaultEventOptionsIsolation(t *testing.T) {
	opts := DefaultEventOptions("input")
	if len(opts.Listen) != 1 || opts.Listen[0] != "change" {
		t.Fatalf("unexpected preset listen list: %v", opts.Listen)
	}
	opts.Listen[0] = "mutated"
	opts.Props[0] = "mutated"

	next := DefaultEventOptions("input")
	if len(next.Listen) != 1 || next.Listen[0] != "change" {
		t.Fatalf("preset listen list leaked mutation: %v", next.Listen)
	}
	if len(next.Props) != 1 || next.Props[0] != "target.value" {
		t.Fatalf("preset props list leaked mutation: %v", next.Props)
	}
}

func TestSanitizeEventList(t *testing.T) {
	result := sanitizeEventList("click", []string{"", " click ", "CLICK", "focus", "change", "focus"})
	if want := []string{"focus", "change"}; !reflect.DeepEqual(result, want) {
		t.Fatalf("sanitizeEventList mismatch: got %v want %v", result, want)
	}
}

func TestSanitizeSelectorList(t *testing.T) {
	result := sanitizeSelectorList([]string{" target.value ", "", "target.value", "target.checked"})
	if want := []string{"target.value", "target.checked"}; !reflect.DeepEqual(result, want) {
		t.Fatalf("sanitizeSelectorList mismatch: got %v want %v", result, want)
	}
}

func TestMergeStringSetFold(t *testing.T) {
	result := mergeStringSet([]string{"Change"}, []string{"change", "Focus"}, true)
	if want := []string{"Change", "Focus"}; !reflect.DeepEqual(result, want) {
		t.Fatalf("mergeStringSet with fold true mismatch: got %v want %v", result, want)
	}
}
