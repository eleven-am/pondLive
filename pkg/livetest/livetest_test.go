package livetest_test

import (
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/pkg"
	"github.com/eleven-am/pondlive/pkg/livetest"
)

type recorder struct {
	testing.TB
	errs []string
}

func (r *recorder) Helper() {}

func (r *recorder) Errorf(format string, args ...any) {
	r.errs = append(r.errs, strings.TrimSpace(fmt.Sprintf(format, args...)))
}

func (r *recorder) Fatalf(format string, args ...any) {
	r.Errorf(format, args...)
}

func TestMain(m *testing.M) { goleak.VerifyTestMain(m) }

func TestAPageThatStartsNothingPasses(t *testing.T) {
	var ran atomic.Int32
	livetest.AssertNothingStartsBeforeConnect(t, pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		pkg.UseEffect(ctx, func() func() { ran.Add(1); return nil })
		count := pkg.UseSignal(ctx, 0)
		return pkg.Div(pkg.Text(count))
	}))
	if ran.Load() != 0 {
		t.Fatalf("effect ran %d times without a socket", ran.Load())
	}
}

func TestAGoroutineStartedWhileRenderingIsReported(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	rec := &recorder{TB: t}
	livetest.AssertNothingStartsBeforeConnect(rec, pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		go func() { <-stop }()
		return pkg.Div(pkg.Text("x"))
	}))
	if len(rec.errs) == 0 || !strings.Contains(strings.Join(rec.errs, "\n"), "started while rendering") {
		t.Fatalf("errors = %v", rec.errs)
	}
}

func TestBackgroundWorkStartedInTheBodyThroughATickerIsReported(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	rec := &recorder{TB: t}
	livetest.AssertNothingStartsBeforeConnect(rec, pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		go func() {
			tick := time.NewTicker(time.Millisecond)
			defer tick.Stop()
			for {
				select {
				case <-tick.C:
				case <-stop:
					return
				}
			}
		}()
		return pkg.Div(pkg.Text("x"))
	}))
	if len(rec.errs) == 0 {
		t.Fatal("a body that starts a ticker goroutine passed")
	}
}

func TestAnEffectThatRunsDuringTheRequestIsReported(t *testing.T) {
	rec := &recorder{TB: t}
	livetest.AssertNothingStartsBeforeConnect(rec, pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		livetest.Mount(t, pkg.Component(func(inner *pkg.Ctx) pkg.Node {
			pkg.UseEffect(inner, func() func() { return nil })
			return pkg.Div()
		}))
		return pkg.Div(pkg.Text("x"))
	}))
	if len(rec.errs) == 0 || !strings.Contains(strings.Join(rec.errs, "\n"), "UseEffect callbacks ran") {
		t.Fatalf("errors = %v", rec.errs)
	}
}

func TestAGoroutineStartedByCleanupDuringShutdownIsReportedAsSurviving(t *testing.T) {
	stop := make(chan struct{})
	defer close(stop)
	rec := &recorder{TB: t}
	livetest.AssertNothingStartsBeforeConnect(rec, pkg.Component(func(ctx *pkg.Ctx) pkg.Node {
		pkg.UseRenderEffect(ctx, func() func() {
			return func() { go func() { <-stop }() }
		})
		return pkg.Div(pkg.Text("x"))
	}), pkg.WithDevMode())
	if len(rec.errs) == 0 || !strings.Contains(strings.Join(rec.errs, "\n"), "survived shutdown") {
		t.Fatalf("errors = %v", rec.errs)
	}
}
