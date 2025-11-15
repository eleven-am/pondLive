package render

import (
	"reflect"
	"runtime"
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestConcurrentChildrenMatchesSequential(t *testing.T) {
	tree := h.WrapComponent("root",
		h.Div(
			h.Div(
				h.Text("static"),
				h.WrapComponent("inner-a", h.Span(h.Text("inner a"))),
			),
			h.Div(
				h.Textf("%d", 42),
				h.Button(h.Text("click")),
			),
			h.WrapComponent("inner-b",
				h.Div(
					h.Text("nested"),
					h.WrapComponent("inner-c", h.P(h.Text("deep child"))),
				),
			),
			h.Div(
				h.Input(h.MutableAttr("value", "hi")),
				h.Text("tail"),
			),
		),
	)

	sequential := renderWithOptions(t, tree, StructuredOptions{})
	concurrent := renderWithOptions(t, tree, StructuredOptions{
		ConcurrentChildren:        true,
		ChildConcurrencyThreshold: 1,
		MaxChildWorkers:           runtime.GOMAXPROCS(0),
	})

	if !reflect.DeepEqual(sequential, concurrent) {
		t.Fatalf("concurrent children render mismatch\nseq=%+v\nconc=%+v", sequential, concurrent)
	}
}

func TestConcurrentRowsMatchesSequential(t *testing.T) {
	row := func(key string) h.Node {
		return h.Li(
			h.Key(key),
			h.WrapComponent("comp-"+key, h.Div(h.Text("row "+key))),
			h.Button(h.Text("btn "+key)),
		)
	}

	tree := h.WrapComponent("root",
		h.Div(
			h.Ul(
				row("a"), row("b"), row("c"), row("d"), row("e"),
			),
		),
	)

	sequential := renderWithOptions(t, tree, StructuredOptions{})
	concurrent := renderWithOptions(t, tree, StructuredOptions{
		ConcurrentRows:          true,
		RowConcurrencyThreshold: 1,
		MaxRowWorkers:           runtime.GOMAXPROCS(0),
	})

	if !reflect.DeepEqual(sequential, concurrent) {
		t.Fatalf("concurrent rows render mismatch\nseq=%+v\nconc=%+v", sequential, concurrent)
	}
}

func renderWithOptions(t *testing.T, node h.Node, opts StructuredOptions) Structured {
	t.Helper()
	structured, err := ToStructuredWithHandlers(node, opts)
	if err != nil {
		t.Fatalf("ToStructuredWithHandlers failed: %v", err)
	}
	return structured
}
