package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestBootHasNoPatch(t *testing.T) {
	ls := NewLiveSession("boot", 2, dummyComponent, nil)
	t.Cleanup(func() { _ = ls.Close() })
	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	raw, err := json.Marshal(res.Boot)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["patch"]; ok {
		t.Fatalf("boot carries a patch: %s", raw)
	}
	if string(fields["ver"]) != "2" {
		t.Fatalf("boot version = %s", fields["ver"])
	}
}

func TestHandlerEventByIDRunsTheHandler(t *testing.T) {
	probe := &clickProbe{}
	ls := startLive(t, probe.component)
	var id uint32
	onSession(t, ls, func() { id = probe.refID() })
	if id == 0 {
		t.Fatal("the button has no id")
	}
	if err := deliver(ls, clickEvt(id, 7)); err != nil {
		t.Fatal(err)
	}
	barrier(t, ls)
	var log []int
	onSession(t, ls, func() { log = append(log, probe.log...) })
	if len(log) != 1 || log[0] != 7 {
		t.Fatalf("log = %v", log)
	}
}

func TestStaleHandlerIDDropped(t *testing.T) {
	probe := &clickProbe{}
	ls := startLive(t, probe.component)
	var phases []string
	var mu sync.Mutex
	ls.Session().Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		mu.Lock()
		phases = append(phases, d.Phase)
		mu.Unlock()
	})
	if err := deliver(ls, clickEvt(987654, 1)); err != nil {
		t.Fatal(err)
	}
	barrier(t, ls)
	mu.Lock()
	got := append([]string(nil), phases...)
	mu.Unlock()
	if len(got) != 1 || got[0] != "handler:stale" {
		t.Fatalf("diagnostics = %v", got)
	}
	if ls.State() != StateLive {
		t.Fatalf("state = %v", ls.State())
	}
	var id uint32
	onSession(t, ls, func() { id = probe.refID() })
	if err := deliver(ls, clickEvt(id, 3)); err != nil {
		t.Fatal(err)
	}
	barrier(t, ls)
	onSession(t, ls, func() {
		if len(probe.log) != 1 {
			t.Errorf("valid click after a stale one was lost: %v", probe.log)
		}
	})
}

func TestConnectedTogglesOnAttachDetach(t *testing.T) {
	comp := func(ctx *runtime.Ctx) work.Node {
		c := runtime.SessionOf(ctx).Connected()
		label := runtime.UseComputed(ctx, func() string {
			if c.Get() {
				return "online"
			}
			return "offline"
		})
		return work.BuildElement("p", work.TextOf(label))
	}
	ls := NewLiveSession("conn", 2, comp, nil)
	t.Cleanup(func() { _ = ls.Close() })
	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, "offline") {
		t.Fatalf("ssr = %s", res.HTML)
	}
	tr := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr, nil, res.Boot.Seq, false); err != nil {
		t.Fatal(err)
	}
	if got := tr.frames; len(got) != 1 || !strings.Contains(string(got[0]), `"online"`) {
		t.Fatalf("attach frames = %s", tr.frames)
	}
	ls.Detach(tr)
	barrier(t, ls)
	tr2 := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr2, nil, 1, false); err != nil {
		t.Fatal(err)
	}
	var texts []string
	for _, f := range tr2.frames {
		texts = append(texts, string(f))
	}
	joined := strings.Join(texts, "|")
	if !strings.Contains(joined, `"offline"`) || !strings.Contains(joined, `"online"`) {
		t.Fatalf("replayed frames = %s", joined)
	}
}

func TestNondeterministicBodyReloads(t *testing.T) {
	var runs atomic.Int32
	comp := func(ctx *runtime.Ctx) work.Node {
		if runs.Add(1) == 1 {
			return work.BuildElement("div", work.NewText("static"))
		}
		return work.BuildElement("div", work.TextOf(runtime.UseComputed(ctx, func() string { return "dynamic" })))
	}
	ls := NewLiveSession("nondet", 2, comp, &Config{DevMode: false})
	t.Cleanup(func() { _ = ls.Close() })
	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	var diags []string
	var mu sync.Mutex
	ls.Session().Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		mu.Lock()
		diags = append(diags, d.Phase)
		mu.Unlock()
	})
	tr := newRecordingTransport()
	_, err := ls.Attach(context.Background(), tr, nil, 0, true)
	if !errors.Is(err, runtime.ErrNondeterministic) {
		t.Fatalf("attach err = %v", err)
	}
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonResetNondeterministic {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
	mu.Lock()
	defer mu.Unlock()
	found := false
	for _, p := range diags {
		found = found || p == "reset:nondeterministic"
	}
	if !found {
		t.Fatalf("diagnostics = %v", diags)
	}
	for _, f := range tr.frames {
		if strings.Contains(string(f), `"reset"`) {
			t.Fatal("a reset frame was sent for a nondeterministic body")
		}
	}
}

func TestFeedResetReloadsFromLoader(t *testing.T) {
	var mu sync.Mutex
	rows := []string{"a", "b"}
	var feed *runtime.Feed[string, string]
	comp := func(ctx *runtime.Ctx) work.Node {
		feed = runtime.NewFeed(func(s string) string { return s }, func() []string {
			mu.Lock()
			defer mu.Unlock()
			return append([]string(nil), rows...)
		})
		return work.BuildElement("ul", runtime.For(feed, runtime.PropsComponent(func(_ *runtime.Ctx, item runtime.ReadSignal[string]) work.Node {
			return work.BuildElement("li", work.NewText(item.Peek()))
		})))
	}
	ls := startLive(t, comp)
	onSession(t, ls, func() {})
	mu.Lock()
	rows = append(rows, "c")
	mu.Unlock()
	tr := newRecordingTransport()
	ar, err := ls.Attach(context.Background(), tr, nil, 0, true)
	if err != nil || ar.Mode != protocol.ModeReset {
		t.Fatalf("attach: %+v %v", ar, err)
	}
	var reset protocol.ResetPayload
	for _, m := range tr.messages() {
		if m.Event == string(protocol.FrameResetAction) {
			raw, _ := json.Marshal(m.Data)
			_ = json.Unmarshal(raw, &reset)
		}
	}
	for _, want := range []string{">a<", ">b<", ">c<"} {
		if !strings.Contains(reset.HTML, want) {
			t.Fatalf("reset html lacks %s: %s", want, reset.HTML)
		}
	}
	sent := 0
	for _, m := range tr.messages() {
		if m.Event == string(protocol.FramePatchAction) {
			sent++
		}
	}
	if sent != 0 {
		t.Fatalf("%d patch frames were sent alongside the reset", sent)
	}
	_ = time.Second
}

func TestHeaderSignalUpdatesThroughAnEditFrameOnUpgrade(t *testing.T) {
	comp := func(ctx *runtime.Ctx) work.Node {
		return work.BuildElement("p", work.TextOf(headers.UseRequest(ctx).Header("X-Tenant")))
	}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Tenant", "acme")
	ls := NewLiveSession("hdr", 2, comp, &Config{RetainedHeaders: []string{"X-Tenant"}}, WithRequest(req))
	t.Cleanup(func() { _ = ls.Close() })
	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(res.HTML, ">acme<") {
		t.Fatalf("ssr = %s", res.HTML)
	}
	tr := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr, http.Header{"X-Tenant": {"globex"}}, res.Boot.Seq, false); err != nil {
		t.Fatal(err)
	}
	if len(tr.frames) != 1 || !strings.Contains(string(tr.frames[0]), `"globex"`) {
		t.Fatalf("attach frames = %s", tr.frames)
	}
	ls.Detach(tr)
	barrier(t, ls)
	tr2 := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr2, http.Header{"X-Tenant": {"globex"}}, 1, false); err != nil {
		t.Fatal(err)
	}
	for _, f := range tr2.frames {
		if strings.Contains(string(f), `"patch"`) {
			t.Fatalf("an unchanged header on reconnect produced a patch: %s", f)
		}
	}
	tr3 := newRecordingTransport()
	ls.Detach(tr2)
	barrier(t, ls)
	if _, err := ls.Attach(context.Background(), tr3, http.Header{"X-Tenant": {"initech"}}, 1, false); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(strings.Join(frameStrings(tr3), "|"), `"initech"`) {
		t.Fatalf("reconnect frames = %v", frameStrings(tr3))
	}
}

func frameStrings(tr *recordingTransport) []string {
	var out []string
	for _, f := range tr.frames {
		out = append(out, string(f))
	}
	return out
}
