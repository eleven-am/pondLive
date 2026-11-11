package runtime_test

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func stringSpec(prop string) runtime.RefSpec[string] {
	return runtime.RefSpec[string]{
		Prop: prop,
		Decode: func(v any) (string, bool) {
			s, ok := v.(string)
			return s, ok
		},
		Seed: func(attrs map[string]string) (string, bool) {
			v, ok := attrs["value"]
			return v, ok
		},
	}
}

func sendValue(app *runtimetest.App, id uint32, prop string, v any) {
	app.On(func() {
		app.S.Bus.Publish(protocol.DOMHandler, string(protocol.DOMValueAction), protocol.DOMValuePayload{ID: id, Prop: prop, Value: v})
	})
	app.Sync()
}

type liveProbe struct {
	ref *runtime.Ref
	sig runtime.ReadSignal[string]
}

func liveApp(t *testing.T, read bool, opts ...runtime.ObserveOpt) (*runtimetest.App, *liveProbe, *domLog) {
	return liveAppFor(t, "currentTime", read, opts...)
}

func liveAppFor(t *testing.T, prop string, read bool, opts ...runtime.ObserveOpt) (*runtimetest.App, *liveProbe, *domLog) {
	p := &liveProbe{}
	var log *domLog
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		p.ref = useRef(ctx, "MediaRef", onlyTag("input"))
		p.sig = runtime.RefSignal(p.ref, stringSpec(prop), opts...)
		kids := []work.Item{runtime.AttachItem(p.ref), work.Bind("start")}
		if read {
			return el("div", el("input", kids...), el("p", bound(p.sig)))
		}
		return el("div", el("input", kids...))
	})
	log = watchDOM(app)
	return app, p, log
}

func TestLiveSignalStartsFromTheHTMLValueAndSubscribesOnFirstRead(t *testing.T) {
	app, p, log := liveApp(t, false)
	if strings.Contains(log.names(), "observe") {
		t.Fatalf("subscribed before any read: %s", log.names())
	}
	app.On(func() {
		if got := p.sig.Peek(); got != "start" {
			t.Errorf("start value = %q, want the HTML value", got)
		}
	})
	got, ok := log.last().(protocol.DOMObservePayload)
	if !ok || got.ID != p.ref.ID() || got.Prop != "currentTime" || got.Ms != 0 {
		t.Fatalf("observe payload = %#v (events %s)", log.last(), log.names())
	}
	app.On(func() { _ = p.sig.Get(); _ = p.sig.Peek() })
	if n := strings.Count(log.names(), "observe"); n != 1 {
		t.Fatalf("observe sent %d times: %s", n, log.names())
	}
}

func TestLiveSignalSeededBeforeAnyReadAndSSRShowsTheStartValue(t *testing.T) {
	app, _, _ := liveApp(t, true)
	if !strings.Contains(app.SSR, ">start<") {
		t.Fatalf("SSR must render the HTML start value: %s", app.SSR)
	}
}

func TestLiveSignalFollowsBrowserValuesWithoutARoundTrip(t *testing.T) {
	app, p, log := liveApp(t, true)
	before := log.names()
	sendValue(app, p.ref.ID(), "currentTime", "typed")
	if !strings.Contains(app.Normalized(), "<p>typed</p>") {
		t.Fatalf("signal did not follow the browser: %s", app.Normalized())
	}
	sendValue(app, p.ref.ID(), "currentTime", 42.0)
	if !strings.Contains(app.Normalized(), "<p>typed</p>") {
		t.Fatalf("an undecodable value must be ignored: %s", app.Normalized())
	}
	if log.names() != before+",value,value" && !strings.HasPrefix(log.names(), before) {
		t.Fatalf("server must not send anything back: %s", log.names())
	}
	for _, e := range strings.Split(log.names(), ",") {
		if e == "query" || e == "async" {
			t.Fatalf("a live signal used a round trip: %s", log.names())
		}
	}
}

func TestLiveSignalUnobservesWhenTheElementUnmounts(t *testing.T) {
	var flag *runtime.Signal[bool]
	var ref *runtime.Ref
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		ref = useRef(ctx, "MediaRef", nil)
		sig := runtime.RefSignal(ref, stringSpec("currentTime"))
		return el("div", runtime.Show(flag, func() Node { return el("input", runtime.AttachItem(ref)) }), el("p", bound(sig)))
	})
	log := watchDOM(app)
	id := ref.ID()
	app.Do(func() { flag.Set(false) })
	got, ok := log.last().(protocol.DOMUnobservePayload)
	if !ok || got.ID != id || got.Prop != "currentTime" {
		t.Fatalf("last = %#v (events %s)", log.last(), log.names())
	}
	app.Do(func() { flag.Set(true) })
	if ob, ok := log.last().(protocol.DOMObservePayload); !ok || ob.ID != ref.ID() || ob.ID == id {
		t.Fatalf("a remounted element must be observed under its new id: %#v", log.last())
	}
}

func TestLiveSignalThrottleOptionReachesTheWire(t *testing.T) {
	cases := []struct {
		opt  runtime.ObserveOpt
		want int
	}{{work.Throttle(50 * time.Millisecond), 50}, {work.Throttle(0), -1}}
	for _, tc := range cases {
		app, p, log := liveApp(t, false, tc.opt)
		app.On(func() { _ = p.sig.Peek() })
		if got := log.last().(protocol.DOMObservePayload); got.Ms != tc.want {
			t.Fatalf("ms = %d, want %d", got.Ms, tc.want)
		}
	}
}

func TestLiveSignalsAreResyncedAfterAReset(t *testing.T) {
	app, p, log := liveApp(t, true)
	before := strings.Count(log.names(), "observe")
	app.On(func() { app.S.ResyncRefs() })
	if after := strings.Count(log.names(), "observe"); after != before+1 {
		t.Fatalf("observe count %d -> %d", before, after)
	}
	if got := log.last().(protocol.DOMObservePayload); got.ID != p.ref.ID() {
		t.Fatalf("resync payload = %#v", got)
	}
}

func TestLiveSignalReadBeforeConnectSubscribesOnceTheSessionGoesLive(t *testing.T) {
	var ref *runtime.Ref
	d := rawStartDeferred(t, false, func(ctx *Ctx) Node {
		ref = useRef(ctx, "MediaRef", nil)
		sig := runtime.RefSignal(ref, stringSpec("currentTime"))
		return el("div", el("input", runtime.AttachItem(ref)), el("p", bound(sig)))
	})
	var events []string
	d.s.Bus.Subscribe(protocol.DOMHandler, func(event string, _ interface{}) { events = append(events, event) })
	if len(events) != 0 {
		t.Fatalf("subscribed before connect: %v", events)
	}
	d.goLive()
	if strings.Join(events, ",") != "observe" {
		t.Fatalf("events after go-live = %v", events)
	}
}

func TestLiveSignalCreatedAfterMountIsStillSeededFromTheHTML(t *testing.T) {
	var ref *runtime.Ref
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		ref = useRef(ctx, "MediaRef", onlyTag("input"))
		return el("input", runtime.AttachItem(ref), work.Bind("start"))
	})
	app.On(func() {
		if got := runtime.RefSignal(ref, stringSpec("currentTime")).Peek(); got != "start" {
			t.Errorf("late signal = %q, want the HTML value", got)
		}
	})
}

func TestObserveRateIsAlwaysExplicitOnTheWire(t *testing.T) {
	cases := []struct {
		name string
		prop string
		opts []runtime.ObserveOpt
		want int
	}{
		{"a per-frame property defaults to a frame", "currentTime", nil, 0},
		{"an immediate property defaults to every event", "volume", nil, -1},
		{"an explicit frame on an immediate property is a frame", "volume", []runtime.ObserveOpt{work.Frame()}, 0},
		{"an explicit immediate on a per-frame property is every event", "currentTime", []runtime.ObserveOpt{work.Immediate()}, -1},
		{"a throttle is its milliseconds", "volume", []runtime.ObserveOpt{work.Throttle(40 * time.Millisecond)}, 40},
	}
	for _, tc := range cases {
		app, p, log := liveAppFor(t, tc.prop, false, tc.opts...)
		app.On(func() { _ = p.sig.Peek() })
		got := log.last().(protocol.DOMObservePayload)
		if got.Ms != tc.want {
			t.Errorf("%s: ms = %d, want %d", tc.name, got.Ms, tc.want)
		}
		raw, err := json.Marshal(got)
		if err != nil || !strings.Contains(string(raw), `"ms":`+strconv.Itoa(tc.want)) {
			t.Errorf("%s: wire = %s (%v)", tc.name, raw, err)
		}
	}
}
