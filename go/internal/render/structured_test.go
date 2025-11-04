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
		h.Button(h.On("click", click), h.Text("0")),
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
	if attrSlots == 0 {
		t.Fatal("expected at least one attr slot")
	}
	// Ensure event handler registered and stored on attr slot and slot indexes annotated.
	found := false
	annotated := 0
	for idx, dyn := range structured.D {
		if dyn.Kind != DynAttrs {
			continue
		}
		if _, ok := dyn.Attrs["data-onclick"]; ok {
			found = true
		}
		if token := strings.TrimSpace(dyn.Attrs["data-slot-index"]); token == "" {
			t.Fatalf("expected data-slot-index on attr slot %d", idx)
		} else {
			annotated++
		}
	}
	if !found {
		t.Fatal("expected button attrs to include data-onclick")
	}
	if annotated == 0 {
		t.Fatal("expected attr slots to be annotated with data-slot-index")
	}
}

func TestToStructuredKeyedList(t *testing.T) {
	nodes := h.Ul(
		h.Li(h.Key("a"), h.Text("first")),
		h.Li(h.Key("b"), h.Text("second")),
	)
	structured := ToStructured(nodes)
	var listSlots int
	attrTagged := false
	var listSlotIndex int
	for i, dyn := range structured.D {
		if dyn.Kind != DynList {
			if dyn.Kind == DynAttrs && dyn.Attrs["data-list-slot"] != "" {
				attrTagged = true
			}
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
	if !attrTagged {
		t.Fatalf("expected parent attrs to be tagged with list slot, structured=%+v", structured.D)
	}
	if listSlotIndex <= 0 {
		t.Fatalf("expected list slot index to be positive, got %d", listSlotIndex)
	}
}

func TestDataSlotIndexesEncodeChildOffsets(t *testing.T) {
	tree := h.Div(
		h.Span(h.Text("first")),
		h.Text(" tail"),
	)
	structured := ToStructured(tree)
	if len(structured.D) < 4 {
		t.Fatalf("expected at least 4 dynamic slots, got %d", len(structured.D))
	}
	if got := structured.D[0].Attrs["data-slot-index"]; got != "0 3@1" {
		t.Fatalf("unexpected div annotation: got %q want %q", got, "0 3@1")
	}
	if got := structured.D[1].Attrs["data-slot-index"]; got != "1 2@0" {
		t.Fatalf("unexpected span annotation: got %q want %q", got, "1 2@0")
	}
}
