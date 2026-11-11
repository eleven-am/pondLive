package pkg

import (
	"reflect"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestEveryTypedHandlerBindsItsEventOnTheElement(t *testing.T) {
	cases := []struct {
		event string
		item  Item
	}{
		{"click", OnClick(func(ClickEvent) {})},
		{"dblclick", OnDoubleClick(func(ClickEvent) {})},
		{"contextmenu", OnContextMenu(func(ClickEvent) {})},
		{"auxclick", OnAuxClick(func(ClickEvent) {})},
		{"mousedown", OnMouseDown(func(MouseEvent) {})},
		{"mouseup", OnMouseUp(func(MouseEvent) {})},
		{"mousemove", OnMouseMove(func(MouseEvent) {})},
		{"mouseenter", OnMouseEnter(func(MouseEvent) {})},
		{"mouseleave", OnMouseLeave(func(MouseEvent) {})},
		{"mouseover", OnMouseOver(func(MouseEvent) {})},
		{"mouseout", OnMouseOut(func(MouseEvent) {})},
		{"focus", OnFocus(func(FocusEvent) {})},
		{"blur", OnBlur(func(FocusEvent) {})},
		{"focusin", OnFocusIn(func(FocusEvent) {})},
		{"focusout", OnFocusOut(func(FocusEvent) {})},
		{"keydown", OnKeyDown(func(KeyboardEvent) {})},
		{"keyup", OnKeyUp(func(KeyboardEvent) {})},
		{"pointerdown", OnPointerDown(func(PointerEvent) {})},
		{"pointerup", OnPointerUp(func(PointerEvent) {})},
		{"pointermove", OnPointerMove(func(PointerEvent) {})},
		{"pointerenter", OnPointerEnter(func(PointerEvent) {})},
		{"pointerleave", OnPointerLeave(func(PointerEvent) {})},
		{"pointercancel", OnPointerCancel(func(PointerEvent) {})},
		{"pointerover", OnPointerOver(func(PointerEvent) {})},
		{"pointerout", OnPointerOut(func(PointerEvent) {})},
		{"lostpointercapture", OnLostPointerCapture(func(PointerEvent) {})},
		{"touchstart", OnTouchStart(func(TouchEvent) {})},
		{"touchend", OnTouchEnd(func(TouchEvent) {})},
		{"touchmove", OnTouchMove(func(TouchEvent) {})},
		{"touchcancel", OnTouchCancel(func(TouchEvent) {})},
		{"drag", OnDrag(func(DragEvent) {})},
		{"dragstart", OnDragStart(func(DragEvent) {})},
		{"dragend", OnDragEnd(func(DragEvent) {})},
		{"dragenter", OnDragEnter(func(DragEvent) {})},
		{"dragleave", OnDragLeave(func(DragEvent) {})},
		{"dragover", OnDragOver(func(DragEvent) {})},
		{"drop", OnDrop(func(DragEvent) {})},
		{"wheel", OnWheel(func(WheelEvent) {})},
		{"scroll", OnScroll(func(ScrollEvent) {})},
		{"scrollend", OnScrollEnd(func(ScrollEvent) {})},
		{"animationstart", OnAnimationStart(func(AnimationEvent) {})},
		{"animationend", OnAnimationEnd(func(AnimationEvent) {})},
		{"animationiteration", OnAnimationIteration(func(AnimationEvent) {})},
		{"animationcancel", OnAnimationCancel(func(AnimationEvent) {})},
		{"transitionstart", OnTransitionStart(func(TransitionEvent) {})},
		{"transitionend", OnTransitionEnd(func(TransitionEvent) {})},
		{"transitionrun", OnTransitionRun(func(TransitionEvent) {})},
		{"transitioncancel", OnTransitionCancel(func(TransitionEvent) {})},
		{"copy", OnCopy(func(ClipboardEvent) {})},
		{"cut", OnCut(func(ClipboardEvent) {})},
		{"paste", OnPaste(func(ClipboardEvent) {})},
		{"compositionstart", OnCompositionStart(func(CompositionEvent) {})},
		{"compositionupdate", OnCompositionUpdate(func(CompositionEvent) {})},
		{"compositionend", OnCompositionEnd(func(CompositionEvent) {})},
		{"submit", OnSubmit(func(FormEvent) {})},
		{"reset", OnReset(func(FormEvent) {})},
		{"play", OnPlay(func(MediaEvent) {})},
		{"pause", OnPause(func(MediaEvent) {})},
		{"ended", OnEnded(func(MediaEvent) {})},
		{"timeupdate", OnTimeUpdate(func(MediaEvent) {})},
		{"volumechange", OnVolumeChange(func(MediaEvent) {})},
		{"seeking", OnSeeking(func(MediaEvent) {})},
		{"seeked", OnSeeked(func(MediaEvent) {})},
		{"loadedmetadata", OnLoadedMetadata(func(MediaEvent) {})},
		{"loadeddata", OnLoadedData(func(MediaEvent) {})},
		{"canplay", OnCanPlay(func(MediaEvent) {})},
		{"canplaythrough", OnCanPlayThrough(func(MediaEvent) {})},
		{"waiting", OnWaiting(func(MediaEvent) {})},
		{"playing", OnPlaying(func(MediaEvent) {})},
		{"stalled", OnStalled(func(MediaEvent) {})},
		{"ratechange", OnRateChange(func(MediaEvent) {})},
		{"durationchange", OnDurationChange(func(MediaEvent) {})},
		{"loadstart", OnLoadStart(func(MediaEvent) {})},
		{"suspend", OnSuspend(func(MediaEvent) {})},
		{"abort", OnAbort(func(MediaEvent) {})},
		{"emptied", OnEmptied(func(MediaEvent) {})},
		{"progress", OnProgress(func(ProgressEvent) {})},
		{"error", OnError(func(ErrorEvent) {})},
		{"load", OnLoad(func(LoadEvent) {})},
		{"toggle", OnToggle(func(ToggleEvent) {})},
		{"beforetoggle", OnBeforeToggle(func(ToggleEvent) {})},
		{"close", OnClose(func(DialogEvent) {})},
		{"cancel", OnCancel(func(DialogEvent) {})},
		{"select", OnSelect(func(SelectionEvent) {})},
		{"selectionchange", OnSelectionChange(func(SelectionEvent) {})},
		{"beforeinput", OnBeforeInput(func(BeforeInputEvent) {})},
		{"fullscreenchange", OnFullscreenChange(func(FullscreenChangeEvent) {})},
		{metadata.EditEvent, OnChange(func(ChangeEvent) {})},
		{"invalid", OnInvalid(func(Event) {})},
	}
	seen := map[string]bool{}
	for _, tc := range cases {
		el := work.BuildElement("div", tc.item)
		if len(el.Events) != 1 || el.Events[0].Event != tc.event {
			t.Errorf("%s: events = %+v", tc.event, el.Events)
		}
		if seen[tc.event] {
			t.Errorf("%s is bound by two helpers", tc.event)
		}
		seen[tc.event] = true
	}
	if len(cases) != 89 {
		t.Fatalf("the table covers %d helpers", len(cases))
	}
}

func optionsOf(t *testing.T, item Item) metadata.EventOptions {
	t.Helper()
	el := work.BuildElement("div", item)
	if len(el.Events) != 1 {
		t.Fatalf("events = %+v", el.Events)
	}
	return el.Events[0].Options
}

func fieldNames(t *testing.T, names []string) map[string]bool {
	t.Helper()
	if err := protocol.ValidateFields(names...); err != nil {
		t.Fatal(err)
	}
	out := map[string]bool{}
	for _, n := range names {
		out[n] = true
	}
	return out
}

func TestTypedHandlersDelegateToTheGenericOnWith(t *testing.T) {
	el := work.BuildElement("form", OnSubmit(func(FormEvent) {}), OnClick(func(ClickEvent) {}, EventOptions{Stop: true}))
	var submit, click work.EventBinding
	for _, b := range el.Events {
		switch b.Event {
		case "submit":
			submit = b
		case "click":
			click = b
		}
	}
	if !submit.Options.Prevent {
		t.Fatalf("submit options = %+v", submit.Options)
	}
	if !click.Options.Stop || len(click.Options.Fields) == 0 {
		t.Fatalf("click options = %+v", click.Options)
	}
	if got := work.BuildElement("div", OnClick(nil), OnInvalid(nil)); len(got.Events) != 0 {
		t.Fatalf("nil handlers must be skipped: %+v", got.Events)
	}
}

func TestLeanDefaultsAndOptInFields(t *testing.T) {
	lean := fieldNames(t, optionsOf(t, OnClick(func(ClickEvent) {})).Fields)
	for _, want := range []string{"clientX", "clientY", "button", "ctrlKey"} {
		if !lean[want] {
			t.Errorf("click default is missing %s", want)
		}
	}
	for _, not := range []string{"pageX", "screenX", "buttons"} {
		if lean[not] {
			t.Errorf("click default carries opt-in field %s", not)
		}
	}
	more := fieldNames(t, optionsOf(t, OnClick(func(ClickEvent) {}, Fields("pageX"))).Fields)
	if !more["pageX"] || !more["clientX"] {
		t.Fatalf("Fields must add to the defaults: %v", more)
	}
	all := fieldNames(t, optionsOf(t, OnClick(func(ClickEvent) {}, AllFields())).Fields)
	if !all["screenY"] || !all["buttons"] {
		t.Fatalf("AllFields = %v", all)
	}
	keyboard := fieldNames(t, optionsOf(t, OnKeyDown(func(KeyboardEvent) {})).Fields)
	if keyboard["value"] || keyboard["location"] {
		t.Fatalf("a key press must not carry the input value by default: %v", keyboard)
	}
}

func TestFieldsRejectsAnUnknownName(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("an unknown field name must panic")
		}
	}()
	Fields("ownerDocument.cookie")
}

func TestEveryShapeDecodesEveryFieldItDeclares(t *testing.T) {
	payload := map[string]any{}
	for _, f := range protocol.EventFields {
		payload[f] = 1.0
	}
	payload["key"], payload["code"], payload["value"], payload["pointerType"] = "k", "c", "v", "mouse"
	payload["inputType"], payload["data"], payload["animationName"], payload["pseudoElement"] = "i", "d", "a", "p"
	payload["propertyName"], payload["returnValue"], payload["text"] = "pn", "rv", "tx"
	payload["selected"] = []any{"a", "b"}
	payload["touches"] = []any{[]any{1.0, 2.0, 3.0}}
	payload["files"] = []any{map[string]any{"n": "f.txt", "s": 4.0, "t": "text/plain", "m": 5.0}}
	for _, name := range []string{"altKey", "ctrlKey", "shiftKey", "metaKey", "repeat", "isComposing", "isPrimary", "paused", "ended", "muted", "open", "lengthComputable", "checked", "fullscreen"} {
		payload[name] = true
	}
	check := func(t *testing.T, st any, dec func(work.Event)) {
		dec(work.Event{Payload: payload})
		v := reflect.ValueOf(st).Elem()
		for i := 0; i < v.NumField(); i++ {
			if _, ok := v.Type().Field(i).Tag.Lookup("ev"); !ok {
				continue
			}
			f := v.Field(i)
			if f.IsZero() {
				t.Errorf("%s.%s stayed zero", v.Type().Name(), v.Type().Field(i).Name)
			}
			switch f.Kind() {
			case reflect.Float64:
				if f.Float() != 1 {
					t.Errorf("%s.%s = %v, want 1", v.Type().Name(), v.Type().Field(i).Name, f.Float())
				}
			case reflect.Int:
				if f.Int() != 1 {
					t.Errorf("%s.%s = %v, want 1", v.Type().Name(), v.Type().Field(i).Name, f.Int())
				}
			}
		}
	}
	run := func(name string, item Item, st any) {
		t.Run(name, func(t *testing.T) {
			el := work.BuildElement("div", item)
			check(t, st, el.Events[0].Fns[0])
		})
	}
	var click ClickEvent
	run("click", OnClick(func(e ClickEvent) { click = e }, AllFields()), &click)
	var mouse MouseEvent
	run("mouse", OnMouseMove(func(e MouseEvent) { mouse = e }, AllFields()), &mouse)
	var key KeyboardEvent
	run("keyboard", OnKeyDown(func(e KeyboardEvent) { key = e }, AllFields()), &key)
	var pointer PointerEvent
	run("pointer", OnPointerDown(func(e PointerEvent) { pointer = e }, AllFields()), &pointer)
	var touch TouchEvent
	run("touch", OnTouchStart(func(e TouchEvent) { touch = e }, AllFields()), &touch)
	var drag DragEvent
	run("drag", OnDrop(func(e DragEvent) { drag = e }, AllFields()), &drag)
	var wheel WheelEvent
	run("wheel", OnWheel(func(e WheelEvent) { wheel = e }, AllFields()), &wheel)
	var scroll ScrollEvent
	run("scroll", OnScroll(func(e ScrollEvent) { scroll = e }, AllFields()), &scroll)
	var change ChangeEvent
	run("change", OnChange(func(e ChangeEvent) { change = e }, AllFields()), &change)
	var before BeforeInputEvent
	run("beforeinput", OnBeforeInput(func(e BeforeInputEvent) { before = e }, AllFields()), &before)
	var media MediaEvent
	run("media", OnTimeUpdate(func(e MediaEvent) { media = e }, AllFields()), &media)
	var anim AnimationEvent
	run("animation", OnAnimationEnd(func(e AnimationEvent) { anim = e }, AllFields()), &anim)
	var trans TransitionEvent
	run("transition", OnTransitionEnd(func(e TransitionEvent) { trans = e }, AllFields()), &trans)
	var clip ClipboardEvent
	run("clipboard", OnPaste(func(e ClipboardEvent) { clip = e }, AllFields()), &clip)
	var comp CompositionEvent
	run("composition", OnCompositionEnd(func(e CompositionEvent) { comp = e }, AllFields()), &comp)
	var sel SelectionEvent
	run("selection", OnSelect(func(e SelectionEvent) { sel = e }, AllFields()), &sel)
	var toggle ToggleEvent
	run("toggle", OnToggle(func(e ToggleEvent) { toggle = e }, AllFields()), &toggle)
	var dialog DialogEvent
	run("dialog", OnClose(func(e DialogEvent) { dialog = e }, AllFields()), &dialog)
	var load LoadEvent
	run("load", OnLoad(func(e LoadEvent) { load = e }, AllFields()), &load)
	var progress ProgressEvent
	run("progress", OnProgress(func(e ProgressEvent) { progress = e }, AllFields()), &progress)
	var fs FullscreenChangeEvent
	run("fullscreen", OnFullscreenChange(func(e FullscreenChangeEvent) { fs = e }, AllFields()), &fs)
	if change.Values[1] != "b" || change.Files[0].Name != "f.txt" || change.Files[0].Size != 4 || touch.Touches[0].ClientY != 3 {
		t.Fatalf("composite fields decoded wrongly: %+v %+v %+v %+v", change, touch, drag, click)
	}
}

func TestKeyFilterIsSentToTheClientAndEnforcedForEveryHandler(t *testing.T) {
	var got []string
	item := OnKeyDown(func(e KeyboardEvent) { got = append(got, e.Key) }, Key("Enter", "Escape"))
	o := optionsOf(t, item)
	if len(o.Keys) != 2 || !fieldNames(t, o.Fields)["key"] {
		t.Fatalf("options = %+v", o)
	}
	el := work.BuildElement("input", item)
	for _, k := range []string{"a", "Enter", "Tab", "Escape"} {
		el.Events[0].Fns[0](work.Event{Name: "keydown", Payload: map[string]any{"key": k}})
	}
	if len(got) != 2 || got[0] != "Enter" || got[1] != "Escape" {
		t.Fatalf("handler saw %v", got)
	}
}

func TestKeyOnANonKeyEventIsRejected(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Fatal("Key on click must panic")
		}
	}()
	OnClick(func(ClickEvent) {}, Key("Enter"))
}

func TestRateOptionsReachTheEventOptions(t *testing.T) {
	cases := []struct {
		name string
		item Item
		db   int
		th   int
	}{
		{"throttle", OnMouseMove(func(MouseEvent) {}, Throttle(100*time.Millisecond)), 0, 100},
		{"debounce", OnClick(func(ClickEvent) {}, Debounce(250*time.Millisecond)), 250, 0},
		{"frame", OnClick(func(ClickEvent) {}, Frame()), 0, metadata.RateFrame},
		{"immediate", OnMouseMove(func(MouseEvent) {}, Immediate()), 0, metadata.RateOff},
		{"throttle zero is immediate", OnMouseMove(func(MouseEvent) {}, Throttle(0)), 0, metadata.RateOff},
	}
	for _, c := range cases {
		o := optionsOf(t, c.item)
		if o.Debounce != c.db || o.Throttle != c.th {
			t.Errorf("%s: debounce %d throttle %d, want %d %d", c.name, o.Debounce, o.Throttle, c.db, c.th)
		}
	}
}

func TestOnChangeUsesTheEditEvent(t *testing.T) {
	o := optionsOf(t, OnChange(func(ChangeEvent) {}))
	names := fieldNames(t, o.Fields)
	if !names["value"] || !names["checked"] || !names["selected"] || !names["files"] {
		t.Fatalf("change fields = %v", names)
	}
	el := work.BuildElement("input", OnChange(func(ChangeEvent) {}))
	if el.Events[0].Event != metadata.EditEvent {
		t.Fatalf("event = %q", el.Events[0].Event)
	}
}

func TestTypedHandlerBuildsTheTypedEvent(t *testing.T) {
	var got KeyboardEvent
	el := work.BuildElement("input", OnKeyDown(func(e KeyboardEvent) { got = e }))
	el.Events[0].Fns[0](work.Event{Name: "keydown", Payload: map[string]any{"key": "Enter"}})
	if got.Key != "Enter" || got.Name != "keydown" {
		t.Fatalf("event = %+v", got)
	}
}
