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
	tree := h.WrapComponent("comp", h.Div(
		h.Class("wrapper"),
		h.Button(h.On("click", click), h.Textf("%d", 0)),
	))

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
	if attrSlots == 0 {
		t.Fatal("expected dynamic attr slot for handler bindings")
	}
	combined := strings.Join(structured.S, "")
	if strings.Contains(combined, "data-slot-index=") {
		t.Fatalf("expected slot annotations to move to dynamic attrs, got %q", combined)
	}
	if strings.Contains(combined, "data-onclick=") {
		t.Fatalf("expected static markup to omit data-onclick attribute, got %q", combined)
	}
	if strings.Contains(combined, "<!---->") {
		t.Fatalf("expected static markup to omit component markers, got %q", combined)
	}
	if len(structured.Bindings) == 0 {
		t.Fatal("expected handler bindings to be recorded")
	}
	var found bool
	for _, binding := range structured.Bindings {
		if binding.Event == "click" && binding.Handler != "" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected click handler binding to be recorded, got %+v", structured.Bindings)
	}
	var slotFound bool
	for _, path := range structured.SlotPaths {
		if path.TextChildIndex == -1 {
			slotFound = true
			break
		}
	}
	if !slotFound {
		t.Fatalf("expected at least one slot path targeting element attributes, got %+v", structured.SlotPaths)
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
	if strings.Contains(combined, "data-list-slot=") {
		t.Fatalf("expected static markup to omit list annotations, got %q", combined)
	}
	if strings.Contains(combined, "<!---->") {
		t.Fatalf("expected static markup to omit component markers, got %q", combined)
	}
	if listSlotIndex < 0 {
		t.Fatalf("expected list slot index to be non-negative, got %d", listSlotIndex)
	}
}

func TestDataSlotIndexesEncodeChildOffsets(t *testing.T) {
	tree := h.WrapComponent("comp", h.Div(
		h.Span(h.Textf("%s", "first")),
		h.Textf("%s", " tail"),
	))
	structured := ToStructured(tree)
	if len(structured.SlotPaths) == 0 {
		t.Fatalf("expected slot paths to be recorded, got %+v", structured.SlotPaths)
	}
	var divChild, spanChild bool
	for _, path := range structured.SlotPaths {
		if path.TextChildIndex == 0 {
			spanChild = true
		}
		if path.TextChildIndex == 1 {
			divChild = true
		}
	}
	if !spanChild || !divChild {
		t.Fatalf("expected slot paths to capture text child offsets, got %+v", structured.SlotPaths)
	}
}

func TestStructuredCapturesSlotPaths(t *testing.T) {
	tree := h.WrapComponent("comp", h.Div(
		h.Textf("%s", "value"),
	))
	structured := ToStructured(tree)
	if len(structured.SlotPaths) != 1 {
		t.Fatalf("expected a single slot path, got %d", len(structured.SlotPaths))
	}
	anchor := structured.SlotPaths[0]
	if anchor.ComponentID != "comp" {
		t.Fatalf("expected slot path component 'comp', got %q", anchor.ComponentID)
	}
	if len(anchor.ElementPath) != 0 {
		t.Fatalf("expected slot path to target the component root element, got %+v", anchor.ElementPath)
	}
	if anchor.TextChildIndex != 0 {
		t.Fatalf("expected text slot to reference first child, got %d", anchor.TextChildIndex)
	}
	var textSlot = -1
	for idx, dyn := range structured.D {
		if dyn.Kind == DynText {
			textSlot = idx
			break
		}
	}
	if textSlot < 0 {
		t.Fatalf("expected dynamic text slot, got %+v", structured.D)
	}
	if anchor.Slot != textSlot {
		t.Fatalf("expected slot path to reference slot %d, got %d", textSlot, anchor.Slot)
	}
	var component ComponentPath
	for _, cp := range structured.ComponentPaths {
		if cp.ComponentID == "comp" {
			component = cp
			break
		}
	}
	if component.ComponentID == "" {
		t.Fatalf("expected component path for 'comp', got %+v", structured.ComponentPaths)
	}
	if component.ParentID != "" {
		t.Fatalf("expected top-level component to have empty parent id, got %q", component.ParentID)
	}
	if len(component.ParentPath) != 0 {
		t.Fatalf("expected top-level component parent path to be empty, got %+v", component.ParentPath)
	}
	if len(component.FirstChild) != 1 || component.FirstChild[0] != 0 {
		t.Fatalf("expected component first child path [0], got %+v", component.FirstChild)
	}
	if len(component.LastChild) != 1 || component.LastChild[0] != 0 {
		t.Fatalf("expected component last child path [0], got %+v", component.LastChild)
	}
}

func TestStructuredCapturesListPaths(t *testing.T) {
	tree := h.WrapComponent("comp", h.Ul(
		h.Li(h.Key("a"), h.Text("first")),
		h.Li(h.Key("b"), h.Text("second")),
	))
	structured := ToStructured(tree)
	if len(structured.ListPaths) != 1 {
		t.Fatalf("expected a single list path, got %d", len(structured.ListPaths))
	}
	anchor := structured.ListPaths[0]
	if anchor.ComponentID != "comp" {
		t.Fatalf("expected list path component 'comp', got %q", anchor.ComponentID)
	}
	if len(anchor.ElementPath) != 0 {
		t.Fatalf("expected list path to reference component root element, got %+v", anchor.ElementPath)
	}
	var listSlot = -1
	for idx, dyn := range structured.D {
		if dyn.Kind == DynList {
			listSlot = idx
			break
		}
	}
	if listSlot < 0 {
		t.Fatalf("expected dynamic list slot, got %+v", structured.D)
	}
	if anchor.Slot != listSlot {
		t.Fatalf("expected list path to reference slot %d, got %d", listSlot, anchor.Slot)
	}
}

func TestKeyedRowCapturesSlotManifest(t *testing.T) {
	tree := h.WrapComponent("comp", h.Ul(
		h.Li(h.Key("a"), h.Textf("%s", "value")),
	))
	structured := ToStructured(tree)
	var row Row
	found := false
	for _, dyn := range structured.D {
		if dyn.Kind != DynList {
			continue
		}
		if len(dyn.List) == 0 {
			t.Fatalf("expected keyed list to include rows, got %+v", dyn.List)
		}
		row = dyn.List[0]
		found = true
		break
	}
	if !found {
		t.Fatalf("expected keyed list dynamics, got %+v", structured.D)
	}
	if len(row.SlotPaths) == 0 {
		t.Fatalf("expected keyed row to capture slot paths, got %+v", row.SlotPaths)
	}
	anchor := row.SlotPaths[0]
	if anchor.ComponentID != "comp" {
		t.Fatalf("expected row slot path to reference component 'comp', got %q", anchor.ComponentID)
	}
	if anchor.Slot < 0 {
		t.Fatalf("expected slot path to reference a valid slot, got %d", anchor.Slot)
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
