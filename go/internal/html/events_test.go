package html

import "testing"

func TestOnAppliesDefaultEventOptions(t *testing.T) {
	callback := func(Event) Updates { return nil }
	el := Video(On("timeupdate", callback))

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
	el := Video(OnWith("timeupdate", EventOptions{
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
