package runtime

import (
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

func newStarted(t *testing.T, root func(*Ctx) work.Node) *Session {
	t.Helper()
	if root == nil {
		root = func(*Ctx) work.Node { return work.BuildElement("div") }
	}
	s := NewSession(Config{SessionID: "sess", Root: root})
	s.Start()
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.Close()
		}()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Errorf("session did not close within 5s")
		}
	})
	if err := s.Ask(5*time.Second, func() error { _, err := s.RenderSnapshot(); return err }); err != nil {
		t.Fatal(err)
	}
	return s
}

func onActor(t testing.TB, s *Session, fn func()) {
	t.Helper()
	if err := s.Ask(5*time.Second, func() error { fn(); return nil }); err != nil {
		t.Fatalf("ask failed: %v", err)
	}
}

func within(t *testing.T, bound time.Duration, what string, fn func()) {
	t.Helper()
	done := make(chan struct{})
	go func() {
		defer close(done)
		fn()
	}()
	select {
	case <-done:
	case <-time.After(bound):
		t.Fatalf("%s did not return within %v", what, bound)
	}
}

func collectDiagnostics(s *Session) func() []protocol.Diagnostic {
	var mu sync.Mutex
	var got []protocol.Diagnostic
	s.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		mu.Lock()
		got = append(got, d)
		mu.Unlock()
	})
	return func() []protocol.Diagnostic {
		mu.Lock()
		defer mu.Unlock()
		return append([]protocol.Diagnostic(nil), got...)
	}
}

func hasPhase(diags []protocol.Diagnostic, phase string) bool {
	for _, d := range diags {
		if d.Phase == phase {
			return true
		}
	}
	return false
}

func counterRoot(sig **reactive.Signal[int]) func(*Ctx) work.Node {
	return func(ctx *Ctx) work.Node {
		*sig = UseSignal(ctx, 0)
		return work.BuildElement("p", work.TextOf(*sig))
	}
}

func TestSettersFromManyGoroutinesDoNotRace(t *testing.T) {
	var n *reactive.Signal[int]
	s := newStarted(t, counterRoot(&n))

	var wg sync.WaitGroup
	for g := 0; g < 8; g++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				n.Update(func(v int) int { return v + 1 })
			}
		}()
	}
	wg.Wait()
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}

	var got int
	onActor(t, s, func() { got = n.Peek() })
	if got != 800 {
		t.Fatalf("lost off-actor updates: %d, want 800", got)
	}
}

func TestOffActorSetIsPostedNotApplied(t *testing.T) {
	var n *reactive.Signal[int]
	s := newStarted(t, counterRoot(&n))
	release := make(chan struct{})
	started := make(chan struct{})
	_ = s.Post(func() {
		close(started)
		<-release
	})
	<-started

	n.Set(5)

	var peek int
	close(release)
	onActor(t, s, func() { peek = n.Peek() })
	if peek != 5 {
		t.Fatalf("posted write was lost: %d", peek)
	}
}

func TestSetterDuringCloseIsNoop(t *testing.T) {
	var n *reactive.Signal[int]
	s := newStarted(t, counterRoot(&n))
	s.Close()

	n.Set(1)

	if err := s.Flush(); err == nil {
		t.Fatal("flush on a closed session must fail")
	}
}

func TestCloseDisposesCleanupsOnce(t *testing.T) {
	var runs atomic.Int32
	s := newStarted(t, func(ctx *Ctx) work.Node {
		UseEffect(ctx, func() func() { return func() { runs.Add(1) } })
		return work.BuildElement("div")
	})
	onActor(t, s, func() { s.GoLive() })
	_ = s.Flush()

	s.Close()
	s.Close()

	if runs.Load() != 1 {
		t.Fatalf("cleanup ran %d times", runs.Load())
	}
}

func TestCascadeBudgetStopsRenderWatchPingPongAndReportsUpdateDepth(t *testing.T) {
	var x, y *reactive.Signal[int]
	s := NewSession(Config{SessionID: "budget", Root: func(ctx *Ctx) work.Node {
		x = UseSignal(ctx, 0)
		y = UseSignal(ctx, 0)
		UseEffect(ctx, func() func() { x.Set(y.Get() + 1); return nil })
		return work.BuildElement("p", work.TextOf(y), work.Attribute("data-x", x))
	}})
	diags := collectDiagnostics(s)
	s.Start()
	t.Cleanup(s.Close)
	onActor(t, s, func() {
		_, _ = s.RenderSnapshot()
		reactive.NewEffect(s.root, reactive.KindRender, func() { y.Set(x.Get() + 1) })
		s.GoLive()
	})

	within(t, 5*time.Second, "flush", func() { _ = s.Flush() })

	if !hasPhase(diags(), "flush:update-depth") {
		t.Fatalf("no update-depth diagnostic: %v", diags())
	}
}

func TestWatchSelfLoopHitsTheRoundLimit(t *testing.T) {
	var n *reactive.Signal[int]
	s := NewSession(Config{SessionID: "loop", Root: func(ctx *Ctx) work.Node {
		n = UseSignal(ctx, 0)
		UseEffect(ctx, func() func() { n.Set(n.Get() + 1); return nil })
		return work.BuildElement("p", work.TextOf(n))
	}})
	diags := collectDiagnostics(s)
	s.DeferEffects()
	s.Start()
	t.Cleanup(s.Close)

	within(t, 5*time.Second, "go live", func() {
		_ = s.Ask(5*time.Second, func() error { _, err := s.RenderSnapshot(); s.GoLive(); return err })
	})

	if !hasPhase(diags(), "flush:render-loop") {
		t.Fatalf("diagnostics = %v", diags())
	}
}

func TestSessionStepPanicClosesSessionInsteadOfWedging(t *testing.T) {
	s := newStarted(t, nil)
	diags := collectDiagnostics(s)

	_ = s.Post(func() { panic("invariant") })

	within(t, 5*time.Second, "Flush after a step panic", func() {
		if err := s.Flush(); err == nil {
			t.Error("Flush after a fatal step panic must report the closed session")
		}
	})
	if !hasPhase(diags(), "actor:panic") {
		t.Fatal("expected an actor:panic diagnostic")
	}
}

func TestFlushBlocksUntilItsFlushHasRun(t *testing.T) {
	var n *reactive.Signal[int]
	s := newStarted(t, counterRoot(&n))
	for i := 1; i <= 20; i++ {
		before := s.FlushCount()
		n.Set(i)
		if err := s.Flush(); err != nil {
			t.Fatal(err)
		}
		if s.FlushCount() <= before {
			t.Fatalf("Flush #%d returned without running a flush", i)
		}
	}
}

func TestClientEventBacklogIsBounded(t *testing.T) {
	s := newStarted(t, nil)
	release := make(chan struct{})
	started := make(chan struct{})
	_ = s.Post(func() {
		close(started)
		<-release
	})
	<-started

	var ran atomic.Int64
	accepted := 0
	var overflow error
	for i := 0; i < maxClientEvents+10; i++ {
		if err := s.PostClientEvent(func() { ran.Add(1) }); err != nil {
			overflow = err
			break
		}
		accepted++
	}
	if accepted != maxClientEvents || overflow != ErrClientBacklog {
		t.Fatalf("accepted %d events, overflow=%v; want %d accepted then ErrClientBacklog", accepted, overflow, maxClientEvents)
	}
	close(release)
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	if ran.Load() != int64(maxClientEvents) {
		t.Fatalf("ran %d client events, want %d", ran.Load(), maxClientEvents)
	}
	if err := s.PostClientEvent(func() {}); err != nil {
		t.Fatalf("the backlog did not drain: %v", err)
	}
}

func TestInboxOverflowClosesSessionAndReportsOnceWithCount(t *testing.T) {
	s := newStarted(t, nil)
	diags := collectDiagnostics(s)
	release := make(chan struct{})
	started := make(chan struct{})
	_ = s.Post(func() {
		close(started)
		<-release
	})
	<-started
	s.mb.mu.Lock()
	s.mb.max = 4
	s.mb.mu.Unlock()

	var overflow error
	for i := 0; i < 200; i++ {
		if err := s.Post(func() {}); err != nil && overflow == nil {
			overflow = err
		}
	}
	if overflow != ErrInboxOverflow {
		t.Fatalf("expected ErrInboxOverflow, got %v", overflow)
	}
	close(release)
	select {
	case <-s.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the session did not close after the inbox overflowed")
	}
	if err := s.Post(func() {}); err != ErrSessionClosed {
		t.Fatalf("post after overflow close returned %v", err)
	}
	count := 0
	var dropped int64
	for _, d := range diags() {
		if d.Phase == "actor:inbox-overflow" {
			count++
			dropped, _ = d.Metadata["dropped"].(int64)
		}
	}
	if count != 1 || dropped < 100 {
		t.Fatalf("overflow diagnostics=%d dropped=%d, want exactly one reporting the dropped count", count, dropped)
	}
}

func TestPanickingDiagnosticSubscriberDoesNotCrashTheProcess(t *testing.T) {
	var clicks atomic.Int64
	s := newStarted(t, func(ctx *Ctx) work.Node {
		return work.BuildElement("button", work.On("click", func(work.Event) {
			clicks.Add(1)
			panic("handler panic")
		}))
	})
	s.Bus.SubscribeToDiagnostics(func(protocol.Diagnostic) { panic("diagnostics sink panic") })

	_ = s.PostClientEvent(func() { _ = s.DispatchHandler(1, "click", nil) })

	if err := s.Flush(); err != nil {
		t.Fatalf("session died after a panicking diagnostics subscriber: %v", err)
	}
	if clicks.Load() != 1 {
		t.Fatalf("handler ran %d times", clicks.Load())
	}
}

func TestStartIsIdempotent(t *testing.T) {
	s := newStarted(t, nil)
	done := s.Done()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			s.Start()
		}()
	}
	wg.Wait()
	if s.Done() != done {
		t.Fatal("Start replaced the running actor")
	}
}

func TestUnstartedSessionHasNoExecutionPath(t *testing.T) {
	s := NewSession(Config{Root: func(*Ctx) work.Node { return work.BuildElement("div") }})

	if err := s.Post(func() {}); err != ErrNotStarted {
		t.Fatalf("Post = %v", err)
	}
	if err := s.Ask(time.Second, func() error { return nil }); err != ErrNotStarted {
		t.Fatalf("Ask = %v", err)
	}
	if err := s.Flush(); err != ErrNotStarted {
		t.Fatalf("Flush = %v", err)
	}
}

func TestCtxRequestCloseFromInsideTheActorDoesNotStall(t *testing.T) {
	var ctx *Ctx
	s := newStarted(t, func(c *Ctx) work.Node {
		ctx = c
		return work.BuildElement("div")
	})

	within(t, 5*time.Second, "RequestClose inside the actor", func() {
		onActor(t, s, func() { _ = ctx.s.RequestClose() })
		select {
		case <-s.Done():
		case <-time.After(3 * time.Second):
			t.Error("session did not close")
		}
	})
}

func TestRunOnActorIssuedFromWatchIsAbsorbed(t *testing.T) {
	var n *reactive.Signal[int]
	var got atomic.Int64
	s := NewSession(Config{SessionID: "r", Root: func(ctx *Ctx) work.Node {
		n = UseSignal(ctx, 0)
		UseEffect(ctx, func() func() {
			v := n.Get()
			RunOnActor(ctx, func() { got.Store(int64(v)) })
			return nil
		})
		return work.BuildElement("p", work.TextOf(n))
	}})
	s.DeferEffects()
	s.Start()
	t.Cleanup(s.Close)
	onActor(t, s, func() { _, _ = s.RenderSnapshot(); s.GoLive() })
	onActor(t, s, func() { n.Set(9) })
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	if err := s.Flush(); err != nil {
		t.Fatal(err)
	}
	if got.Load() != 9 {
		t.Fatalf("got = %d", got.Load())
	}
}

func TestTimersFireOnTheActorAndStopsAreHonoured(t *testing.T) {
	s := newStarted(t, nil)
	var fired, stopped atomic.Int32
	onActor(t, s, func() {
		s.After(5*time.Millisecond, func() { fired.Add(1) })
		tm := s.After(5*time.Millisecond, func() { stopped.Add(1) })
		tm.Stop()
	})
	deadline := time.Now().Add(2 * time.Second)
	for fired.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	time.Sleep(20 * time.Millisecond)
	if fired.Load() != 1 || stopped.Load() != 0 {
		t.Fatalf("fired=%d stopped=%d", fired.Load(), stopped.Load())
	}
}

func TestDeferredTimersWaitForGoLive(t *testing.T) {
	s := NewSession(Config{Root: func(*Ctx) work.Node { return work.BuildElement("div") }})
	s.DeferEffects()
	s.Start()
	t.Cleanup(s.Close)
	var fired atomic.Int32
	onActor(t, s, func() {
		_, _ = s.RenderSnapshot()
		s.After(time.Millisecond, func() { fired.Add(1) })
	})
	time.Sleep(30 * time.Millisecond)
	if fired.Load() != 0 {
		t.Fatal("timer fired before GoLive")
	}
	onActor(t, s, func() { s.GoLive() })
	deadline := time.Now().Add(2 * time.Second)
	for fired.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(2 * time.Millisecond)
	}
	if fired.Load() != 1 {
		t.Fatalf("fired = %d", fired.Load())
	}
}

func TestAskReportsBusyAndClosed(t *testing.T) {
	s := newStarted(t, nil)
	release := make(chan struct{})
	started := make(chan struct{})
	_ = s.Post(func() {
		close(started)
		<-release
	})
	<-started
	if err := s.Ask(20*time.Millisecond, func() error { return nil }); err != ErrSessionBusy {
		t.Fatalf("Ask on a busy actor = %v", err)
	}
	close(release)
	s.Close()
	if err := s.Ask(time.Second, func() error { return nil }); err == nil {
		t.Fatal("Ask on a closed session must fail")
	}
}

func TestStepRunningReportsTheCurrentStep(t *testing.T) {
	s := newStarted(t, nil)
	release := make(chan struct{})
	started := make(chan struct{})
	_ = s.Post(func() {
		close(started)
		<-release
	})
	<-started
	_, seq, running := s.StepRunning()
	close(release)
	if !running || seq == 0 {
		t.Fatalf("running=%v seq=%d", running, seq)
	}
	_ = fmt.Sprint
}
