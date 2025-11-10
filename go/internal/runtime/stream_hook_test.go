package runtime

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type listItem struct {
	Label string
}

type streamHandleCapture struct {
	handle StreamHandle[listItem]
}

func streamComponent(ctx Ctx, props *streamHandleCapture) h.Node {
	list, handle := UseStream[listItem](ctx, func(item StreamItem[listItem]) h.Node {
		return h.Li(h.Text(item.Value.Label))
	})
	if props != nil {
		props.handle = handle
	}
	return h.Ul(list)
}

type opSink struct {
	ops []diff.Op
}

func (s *opSink) send(ops []diff.Op) error {
	s.ops = append(s.ops, ops...)
	return nil
}

func (s *opSink) take() []diff.Op {
	out := append([]diff.Op(nil), s.ops...)
	s.ops = nil
	return out
}

func (s *opSink) takeList(slot int) []diff.List {
	ops := s.take()
	lists := make([]diff.List, 0, len(ops))
	for _, op := range ops {
		list, ok := op.(diff.List)
		if !ok || list.Slot != slot {
			continue
		}
		lists = append(lists, list)
	}
	return lists
}

func TestUseStreamListOperations(t *testing.T) {
	capture := &streamHandleCapture{}
	sess := NewSession(streamComponent, capture)
	sink := &opSink{}
	sess.SetPatchSender(sink.send)
	sess.InitialStructured()
	slot := -1
	handle := capture.handle
	if handle == nil {
		t.Fatal("expected handle to be captured during initial render")
	}

	if !handle.Append(StreamItem[listItem]{Key: "a", Value: listItem{Label: "alpha"}}) {
		t.Fatal("append should report change")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after append a: %v", err)
	}
	slot = findListSlot(sess.prev)
	if slot == -1 {
		t.Fatal("expected list slot after first append")
	}
	lists := sink.takeList(slot)
	if len(lists) > 0 {
		requireIns(t, lists, "a", 0)
	}
	assertRowOrder(t, sess.prev, slot, []string{"a"})

	if !handle.Append(StreamItem[listItem]{Key: "b", Value: listItem{Label: "beta"}}) {
		t.Fatal("append second item should report change")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after append b: %v", err)
	}
	slot = findListSlot(sess.prev)
	if slot == -1 {
		t.Fatal("expected list slot after second append")
	}
	lists = sink.takeList(slot)
	if len(lists) > 0 {
		requireIns(t, lists, "b", 1)
	}
	assertRowOrder(t, sess.prev, slot, []string{"a", "b"})

	if !handle.Upsert(StreamItem[listItem]{Key: "b", Value: listItem{Label: "beta v2"}}) {
		t.Fatal("upsert should report change")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after upsert: %v", err)
	}
	slot = findListSlot(sess.prev)
	if slot == -1 {
		t.Fatal("expected list slot after upsert")
	}
	sink.takeList(slot)
	if got := rowText(sess.prev, slot, "b"); got != "beta v2" {
		t.Fatalf("expected row text for b to update, got %q", got)
	}

	if !handle.InsertBefore("a", StreamItem[listItem]{Key: "b", Value: listItem{Label: "beta v3"}}) {
		t.Fatal("insert before should report change")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after reorder: %v", err)
	}
	slot = findListSlot(sess.prev)
	if slot == -1 {
		t.Fatal("expected list slot after reorder")
	}
	lists = sink.takeList(slot)
	boots := sess.consumeComponentBoots()
	if len(boots) > 0 {
		if !bootHasListOrder(boots, slot, []string{"b", "a"}) {
			t.Fatalf("expected component boot to reorder rows, got %+v", boots)
		}
	} else if !hasMov(lists, 1, 0) {
		t.Fatalf("expected move operation when reordering rows, got %+v", lists)
	}
	assertRowOrder(t, sess.prev, slot, []string{"b", "a"})
	if got := rowText(sess.prev, slot, "b"); got != "beta v3" {
		t.Fatalf("expected reordered row text to update, got %q", got)
	}

	if !handle.Delete("a") {
		t.Fatal("delete should report change for existing key")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after delete: %v", err)
	}
	slot = findListSlot(sess.prev)
	if slot == -1 {
		t.Fatal("expected list slot after delete")
	}
	lists = sink.takeList(slot)
	if len(lists) > 0 && !hasDel(lists, "a") {
		t.Fatalf("expected delete operation for key 'a', got %+v", lists)
	}
	assertRowOrder(t, sess.prev, slot, []string{"b"})

	if !handle.Clear() {
		t.Fatal("clear should report change when list not empty")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after clear: %v", err)
	}
	slot = findListSlot(sess.prev)
	lists = sink.takeList(slot)
	if len(lists) > 0 && !hasDel(lists, "b") {
		t.Fatalf("expected delete operation for remaining key 'b', got %+v", lists)
	}
	assertRowOrder(t, sess.prev, slot, nil)
}

func TestUseStreamResetAndItems(t *testing.T) {
	capture := &streamHandleCapture{}
	sess := NewSession(streamComponent, capture)
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()
	handle := capture.handle
	if handle == nil {
		t.Fatal("expected handle to be captured during initial render")
	}

	if !handle.Append(StreamItem[listItem]{Key: "seed", Value: listItem{Label: "seed"}}) {
		t.Fatal("expected seed append to report change")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after seed append: %v", err)
	}

	items := []StreamItem[listItem]{
		{Key: "x", Value: listItem{Label: "ex"}},
	}
	if !handle.Reset(items) {
		t.Fatal("expected reset to report change")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush after reset: %v", err)
	}
	got := handle.Items()
	if len(got) != 1 || got[0].Key != "x" {
		t.Fatalf("unexpected items snapshot: %+v", got)
	}
	got[0].Key = "mutated"
	if items := handle.Items(); items[0].Key != "x" {
		t.Fatalf("expected Items to return a copy, got %+v", items)
	}
}

func TestUseStreamPanicsOnInvalidUsage(t *testing.T) {
	t.Run("duplicate initial keys", func(t *testing.T) {
		component := func(ctx Ctx, _ struct{}) h.Node {
			UseStream[listItem](ctx, func(item StreamItem[listItem]) h.Node {
				return h.Li(h.Text(item.Value.Label))
			},
				StreamItem[listItem]{Key: "dup", Value: listItem{Label: "first"}},
				StreamItem[listItem]{Key: "dup", Value: listItem{Label: "second"}},
			)
			return h.Div()
		}
		sess := NewSession(component, struct{}{})
		_ = sess.InitialStructured()
		if !sess.errored {
			t.Fatal("expected session to be marked errored")
		}
		if sess.lastDiagnostic == nil || sess.lastDiagnostic.Message == "" {
			t.Fatal("expected diagnostic for duplicate keys")
		}
		if !strings.Contains(sess.lastDiagnostic.Message, "duplicate stream key") {
			t.Fatalf("expected duplicate key diagnostic, got %q", sess.lastDiagnostic.Message)
		}
	})

	t.Run("non element renderer", func(t *testing.T) {
		component := func(ctx Ctx, _ struct{}) h.Node {
			UseStream[listItem](ctx, func(StreamItem[listItem]) h.Node {
				return h.Text("oops")
			}, StreamItem[listItem]{Key: "bad", Value: listItem{Label: "bad"}})
			return h.Div()
		}
		sess := NewSession(component, struct{}{})
		_ = sess.InitialStructured()
		if !sess.errored {
			t.Fatal("expected session to be marked errored for invalid renderer")
		}
		if sess.lastDiagnostic == nil || sess.lastDiagnostic.Message == "" {
			t.Fatal("expected diagnostic when renderer does not return element")
		}
		if !strings.Contains(sess.lastDiagnostic.Message, "must return an *html.Element") {
			t.Fatalf("unexpected diagnostic message: %q", sess.lastDiagnostic.Message)
		}
	})

	t.Run("empty key on mutation", func(t *testing.T) {
		capture := &streamHandleCapture{}
		sess := NewSession(streamComponent, capture)
		sess.SetPatchSender(func([]diff.Op) error { return nil })
		sess.InitialStructured()
		if capture.handle == nil {
			t.Fatal("expected handle from initial render")
		}
		defer func() {
			if r := recover(); r == nil {
				t.Fatal("expected panic on empty key append")
			}
		}()
		capture.handle.Append(StreamItem[listItem]{Value: listItem{Label: "no key"}})
	})
}

func findListSlot(structured render.Structured) int {
	for i, dyn := range structured.D {
		if dyn.Kind == render.DynList {
			return i
		}
	}
	return -1
}

func assertRowOrder(t *testing.T, structured render.Structured, slot int, expected []string) {
	t.Helper()
	if len(expected) == 0 {
		if slot < 0 {
			return
		}
	} else {
		if slot < 0 || slot >= len(structured.D) {
			t.Fatalf("invalid slot index %d", slot)
		}
	}
	rows := []render.Row{}
	if slot >= 0 && slot < len(structured.D) {
		rows = structured.D[slot].List
	}
	if len(rows) != len(expected) {
		t.Fatalf("expected %d rows, got %d", len(expected), len(rows))
	}
	for i, row := range rows {
		if row.Key != expected[i] {
			t.Fatalf("expected row %d key %q, got %q", i, expected[i], row.Key)
		}
	}
}

func rowText(structured render.Structured, slot int, key string) string {
	if slot < 0 || slot >= len(structured.D) {
		return ""
	}
	rows := structured.D[slot].List
	for _, row := range rows {
		if row.Key != key || len(row.Slots) == 0 {
			continue
		}
		for _, idx := range row.Slots {
			if idx < 0 || idx >= len(structured.D) {
				continue
			}
			dyn := structured.D[idx]
			if dyn.Kind == render.DynText {
				return dyn.Text
			}
		}
	}
	return ""
}

func requireIns(t *testing.T, lists []diff.List, key string, pos int) {
	t.Helper()
	for _, list := range lists {
		for _, op := range list.Ops {
			ins, ok := op.(diff.Ins)
			if !ok {
				continue
			}
			if ins.Row.Key == key && ins.Pos == pos {
				return
			}
		}
	}
	t.Fatalf("expected insert for key %q at %d, got %+v", key, pos, lists)
}

func hasMov(lists []diff.List, from, to int) bool {
	for _, list := range lists {
		for _, op := range list.Ops {
			mov, ok := op.(diff.Mov)
			if !ok {
				continue
			}
			if mov.From == from && mov.To == to {
				return true
			}
		}
	}
	return false
}

func bootHasListOrder(boots []componentTemplateUpdate, slot int, expected []string) bool {
	for _, boot := range boots {
		if len(boot.listSlots) == 0 {
			continue
		}
		for _, listSlot := range boot.listSlots {
			if listSlot != slot {
				continue
			}
			idx := -1
			for i, slotMeta := range boot.slots {
				if slotMeta.AnchorID == listSlot {
					idx = i
					break
				}
			}
			if idx < 0 || idx >= len(boot.dynamics) {
				continue
			}
			rows := boot.dynamics[idx].List
			if len(rows) != len(expected) {
				continue
			}
			match := true
			for i, row := range rows {
				if row.Key != expected[i] {
					match = false
					break
				}
			}
			if match {
				return true
			}
		}
	}
	return false
}

func hasDel(lists []diff.List, key string) bool {
	for _, list := range lists {
		for _, op := range list.Ops {
			del, ok := op.(diff.Del)
			if !ok {
				continue
			}
			if del.Key == key {
				return true
			}
		}
	}
	return false
}
