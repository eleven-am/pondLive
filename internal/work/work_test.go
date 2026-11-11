package work

import (
	"reflect"
	"testing"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/reactive"
)

type sig[T any] struct{ v T }

func (s *sig[T]) Get() T  { return s.v }
func (s *sig[T]) Peek() T { return s.v }

func TestTextOfSwitchesOnTheValueType(t *testing.T) {
	if n, ok := TextOf("a").(*Text); !ok || n.Value != "a" {
		t.Fatalf("string: %#v", TextOf("a"))
	}
	if n, ok := TextOf(5).(*Text); !ok || n.Value != "5" {
		t.Fatalf("int: %#v", TextOf(5))
	}
	if n, ok := TextOf(true).(*Text); !ok || n.Value != "true" {
		t.Fatalf("bool: %#v", TextOf(true))
	}
	for name, node := range map[string]Node{
		"string signal": TextOf(reactive.Readable[string](&sig[string]{"x"})),
		"int signal":    TextOf(reactive.Readable[int](&sig[int]{1})),
		"bool signal":   TextOf(reactive.Readable[bool](&sig[bool]{true})),
		"float signal":  TextOf(reactive.Readable[float64](&sig[float64]{1.5})),
		"other signal":  TextOf(&sig[uint8]{3}),
	} {
		bt, ok := node.(*BoundText)
		if !ok {
			t.Fatalf("%s: %#v", name, node)
		}
		_ = bt.Source.Get()
	}
	if got := TextOf(reactive.Readable[int](&sig[int]{42})).(*BoundText).Source.Get(); got != "42" {
		t.Fatalf("int signal text = %q", got)
	}
	if got := TextOf(&sig[uint8]{3}).(*BoundText).Source.Peek(); got != "3" {
		t.Fatalf("reflected signal text = %q", got)
	}
}

func TestUnsupportedValuesPanicWithATypedError(t *testing.T) {
	defer func() {
		r := recover()
		if _, ok := r.(*UnsupportedValueError); !ok {
			t.Fatalf("recovered %#v", r)
		}
	}()
	TextOf(struct{}{})
}

func TestAttributeKinds(t *testing.T) {
	el := BuildElement("input",
		Attribute("a", "x"), Attribute("b", true), Attribute("c", false),
		Attribute("d", reactive.Readable[string](&sig[string]{"s"})),
		Attribute("e", reactive.Readable[bool](&sig[bool]{true})),
		Attribute("a", "y"),
	)
	kinds := map[string]AttrKind{}
	for _, a := range el.Attrs {
		kinds[a.Name] = a.Kind
	}
	if kinds["a"] != AttrStatic || kinds["b"] != AttrPresent || kinds["c"] != AttrAbsent || kinds["d"] != AttrDynamic || kinds["e"] != AttrDynamicPresence {
		t.Fatalf("kinds = %v", kinds)
	}
	if len(el.Attrs) != 5 || el.Attrs[0].Value != "y" {
		t.Fatalf("a later attribute must replace an earlier one in place: %+v", el.Attrs)
	}
	if !el.Dynamic() {
		t.Fatal("an element with bound attributes is dynamic")
	}
	if BuildElement("p", Attribute("a", "x")).Dynamic() {
		t.Fatal("a static element is not dynamic")
	}
}

func TestClassPartsAndStylesDeclarationOrder(t *testing.T) {
	on := reactive.Readable[string](&sig[string]{"c"})
	el := BuildElement("div", Class("a b", on, nil), Class("d"), Style("x", "1"), Style("y", 2), Style("x", "3"))
	if len(el.Classes) != 4 || !el.Classes[2].Dynamic() || el.Classes[3].Static != "d" {
		t.Fatalf("classes = %+v", el.Classes)
	}
	if len(el.Styles) != 2 || el.Styles[0].Prop != "x" || el.Styles[0].Value != "3" || el.Styles[1].Value != "2" {
		t.Fatalf("styles = %+v", el.Styles)
	}
}

func TestStylesMapIsDeterministic(t *testing.T) {
	m := map[string]string{"z": "1", "a": "2", "m": "3"}
	for i := 0; i < 20; i++ {
		el := BuildElement("div", Styles(m))
		if el.Styles[0].Prop != "a" || el.Styles[1].Prop != "m" || el.Styles[2].Prop != "z" {
			t.Fatalf("styles = %+v", el.Styles)
		}
	}
}

func TestEventsMergeByNameAndOptions(t *testing.T) {
	calls := 0
	fn := func(Event) { calls++ }
	el := BuildElement("button",
		OnWith("click", metadata.EventOptions{Prevent: true, Debounce: 300, Fields: []string{"a", "b"}}, fn),
		OnWith("click", metadata.EventOptions{Stop: true, Debounce: 100, Fields: []string{"b", "c"}}, fn),
		On("input", fn),
	)
	if len(el.Events) != 2 {
		t.Fatalf("events = %+v", el.Events)
	}
	click := el.Events[0]
	if len(click.Fns) != 2 || !click.Options.Prevent || !click.Options.Stop || click.Options.Debounce != 100 ||
		len(click.Options.Fields) != 3 || !click.Options.RateMixed {
		t.Fatalf("merged click = %+v", click)
	}
}

func TestFragmentAttrsApplyToTheParent(t *testing.T) {
	el := BuildElement("div", NewFragment(Attribute("a", "1"), NewText("x")))
	if len(el.Attrs) != 1 || len(el.Children) != 1 {
		t.Fatalf("attrs=%v children=%d", el.Attrs, len(el.Children))
	}
}

func TestRemovedEventAndOptionMembersStayRemoved(t *testing.T) {
	for _, c := range []struct {
		typ    reflect.Type
		fields []string
	}{
		{reflect.TypeFor[Event](), []string{"Value", "Form", "Mods"}},
		{reflect.TypeFor[metadata.EventOptions](), []string{"Props"}},
	} {
		for _, name := range c.fields {
			if _, ok := c.typ.FieldByName(name); ok {
				t.Errorf("%s still has the field %s", c.typ, name)
			}
		}
	}
	if reflect.TypeFor[Event]().NumField() != 2 {
		t.Errorf("work.Event must be exactly Name and Payload")
	}
}
