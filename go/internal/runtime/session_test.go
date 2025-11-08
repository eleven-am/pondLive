package runtime

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	handlers "github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/internal/html"
)

type patchSink struct {
	ops []diff.Op
}

func (s *patchSink) send(ops []diff.Op) error {
	s.ops = append(s.ops, ops...)
	return nil
}

func TestSessionUseStateAndFlush(t *testing.T) {
	type props struct{}
	counter := func(ctx Ctx, _ props) h.Node {
		get, set := UseState(ctx, 0)
		handler := func(h.Event) h.Updates {
			set(get() + 1)
			return nil
		}
		return h.Div(
			h.Button(
				h.On("click", handler),
				h.Textf("%d", get()),
			),
		)
	}
	sess := NewSession(counter, props{})
	sink := &patchSink{}
	sess.SetPatchSender(sink.send)
	structured := sess.InitialStructured()
	if len(structured.D) == 0 {
		t.Fatalf("expected structured render to have dynamics")
	}
	handlerID := findHandlerAttr(structured, "data-onclick")
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}
	if err := sess.DispatchEvent(handlerID, handlers.Event{Name: "click"}); err != nil {
		t.Fatalf("DispatchEvent: %v", err)
	}
	if len(sink.ops) != 1 {
		b, _ := json.Marshal(sink.ops)
		t.Fatalf("expected one op, got %d (%s)", len(sink.ops), string(b))
	}
	set, ok := sink.ops[0].(diff.SetText)
	if !ok {
		t.Fatalf("expected SetText, got %T", sink.ops[0])
	}
	if set.Text != "1" {
		t.Fatalf("expected count to increment to 1, got %q", set.Text)
	}
}

func TestUseStateNoOpOnEqual(t *testing.T) {
	type props struct{}
	setter := new(func(int))
	valueGetter := new(func() int)
	component := func(ctx Ctx, _ props) h.Node {
		get, set := UseState(ctx, 5)
		*setter = set
		*valueGetter = get
		return h.Div(h.Textf("%d", get()))
	}
	sess := NewSession(component, props{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()
	(*setter)(5)
	if sess.Dirty() {
		t.Fatal("expected no dirty flag when setting identical value")
	}
	(*setter)(6)
	if !sess.Dirty() {
		t.Fatal("expected session to be marked dirty after change")
	}
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if (*valueGetter)() != 6 {
		t.Fatalf("expected getter to read updated value, got %d", (*valueGetter)())
	}
}

func TestSessionDirtyReflectsPendingFlush(t *testing.T) {
	type props struct{}
	component := func(ctx Ctx, _ props) h.Node { return h.Div() }
	sess := NewSession(component, props{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()

	if sess.Dirty() {
		t.Fatal("expected clean session after initial render")
	}

	triggered := false
	sess.enqueuePubsub(func() { triggered = true })
	if !sess.Dirty() {
		t.Fatal("expected pending pubsub task to mark session dirty")
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush: %v", err)
	}
	if !triggered {
		t.Fatal("expected pubsub task to run during flush")
	}
	if sess.Dirty() {
		t.Fatal("expected session to be clean after flush")
	}
}

func TestHookOrderMismatchPanics(t *testing.T) {
	type props struct{}
	toggle := true
	component := func(ctx Ctx, _ props) h.Node {
		if toggle {
			UseState(ctx, 1)
		}
		UseEffect(ctx, func() Cleanup { return nil })
		toggle = false
		return h.Div()
	}
	sess := NewSession(component, props{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()
	toggle = false
	sess.dirtyRoot = true
	err := sess.Flush()
	if err == nil {
		t.Fatal("expected flush to fail due to hook mismatch")
	}
	diag, ok := AsDiagnosticError(err)
	if !ok {
		t.Fatalf("expected diagnostic error, got %T", err)
	}
	if diag.Code != "flush_panic" {
		t.Fatalf("expected flush panic code, got %q", diag.Code)
	}
	if !strings.Contains(diag.Message, "hooks mismatch") {
		t.Fatalf("expected hooks mismatch message, got %q", diag.Message)
	}
	if diag.Hook != "UseEffect" {
		t.Fatalf("expected hook name UseEffect, got %q", diag.Hook)
	}
}

func TestComponentSessionResetClearsError(t *testing.T) {
	type props struct{}
	component := func(ctx Ctx, _ props) h.Node { return h.Div() }

	sess := NewSession(component, props{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()

	sess.handlePanic("flush", errors.New("boom"))
	if !sess.errored {
		t.Fatal("expected session to be marked errored")
	}

	if ok := sess.Reset(); !ok {
		t.Fatal("expected reset to succeed")
	}
	if sess.errored {
		t.Fatal("expected errored flag to clear after reset")
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("expected flush after reset to succeed, got %v", err)
	}
}

func TestKeyedChildStatePersistsOnReorder(t *testing.T) {
	type parentProps struct{}
	type childProps struct{ Key string }
	getters := map[string]func() int{}
	setters := map[string]func(int){}
	var setOrder func([]string)
	var child Component[childProps]
	parent := func(ctx Ctx, _ parentProps) h.Node {
		orderGet, orderSet := UseState(ctx, []string{"a", "b", "c"})
		setOrder = orderSet
		order := orderGet()
		items := make([]h.Item, 0, len(order))
		for _, key := range order {
			node := Render(ctx, child, childProps{Key: key}, WithKey(key))
			item, ok := node.(h.Item)
			if !ok {
				t.Fatalf("child node %T does not implement html.Item", node)
			}
			items = append(items, item)
		}
		return h.Ul(items...)
	}
	child = func(ctx Ctx, props childProps) h.Node {
		get, set := UseState(ctx, len(props.Key))
		getters[props.Key] = get
		setters[props.Key] = set
		return h.Li(
			h.Class(props.Key),
			h.Textf("%s:%d", props.Key, get()),
		)
	}
	sess := NewSession(parent, parentProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()
	if setters["b"] == nil {
		t.Fatalf("expected child setter to be captured")
	}
	setters["b"](42)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if getters["b"] == nil || getters["b"]() != 42 {
		t.Fatalf("expected child 'b' state to be 42, got %v", getters["b"]())
	}
	setOrder([]string{"c", "b", "a"})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if getters["b"] == nil || getters["b"]() != 42 {
		t.Fatalf("expected child 'b' state to persist after reorder, got %v", getters["b"]())
	}
}
