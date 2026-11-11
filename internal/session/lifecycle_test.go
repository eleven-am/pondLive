package session

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type fakeClock struct {
	mu         sync.Mutex
	now        time.Duration
	timers     []*fakeTimer
	ignoreStop bool
}

type fakeTimer struct {
	clock   *fakeClock
	at      time.Duration
	f       func()
	stopped bool
	fired   bool
}

func (c *fakeClock) AfterFunc(d time.Duration, f func()) Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{clock: c, at: c.now + d, f: f}
	c.timers = append(c.timers, t)
	return t
}

func (t *fakeTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	if t.clock.ignoreStop {
		return false
	}
	was := !t.stopped && !t.fired
	t.stopped = true
	return was
}

func (c *fakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now += d
	var due []*fakeTimer
	for _, t := range c.timers {
		if !t.fired && t.at <= c.now && (c.ignoreStop || !t.stopped) {
			t.fired = true
			due = append(due, t)
		}
	}
	c.mu.Unlock()
	for _, t := range due {
		t.f()
	}
}

func (c *fakeClock) Armed() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, t := range c.timers {
		if !t.fired && !t.stopped {
			n++
		}
	}
	return n
}

const testTTL = 2 * time.Minute

func newClocked(t *testing.T, comp Component, clock *fakeClock, mutate func(*Config)) *LiveSession {
	t.Helper()
	cfg := &Config{PendingTTL: testTTL, DisconnectTTL: testTTL, Clock: clock}
	if mutate != nil {
		mutate(cfg)
	}
	ls := NewLiveSession("lifecycle", 1, comp, cfg, WithRequest(httptest.NewRequest(http.MethodGet, "/", nil)))
	t.Cleanup(func() { _ = ls.Close() })
	barrier(t, ls)
	return ls
}

func barrier(t *testing.T, ls *LiveSession) {
	t.Helper()
	if err := ls.Flush(); err != nil && !errors.Is(err, runtime.ErrSessionClosed) {
		t.Fatalf("barrier: %v", err)
	}
}

func waitClosed(t *testing.T, ls *LiveSession) {
	t.Helper()
	select {
	case <-ls.Done():
	case <-time.After(5 * time.Second):
		t.Fatalf("session still %v", ls.State())
	}
}

func holdActor(t *testing.T, ls *LiveSession) (release func()) {
	t.Helper()
	held := make(chan struct{})
	hold := make(chan struct{})
	if err := ls.Session().Post(func() {
		close(held)
		<-hold
	}); err != nil {
		t.Fatal(err)
	}
	<-held
	var once sync.Once
	release = func() { once.Do(func() { close(hold) }) }
	t.Cleanup(release)
	return release
}

func TestPendingSessionExpires(t *testing.T) {
	clock := &fakeClock{}
	ls := newClocked(t, dummyComponent, clock, nil)
	if ls.State() != StatePending {
		t.Fatalf("state = %v", ls.State())
	}

	clock.Advance(testTTL - time.Second)
	barrier(t, ls)
	if ls.State() != StatePending {
		t.Fatalf("expired early: %v", ls.State())
	}

	clock.Advance(time.Second)
	waitClosed(t, ls)
	if ls.State() != StateClosed || ls.CloseReason() != ReasonPendingExpired {
		t.Fatalf("state=%v reason=%v", ls.State(), ls.CloseReason())
	}
}

func TestDisconnectedSessionExpires(t *testing.T) {
	clock := &fakeClock{}
	ls := newClocked(t, dummyComponent, clock, nil)
	tr := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	if ls.State() != StateLive {
		t.Fatalf("state = %v", ls.State())
	}
	clock.Advance(10 * testTTL)
	barrier(t, ls)
	if ls.State() != StateLive {
		t.Fatalf("a live session expired through the pending timer: %v", ls.State())
	}

	ls.Detach(tr)
	barrier(t, ls)
	if ls.State() != StateDisconnected {
		t.Fatalf("state = %v", ls.State())
	}
	clock.Advance(testTTL - time.Second)
	barrier(t, ls)
	if ls.State() != StateDisconnected {
		t.Fatalf("expired early: %v", ls.State())
	}
	clock.Advance(time.Second)
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonDisconnectExpired {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
}

func TestReattachWithinGraceKeepsSession(t *testing.T) {
	for _, ignoreStop := range []bool{false, true} {
		name := "stopped timer"
		if ignoreStop {
			name = "timer that fires despite Stop"
		}
		t.Run(name, func(t *testing.T) {
			clock := &fakeClock{ignoreStop: ignoreStop}
			ls := newClocked(t, dummyComponent, clock, nil)
			first := newRecordingTransport()
			if _, err := ls.Attach(context.Background(), first, nil, 0, false); err != nil {
				t.Fatal(err)
			}
			ls.Detach(first)
			barrier(t, ls)

			clock.Advance(testTTL - time.Second)
			second := newRecordingTransport()
			ar, err := ls.Attach(context.Background(), second, nil, 0, false)
			if err != nil {
				t.Fatal(err)
			}
			if ar.Gen != 2 || ls.State() != StateLive {
				t.Fatalf("gen=%d state=%v", ar.Gen, ls.State())
			}

			clock.Advance(testTTL)
			barrier(t, ls)
			if ls.State() != StateLive {
				t.Fatalf("a stale disconnect timer closed the session: %v", ls.State())
			}
			if second.Closed() {
				t.Fatal("the reattached transport was terminated")
			}
		})
	}
}

func TestStaleDisconnectTimerDoesNotCutTheNextGracePeriodShort(t *testing.T) {
	clock := &fakeClock{ignoreStop: true}
	ls := newClocked(t, dummyComponent, clock, nil)
	first, second := newRecordingTransport(), newRecordingTransport()
	if _, err := ls.Attach(context.Background(), first, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	ls.Detach(first)
	barrier(t, ls)

	clock.Advance(testTTL - time.Second)
	if _, err := ls.Attach(context.Background(), second, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	ls.Detach(second)
	barrier(t, ls)

	clock.Advance(2 * time.Second)
	barrier(t, ls)
	if ls.State() != StateDisconnected {
		t.Fatalf("the first disconnect's timer closed the session %v into its second grace period", ls.State())
	}

	clock.Advance(testTTL)
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonDisconnectExpired {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
}

func TestAbsoluteLifetimeCloses(t *testing.T) {
	clock := &fakeClock{}
	ls := newClocked(t, dummyComponent, clock, func(c *Config) { c.MaxLifetime = time.Hour })
	tr := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	clock.Advance(time.Hour - time.Second)
	barrier(t, ls)
	if ls.State() != StateLive {
		t.Fatalf("closed early: %v", ls.State())
	}
	clock.Advance(time.Second)
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonLifetimeExpired {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
	if !tr.Closed() || tr.code.Load() != int64(websocket.StatusGoingAway) {
		t.Fatalf("transport closed=%v code=%d, want closed with 1001", tr.Closed(), tr.code.Load())
	}
	if clock.Armed() != 0 {
		t.Fatalf("%d timers still armed after close", clock.Armed())
	}
}

func TestLifetimeClosesEvenWhenTheActorIsBusy(t *testing.T) {
	clock := &fakeClock{}
	ls := newClocked(t, dummyComponent, clock, func(c *Config) { c.MaxLifetime = time.Hour })
	release := holdActor(t, ls)
	for i := 0; i < 50; i++ {
		_ = ls.Session().Post(func() {})
	}
	clock.Advance(time.Hour)
	release()
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonLifetimeExpired {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
}

func TestPendingSessionRunsNoEffectsAndTimersUntilTheFirstAttach(t *testing.T) {
	var effects, cleanups, fired atomic.Int64
	comp := func(ctx *runtime.Ctx) work.Node {
		runtime.UseEffect(ctx, func() func() {
			effects.Add(1)
			ctx.After(10*time.Millisecond, func() { fired.Add(1) })
			return func() { cleanups.Add(1) }
		})
		return &work.Element{Tag: "div"}
	}
	ls := NewLiveSession("pending", 1, comp, nil)
	t.Cleanup(func() { _ = ls.Close() })

	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	barrier(t, ls)
	if effects.Load() != 0 || fired.Load() != 0 {
		t.Fatalf("pending render ran %d effects and %d timers", effects.Load(), fired.Load())
	}

	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	if effects.Load() != 1 {
		t.Fatalf("effects after first attach = %d, want 1", effects.Load())
	}
	deadline := time.Now().Add(2 * time.Second)
	for fired.Load() != 1 {
		if time.Now().After(deadline) {
			t.Fatal("the timer created by the effect never fired after attach")
		}
		time.Sleep(time.Millisecond)
	}

	second := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), second, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	if effects.Load() != 1 {
		t.Fatalf("a re-attach re-ran effects: %d", effects.Load())
	}
}

func TestTimerCreatedDuringThePendingRenderStartsOnTheFirstAttach(t *testing.T) {
	var once sync.Once
	var fired atomic.Int64
	comp := func(ctx *runtime.Ctx) work.Node {
		once.Do(func() { ctx.After(10*time.Millisecond, func() { fired.Add(1) }) })
		return &work.Element{Tag: "div"}
	}
	ls := NewLiveSession("pending", 1, comp, nil)
	t.Cleanup(func() { _ = ls.Close() })
	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	barrier(t, ls)
	if fired.Load() != 0 {
		t.Fatal("a timer created while pending fired before any client attached")
	}
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for fired.Load() != 1 {
		if time.Now().After(deadline) {
			t.Fatal("the deferred timer never fired after attach")
		}
		time.Sleep(time.Millisecond)
	}
}

func TestDeferredTimerStoppedWhilePendingNeverFires(t *testing.T) {
	var fired atomic.Int64
	var timer *runtime.Timer
	comp := func(ctx *runtime.Ctx) work.Node {
		runtime.UseEffect(ctx, func() func() {
			timer = ctx.After(10*time.Millisecond, func() { fired.Add(1) })
			return nil
		})
		return &work.Element{Tag: "div"}
	}
	ls := NewLiveSession("pending", 1, comp, nil)
	t.Cleanup(func() { _ = ls.Close() })
	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	if err := ls.Session().Ask(time.Second, func() error { timer.Stop(); return nil }); err != nil {
		t.Fatal(err)
	}
	time.Sleep(60 * time.Millisecond)
	if fired.Load() != 0 {
		t.Fatal("a stopped timer fired")
	}
}

type orderedTransport struct {
	mu   sync.Mutex
	msgs []string
	recordingTransport
}

func (t *orderedTransport) SendFrame(frame []byte) error {
	var m protocol.Message
	if err := json.Unmarshal(frame, &m); err != nil {
		return err
	}
	t.mu.Lock()
	t.msgs = append(t.msgs, m.Topic+"/"+m.Event)
	t.mu.Unlock()
	return nil
}

func (t *orderedTransport) log() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return append([]string(nil), t.msgs...)
}

func TestMessagesBeforeJoinAreDeliveredAfterJoined(t *testing.T) {
	var bump func(int)
	comp := func(ctx *runtime.Ctx) work.Node {
		sig := runtime.UseSignal(ctx, 0)
		bump = sig.Set
		letter := runtime.UseComputed(ctx, func() string { return string(rune('a' + sig.Get())) })
		return work.BuildElement("p", work.TextOf(letter))
	}
	ls := NewLiveSession("pending", 1, comp, nil)
	t.Cleanup(func() { _ = ls.Close() })

	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.HTML == "" || res.Boot.Seq != 0 {
		t.Fatalf("render result = %+v", res)
	}
	if n, _ := ls.Retained(); n != 0 {
		t.Fatalf("the SSR render itself left %d messages for the client; render-only pending must produce none", n)
	}

	var set func(int)
	if err := ls.Session().Ask(time.Second, func() error { set = bump; return nil }); err != nil {
		t.Fatal(err)
	}
	set(1)
	barrier(t, ls)
	if n, _ := ls.Retained(); n != 1 {
		t.Fatalf("a state change between SSR and join must be buffered, got %d messages", n)
	}

	tr := &orderedTransport{}
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	got := tr.log()
	if len(got) != 1 || got[0] != "frame/patch" {
		t.Fatalf("messages delivered at attach = %v, want the buffered frame", got)
	}
	if n, _ := ls.Retained(); n != 1 {
		t.Fatalf("the delivered frame must stay in the ring until acked, retained %d", n)
	}
	if err := ls.Ack(ls.Generation(), 1); err != nil {
		t.Fatal(err)
	}
	if n, b := ls.Retained(); n != 0 || b != 0 {
		t.Fatalf("an acked frame stayed in the ring: %d frames %d bytes", n, b)
	}

	set(2)
	barrier(t, ls)
	if got := tr.log(); len(got) != 2 {
		t.Fatalf("live frames after attach = %v", got)
	}
}

func TestAttachAbandonedAfterDeadlineIsSkipped(t *testing.T) {
	ls := newClocked(t, dummyComponent, &fakeClock{}, nil)
	release := holdActor(t, ls)

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	tr := newRecordingTransport()
	started := time.Now()
	_, err := ls.Attach(ctx, tr, nil, 0, false)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Attach returned %v, want the deadline", err)
	}
	if time.Since(started) > 2*time.Second {
		t.Fatal("Attach did not return at the deadline")
	}

	release()
	barrier(t, ls)
	if ls.Generation() != 0 || ls.State() != StatePending {
		t.Fatalf("the abandoned attach ran: gen=%d state=%v", ls.Generation(), ls.State())
	}
	if tr.sent.Load() != 0 {
		t.Fatal("frames were sent to the abandoned joiner")
	}
}

func TestAttachWhoseStepIsRunningReturnsAtTheDeadline(t *testing.T) {
	var blockRender atomic.Bool
	gate := make(chan struct{})
	comp := func(ctx *runtime.Ctx) work.Node {
		runtime.UseEffect(ctx, func() func() {
			if blockRender.Load() {
				<-gate
			}
			return nil
		})
		return &work.Element{Tag: "div"}
	}
	ls := newClocked(t, comp, &fakeClock{}, nil)
	blockRender.Store(true)
	defer close(gate)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	started := time.Now()
	_, err := ls.Attach(ctx, newRecordingTransport(), nil, 0, false)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("Attach returned %v, want the deadline", err)
	}
	if time.Since(started) > 2*time.Second {
		t.Fatalf("a render that never returns held the joiner for %v", time.Since(started))
	}
}

func TestDeadJoinerDoesNotTakeOver(t *testing.T) {
	ls := newClocked(t, dummyComponent, &fakeClock{}, nil)
	live := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), live, nil, 0, false); err != nil {
		t.Fatal(err)
	}

	release := holdActor(t, ls)
	dead := newRecordingTransport()
	result := make(chan error, 1)
	go func() {
		_, err := ls.Attach(context.Background(), dead, nil, 0, false)
		result <- err
	}()
	time.Sleep(20 * time.Millisecond)
	dead.Terminate(websocket.StatusGoingAway, "gone")
	release()

	if err := <-result; !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Attach of a dead transport returned %v", err)
	}
	if live.Closed() {
		t.Fatal("a dead joiner replaced the live connection")
	}
	if ls.Generation() != 1 || ls.State() != StateLive {
		t.Fatalf("gen=%d state=%v", ls.Generation(), ls.State())
	}
}

func TestAttachReplacesTheCurrentTransportWith4409(t *testing.T) {
	ls := newClocked(t, dummyComponent, &fakeClock{}, nil)
	first, second := newRecordingTransport(), newRecordingTransport()
	if _, err := ls.Attach(context.Background(), first, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	ar, err := ls.Attach(context.Background(), second, nil, 0, false)
	if err != nil || ar.Gen != 2 {
		t.Fatalf("gen=%d err=%v", ar.Gen, err)
	}
	if !first.Closed() || first.code.Load() != int64(CloseReplaced) {
		t.Fatalf("replaced transport closed=%v code=%d", first.Closed(), first.code.Load())
	}
	ls.Detach(first)
	barrier(t, ls)
	if ls.State() != StateLive || second.Closed() {
		t.Fatal("the stale transport's detach affected the live connection")
	}
}

func TestDetachedOverflowDiscardsWithoutClosingAndNextAttachResets(t *testing.T) {
	ls := newClocked(t, dummyComponent, &fakeClock{}, func(c *Config) { c.ReplayFrames = 5 })
	bus := ls.Session().Bus
	for i := 0; i < 50; i++ {
		bus.Publish(protocol.TopicFrame, "patch", []any{})
	}
	barrier(t, ls)
	if ls.State() == StateClosing || ls.State() == StateClosed {
		t.Fatalf("overflow closed the session: %v", ls.State())
	}
	if n, b := ls.Retained(); n != 0 || b != 0 {
		t.Fatalf("a discarding ring retained %d frames %d bytes", n, b)
	}
	tr := newRecordingTransport()
	ar, err := ls.Attach(context.Background(), tr, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if ar.Mode != protocol.ModeReset {
		t.Fatalf("mode = %q, want reset", ar.Mode)
	}
	msgs := tr.messages()
	if len(msgs) != 1 || msgs[0].Event != string(protocol.FrameResetAction) || msgs[0].Seq <= 50 {
		t.Fatalf("attach sent %+v", msgs)
	}
}

func TestSetterAfterCloseIsNoop(t *testing.T) {
	var set func(int)
	var renders atomic.Int64
	comp := func(ctx *runtime.Ctx) work.Node {
		renders.Add(1)
		sig := runtime.UseSignal(ctx, 0)
		set = sig.Set
		return work.BuildElement("div", work.TextOf(sig))
	}
	ls := NewLiveSession("closed", 1, comp, nil)
	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	var local func(int)
	if err := ls.Session().Ask(time.Second, func() error { local = set; return nil }); err != nil {
		t.Fatal(err)
	}
	_ = ls.Close()
	before := renders.Load()
	for i := 0; i < 100; i++ {
		local(i)
	}
	if renders.Load() != before {
		t.Fatal("a setter after close rendered")
	}
}

func TestRenderRedirectClosesTheSessionAndKeepsSetCookie(t *testing.T) {
	comp := func(ctx *runtime.Ctx) work.Node {
		if rs := headers.UseRequestState(ctx); rs != nil {
			rs.SetRedirect("/login", http.StatusSeeOther)
			rs.AddResponseHeader("Set-Cookie", "flash=1; Path=/")
		}
		return &work.Element{Tag: "div"}
	}
	ls := NewLiveSession("redirect", 1, comp, nil, WithRequest(httptest.NewRequest(http.MethodGet, "/", nil)))
	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Redirect == nil || res.Redirect.URL != "/login" || res.Status != http.StatusSeeOther {
		t.Fatalf("result = %+v", res)
	}
	if got := res.Header.Values("Set-Cookie"); len(got) != 1 || got[0] != "flash=1; Path=/" {
		t.Fatalf("Set-Cookie = %v", got)
	}
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonRedirected {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
}

func TestRenderCrashClosesTheSession(t *testing.T) {
	comp := func(ctx *runtime.Ctx) work.Node { panic("boom") }
	ls := NewLiveSession("crash", 1, comp, &Config{DevMode: true})
	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusInternalServerError || res.Crash == nil || res.Crash.Message != "boom" {
		t.Fatalf("result = %+v", res)
	}
	if res.HTML == "" {
		t.Fatal("a crash must still carry the rendered crash page")
	}
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonRenderFailed {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
}

func TestUserBoundaryThatHandlesTheErrorIsNotACrash(t *testing.T) {
	child := runtime.Component(func(ctx *runtime.Ctx) work.Node { panic("inner") })
	guard := runtime.Component(func(ctx *runtime.Ctx) work.Node {
		boundary := runtime.UseErrorBoundary(ctx)
		failed := runtime.UseComputed(ctx, func() bool { return boundary.Errors().Get().HasErrors() })
		return work.BuildElement("div", runtime.Show(failed,
			func() work.Node { return work.BuildElement("p", work.NewText("fallback")) },
			runtime.Else(func() work.Node { return work.BuildElement("section", child()) }),
		))
	})
	comp := func(ctx *runtime.Ctx) work.Node { return guard() }
	ls := NewLiveSession("guarded", 1, comp, nil)
	t.Cleanup(func() { _ = ls.Close() })
	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != http.StatusOK || res.Crash != nil {
		t.Fatalf("an error handled by a user boundary produced status %d", res.Status)
	}
}

func TestRenderOnClosedSessionFails(t *testing.T) {
	ls := NewLiveSession("closed", 1, dummyComponent, nil)
	_ = ls.Close()
	if _, err := ls.Render(context.Background()); err == nil {
		t.Fatal("Render on a closed session must fail")
	}
}

func TestRenderCancelledByTheClientClosesTheSession(t *testing.T) {
	ls := NewLiveSession("cancel", 1, dummyComponent, nil)
	release := holdActor(t, ls)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := ls.Render(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("Render returned %v", err)
	}
	release()
	waitClosed(t, ls)
}

type killer struct {
	victim atomic.Pointer[recordingTransport]
	extra  atomic.Pointer[recordingTransport]
}

func cookieHeader(n int) http.Header {
	return http.Header{"Cookie": {"n=" + strconv.Itoa(n)}}
}

func (k *killer) component(ctx *runtime.Ctx) work.Node {
	n := headers.UseRequest(ctx).Cookie("n")
	runtime.UseEffect(ctx, func() func() {
		_ = n.Get()
		if v := k.victim.Load(); v != nil {
			v.Terminate(websocket.StatusGoingAway, "died mid-attach")
			if e := k.extra.Load(); e != nil {
				e.Terminate(websocket.StatusGoingAway, "died mid-attach")
			}
		}
		return nil
	})
	return &work.Element{Tag: "div"}
}

func TestReplacingJoinerThatDiesMidAttachKeepsTheOldConnection(t *testing.T) {
	k := &killer{}
	ls := newClocked(t, k.component, &fakeClock{}, nil)
	old, joiner := newRecordingTransport(), newRecordingTransport()
	if _, err := ls.Attach(context.Background(), old, cookieHeader(1), 0, false); err != nil {
		t.Fatal(err)
	}
	k.victim.Store(joiner)
	if _, err := ls.Attach(context.Background(), joiner, cookieHeader(2), 0, false); err == nil {
		t.Fatal("an attach whose transport died must fail")
	}
	barrier(t, ls)
	if old.Closed() {
		t.Fatal("the old connection was kicked for a joiner that never completed")
	}
	if ls.State() != StateLive || ls.Generation() != 1 {
		t.Fatalf("state=%v gen=%d", ls.State(), ls.Generation())
	}
}

func TestReplacingJoinerThatDiesWithTheOldOneFallsBackToDisconnectedWithAGraceTimer(t *testing.T) {
	k := &killer{}
	clock := &fakeClock{}
	ls := newClocked(t, k.component, clock, nil)
	old, joiner := newRecordingTransport(), newRecordingTransport()
	if _, err := ls.Attach(context.Background(), old, cookieHeader(1), 0, false); err != nil {
		t.Fatal(err)
	}
	k.victim.Store(joiner)
	k.extra.Store(old)
	if _, err := ls.Attach(context.Background(), joiner, cookieHeader(2), 0, false); err == nil {
		t.Fatal("an attach whose transport died must fail")
	}
	barrier(t, ls)
	if ls.State() != StateDisconnected {
		t.Fatalf("state = %v, want disconnected (no zombie live session without a transport)", ls.State())
	}
	clock.Advance(testTTL)
	waitClosed(t, ls)
	if ls.CloseReason() != ReasonDisconnectExpired {
		t.Fatalf("reason = %v", ls.CloseReason())
	}
}

func TestFirstJoinerThatDiesAfterGoLiveFallsBackToDisconnected(t *testing.T) {
	k := &killer{}
	clock := &fakeClock{}
	ls := newClocked(t, k.component, clock, nil)
	joiner := newRecordingTransport()
	k.victim.Store(joiner)
	if _, err := ls.Attach(context.Background(), joiner, nil, 0, false); err == nil {
		t.Fatal("an attach whose transport died must fail")
	}
	barrier(t, ls)
	if ls.State() != StateDisconnected {
		t.Fatalf("state = %v", ls.State())
	}
	clock.Advance(testTTL)
	waitClosed(t, ls)
}

func TestClosingWithAPendingDOMQueryDoesNotWaitForTheBrowser(t *testing.T) {
	var ref *runtime.Ref
	var called atomic.Bool
	comp := func(ctx *runtime.Ctx) work.Node {
		ref = runtime.UseCell(ctx, func() *runtime.Ref { return runtime.NewRef(ctx, "InputRef", nil) })
		return work.BuildElement("input", runtime.AttachItem(ref), work.On("click", func(work.Event) {
			ref.Query(func(map[string]any, error) { called.Store(true) }, "selectionStart")
		}))
	}
	ls := NewLiveSession("dom", 1, comp, &Config{DOMTimeout: 30 * time.Second})
	tr := &domProbeTransport{queries: make(chan protocol.DOMQueryPayload, 1)}
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	var refID uint32
	onSession(t, ls, func() { refID = ref.ID() })
	if err := deliver(ls, clickEvt(refID, 0)); err != nil {
		t.Fatal(err)
	}
	select {
	case <-tr.queries:
	case <-time.After(2 * time.Second):
		t.Fatal("the handler never issued its query")
	}

	begin := time.Now()
	ls.RequestClose(ReasonShutdown)
	waitClosed(t, ls)
	if took := time.Since(begin); took > 3*time.Second {
		t.Fatalf("close waited %v for a DOM query that can no longer be answered", took)
	}
	if called.Load() {
		t.Fatal("the callback of an unanswered query ran after the session closed")
	}
}
