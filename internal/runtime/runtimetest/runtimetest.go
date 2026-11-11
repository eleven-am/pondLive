package runtimetest

import (
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

const closeBound = 5 * time.Second

func Start(t testing.TB, s *runtime.Session) *runtime.Session {
	t.Helper()
	s.Start()
	t.Cleanup(func() {
		done := make(chan struct{})
		go func() {
			defer close(done)
			s.Close()
		}()
		select {
		case <-done:
		case <-time.After(closeBound):
			t.Errorf("session did not close within %v", closeBound)
		}
	})
	return s
}

type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*fakeTimer
}

type fakeTimer struct {
	at      time.Time
	fn      func()
	stopped bool
	fired   bool
}

func (t *fakeTimer) Stop() bool {
	was := !t.stopped && !t.fired
	t.stopped = true
	return was
}

func NewFakeClock() *FakeClock { return &FakeClock{now: time.Unix(1_700_000_000, 0)} }

func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FakeClock) AfterFunc(d time.Duration, f func()) runtime.Stopper {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &fakeTimer{at: c.now.Add(d), fn: f}
	c.timers = append(c.timers, t)
	return t
}

func (c *FakeClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)
	var due []*fakeTimer
	for _, t := range c.timers {
		if !t.stopped && !t.fired && !t.at.After(c.now) {
			t.fired = true
			due = append(due, t)
		}
	}
	c.mu.Unlock()
	for _, t := range due {
		t.fn()
	}
}

func (c *FakeClock) Pending() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, t := range c.timers {
		if !t.stopped && !t.fired {
			n++
		}
	}
	return n
}

type App struct {
	T      testing.TB
	S      *runtime.Session
	Clock  *FakeClock
	Model  *Model
	Frames [][]protocol.Edit
	Diags  []protocol.Diagnostic
	SSR    string

	mu      sync.Mutex
	pending [][]protocol.Edit
	diagMu  sync.Mutex
}

type Option func(*runtime.Config)

func Dev(on bool) Option { return func(c *runtime.Config) { c.DevMode = on } }

func New(t testing.TB, root func(*runtime.Ctx) work.Node, opts ...Option) *App {
	t.Helper()
	a := &App{T: t, Clock: NewFakeClock()}
	cfg := runtime.Config{SessionID: "test", Root: root, DevMode: true, Clock: a.Clock}
	for _, o := range opts {
		o(&cfg)
	}
	a.S = runtime.NewSession(cfg)
	a.S.Bus.SubscribeToFrameEdits(func(edits []protocol.Edit) {
		a.mu.Lock()
		a.pending = append(a.pending, edits)
		a.mu.Unlock()
	})
	a.S.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		a.diagMu.Lock()
		a.Diags = append(a.Diags, d)
		a.diagMu.Unlock()
	})
	a.S.DeferEffects()
	Start(t, a.S)
	a.On(func() {
		snap, err := a.S.RenderSnapshot()
		if err != nil {
			t.Fatalf("render: %v", err)
		}
		if snap.Crashed {
			t.Fatalf("session crashed: %v", snap.Crash)
		}
		a.SSR = snap.HTML
	})
	a.drainPending()
	model, err := NewModel(a.SSR)
	if err != nil {
		t.Fatalf("ssr is not indexable: %v\n%s", err, a.SSR)
	}
	a.Model = model
	a.Frames = nil
	a.On(func() { a.S.GoLive() })
	a.Sync()
	return a
}

func (a *App) drainPending() [][]protocol.Edit {
	a.mu.Lock()
	defer a.mu.Unlock()
	out := a.pending
	a.pending = nil
	return out
}

func (a *App) On(fn func()) {
	a.T.Helper()
	if err := a.S.Ask(closeBound, func() error { fn(); return nil }); err != nil {
		a.T.Fatalf("ask: %v", err)
	}
}

func (a *App) Settle() {
	a.T.Helper()
	a.On(func() {
		if err := a.S.SettleIfNeeded(); err != nil {
			a.T.Fatalf("settle: %v", err)
		}
	})
}

func (a *App) Sync() []protocol.Edit {
	a.T.Helper()
	a.Settle()
	var all []protocol.Edit
	for _, frame := range a.drainPending() {
		a.Frames = append(a.Frames, frame)
		if err := a.Model.Apply(frame); err != nil {
			a.T.Fatalf("client rejected frame %v: %v", frame, err)
		}
		all = append(all, frame...)
	}
	a.CheckReset()
	return all
}

func (a *App) Pump() []protocol.Edit {
	a.T.Helper()
	a.Settle()
	var all []protocol.Edit
	for _, frame := range a.drainPending() {
		a.Frames = append(a.Frames, frame)
		if err := a.Model.Apply(frame); err != nil {
			a.T.Fatalf("client rejected frame %v: %v", frame, err)
		}
		all = append(all, frame...)
	}
	return all
}

func (a *App) ResetHTML() string {
	a.T.Helper()
	var html string
	var err error
	a.On(func() { html, err = a.S.ResetStep() })
	if err != nil {
		a.T.Fatalf("reset step: %v", err)
	}
	for _, frame := range a.drainPending() {
		a.Frames = append(a.Frames, frame)
		if err := a.Model.Apply(frame); err != nil {
			a.T.Fatalf("client rejected the resync frame %v: %v", frame, err)
		}
	}
	return html
}

func (a *App) CheckReset() {
	a.T.Helper()
	html := a.ResetHTML()
	want, cerr := Canonical(html)
	if cerr != nil {
		a.T.Fatalf("reset html is not parseable: %v", cerr)
	}
	if got := a.Model.HTML(); got != want {
		a.T.Fatalf("serialize pass differs from the live DOM\nlive:      %s\nserialize: %s", got, want)
	}
}

func (a *App) Do(fn func()) []protocol.Edit {
	a.T.Helper()
	a.On(fn)
	return a.Sync()
}

func (a *App) Dispatch(id uint32, event string, payload map[string]any) error {
	a.T.Helper()
	var err error
	a.On(func() { err = a.S.DispatchHandler(id, event, payload) })
	a.Sync()
	return err
}

func (a *App) Advance(d time.Duration) []protocol.Edit {
	a.T.Helper()
	a.Clock.Advance(d)
	return a.Sync()
}

func (a *App) Fresh() string { return a.Normalized() }

func (a *App) Normalized() string { return a.Model.Normalized() }

func (a *App) Diagnostics(phase string) []protocol.Diagnostic {
	a.diagMu.Lock()
	defer a.diagMu.Unlock()
	var out []protocol.Diagnostic
	for _, d := range a.Diags {
		if d.Phase == phase {
			out = append(out, d)
		}
	}
	return out
}

func FindID(markup, needle string) (uint32, error) {
	m, err := NewModel(markup)
	if err != nil {
		return 0, err
	}
	for _, id := range m.IDs() {
		if n := m.Node(id); n != nil && n.Type == 3 && fmt.Sprint(n.Data) == needle {
			return id, nil
		}
	}
	return 0, fmt.Errorf("no node with data %q", needle)
}
