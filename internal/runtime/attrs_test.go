package runtime_test

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestAttrClassAndStyleAreRoutedToClassAndStyle(t *testing.T) {
	for _, name := range []string{"class", "style", "Class"} {
		body := func(ctx *Ctx) Node { return el("div", attr(name, "x"), attr("title", "ok")) }
		dev, _ := rawStart(t, true, body)
		if dev.html != "" {
			t.Fatalf("%s: dev must refuse Attr(%q), got %q", name, name, dev.html)
		}
		prod, err := rawStart(t, false, body)
		if err != nil {
			t.Fatal(err)
		}
		requireEqual(t, prod.html, `<div title="ok"></div>`)
		if !strings.Contains(prod.phases(), "attribute:routed") {
			t.Fatalf("%s: diagnostics = %s", name, prod.phases())
		}
	}
}

func TestAttrOnStateTheUserCanChangeIsRefusedAndBindOwnsIt(t *testing.T) {
	cases := []struct {
		name  string
		tag   string
		attrs []Item
		owned bool
	}{
		{"input value", "input", []Item{attr("value", "x")}, true},
		{"text input value", "input", []Item{attr("type", "text"), attr("value", "x")}, true},
		{"number input value", "input", []Item{attr("type", "number"), attr("value", "1")}, true},
		{"checkbox checked", "input", []Item{attr("type", "checkbox"), attr("checked", true)}, true},
		{"radio checked", "input", []Item{attr("type", "radio"), attr("checked", true)}, true},
		{"textarea value", "textarea", []Item{attr("value", "x")}, true},
		{"select value", "select", []Item{attr("value", "x")}, true},
		{"option selected", "option", []Item{attr("selected", true)}, true},
		{"details open", "details", []Item{attr("open", true)}, true},
		{"dialog open", "dialog", []Item{attr("open", true)}, true},
		{"checkbox value is the submitted value", "input", []Item{attr("type", "checkbox"), attr("value", "yes")}, false},
		{"radio value", "input", []Item{attr("type", "radio"), attr("value", "a")}, false},
		{"hidden value", "input", []Item{attr("type", "hidden"), attr("value", "a")}, false},
		{"submit value", "input", []Item{attr("type", "submit"), attr("value", "Go")}, false},
		{"option value", "option", []Item{attr("value", "a")}, false},
		{"button value", "button", []Item{attr("value", "a")}, false},
		{"div open", "div", []Item{attr("open", true)}, false},
	}
	for _, c := range cases {
		body := func(ctx *Ctx) Node { return el(c.tag, c.attrs...) }
		dev, _ := rawStart(t, true, body)
		if c.owned && dev.html != "" {
			t.Errorf("%s: dev must refuse the attribute, got %q", c.name, dev.html)
		}
		if !c.owned && dev.html == "" {
			t.Errorf("%s: dev refused a plain attribute", c.name)
		}
		prod, err := rawStart(t, false, body)
		if err != nil {
			t.Fatal(err)
		}
		reported := strings.Contains(prod.phases(), "attribute:bind-owned")
		if reported != c.owned {
			t.Errorf("%s: prod reported=%v, want %v (%s)", c.name, reported, c.owned, prod.phases())
		}
		if prod.html == "" {
			t.Errorf("%s: prod must still render", c.name)
		}
	}
}

func TestBindOpenOnDetailsAndDialogIsTwoWayWithoutAnEcho(t *testing.T) {
	for _, tag := range []string{"details", "dialog"} {
		var open *runtime.Signal[bool]
		app := runtimetest.New(t, func(ctx *Ctx) Node {
			open = runtime.UseSignal(ctx, true)
			return el("div", el(tag, work.Bind(open)), el("p", bound(open)))
		})
		wantSpec := map[string]string{
			"details": `[{"e":"toggle","p":"open"}]`,
			"dialog":  `[{"e":"close","p":"open"},{"e":"toggle","p":"open"}]`,
		}[tag]
		if !strings.Contains(app.SSR, ` open=""`) || !strings.Contains(app.SSR, wantSpec) {
			t.Fatalf("%s: ssr = %s", tag, app.SSR)
		}
		id := app.FindByAttr("data-pon", wantSpec)
		edits := app.Toggle(id, false)
		if len(attrEdits(edits, "open")) != 0 || open.Peek() {
			t.Fatalf("%s: the toggling element must not get its own state back: %v, open=%v", tag, edits, open.Peek())
		}
		edits = app.Do(func() { open.Set(true) })
		if len(attrEdits(edits, "open")) != 1 {
			t.Fatalf("%s: a server write must reach the element: %v", tag, edits)
		}
	}
}

func TestBindOpenOnADialogFollowsAnOpenTheServerDidNotAskFor(t *testing.T) {
	var open *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		open = runtime.UseSignal(ctx, false)
		return el("div", el("dialog", work.Bind(open)), el("p", bound(open)))
	})
	id := app.FindByAttr("data-pon", `[{"e":"close","p":"open"},{"e":"toggle","p":"open"}]`)

	app.Toggle(id, true)
	if !open.Peek() {
		t.Fatal("a dialog opened through show() or showModal() must set the bound signal")
	}
	app.Do(func() { open.Set(open.Peek()) })
	if !open.Peek() {
		t.Fatal("a reset must keep the dialog open")
	}
	app.Toggle(id, false)
	if open.Peek() {
		t.Fatal("closing the dialog must clear the bound signal")
	}
}

func TestBindOpenWantsABool(t *testing.T) {
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		return el("div", el("details", work.Bind("yes")))
	}, runtimetest.Dev(false))
	if len(app.Diagnostics("warning:BIND_TYPE_MISMATCH")) != 1 {
		t.Fatalf("diagnostics = %v", app.Diags)
	}
}

func TestBindRunsBeforeAnOnChangeHandlerOnTheSameElement(t *testing.T) {
	var sig *runtime.Signal[string]
	var seen []string
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		sig = runtime.UseSignal(ctx, "")
		return el("input", on(metadata.EditEvent, func(evt work.Event) {
			seen = append(seen, evt.Payload["value"].(string)+"|"+sig.Peek())
		}), work.Bind(sig))
	})
	if strings.Count(app.SSR, `"e":"~change"`) != 1 || !strings.Contains(app.SSR, `"p":"value"`) {
		t.Fatalf("one shared listener expected: %s", app.SSR)
	}
	app.Type(app.FindByAttr("data-pon", `[{"e":"~change","p":"value"}]`), "hello")
	if len(seen) != 1 || seen[0] != "hello|hello" {
		t.Fatalf("the handler must see the bound signal already updated: %v", seen)
	}
}

func specOf(t *testing.T, tag string, items ...Item) string {
	t.Helper()
	app := runtimetest.New(t, func(ctx *Ctx) Node { return el(tag, items...) })
	start := strings.Index(app.SSR, "data-pon='")
	if start < 0 {
		t.Fatalf("no data-pon in %s", app.SSR)
	}
	rest := app.SSR[start+len("data-pon='"):]
	return rest[:strings.Index(rest, "'")]
}

func TestNoisyEventsAreThrottledToAFrameAndScrollIsPassiveByDefault(t *testing.T) {
	for event, want := range map[string]string{
		"mousemove":   `[{"e":"mousemove","th":0}]`,
		"pointermove": `[{"e":"pointermove","th":0}]`,
		"touchmove":   `[{"e":"touchmove","ps":1,"th":0}]`,
		"scroll":      `[{"e":"scroll","ps":1,"th":0}]`,
		"wheel":       `[{"e":"wheel","ps":1,"th":0}]`,
		"drag":        `[{"e":"drag","th":0}]`,
		"dragover":    `[{"e":"dragover","th":0}]`,
		"timeupdate":  `[{"e":"timeupdate","th":0}]`,
		"click":       `[{"e":"click"}]`,
		"touchstart":  `[{"e":"touchstart","ps":1}]`,
	} {
		if got := specOf(t, "div", on(event, func(work.Event) {})); got != want {
			t.Errorf("%s: %s, want %s", event, got, want)
		}
	}
}

func TestExplicitRatesOverrideTheDefaults(t *testing.T) {
	noop := func(work.Event) {}
	cases := []struct {
		name string
		opts metadata.EventOptions
		want string
	}{
		{"ms", metadata.EventOptions{Throttle: 100}, `[{"e":"mousemove","th":100}]`},
		{"immediate", metadata.EventOptions{Throttle: metadata.RateOff}, `[{"e":"mousemove"}]`},
		{"frame", metadata.EventOptions{Throttle: metadata.RateFrame}, `[{"e":"mousemove","th":0}]`},
		{"debounce wins", metadata.EventOptions{Debounce: 50}, `[{"e":"mousemove","db":50}]`},
		{"keys and capture", metadata.EventOptions{Keys: []string{"a"}, CapturePointer: true, Throttle: metadata.RateOff}, `[{"e":"mousemove","cp":1,"k":["a"]}]`},
		{"prevent beats default passive", metadata.EventOptions{Prevent: true, Throttle: metadata.RateOff}, `[{"e":"mousemove","pv":1}]`},
	}
	for _, c := range cases {
		if got := specOf(t, "div", work.OnWith("mousemove", c.opts, noop)); got != c.want {
			t.Errorf("%s: %s, want %s", c.name, got, c.want)
		}
	}
	if got := specOf(t, "div", work.OnWith("touchmove", metadata.EventOptions{Prevent: true}, noop)); got != `[{"e":"touchmove","pv":1,"th":0}]` {
		t.Errorf("prevent on touchmove: %s", got)
	}
}

func TestRateMistakesAreReportedOnce(t *testing.T) {
	noop := func(work.Event) {}
	both := runtimetest.New(t, func(ctx *Ctx) Node {
		return el("div", work.OnWith("input", metadata.EventOptions{Debounce: 50, Throttle: 100}, noop))
	}, runtimetest.Dev(false))
	if len(both.Diagnostics("warning:EVENT_RATE_BOTH")) != 1 {
		t.Fatalf("diagnostics = %v", both.Diags)
	}
	mixed := runtimetest.New(t, func(ctx *Ctx) Node {
		return el("div", work.OnWith("input", metadata.EventOptions{Debounce: 50}, noop), work.OnWith("input", metadata.EventOptions{}, noop))
	}, runtimetest.Dev(false))
	if len(mixed.Diagnostics("warning:EVENT_RATE_CONFLICT")) != 1 {
		t.Fatalf("diagnostics = %v", mixed.Diags)
	}
	calm := runtimetest.New(t, func(ctx *Ctx) Node {
		return el("div", work.OnWith("input", metadata.EventOptions{Debounce: 50}, noop), work.OnWith("input", metadata.EventOptions{Debounce: 50}, noop))
	}, runtimetest.Dev(false))
	if len(calm.Diagnostics("warning:EVENT_RATE_CONFLICT"))+len(calm.Diagnostics("warning:EVENT_RATE_BOTH")) != 0 {
		t.Fatalf("diagnostics = %v", calm.Diags)
	}
}
