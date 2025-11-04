package diff

import (
	"testing"
	"time"

	render "github.com/eleven-am/liveui/internal/render"
)

func TestDiffTextChange(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "A"}}}
	next := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "B"}}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	set, ok := ops[0].(SetText)
	if !ok {
		t.Fatalf("expected SetText op, got %T", ops[0])
	}
	if set.Slot != 0 || set.Text != "B" {
		t.Fatalf("unexpected payload: %+v", set)
	}
}

func TestDiffTextNoChange(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "same"}}}
	next := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "same"}}}

	if ops := Diff(prev, next); len(ops) != 0 {
		t.Fatalf("expected no ops, got %v", ops)
	}
}

func TestDiffTextMultipleSlotsSingleChange(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{
		{Kind: render.DynText, Text: "A"},
		{Kind: render.DynText, Text: "X"},
		{Kind: render.DynText, Text: "C"},
	}}
	next := render.Structured{D: []render.Dyn{
		{Kind: render.DynText, Text: "A"},
		{Kind: render.DynText, Text: "Y"},
		{Kind: render.DynText, Text: "C"},
	}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	set, ok := ops[0].(SetText)
	if !ok {
		t.Fatalf("expected SetText op, got %T", ops[0])
	}
	if set.Slot != 1 || set.Text != "Y" {
		t.Fatalf("unexpected payload: %+v", set)
	}
}

func TestDiffAttrsAddChangeRemove(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{
		Kind:  render.DynAttrs,
		Attrs: map[string]string{"class": "x", "data-old": "remove"},
	}}}
	next := render.Structured{D: []render.Dyn{{
		Kind:  render.DynAttrs,
		Attrs: map[string]string{"class": "y", "title": "new"},
	}}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	set, ok := ops[0].(SetAttrs)
	if !ok {
		t.Fatalf("expected SetAttrs op, got %T", ops[0])
	}
	if set.Slot != 0 {
		t.Fatalf("unexpected slot: %d", set.Slot)
	}
	if len(set.Upsert) != 2 || set.Upsert["class"] != "y" || set.Upsert["title"] != "new" {
		t.Fatalf("unexpected upsert payload: %+v", set.Upsert)
	}
	if len(set.Remove) != 1 || set.Remove[0] != "data-old" {
		t.Fatalf("unexpected remove payload: %+v", set.Remove)
	}
}

func TestDiffAttrsNoChange(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{Kind: render.DynAttrs, Attrs: map[string]string{"class": "x"}}}}
	next := render.Structured{D: []render.Dyn{{Kind: render.DynAttrs, Attrs: map[string]string{"class": "x"}}}}

	if ops := Diff(prev, next); len(ops) != 0 {
		t.Fatalf("expected no ops, got %v", ops)
	}
}

func TestDiffAttrsNilVsEmpty(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{Kind: render.DynAttrs, Attrs: nil}}}
	next := render.Structured{D: []render.Dyn{{Kind: render.DynAttrs, Attrs: map[string]string{}}}}

	if ops := Diff(prev, next); len(ops) != 0 {
		t.Fatalf("expected no ops for nil vs empty attrs, got %v", ops)
	}
}

func TestDiffMixedKinds(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{
		{Kind: render.DynText, Text: "Hello"},
		{Kind: render.DynAttrs, Attrs: map[string]string{"class": "x"}},
	}}
	next := render.Structured{D: []render.Dyn{
		{Kind: render.DynText, Text: "Hello, Bob"},
		{Kind: render.DynAttrs, Attrs: map[string]string{"class": "x"}},
	}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	set, ok := ops[0].(SetText)
	if !ok {
		t.Fatalf("expected SetText op, got %T", ops[0])
	}
	if set.Slot != 0 || set.Text != "Hello, Bob" {
		t.Fatalf("unexpected payload: %+v", set)
	}
}

func TestDiffKindMismatchPanics(t *testing.T) {
	prevStrict := StrictMismatch
	StrictMismatch = true
	defer func() {
		StrictMismatch = prevStrict
		if r := recover(); r == nil {
			t.Fatalf("expected panic for kind mismatch")
		}
	}()

	prev := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "A"}}}
	next := render.Structured{D: []render.Dyn{{Kind: render.DynAttrs, Attrs: map[string]string{"class": "x"}}}}

	Diff(prev, next)
}

func TestDiffLengthMismatchPanics(t *testing.T) {
	prevStrict := StrictMismatch
	StrictMismatch = true
	defer func() {
		StrictMismatch = prevStrict
		if r := recover(); r == nil {
			t.Fatalf("expected panic for slot count mismatch")
		}
	}()

	prev := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "A"}}}
	next := render.Structured{D: []render.Dyn{
		{Kind: render.DynText, Text: "A"},
		{Kind: render.DynText, Text: "B"},
	}}

	Diff(prev, next)
}

func TestDiffStrictMismatchDisabledIgnoresTemplateShift(t *testing.T) {
	prevStrict := StrictMismatch
	StrictMismatch = false
	defer func() { StrictMismatch = prevStrict }()

	prev := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "A"}}}
	next := render.Structured{D: []render.Dyn{{Kind: render.DynAttrs, Attrs: map[string]string{"class": "x"}}}}

	if ops := Diff(prev, next); ops != nil {
		t.Fatalf("expected nil ops when mismatch ignored, got %v", ops)
	}
}

func TestDiffAttrsHandlesNilNextAsRemoval(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{
		Kind:  render.DynAttrs,
		Attrs: map[string]string{"class": "x", "id": "remove-me"},
	}}}
	next := render.Structured{D: []render.Dyn{{
		Kind:  render.DynAttrs,
		Attrs: nil,
	}}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	set, ok := ops[0].(SetAttrs)
	if !ok {
		t.Fatalf("expected SetAttrs op, got %T", ops[0])
	}
	if len(set.Upsert) != 0 {
		t.Fatalf("expected no upserts, got %+v", set.Upsert)
	}
	expectedRemovals := map[string]struct{}{"class": {}, "id": {}}
	if len(set.Remove) != len(expectedRemovals) {
		t.Fatalf("expected %d removals, got %v", len(expectedRemovals), set.Remove)
	}
	for _, key := range set.Remove {
		if _, ok := expectedRemovals[key]; !ok {
			t.Fatalf("unexpected removal key %q", key)
		}
		delete(expectedRemovals, key)
	}
	if len(expectedRemovals) != 0 {
		t.Fatalf("missing removals for keys %v", expectedRemovals)
	}
}

func TestMismatchHandlerInvoked(t *testing.T) {
	prevStrict := StrictMismatch
	StrictMismatch = false
	defer func() { StrictMismatch = prevStrict; RegisterMismatchHandler(nil) }()

	done := make(chan struct{}, 1)
	RegisterMismatchHandler(func(err error) {
		if err == nil {
			t.Fatalf("expected error payload")
		}
		done <- struct{}{}
	})
	prev := render.Structured{D: []render.Dyn{{Kind: render.DynText, Text: "A"}}}
	next := render.Structured{D: []render.Dyn{{Kind: render.DynAttrs, Attrs: map[string]string{"class": "x"}}}}

	Diff(prev, next)

	select {
	case <-done:
	case <-time.After(100 * time.Millisecond):
		t.Fatalf("expected mismatch handler to be invoked")
	}
}

func TestDiffListInsert(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{
		Kind: render.DynList,
		List: []render.Row{{Key: "a", HTML: "<li>A</li>"}},
	}}}
	next := render.Structured{D: []render.Dyn{{
		Kind: render.DynList,
		List: []render.Row{{Key: "a", HTML: "<li>A</li>"}, {Key: "b", HTML: "<li>B</li>"}},
	}}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	list, ok := ops[0].(List)
	if !ok {
		t.Fatalf("expected List op, got %T", ops[0])
	}
	if list.Slot != 0 {
		t.Fatalf("expected list slot 0, got %d", list.Slot)
	}
	if len(list.Ops) != 1 {
		t.Fatalf("expected single child op, got %d", len(list.Ops))
	}
	ins, ok := list.Ops[0].(Ins)
	if !ok {
		t.Fatalf("expected insertion op, got %T", list.Ops[0])
	}
	if ins.Pos != 1 || ins.Row.Key != "b" || ins.Row.HTML == "" {
		t.Fatalf("unexpected insertion payload: %+v", ins)
	}
}

func TestDiffListDeleteAndMove(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{
		Kind: render.DynList,
		List: []render.Row{
			{Key: "a", HTML: "<li>A</li>"},
			{Key: "b", HTML: "<li>B</li>"},
			{Key: "c", HTML: "<li>C</li>"},
		},
	}}}
	next := render.Structured{D: []render.Dyn{{
		Kind: render.DynList,
		List: []render.Row{
			{Key: "c", HTML: "<li>C</li>"},
			{Key: "a", HTML: "<li>A</li>"},
		},
	}}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	list, ok := ops[0].(List)
	if !ok {
		t.Fatalf("expected List op, got %T", ops[0])
	}
	if len(list.Ops) != 2 {
		t.Fatalf("expected two child ops, got %d", len(list.Ops))
	}
	if del, ok := list.Ops[0].(Del); !ok || del.Key != "b" {
		t.Fatalf("expected delete for key b, got %#v", list.Ops[0])
	}
	if mov, ok := list.Ops[1].(Mov); !ok || mov.From != 1 || mov.To != 0 {
		t.Fatalf("expected move from 1 to 0, got %#v", list.Ops[1])
	}
}

func TestDiffAttrsHandlesNilPrevAsAddition(t *testing.T) {
	prev := render.Structured{D: []render.Dyn{{
		Kind:  render.DynAttrs,
		Attrs: nil,
	}}}
	next := render.Structured{D: []render.Dyn{{
		Kind:  render.DynAttrs,
		Attrs: map[string]string{"class": "x", "title": "hello"},
	}}}

	ops := Diff(prev, next)
	if len(ops) != 1 {
		t.Fatalf("expected 1 op, got %d", len(ops))
	}
	set, ok := ops[0].(SetAttrs)
	if !ok {
		t.Fatalf("expected SetAttrs op, got %T", ops[0])
	}
	expected := map[string]string{"class": "x", "title": "hello"}
	if len(set.Upsert) != len(expected) {
		t.Fatalf("expected %d upserts, got %+v", len(expected), set.Upsert)
	}
	for k, v := range expected {
		if set.Upsert[k] != v {
			t.Fatalf("expected %s=%q, got %+v", k, v, set.Upsert)
		}
	}
	if len(set.Remove) != 0 {
		t.Fatalf("expected no removals, got %+v", set.Remove)
	}
}

func TestDiffStaticsMismatchPanics(t *testing.T) {
	prevStrict := StrictMismatch
	StrictMismatch = true
	defer func() {
		StrictMismatch = prevStrict
		if r := recover(); r == nil {
			t.Fatalf("expected panic for statics mismatch")
		}
	}()

	prev := render.Structured{S: []string{"<div>"}, D: []render.Dyn{{Kind: render.DynText, Text: "A"}}}
	next := render.Structured{S: []string{"<span>"}, D: []render.Dyn{{Kind: render.DynText, Text: "A"}}}

	Diff(prev, next)
}
