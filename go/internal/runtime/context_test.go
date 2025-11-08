package runtime

import (
	"fmt"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type emptyProps struct{}

func firstText(structured render.Structured) string {
	for _, dyn := range structured.D {
		if dyn.Kind == render.DynText {
			return dyn.Text
		}
	}
	combined := strings.Join(structured.S, "")
	if combined == "" {
		return ""
	}
	var segment strings.Builder
	inTag := false
	inComment := false
	for i := 0; i < len(combined); i++ {
		if inComment {
			if i+2 < len(combined) && combined[i] == '-' && combined[i+1] == '-' && combined[i+2] == '>' {
				inComment = false
				i += 2
			}
			continue
		}
		ch := combined[i]
		if ch == '<' {
			if segment.Len() > 0 {
				text := strings.TrimSpace(segment.String())
				segment.Reset()
				if text != "" {
					return text
				}
			}
			inTag = true
			if i+3 < len(combined) && combined[i+1] == '!' && combined[i+2] == '-' && combined[i+3] == '-' {
				inComment = true
			}
			continue
		}
		if ch == '>' && inTag {
			inTag = false
			continue
		}
		if inTag {
			continue
		}
		segment.WriteByte(ch)
	}
	if segment.Len() > 0 {
		text := strings.TrimSpace(segment.String())
		if text != "" {
			return text
		}
	}
	return ""
}

type opRecorder struct {
	ops []diff.Op
}

func (r *opRecorder) send(ops []diff.Op) error {
	r.ops = append(r.ops, ops...)
	return nil
}

func TestContextProvideUsePair(t *testing.T) {
	theme := NewContext[string]("light")
	child := func(ctx Ctx, _ emptyProps) h.Node {
		return h.Span(h.Text(theme.Use(ctx)))
	}
	var setTheme func(string)
	parent := func(ctx Ctx, _ emptyProps) h.Node {
		get, set := theme.UsePair(ctx)
		setTheme = set
		return theme.Provide(ctx, get(), func() h.Node {
			return Render(ctx, child, emptyProps{})
		})
	}
	sess := NewSession(parent, emptyProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	structured := sess.InitialStructured()
	if got := firstText(structured); got != "light" {
		t.Fatalf("expected initial theme 'light', got %q", got)
	}
	if setTheme == nil {
		t.Fatalf("expected setter to be captured")
	}
	setTheme("dark")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if got := firstText(sess.prev); got != "dark" {
		t.Fatalf("expected updated theme 'dark', got %q", got)
	}
}

func TestContextProvideStaticValue(t *testing.T) {
	theme := NewContext[string]("light")
	child := func(ctx Ctx, _ emptyProps) h.Node {
		return h.Span(h.Text(theme.Use(ctx)))
	}
	parent := func(ctx Ctx, _ emptyProps) h.Node {
		return theme.Provide(ctx, "dark", func() h.Node {
			return Render(ctx, child, emptyProps{})
		})
	}
	sess := NewSession(parent, emptyProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	structured := sess.InitialStructured()
	if got := firstText(structured); got != "dark" {
		t.Fatalf("expected child to see provided value 'dark', got %q", got)
	}
}

func TestContextProvideFunc(t *testing.T) {
	text := NewContext[string]("zero")
	child := func(ctx Ctx, _ emptyProps) h.Node {
		return h.Span(h.Text(text.Use(ctx)))
	}
	var setCount func(int)
	parent := func(ctx Ctx, _ emptyProps) h.Node {
		count, set := UseState(ctx, 1)
		setCount = func(v int) { set(v) }
		compute := func() string {
			if count() == 1 {
				return "one"
			}
			if count() == 2 {
				return "two"
			}
			return "other"
		}
		deps := []any{count()}
		return text.ProvideFunc(ctx, compute, deps, func() h.Node {
			return Render(ctx, child, emptyProps{})
		})
	}
	sess := NewSession(parent, emptyProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	structured := sess.InitialStructured()
	if got := firstText(structured); got != "one" {
		t.Fatalf("expected initial derived value 'one', got %q", got)
	}
	if setCount == nil {
		t.Fatalf("expected setter to be captured")
	}
	setCount(2)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if got := firstText(sess.prev); got != "two" {
		t.Fatalf("expected derived value to update to 'two', got %q", got)
	}
}

func TestContextProvideFuncSetter(t *testing.T) {
	text := NewContext[string]("zero")
	var setText func(string)
	child := func(ctx Ctx, _ emptyProps) h.Node {
		get, set := text.UsePair(ctx)
		setText = set
		return h.Span(h.Text(get()))
	}
	var setCount func(int)
	parent := func(ctx Ctx, _ emptyProps) h.Node {
		count, update := UseState(ctx, 0)
		setCount = update
		compute := func() string {
			return fmt.Sprintf("count:%d", count())
		}
		deps := []any{count()}
		return text.ProvideFunc(ctx, compute, deps, func() h.Node {
			return Render(ctx, child, emptyProps{})
		})
	}
	sess := NewSession(parent, emptyProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	structured := sess.InitialStructured()
	if got := firstText(structured); got != "count:0" {
		t.Fatalf("expected initial derived value 'count:0', got %q", got)
	}
	if setText == nil {
		t.Fatalf("expected context setter to be captured")
	}
	setText("manual")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if got := firstText(sess.prev); got != "manual" {
		t.Fatalf("expected manual override 'manual', got %q", got)
	}
	if setCount == nil {
		t.Fatalf("expected state setter to be captured")
	}
	setCount(1)
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if got := firstText(sess.prev); got != "count:1" {
		t.Fatalf("expected derived value to update to 'count:1', got %q", got)
	}
}

func TestContextUsePairLocalFallback(t *testing.T) {
	theme := NewContext[string]("light")
	child := func(ctx Ctx, _ emptyProps) h.Node {
		return h.Span(h.Text(theme.Use(ctx)))
	}
	var getLocal func() string
	var setLocal func(string)
	parent := func(ctx Ctx, _ emptyProps) h.Node {
		get, set := theme.UsePair(ctx)
		getLocal, setLocal = get, set
		return Render(ctx, child, emptyProps{})
	}
	sess := NewSession(parent, emptyProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	structured := sess.InitialStructured()
	if got := firstText(structured); got != "light" {
		t.Fatalf("expected child to see default value 'light', got %q", got)
	}
	if getLocal == nil || setLocal == nil {
		t.Fatalf("expected local getters and setters")
	}
	setLocal("dark")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if got := getLocal(); got != "dark" {
		t.Fatalf("expected local state to update to 'dark', got %q", got)
	}
	if got := firstText(sess.prev); got != "light" {
		t.Fatalf("expected child to continue seeing default 'light', got %q", got)
	}
}

func TestContextNestedProviders(t *testing.T) {
	theme := NewContext[string]("light")
	inner := func(ctx Ctx, _ emptyProps) h.Node {
		return h.Span(h.Text(theme.Use(ctx)))
	}
	outer := func(ctx Ctx, _ emptyProps) h.Node {
		get, set := theme.UsePair(ctx)
		_ = set
		return theme.Provide(ctx, get(), func() h.Node {
			return Render(ctx, inner, emptyProps{})
		})
	}
	wrapper := func(ctx Ctx, _ emptyProps) h.Node {
		return theme.Provide(ctx, "outer", func() h.Node {
			return Render(ctx, outer, emptyProps{})
		})
	}
	sess := NewSession(wrapper, emptyProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	structured := sess.InitialStructured()
	if got := firstText(structured); got != "outer" {
		t.Fatalf("expected inner to inherit outer state 'outer', got %q", got)
	}
	// Update outer provider
	var outerSetter func(string)
	outerWithState := func(ctx Ctx, _ emptyProps) h.Node {
		get, set := theme.UsePair(ctx)
		outerSetter = set
		return theme.Provide(ctx, get(), func() h.Node {
			return Render(ctx, inner, emptyProps{})
		})
	}
	wrapperWithState := func(ctx Ctx, _ emptyProps) h.Node {
		return theme.Provide(ctx, "outer", func() h.Node {
			return Render(ctx, outerWithState, emptyProps{})
		})
	}
	sess = NewSession(wrapperWithState, emptyProps{})
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()
	if outerSetter == nil {
		t.Fatalf("expected to capture setter from outer provider")
	}
	outerSetter("inner")
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if got := firstText(sess.prev); got != "inner" {
		t.Fatalf("expected nested provider update to propagate, got %q", got)
	}
}

func TestContextUseSelect(t *testing.T) {
	type data struct {
		A string
		B string
	}
	store := NewContext[data](data{A: "one", B: "alpha"})
	renders := 0
	child := func(ctx Ctx, _ emptyProps) h.Node {
		renders++
		val := UseSelect(ctx, store, func(d data) string { return d.A }, func(a, b string) bool { return a == b })
		return h.Div(h.Text(val))
	}
	var setStore func(data)
	parent := func(ctx Ctx, _ emptyProps) h.Node {
		get, set := store.UsePair(ctx)
		setStore = set
		return store.Provide(ctx, get(), func() h.Node {
			return Render(ctx, child, emptyProps{})
		})
	}
	sess := NewSession(parent, emptyProps{})
	recorder := &opRecorder{}
	sess.SetPatchSender(recorder.send)
	sess.InitialStructured()
	if renders != 1 {
		t.Fatalf("expected initial render once, got %d", renders)
	}
	setStore(data{A: "one", B: "beta"})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if len(recorder.ops) != 0 {
		t.Fatalf("expected no ops when derived value unchanged, got %d", len(recorder.ops))
	}
	recorder.ops = nil
	setStore(data{A: "two", B: "beta"})
	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if len(recorder.ops) == 0 {
		boots := sess.consumeComponentBoots()
		if len(boots) == 0 {
			t.Fatalf("expected ops or component boot when derived value changes")
		}
	}
}

func TestContextRequirePanicsWithoutProvider(t *testing.T) {
	theme := NewContext[string]("light")
	component := func(ctx Ctx, _ emptyProps) h.Node {
		theme.Require(ctx)
		return h.Div()
	}
	sess := NewSession(component, emptyProps{})
	_ = sess.InitialStructured()
	if sess.lastDiagnostic == nil {
		t.Fatal("expected diagnostic when provider missing")
	}
	if !sess.errored {
		t.Fatal("expected session to be marked errored")
	}
	diag := sess.lastDiagnostic
	if diag.Code != "initial_panic" {
		t.Fatalf("expected initial panic code, got %q", diag.Code)
	}
	if !strings.Contains(diag.Message, "missing provider for context") {
		t.Fatalf("expected missing provider message, got %q", diag.Message)
	}
}

func TestContextDefaultWhenMissing(t *testing.T) {
	theme := NewContext[string]("light")
	component := func(ctx Ctx, _ emptyProps) h.Node {
		return h.Div(h.Text(theme.Use(ctx)))
	}
	sess := NewSession(component, emptyProps{})
	structured := sess.InitialStructured()
	if got := firstText(structured); got != "light" {
		t.Fatalf("expected default value 'light', got %q", got)
	}
}
