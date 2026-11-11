package runtime_test

import (
	"fmt"
	goruntime "runtime"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
	"weak"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/shared"
)

func block(t *testing.T, s *runtime.Session) (release func()) {
	t.Helper()
	started := make(chan struct{})
	rel := make(chan struct{})
	if err := s.Post(func() {
		close(started)
		<-rel
	}); err != nil {
		t.Fatal(err)
	}
	<-started
	var once sync.Once
	release = func() { once.Do(func() { close(rel) }) }
	t.Cleanup(release)
	return release
}

func sharedApp(t *testing.T, sh *shared.Shared[int], opts ...runtimetest.Option) *runtimetest.App {
	t.Helper()
	return runtimetest.New(t, func(ctx *Ctx) Node {
		return el("p", bound(runtime.UseShared(ctx, sh)))
	}, opts...)
}

func TestManySessionsOnOneSharedConvergeToTheLastValue(t *testing.T) {
	sh := shared.NewShared(0)
	apps := make([]*runtimetest.App, 40)
	for i := range apps {
		apps[i] = sharedApp(t, sh)
	}
	var wg sync.WaitGroup
	for g := range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range 200 {
				sh.Set(g*1000 + i)
			}
		}()
	}
	wg.Wait()
	sh.Set(424242)
	for i, a := range apps {
		a.Settle()
		a.Sync()
		if got := a.Normalized(); !strings.Contains(got, "424242") {
			t.Fatalf("session %d did not converge: %s", i, got)
		}
	}
}

func TestSharedUpdateFromManySessionsLosesNothing(t *testing.T) {
	sh := shared.NewShared(0)
	apps := make([]*runtimetest.App, 10)
	for i := range apps {
		apps[i] = sharedApp(t, sh)
	}
	var wg sync.WaitGroup
	for _, a := range apps {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 500 {
				a.S.RunOnActor(func() { sh.Update(func(n int) int { return n + 1 }) })
			}
		}()
	}
	wg.Wait()
	for _, a := range apps {
		a.Settle()
	}
	for _, a := range apps {
		a.Sync()
		if got := a.Normalized(); !strings.Contains(got, "5000") {
			t.Fatalf("got %s", got)
		}
	}
}

func TestSlowSessionSeesLatestOnly(t *testing.T) {
	sh := shared.NewShared(0)
	app := sharedApp(t, sh)
	release := block(t, app.S)
	for i := 1; i <= 1000; i++ {
		sh.Set(i)
	}
	if n := runtime.MailboxLen(app.S); n > 2 {
		t.Fatalf("slow session has %d queued messages, want at most one pending delivery", n)
	}
	release()
	edits := app.Sync()
	if len(edits) != 1 {
		t.Fatalf("edits = %v, want exactly one text edit for the latest value", edits)
	}
	if got := app.Normalized(); !strings.Contains(got, "1000") {
		t.Fatalf("got %s", got)
	}
}

func TestBlockedSessionDoesNotStallOthers(t *testing.T) {
	sh := shared.NewShared(0)
	slow := sharedApp(t, sh)
	fast := sharedApp(t, sh)
	release := block(t, slow.S)

	done := make(chan struct{})
	go func() {
		defer close(done)
		for i := 1; i <= 100; i++ {
			sh.Set(i)
		}
	}()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("publishing stalled behind a blocked session")
	}
	fast.Sync()
	if got := fast.Normalized(); !strings.Contains(got, "100") {
		t.Fatalf("fast session did not converge while the other was blocked: %s", got)
	}
	release()
	slow.Sync()
	if got := slow.Normalized(); !strings.Contains(got, "100") {
		t.Fatalf("slow session did not converge after release: %s", got)
	}
}

type broadcastProbe struct {
	got      []int
	onActor  []bool
	counter  *runtime.Signal[int]
	boundary *runtime.Boundary
}

func broadcastApp(t *testing.T, b *shared.Broadcast[int], probe *broadcastProbe, opts ...runtimetest.Option) *runtimetest.App {
	t.Helper()
	var app *runtimetest.App
	app = runtimetest.New(t, func(ctx *Ctx) Node {
		probe.counter = runtime.UseSignal(ctx, 0)
		runtime.UseBroadcast(ctx, b, func(v int) {
			probe.got = append(probe.got, v)
			probe.onActor = append(probe.onActor, runtime.OnActor(app.S))
			probe.counter.Set(len(probe.got))
		})
		return el("p", bound(probe.counter))
	}, opts...)
	return app
}

func TestBroadcastHandlerRunsOnTheActorInOrder(t *testing.T) {
	b := shared.NewBroadcast[int]()
	probe := &broadcastProbe{}
	app := broadcastApp(t, b, probe)
	for i := range 20 {
		b.Send(i)
	}
	app.Sync()
	app.On(func() {
		if len(probe.got) != 20 {
			t.Errorf("got %v", probe.got)
		}
		for i, v := range probe.got {
			if v != i || !probe.onActor[i] {
				t.Errorf("event %d = %d onActor=%v", i, v, probe.onActor[i])
			}
		}
	})
	if got := app.Normalized(); !strings.Contains(got, "20") {
		t.Fatalf("handler write did not reach the DOM: %s", got)
	}
}

func TestBroadcastInboxDropsOldest(t *testing.T) {
	b := shared.NewBroadcast[int]()
	probe := &broadcastProbe{}
	app := broadcastApp(t, b, probe)
	release := block(t, app.S)
	for i := range 300 {
		b.Send(i)
	}
	release()
	app.Sync()
	app.On(func() {
		if len(probe.got) != 256 || probe.got[0] != 44 || probe.got[255] != 299 {
			t.Errorf("kept %d events, first %v", len(probe.got), probe.got[:min(3, len(probe.got))])
		}
		for i := 1; i < len(probe.got); i++ {
			if probe.got[i] != probe.got[i-1]+1 {
				t.Errorf("order broken at %d", i)
				break
			}
		}
	})
	diags := app.Diagnostics("broadcast:drop")
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %d, want one per burst", len(diags))
	}
	if diags[0].Metadata["dropped"] != 44 {
		t.Fatalf("metadata = %v", diags[0].Metadata)
	}

	release2 := block(t, app.S)
	for i := range 300 {
		b.Send(i)
	}
	release2()
	app.Sync()
	if n := len(app.Diagnostics("broadcast:drop")); n != 2 {
		t.Fatalf("second burst reported %d total diagnostics, want 2", n)
	}
}

func TestBroadcastUnderFullLoadAccountsForEveryEvent(t *testing.T) {
	b := shared.NewBroadcast[int]()
	const sessions, senders, per = 4, 8, 300
	probes := make([]*broadcastProbe, sessions)
	apps := make([]*runtimetest.App, sessions)
	for i := range apps {
		probes[i] = &broadcastProbe{}
		apps[i] = broadcastApp(t, b, probes[i])
	}
	var wg sync.WaitGroup
	for g := range senders {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range per {
				b.Send(g*per + i)
			}
		}()
	}
	wg.Wait()
	for i, a := range apps {
		a.Sync()
		dropped := 0
		for _, d := range a.Diagnostics("broadcast:drop") {
			dropped += d.Metadata["dropped"].(int)
		}
		var received int
		a.On(func() { received = len(probes[i].got) })
		if received+dropped != senders*per {
			t.Fatalf("session %d: received %d + dropped %d != %d", i, received, dropped, senders*per)
		}
	}
}

func TestBroadcastHandlerPanicReachesTheBoundary(t *testing.T) {
	b := shared.NewBroadcast[int]()
	var boundary *runtime.Boundary
	var mounted *runtime.Signal[bool]
	broken := runtime.Component(func(ctx *Ctx) Node {
		runtime.UseBroadcast(ctx, b, func(v int) {
			if v < 0 {
				panic("broadcast boom")
			}
		})
		return el("p", txt("ok"))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		mounted = runtime.UseSignal(ctx, true)
		boundary = runtime.UseErrorBoundary(ctx)
		failed := runtime.UseComputed(ctx, func() bool { return boundary.Errors().Get().HasErrors() })
		return el("main", runtime.Show(failed,
			func() Node { return el("aside", txt("failed")) },
			runtime.Else(func() Node { return runtime.Show(mounted, func() Node { return broken() }) })))
	}, runtimetest.Dev(false))
	b.Send(1)
	app.Sync()
	if got := app.Normalized(); strings.Contains(got, "failed") {
		t.Fatalf("a healthy event failed the boundary: %s", got)
	}
	b.Send(-1)
	app.Sync()
	if got := app.Normalized(); !strings.Contains(got, "<aside>failed</aside>") {
		t.Fatalf("fallback not shown: %s", got)
	}
}

func TestNoDeliveryAfterUnmount(t *testing.T) {
	sh := shared.NewShared(1)
	b := shared.NewBroadcast[int]()
	var mounted *runtime.Signal[bool]
	var calls atomic.Int64
	child := runtime.Component(func(ctx *Ctx) Node {
		v := runtime.UseShared(ctx, sh)
		runtime.UseBroadcast(ctx, b, func(int) { calls.Add(1) })
		return el("p", bound(v))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		mounted = runtime.UseSignal(ctx, true)
		return el("main", runtime.Show(mounted, func() Node { return child() }))
	})
	b.Send(1)
	app.Sync()
	if calls.Load() != 1 || sh.Subscribers() != 1 || b.Subscribers() != 1 {
		t.Fatalf("calls=%d shared=%d broadcast=%d", calls.Load(), sh.Subscribers(), b.Subscribers())
	}
	app.Do(func() { mounted.Set(false) })
	if sh.Subscribers() != 0 || b.Subscribers() != 0 {
		t.Fatalf("still subscribed after unmount: shared=%d broadcast=%d", sh.Subscribers(), b.Subscribers())
	}
	var externals int
	app.On(func() { externals = runtime.ExternalCount(app.S) })
	if externals != 0 {
		t.Fatalf("session still tracks %d external cells", externals)
	}
	b.Send(2)
	sh.Set(2)
	app.Sync()
	if calls.Load() != 1 {
		t.Fatalf("handler ran %d times, want 1", calls.Load())
	}
	app.Do(func() { mounted.Set(true) })
	if got := app.Normalized(); !strings.Contains(got, "<p>2</p>") {
		t.Fatalf("remount did not start from the latest value: %s", got)
	}
}

func TestNoDeliveryAfterSessionEnds(t *testing.T) {
	sh := shared.NewShared(1)
	b := shared.NewBroadcast[int]()
	probe := &broadcastProbe{}
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		v := runtime.UseShared(ctx, sh)
		runtime.UseBroadcast(ctx, b, func(n int) { probe.got = append(probe.got, n) })
		return el("p", bound(v))
	})
	if sh.Subscribers() != 1 || b.Subscribers() != 1 {
		t.Fatalf("shared=%d broadcast=%d", sh.Subscribers(), b.Subscribers())
	}
	app.S.Close()
	if sh.Subscribers() != 0 || b.Subscribers() != 0 {
		t.Fatalf("still subscribed after close: shared=%d broadcast=%d", sh.Subscribers(), b.Subscribers())
	}
	b.Send(1)
	sh.Set(2)
	if len(probe.got) != 0 {
		t.Fatalf("handler ran after close: %v", probe.got)
	}
}

func TestFamilyKeyFreedWhenLastSessionLeaves(t *testing.T) {
	fam := shared.NewSharedFamily(func(k string) int { return len(k) })
	mk := func(key string) *runtimetest.App {
		return runtimetest.New(t, func(ctx *Ctx) Node {
			return el("p", bound(runtime.UseShared(ctx, fam.Key(key))))
		})
	}
	a, b, c := mk("aa"), mk("aa"), mk("bbb")
	if fam.Len() != 2 {
		t.Fatalf("len = %d", fam.Len())
	}
	fam.Key("aa").Set(50)
	a.Sync()
	b.Sync()
	c.Sync()
	if !strings.Contains(a.Normalized(), "50") || !strings.Contains(b.Normalized(), "50") || !strings.Contains(c.Normalized(), "3") {
		t.Fatalf("a=%s b=%s c=%s", a.Normalized(), b.Normalized(), c.Normalized())
	}
	a.S.Close()
	if fam.Len() != 2 {
		t.Fatalf("len = %d with one subscriber left on aa", fam.Len())
	}
	b.S.Close()
	if fam.Len() != 1 {
		t.Fatalf("len = %d, aa should be freed", fam.Len())
	}
	fam.Key("aa").Set(77)
	if got := fam.Key("aa").Get(); got != 2 {
		t.Fatalf("write to a freed key leaked: %d", got)
	}
	c.S.Close()
	if fam.Len() != 0 {
		t.Fatalf("len = %d", fam.Len())
	}
}

func TestSerializeCreatesNoSubscription(t *testing.T) {
	sh := shared.NewShared(1)
	b := shared.NewBroadcast[int]()
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		v := runtime.UseShared(ctx, sh)
		runtime.UseBroadcast(ctx, b, func(int) {})
		return el("p", bound(v))
	})
	before := runtime.GraphCounts(app.S)
	for range 5 {
		app.CheckReset()
		app.Do(func() { sh.Set(sh.Get() + 1) })
	}
	app.On(func() {
		if _, err := app.S.ResetStep(); err != nil {
			t.Error(err)
		}
	})
	after := runtime.GraphCounts(app.S)
	if sh.Subscribers() != 1 || b.Subscribers() != 1 {
		t.Fatalf("shared=%d broadcast=%d", sh.Subscribers(), b.Subscribers())
	}
	var externals int
	app.On(func() { externals = runtime.ExternalCount(app.S) })
	if externals != 1 {
		t.Fatalf("externals = %d", externals)
	}
	if before.Handlers != after.Handlers || before.Nodes != after.Nodes {
		t.Fatalf("serialize created graph state: before %+v after %+v", before, after)
	}
}

func TestSharedChangedBeforeStartIsCaughtUp(t *testing.T) {
	sh := shared.NewShared(1)
	s := runtime.NewSession(runtime.Config{SessionID: "late", DevMode: true, Root: func(ctx *Ctx) Node {
		return el("p", bound(runtime.UseShared(ctx, sh)))
	}})
	if _, err := s.RenderSnapshot(); err != nil {
		t.Fatal(err)
	}
	sh.Set(9)
	s.Start()
	defer s.Close()
	var html string
	err := s.Ask(5*time.Second, func() error {
		if err := s.SettleIfNeeded(); err != nil {
			return err
		}
		snap, err := s.RenderSnapshot()
		html = snap.HTML
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(html, "9") {
		t.Fatalf("mirror stayed stale across Start: %s", html)
	}
}

func TestThousandSessionsLeaveNothingBehind(t *testing.T) {
	sh := shared.NewShared(0)
	b := shared.NewBroadcast[int]()
	const n = 1000
	refs := make([]weak.Pointer[runtime.Session], 0, n)
	sessions := make([]*runtime.Session, 0, n)
	for i := range n {
		s := runtime.NewSession(runtime.Config{SessionID: fmt.Sprint(i), Root: func(ctx *Ctx) Node {
			v := runtime.UseShared(ctx, sh)
			runtime.UseBroadcast(ctx, b, func(int) {})
			return el("p", bound(v))
		}})
		s.Start()
		if err := s.Ask(5*time.Second, func() error { _, err := s.RenderSnapshot(); return err }); err != nil {
			t.Fatal(err)
		}
		sessions = append(sessions, s)
		refs = append(refs, weak.Make(s))
	}
	if sh.Subscribers() != n || b.Subscribers() != n {
		t.Fatalf("shared=%d broadcast=%d", sh.Subscribers(), b.Subscribers())
	}
	sh.Set(1)
	b.Send(1)
	for _, s := range sessions {
		s.Close()
	}
	sessions = nil
	if sh.Subscribers() != 0 || b.Subscribers() != 0 {
		t.Fatalf("shared=%d broadcast=%d after close", sh.Subscribers(), b.Subscribers())
	}
	for range 3 {
		goruntime.GC()
	}
	alive := 0
	for _, r := range refs {
		if r.Value() != nil {
			alive++
		}
	}
	if alive != 0 {
		t.Fatalf("%d sessions still reachable after close", alive)
	}
}
