package runtime

import (
	"reflect"
	"testing"
)

func TestTemplateInternDeduplicatesStatics(t *testing.T) {
	cache := newTemplateInternCache()
	first := cache.InternStatics([]string{"<div>", "</div>"})
	second := cache.InternStatics([]string{"<div>", "</div>"})
	if len(first) == 0 || len(second) == 0 {
		t.Fatalf("expected interned statics to be non-empty: first=%v second=%v", first, second)
	}
	if reflect.ValueOf(first).Pointer() != reflect.ValueOf(second).Pointer() {
		t.Fatalf("expected statics to share backing array")
	}
	original := []string{"<span>", "</span>"}
	interned := cache.InternStatics(original)
	original[0] = "mutated"
	if interned[0] == "mutated" {
		t.Fatalf("expected interned statics to be isolated from caller mutation, got %q", interned[0])
	}
}
