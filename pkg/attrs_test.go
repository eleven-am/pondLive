package pkg

import (
	"regexp"
	"sort"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

var classPattern = regexp.MustCompile(`class="([^"]*)"`)

func classesOf(t *testing.T, html string) []string {
	t.Helper()
	m := classPattern.FindStringSubmatch(html)
	if m == nil {
		t.Fatalf("no class in %s", html)
	}
	out := strings.Fields(m[1])
	sort.Strings(out)
	return out
}

func sameClasses(t *testing.T, got []string, want ...string) {
	t.Helper()
	sort.Strings(want)
	if strings.Join(got, " ") != strings.Join(want, " ") {
		t.Fatalf("classes = %v, want %v", got, want)
	}
}

func TestClassMergesTailwindConflictsAcrossStaticAndReactiveParts(t *testing.T) {
	var pad *Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		pad = UseSignal(ctx, "px-4")
		return Div(Class("px-2 py-1 text-sm", pad, "text-lg"))
	})
	sameClasses(t, classesOf(t, app.SSR), "py-1", "px-4", "text-lg")

	app.Do(func() { pad.Set("px-8 py-3") })
	sameClasses(t, classesOf(t, app.Model.HTML()), "px-8", "py-3", "text-lg")

	app.Do(func() { pad.Set("") })
	sameClasses(t, classesOf(t, app.Model.HTML()), "px-2", "py-1", "text-lg")
}

func TestClassDropsExactDuplicatesOfCustomClasses(t *testing.T) {
	var extra *Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		extra = UseSignal(ctx, "btn")
		return Div(Class("btn", "btn-primary", extra))
	})
	sameClasses(t, classesOf(t, app.SSR), "btn", "btn-primary")
	app.Do(func() { extra.Set("btn extra") })
	sameClasses(t, classesOf(t, app.Model.HTML()), "btn", "btn-primary", "extra")
}

func TestAConditionalClassIsAComputedString(t *testing.T) {
	var active *Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		active = UseSignal(ctx, false)
		on := UseComputed(ctx, func() string {
			if active.Get() {
				return "bg-red-500"
			}
			return ""
		})
		return Div(Class("p-2 bg-blue-500", on))
	})
	sameClasses(t, classesOf(t, app.SSR), "p-2", "bg-blue-500")
	app.Do(func() { active.Set(true) })
	sameClasses(t, classesOf(t, app.Model.HTML()), "p-2", "bg-red-500")
	app.Do(func() { active.Set(false) })
	sameClasses(t, classesOf(t, app.Model.HTML()), "p-2", "bg-blue-500")
}

func TestBooleanHelpersTakeABoolOrASignalAndNothingElse(t *testing.T) {
	var saving *Signal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		saving = UseSignal(ctx, true)
		return Div(
			Button(Disabled(saving), Text("a")),
			Button(Disabled(false), Text("b")),
			Input(Required(true), Readonly(saving), Autofocus(true)),
			Video(Muted(true), Autoplay(true), Controls(true), Loop(false)),
			Select(Multiple(true)),
		)
	})
	html := app.SSR
	for _, want := range []string{`<button data-pi="1" disabled="">a</button>`, `<button>b</button>`, `required=""`, `autofocus=""`, `muted=""`, `autoplay=""`, `controls=""`, `multiple=""`} {
		if !strings.Contains(html, want) {
			t.Errorf("missing %q in %s", want, html)
		}
	}
	if strings.Contains(html, "loop") {
		t.Errorf("Loop(false) must not render: %s", html)
	}
	edits := app.Do(func() { saving.Set(false) })
	if len(edits) == 0 || strings.Contains(app.Model.HTML(), `disabled=""`) {
		t.Fatalf("the signal must remove the attribute: %v %s", edits, app.Model.HTML())
	}
	for _, bad := range []func(){
		func() { Disabled("yes") },
		func() { Required(1) },
		func() { Muted([]int{1}) },
	} {
		func() {
			defer func() {
				if recover() == nil {
					t.Fatal("a non-bool value must panic")
				}
			}()
			bad()
		}()
	}
}

func TestNamedHelpersAreAttrWithTheNameFilledIn(t *testing.T) {
	cases := []struct {
		name string
		item Item
		want string
	}{
		{"id", ID("a"), "id"},
		{"href", Href("/x"), "href"},
		{"src", Src("/x"), "src"},
		{"target", Target("_blank"), "target"},
		{"rel", Rel("noopener"), "rel"},
		{"title", Title("t"), "title"},
		{"alt", Alt("a"), "alt"},
		{"type", Type("text"), "type"},
		{"name", Name("n"), "name"},
		{"placeholder", Placeholder("p"), "placeholder"},
		{"data", Data("k", "v"), "data-k"},
		{"aria", Aria("label", "v"), "aria-label"},
		{"disabled", Disabled(true), "disabled"},
		{"required", Required(true), "required"},
		{"readonly", Readonly(true), "readonly"},
		{"autofocus", Autofocus(true), "autofocus"},
		{"autoplay", Autoplay(true), "autoplay"},
		{"controls", Controls(true), "controls"},
		{"loop", Loop(true), "loop"},
		{"muted", Muted(true), "muted"},
		{"multiple", Multiple(true), "multiple"},
	}
	for _, c := range cases {
		el := work.BuildElement("div", c.item)
		if len(el.Attrs) != 1 || el.Attrs[0].Name != c.want {
			t.Errorf("%s: attrs = %+v, want name %s", c.name, el.Attrs, c.want)
		}
	}
}

func TestStylesWrapsStyleSoValuesCanBeSignalsAndOrderIsFixed(t *testing.T) {
	var width *Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		width = UseSignal(ctx, "10px")
		return Div(Styles(map[string]any{"width": width, "color": "red", "order": 3, "a-first": "1"}))
	})
	requireHTML(t, app.SSR, `<div data-pi="1" style="a-first: 1; color: red; order: 3; width: 10px;"></div>`)
	app.Do(func() { width.Set("20px") })
	if !strings.Contains(app.Model.HTML(), "width: 20px") {
		t.Fatalf("a signal style must follow: %s", app.Model.HTML())
	}
	for i := 0; i < 50; i++ {
		el := work.BuildElement("div", Styles(map[string]any{"z": 1, "m": 2, "a": 3, "q": 4}))
		var got []string
		for _, s := range el.Styles {
			got = append(got, s.Prop)
		}
		if strings.Join(got, "") != "amqz" {
			t.Fatalf("order = %v", got)
		}
	}
}

func TestRemovedHelpersAreGone(t *testing.T) {
	src, err := removedNames()
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"ClassIf", "CN", "ClassPart", "InlineStyles", "Checked", "Selected", "Value", "Open", "OnKeyPress", "OnFileChange", "OnInput", "OnWithProps"} {
		if src[name] {
			t.Errorf("pkg still declares %s", name)
		}
	}
	for _, name := range []string{"Class", "Styles", "Throttle", "Debounce", "Frame", "Immediate", "Lazy", "Key", "CapturePointer", "Fields", "AllFields", "Bind", "OnChange"} {
		if !src[name] {
			t.Errorf("pkg is missing %s", name)
		}
	}
}

func TestAttrOmitEmptyDropsAnEmptyValueAndFollowsASignal(t *testing.T) {
	var title *Signal[string]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		title = UseSignal(ctx, "")
		return Div(Attr("title", title, OmitEmpty()), Attr("lang", "", OmitEmpty()), Attr("dir", "ltr", OmitEmpty()), Text("x"))
	})
	requireHTML(t, app.SSR, `<div data-pi="1" dir="ltr">x</div>`)
	app.Do(func() { title.Set("hello") })
	if !strings.Contains(app.Model.HTML(), `title="hello"`) {
		t.Fatalf("html = %s", app.Model.HTML())
	}
	app.Do(func() { title.Set("") })
	if strings.Contains(app.Model.HTML(), "title=") {
		t.Fatalf("an emptied value must remove the attribute: %s", app.Model.HTML())
	}
}
