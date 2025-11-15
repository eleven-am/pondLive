package render

import (
	"reflect"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestComponentAnalyzerConcurrentMatchesSequential(t *testing.T) {
	tree := buildComplexTree()

	seq := NewComponentAnalyzer().Analyze(tree)
	concurrent := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 1,
	}).Analyze(tree)

	if !reflect.DeepEqual(seq, concurrent) {
		t.Fatalf("concurrent analysis mismatch:\nseq=%+v\nconcurrent=%+v", seq, concurrent)
	}

	if _, ok := seq.Components["root"]; !ok {
		t.Fatalf("expected root component span, got %+v", seq.Components)
	}
}

func TestComponentAnalyzerDetectsKeyedChildren(t *testing.T) {
	keyedTree := h.WrapComponent("root",
		h.Div(
			h.Ul(
				h.Li(h.Key("a"), h.Text("one")),
				h.Li(h.Key("b"), h.Text("two")),
			),
		),
	)
	plainTree := h.WrapComponent("root",
		h.Div(
			h.Ul(
				h.Li(h.Text("one")),
				h.Li(h.Text("two")),
			),
		),
	)

	keyed := NewComponentAnalyzer().Analyze(keyedTree)
	plain := NewComponentAnalyzer().Analyze(plainTree)

	if keyed.DynamicsCapacity <= plain.DynamicsCapacity {
		t.Fatalf("expected keyed list to require more dynamics (%d <= %d)", keyed.DynamicsCapacity, plain.DynamicsCapacity)
	}
}

func TestComponentAnalyzerFragmentKeyedChildren(t *testing.T) {
	keyedFragment := h.Fragment(
		h.Div(h.Key("v1"), h.Text("one")),
		h.Div(h.Key("v2"), h.Text("two")),
	)
	plainFragment := h.Fragment(
		h.Div(h.Text("one")),
		h.Div(h.Text("two")),
	)

	keyedTree := h.WrapComponent("root", h.Div(keyedFragment))
	plainTree := h.WrapComponent("root", h.Div(plainFragment))

	keyed := NewComponentAnalyzer().Analyze(keyedTree)
	plain := NewComponentAnalyzer().Analyze(plainTree)

	if keyed.DynamicsCapacity <= plain.DynamicsCapacity {
		t.Fatalf("expected keyed fragment to require more dynamics (%d <= %d)", keyed.DynamicsCapacity, plain.DynamicsCapacity)
	}
}

func TestComponentAnalyzerDynamicAttrsFromUploadsAndRefs(t *testing.T) {
	plainInput := h.Input()

	enhancedInput := h.Input()
	enhancedInput.RefID = "input-ref"
	enhancedInput.UploadBindings = []dom.UploadBinding{{
		UploadID: "upload-slot",
	}}

	baseTree := h.WrapComponent("root", h.Div(plainInput))
	enhancedTree := h.WrapComponent("root", h.Div(enhancedInput))

	base := NewComponentAnalyzer().Analyze(baseTree)
	enhanced := NewComponentAnalyzer().Analyze(enhancedTree)

	if enhanced.DynamicsCapacity <= base.DynamicsCapacity {
		t.Fatalf("expected ref/upload attrs to increase dynamics (%d <= %d)", enhanced.DynamicsCapacity, base.DynamicsCapacity)
	}
}

func buildComplexTree() h.Node {
	keyedFragment := h.Fragment(
		h.Div(h.Key("alpha"), h.Text("A")),
		h.Div(h.Key("beta"), h.Text("B")),
		h.Div(h.Key("gamma"), h.Text("C")),
		h.Div(h.Key("delta"), h.Text("D")),
		h.Div(h.Key("epsilon"), h.Text("E")),
	)

	keyedList := h.Ul(
		h.Li(h.Key("one"), h.Text("item 1")),
		h.Li(h.Key("two"), h.Textf("%d", 2)),
	)

	button := h.Button(
		h.MutableAttr("data-count", "0"),
		h.Text("Click"),
	)
	button.HandlerAssignments = map[string]dom.EventAssignment{
		"click": {ID: "handler", Listen: []string{"target.value"}},
	}

	return h.WrapComponent("root",
		h.Div(
			h.Text("static"),
			h.Textf("%d", 42),
			keyedList,
			keyedFragment,
			button,
		),
	)
}
