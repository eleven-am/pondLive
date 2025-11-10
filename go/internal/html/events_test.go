package html

import "testing"

func TestOnAppliesDefaultEventOptions(t *testing.T) {
	callback := func(Event) Updates { return nil }
	el := El(HTMLVideoElement{}, "video", On("timeupdate", callback))

	binding, ok := el.Events["timeupdate"]
	if !ok {
		t.Fatal("expected binding for timeupdate")
	}

	if !contains(binding.Listen, "play") || !contains(binding.Listen, "pause") {
		t.Fatalf("expected default listen metadata, got %+v", binding.Listen)
	}

	expectedProps := []string{"target.currentTime", "target.duration", "target.paused"}
	for _, prop := range expectedProps {
		if !contains(binding.Props, prop) {
			t.Fatalf("expected %q in default props, got %+v", prop, binding.Props)
		}
	}
}

func TestOnWithMergesDefaultAndCustomOptions(t *testing.T) {
	callback := func(Event) Updates { return nil }
	el := El(HTMLVideoElement{}, "video", OnWith("timeupdate", EventOptions{
		Listen: []string{"seeking", "pause"},
		Props:  []string{"target.buffered"},
	}, callback))

	binding := el.Events["timeupdate"]

	if !contains(binding.Listen, "play") {
		t.Fatalf("expected base metadata to include play, got %+v", binding.Listen)
	}
	if !contains(binding.Listen, "pause") {
		t.Fatalf("expected pause only once even if provided explicitly, got %+v", binding.Listen)
	}
	if !contains(binding.Listen, "seeking") {
		t.Fatalf("expected custom listen metadata to include seeking, got %+v", binding.Listen)
	}

	if !contains(binding.Props, "target.buffered") {
		t.Fatalf("expected custom prop selector to be merged, got %+v", binding.Props)
	}
	if !contains(binding.Props, "target.currentTime") {
		t.Fatalf("expected base prop selectors to remain, got %+v", binding.Props)
	}
}

func contains(list []string, target string) bool {
	for _, item := range list {
		if item == target {
			return true
		}
	}
	return false
}

func TestOnWithSetsCustomKey(t *testing.T) {
	callback := func(Event) Updates { return nil }
	el := El(HTMLDivElement{}, "div", OnWith("click", EventOptions{
		Key: "my-custom-key",
	}, callback))

	binding, ok := el.Events["click"]
	if !ok {
		t.Fatal("expected binding for click event")
	}

	if binding.Key != "my-custom-key" {
		t.Errorf("binding.Key = %q, want %q", binding.Key, "my-custom-key")
	}
}

func TestOnWithoutKeyLeavesKeyEmpty(t *testing.T) {
	callback := func(Event) Updates { return nil }
	el := El(HTMLDivElement{}, "div", OnWith("click", EventOptions{
		Props: []string{"target.value"},
	}, callback))

	binding, ok := el.Events["click"]
	if !ok {
		t.Fatal("expected binding for click event")
	}

	if binding.Key != "" {
		t.Errorf("binding.Key should be empty when not provided, got %q", binding.Key)
	}
}

func TestOnWithKeyAndOtherOptions(t *testing.T) {
	callback := func(Event) Updates { return nil }
	el := El(HTMLInputElement{}, "input", OnWith("input", EventOptions{
		Key:   "input-handler-1",
		Props: []string{"target.value"},
	}, callback))

	binding, ok := el.Events["input"]
	if !ok {
		t.Fatal("expected binding for input event")
	}

	if binding.Key != "input-handler-1" {
		t.Errorf("binding.Key = %q, want %q", binding.Key, "input-handler-1")
	}

	if !contains(binding.Props, "target.value") {
		t.Errorf("expected custom prop selector to be set, got %+v", binding.Props)
	}
}

func TestMergeEventOptionsPreservesKey(t *testing.T) {
	base := EventOptions{
		Key:   "base-key",
		Props: []string{"prop1"},
	}
	extra := EventOptions{
		Props: []string{"prop2"},
	}

	merged := mergeEventOptions(base, extra)

	if merged.Key != "base-key" {
		t.Errorf("merged.Key = %q, want %q (should preserve base when extra is empty)", merged.Key, "base-key")
	}
}

func TestMergeEventOptionsPreferExtraKey(t *testing.T) {
	base := EventOptions{
		Key:   "base-key",
		Props: []string{"prop1"},
	}
	extra := EventOptions{
		Key:   "extra-key",
		Props: []string{"prop2"},
	}

	merged := mergeEventOptions(base, extra)

	if merged.Key != "extra-key" {
		t.Errorf("merged.Key = %q, want %q (should prefer extra when both provided)", merged.Key, "extra-key")
	}
}
