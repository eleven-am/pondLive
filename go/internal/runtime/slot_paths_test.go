package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// TestListContainerSlotPathsIncluded verifies that list container slots
// appear in both slotPaths and listPaths to prevent client-side resolution failures.
//
// Bug: When a keyed list is rendered (e.g., router slots), the server creates
// a SlotMeta for the list container but only adds it to listPaths, not slotPaths.
// This causes the client to fail when resolving slot boundary nodes.
func TestListContainerSlotPathsIncluded(t *testing.T) {

	component := h.WrapComponent("test-component",
		h.Div(
			h.Button(
				h.Text("Increment State ("),
				&h.TextNode{Value: "0", Mutable: true},
				h.Text(")"),
			),
			h.Nav(

				h.A(h.Key("/"), h.Href("/"), h.Span(h.Text("Home"))),
				h.A(h.Key("/about"), h.Href("/about"), h.Span(h.Text("About"))),
				h.A(h.Key("/users/123"), h.Href("/users/123"), h.Span(h.Text("User 123"))),
			),

			h.Span(
				&h.TextNode{Value: "Current path", Mutable: true},
			),
		),
	)

	structured, err := render.ToStructured(component)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	t.Logf("Dynamics count: %d", len(structured.D))
	for i, d := range structured.D {
		t.Logf("  Slot %d: kind=%v", i, d.Kind)
	}
	t.Logf("ListPaths count: %d", len(structured.ListPaths))
	for i, lp := range structured.ListPaths {
		t.Logf("  ListPath %d: slot=%d componentID=%s", i, lp.Slot, lp.ComponentID)
	}
	t.Logf("SlotPaths count: %d", len(structured.SlotPaths))
	for i, sp := range structured.SlotPaths {
		t.Logf("  SlotPath %d: slot=%d componentID=%s", i, sp.Slot, sp.ComponentID)
	}

	listSlots := make(map[int]bool)
	for _, listPath := range structured.ListPaths {
		listSlots[listPath.Slot] = true
	}

	if len(listSlots) == 0 {
		t.Fatal("Expected at least one list container slot, got none")
	}

	pathSlots := make(map[int]bool)
	for _, slotPath := range structured.SlotPaths {
		pathSlots[slotPath.Slot] = true
	}

	if len(structured.D) == 0 {
		t.Fatal("Expected dynamic slots, got none")
	}

	for listSlot := range listSlots {
		if !pathSlots[listSlot] {
			t.Errorf("List container slot %d is in listPaths but MISSING from slotPaths - this causes client-side resolution failures", listSlot)
		}
	}

	t.Logf("Total dynamic slots: %d", len(structured.D))
	t.Logf("SlotPaths count: %d", len(structured.SlotPaths))
	t.Logf("ListPaths count: %d", len(structured.ListPaths))
	t.Logf("List container slots: %v", getKeys(listSlots))
	t.Logf("Slots with paths: %v", getKeys(pathSlots))

	for i := 0; i < len(structured.D); i++ {
		inSlotPaths := pathSlots[i]
		inListPaths := listSlots[i]

		if !inSlotPaths && !inListPaths {
			t.Errorf("Slot %d has no path in either slotPaths or listPaths", i)
		}

		if inListPaths && !inSlotPaths {
			t.Errorf("Slot %d is a list container (in listPaths) but missing from slotPaths - CLIENT WILL FAIL", i)
		}
	}
}

// TestBootPayloadSlotConsistency verifies that the boot payload has consistent
// slot numbering between the slots array and slotPaths array.
func TestBootPayloadSlotConsistency(t *testing.T) {

	component := h.WrapComponent("boot-test",
		h.Div(
			h.Button(&h.TextNode{Value: "click me", Mutable: true}),
			h.Nav(
				h.A(h.Key("a"), h.Text("Link A")),
				h.A(h.Key("b"), h.Text("Link B")),
			),
			h.Span(&h.TextNode{Value: "status", Mutable: true}),
		),
	)

	structured, err := render.ToStructured(component)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	dynamics := encodeDynamics(structured.D)
	slotPaths := encodeSlotPaths(structured.SlotPaths)
	listPaths := encodeListPaths(structured.ListPaths)

	slotCount := len(dynamics)

	slotsWithPaths := make(map[int]bool)
	for _, sp := range slotPaths {
		slotsWithPaths[sp.Slot] = true
	}
	for _, lp := range listPaths {
		slotsWithPaths[lp.Slot] = true
	}

	missingSlots := []int{}
	for i := 0; i < slotCount; i++ {
		found := false
		for _, sp := range slotPaths {
			if sp.Slot == i {
				found = true
				break
			}
		}
		if !found {
			missingSlots = append(missingSlots, i)
		}
	}

	if len(missingSlots) > 0 {
		t.Errorf("Slots %v are missing from slotPaths (only in listPaths) - client cannot resolve boundaries", missingSlots)

		for _, slot := range missingSlots {
			inListPaths := false
			for _, lp := range listPaths {
				if lp.Slot == slot {
					inListPaths = true
					break
				}
			}
			if inListPaths {
				t.Logf("  Slot %d IS in listPaths but NOT in slotPaths ← BUG", slot)
			} else {
				t.Logf("  Slot %d is in neither slotPaths nor listPaths ← CRITICAL", slot)
			}
		}
	}

	t.Logf("Boot payload stats:")
	t.Logf("  Total slots (SlotMeta): %d", slotCount)
	t.Logf("  Slots with slotPaths: %d", len(slotPaths))
	t.Logf("  Slots with listPaths: %d", len(listPaths))
	t.Logf("  Missing from slotPaths: %v", missingSlots)
}

// TestRouterSlotScenario reproduces the exact router bug scenario
func TestRouterSlotScenario(t *testing.T) {

	component := h.WrapComponent("router-test",
		h.Div(
			h.H1(h.Text("PondLive Router Test")),
			h.Button(
				&h.TextNode{Value: "Increment State (0)", Mutable: true},
			),
			h.Hr(),
			h.H2(&h.TextNode{Value: "Set State (0)", Mutable: true}),
			h.Nav(

				h.A(h.Key("/"), h.Href("/"), h.Span(h.Text("Home"))),
				h.A(h.Key("/about"), h.Href("/about"), h.Span(h.Text("About"))),
				h.A(h.Key("/users/123"), h.Href("/users/123"), h.Span(h.Text("User 123"))),
				h.A(h.Key("/search"), h.Href("/search?filter=active&q=test"), h.Span(h.Text("Search"))),
				h.A(h.Key("/parent"), h.Href("/parent"), h.Span(h.Text("Parent"))),
				h.A(h.Key("/refs"), h.Href("/refs"), h.Span(h.Text("Refs Test"))),
			),

			h.Span(
				h.Div(
					h.H2(h.Text("Home Page")),
					h.Div(
						h.P(&h.TextNode{Value: "Current path: /", Mutable: true}),
						h.P(&h.TextNode{Value: "Current hash: ", Mutable: true}),
						h.P(
							h.Text("Click count: "),
							h.Span(&h.TextNode{Value: "0", Mutable: true}),
						),
						h.Button(&h.TextNode{Value: "Increment & Navigate to User", Mutable: true}),
					),
				),
			),
		),
	)

	structured, err := render.ToStructured(component)
	if err != nil {
		t.Fatalf("Failed to render: %v", err)
	}

	var navListSlot int = -1
	for _, listPath := range structured.ListPaths {
		navListSlot = listPath.Slot
		break
	}

	if navListSlot < 0 {
		t.Fatal("Expected to find navigation list slot")
	}

	foundInSlotPaths := false
	for _, slotPath := range structured.SlotPaths {
		if slotPath.Slot == navListSlot {
			foundInSlotPaths = true
			break
		}
	}

	if !foundInSlotPaths {
		t.Errorf("Navigation list slot %d is MISSING from slotPaths - this is the bug that breaks the router", navListSlot)
		t.Logf("This slot is in listPaths but not slotPaths")
		t.Logf("Client will hit text nodes when resolving slot %d boundaries", navListSlot)
		t.Logf("Result: Event handlers won't work, client becomes unresponsive")
	}
}

func getKeys(m map[int]bool) []int {
	keys := make([]int, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
