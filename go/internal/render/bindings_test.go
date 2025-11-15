package render

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestBindingExtractorDedupSlotPaths(t *testing.T) {
	extractor := NewBindingExtractor()
	frame := elementFrame{
		componentID:   "cmp",
		basePath:      []int{1},
		componentPath: []int{2},
		bindings: []slotBinding{
			{slot: 4, childIndex: -1},
			{slot: 4, childIndex: -1},
			{slot: 4, childIndex: 3},
			{slot: 4, childIndex: 3},
		},
	}

	extractor.ExtractSlotPaths(frame)
	slots := extractor.SlotPaths()
	if len(slots) != 2 {
		t.Fatalf("expected 2 unique slot anchors, got %d (%v)", len(slots), slots)
	}
	if slots[0].ComponentID != "cmp" {
		t.Fatalf("expected component id cmp, got %s", slots[0].ComponentID)
	}
	if slots[0].Path[0].Kind != PathRangeOffset || slots[0].Path[0].Index != 1 {
		t.Fatalf("unexpected path segment %v", slots[0].Path[0])
	}
}

func TestBindingExtractorHandlerBindingsCloneData(t *testing.T) {
	extractor := NewBindingExtractor()
	element := h.Button()
	element.HandlerAssignments = map[string]dom.EventAssignment{
		"click": {ID: "handler-id", Listen: []string{"target.value"}},
	}
	frame := elementFrame{
		element:  element,
		attrSlot: 7,
	}

	extractor.ExtractHandlerBindings(frame)
	bindings := extractor.HandlerBindings()
	if len(bindings) != 1 {
		t.Fatalf("expected 1 handler binding, got %d", len(bindings))
	}
	element.HandlerAssignments["click"] = dom.EventAssignment{ID: "mutated"}
	if bindings[0].Handler != "handler-id" {
		t.Fatalf("expected handler id to be cloned, got %s", bindings[0].Handler)
	}
	if bindings[0].Listen[0] != "target.value" {
		t.Fatalf("expected listen slice to be cloned, got %v", bindings[0].Listen)
	}
}

func TestBindingExtractorUploadBindingsCloneAcceptSlice(t *testing.T) {
	extractor := NewBindingExtractor()
	element := h.Input()
	element.UploadBindings = []dom.UploadBinding{{
		UploadID: "upload-1",
		Accept:   []string{"image/png"},
	}}
	frame := elementFrame{
		element:       element,
		componentID:   "cmp",
		basePath:      []int{0},
		componentPath: []int{5},
	}

	extractor.ExtractUploadBindings(frame)
	bindings := extractor.UploadBindings()
	if len(bindings) != 1 {
		t.Fatalf("expected 1 upload binding, got %d", len(bindings))
	}

	element.UploadBindings[0].Accept[0] = "mutated"
	if bindings[0].Accept[0] != "image/png" {
		t.Fatalf("expected accept slice to be cloned, got %v", bindings[0].Accept)
	}
	if bindings[0].Path[0].Kind != PathRangeOffset {
		t.Fatalf("expected typed path to start with range offset, got %v", bindings[0].Path)
	}
}
