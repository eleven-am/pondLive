package portal

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestPortalCollectsNodes(t *testing.T) {
	state := &portalState{
		nodes: make([]work.Node, 0),
	}

	state.nodes = append(state.nodes, &work.Element{Tag: "div"})
	state.nodes = append(state.nodes, &work.Element{Tag: "span"})

	if len(state.nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %d", len(state.nodes))
	}

	if state.nodes[0].(*work.Element).Tag != "div" {
		t.Errorf("expected first node to be div, got %s", state.nodes[0].(*work.Element).Tag)
	}
	if state.nodes[1].(*work.Element).Tag != "span" {
		t.Errorf("expected second node to be span, got %s", state.nodes[1].(*work.Element).Tag)
	}
}

func TestPortalEmptyState(t *testing.T) {
	state := &portalState{
		nodes: make([]work.Node, 0),
	}

	if len(state.nodes) != 0 {
		t.Errorf("expected 0 nodes, got %d", len(state.nodes))
	}
}

func TestPortalMultipleAppends(t *testing.T) {
	state := &portalState{
		nodes: make([]work.Node, 0),
	}

	nodes1 := []work.Node{
		&work.Element{Tag: "div"},
		&work.Element{Tag: "span"},
	}
	nodes2 := []work.Node{
		&work.Element{Tag: "p"},
	}

	state.nodes = append(state.nodes, nodes1...)
	state.nodes = append(state.nodes, nodes2...)

	if len(state.nodes) != 3 {
		t.Fatalf("expected 3 nodes, got %d", len(state.nodes))
	}

	expectedTags := []string{"div", "span", "p"}
	for i, tag := range expectedTags {
		if state.nodes[i].(*work.Element).Tag != tag {
			t.Errorf("expected node %d to be %s, got %s", i, tag, state.nodes[i].(*work.Element).Tag)
		}
	}
}

func TestPortalContextDefaultIsNil(t *testing.T) {
	ctx := portalCtx
	if ctx == nil {
		t.Error("expected portalCtx to be initialized")
	}
}

func TestPortalStateClearsOnRerender(t *testing.T) {
	state := &portalState{
		nodes: make([]work.Node, 0),
	}

	state.nodes = append(state.nodes, &work.Element{Tag: "div"})
	state.nodes = append(state.nodes, &work.Element{Tag: "span"})

	if len(state.nodes) != 2 {
		t.Fatalf("expected 2 nodes before clear, got %d", len(state.nodes))
	}

	state.nodes = make([]work.Node, 0)

	if len(state.nodes) != 0 {
		t.Errorf("expected 0 nodes after clear, got %d", len(state.nodes))
	}

	state.nodes = append(state.nodes, &work.Element{Tag: "p"})

	if len(state.nodes) != 1 {
		t.Errorf("expected 1 node after re-adding, got %d", len(state.nodes))
	}
	if state.nodes[0].(*work.Element).Tag != "p" {
		t.Errorf("expected node to be p, got %s", state.nodes[0].(*work.Element).Tag)
	}
}
