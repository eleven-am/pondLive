package render

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestToStructuredWithHandlersProducesStableSlots(t *testing.T) {
	reg := handlers.NewRegistry()
	click := func(h.Event) h.Updates { return h.Rerender() }
	tree := h.Div(
		h.Class("wrapper"),
		h.Button(h.On("click", click), h.Textf("%d", 0)),
	)

	structured := ToStructuredWithHandlers(tree, reg)
	if len(structured.S) == 0 {
		t.Fatal("expected statics to be populated")
	}
	if len(structured.D) == 0 {
		t.Fatal("expected dynamics to be populated")
	}
	var textSlots int
	var attrSlots int
	for _, dyn := range structured.D {
		switch dyn.Kind {
		case DynText:
			textSlots++
		case DynAttrs:
			attrSlots++
		}
	}
	if textSlots == 0 {
		t.Fatal("expected at least one text slot")
	}
	if attrSlots != 0 {
		t.Fatalf("expected no dynamic attr slots, got %d", attrSlots)
	}

	combined := strings.Join(structured.S, "")
	if !strings.Contains(combined, "data-onclick=") {
		t.Fatalf("expected static markup to include data-onclick attribute, got %q", combined)
	}
	if !strings.Contains(combined, "data-slot-index=") {
		t.Fatalf("expected static markup to include data-slot-index, got %q", combined)
	}
}

func TestToStructuredKeyedList(t *testing.T) {
	nodes := h.Ul(
		h.Li(h.Key("a"), h.Text("first")),
		h.Li(h.Key("b"), h.Text("second")),
	)
	structured := ToStructured(nodes)
	var listSlots int
	var listSlotIndex int
	for i, dyn := range structured.D {
		if dyn.Kind != DynList {
			continue
		}
		listSlots++
		listSlotIndex = i
		if len(dyn.List) != 2 {
			t.Fatalf("expected 2 rows, got %d", len(dyn.List))
		}
		if dyn.List[0].Key != "a" || dyn.List[1].Key != "b" {
			t.Fatalf("unexpected row keys: %+v", dyn.List)
		}
		for _, row := range dyn.List {
			for _, slot := range row.Slots {
				if slot <= i {
					t.Fatalf("expected row slot %d to appear after list slot %d", slot, i)
				}
			}
		}
	}
	if listSlots != 1 {
		t.Fatalf("expected a single list slot, got %d", listSlots)
	}
	combined := strings.Join(structured.S, "")
	if !strings.Contains(combined, "data-list-slot=") {
		t.Fatalf("expected static markup to include data-list-slot attribute, got %q", combined)
	}
	if listSlotIndex < 0 {
		t.Fatalf("expected list slot index to be non-negative, got %d", listSlotIndex)
	}
}

func TestDataSlotIndexesEncodeChildOffsets(t *testing.T) {
	tree := h.Div(
		h.Span(h.Textf("%s", "first")),
		h.Textf("%s", " tail"),
	)
	structured := ToStructured(tree)
	combined := strings.Join(structured.S, "")
	if !strings.Contains(combined, `data-slot-index="1@1"`) {
		t.Fatalf("expected div markup to include slot annotation, got %q", combined)
	}
	if !strings.Contains(combined, `data-slot-index="0@0"`) {
		t.Fatalf("expected span markup to include slot annotation, got %q", combined)
	}
}

func TestStaticTextDefaultsToStatics(t *testing.T) {
	tree := h.Div(h.Text("hello"))
	structured := ToStructured(tree)
	for _, dyn := range structured.D {
		if dyn.Kind == DynText {
			t.Fatalf("expected no dynamic text slots, got %+v", structured.D)
		}
	}
}

type recordingTracker struct {
	decisions []promotionCall
	promote   bool
}

type promotionCall struct {
	id      string
	path    []int
	value   string
	mutable bool
}

func (r *recordingTracker) ResolveTextPromotion(id string, path []int, value string, mutable bool) bool {
	call := promotionCall{id: id, path: append([]int(nil), path...), value: value, mutable: mutable}
	r.decisions = append(r.decisions, call)
	return r.promote
}

func (r *recordingTracker) ResolveAttrPromotion(string, []int, map[string]string, map[string]bool) bool {
	return r.promote
}

func TestPromotionTrackerControlsDynamicText(t *testing.T) {
	component := h.WrapComponent("comp", h.Div(h.Text("static")))
	tracker := &recordingTracker{}
	structured := ToStructuredWithOptions(component, StructuredOptions{Promotions: tracker})
	if len(tracker.decisions) != 1 {
		t.Fatalf("expected tracker to see one text node, got %d", len(tracker.decisions))
	}
	if tracker.decisions[0].id != "comp" {
		t.Fatalf("expected component id 'comp', got %q", tracker.decisions[0].id)
	}
	if len(tracker.decisions[0].path) != 1 || tracker.decisions[0].path[0] != 0 {
		t.Fatalf("unexpected path: %+v", tracker.decisions[0].path)
	}
	for _, dyn := range structured.D {
		if dyn.Kind == DynText {
			t.Fatalf("expected no dynamic text slots when tracker does not promote, got %+v", structured.D)
		}
	}

	promoteTracker := &recordingTracker{promote: true}
	promoted := ToStructuredWithOptions(component, StructuredOptions{Promotions: promoteTracker})
	foundText := false
	for _, dyn := range promoted.D {
		if dyn.Kind == DynText {
			foundText = true
			break
		}
	}
	if !foundText {
		t.Fatalf("expected promoted render to include a dynamic text slot, got %+v", promoted.D)
	}
}
