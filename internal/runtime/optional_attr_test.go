package runtime_test

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestOmitEmptyAttributeIsAbsentWhenEmptyInSSRAndSerialize(t *testing.T) {
	var title *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		title = runtime.UseSignal(ctx, "")
		return el("div", work.Attribute("title", title, work.OmitEmpty()), work.Attribute("lang", "", work.OmitEmpty()), txt("x"))
	})

	requireEqual(t, app.SSR, `<div data-pi="1">x</div>`)
	requireEqual(t, app.Model.HTML(), `<div data-pi="1">x</div>`)
}

func TestOmitEmptyAttributeEditsUseAttrAndRemoveAttr(t *testing.T) {
	var title *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		title = runtime.UseSignal(ctx, "")
		return el("div", work.Attribute("title", title, work.OmitEmpty()), txt("x"))
	})

	edits := app.Do(func() { title.Set("hi") })
	if len(edits) != 1 || edits[0] != protocol.AttrEdit(1, "title", "hi") {
		t.Fatalf("edits = %v", edits)
	}
	requireEqual(t, app.Model.HTML(), `<div data-pi="1" title="hi">x</div>`)

	edits = app.Do(func() { title.Set("") })
	if len(edits) != 1 || edits[0] != protocol.RemoveAttrEdit(1, "title") {
		t.Fatalf("edits = %v", edits)
	}
	requireEqual(t, app.Model.HTML(), `<div data-pi="1">x</div>`)
}

func TestOmitEmptyAttributeStartingNonEmptyThenEmptyMatchesSerialize(t *testing.T) {
	var title *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		title = runtime.UseSignal(ctx, "a")
		return el("div", work.Attribute("title", title, work.OmitEmpty()), txt("x"))
	})

	requireEqual(t, app.SSR, `<div data-pi="1" title="a">x</div>`)
	app.Do(func() { title.Set("") })
	requireEqual(t, app.Model.HTML(), `<div data-pi="1">x</div>`)
}

func TestDynamicClassIsAbsentWhenEmpty(t *testing.T) {
	var cls *runtime.Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		cls = runtime.UseSignal(ctx, "")
		return el("div", work.Class(cls), txt("x"))
	})

	requireEqual(t, app.SSR, `<div data-pi="1">x</div>`)
	edits := app.Do(func() { cls.Set("on") })
	if len(edits) != 1 || edits[0] != protocol.AttrEdit(1, "class", "on") {
		t.Fatalf("edits = %v", edits)
	}
	edits = app.Do(func() { cls.Set("") })
	if len(edits) != 1 || edits[0] != protocol.RemoveAttrEdit(1, "class") {
		t.Fatalf("edits = %v", edits)
	}
	requireEqual(t, app.Model.HTML(), `<div data-pi="1">x</div>`)
}

func TestStaticEmptyClassIsOmitted(t *testing.T) {
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		return el("div", work.Class(""), txt("x"))
	})
	requireEqual(t, app.SSR, `<div>x</div>`)
}
