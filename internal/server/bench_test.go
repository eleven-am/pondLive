package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/session"

	pl "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

const benchAbandonedRequests = 1000

func benchPage(ctx *pl.Ctx) work.Node {
	count := pl.UseSignal(ctx, 0)
	rows := make([]work.Item, 0, 20)
	for i := 0; i < 20; i++ {
		rows = append(rows, work.BuildElement("li",
			work.Attribute("class", "row"),
			work.On("click", func(work.Event) { count.Update(func(n int) int { return n + 1 }) }),
			work.NewText("row "+strconv.Itoa(i)),
		))
	}
	return work.BuildElement("html",
		work.BuildElement("head"),
		work.BuildElement("body",
			work.BuildElement("ul", rows...),
			work.BuildElement("span", work.TextOf(count)),
		),
	)
}

func benchHeapInUse() (heap, stack uint64) {
	runtime.GC()
	runtime.GC()
	var ms runtime.MemStats
	runtime.ReadMemStats(&ms)
	return ms.HeapAlloc, ms.StackInuse
}

func BenchmarkAbandonedSSRRetainedHeapCounter20RowsHandlers(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		app, err := New(Config{Component: benchPage})
		if err != nil {
			b.Fatal(err)
		}
		handler := app.Handler()
		warm := httptest.NewRecorder()
		handler.ServeHTTP(warm, httptest.NewRequest(http.MethodGet, "/", nil))
		if warm.Code != http.StatusOK {
			b.Fatalf("warmup status %d", warm.Code)
		}
		before, stackBefore := benchHeapInUse()
		b.StartTimer()
		for j := 0; j < benchAbandonedRequests; j++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if rec.Code != http.StatusOK {
				b.Fatalf("status %d", rec.Code)
			}
		}
		b.StopTimer()
		after, stackAfter := benchHeapInUse()
		retained := int64(after) - int64(before)
		stackGrown := int64(stackAfter) - int64(stackBefore)
		b.ReportMetric(float64(retained)/float64(benchAbandonedRequests), "retained-B/req")
		b.ReportMetric(float64(stackGrown)/float64(benchAbandonedRequests), "stack-B/req")
		b.ReportMetric(float64(retained+stackGrown)/float64(benchAbandonedRequests), "heap+stack-B/req")
		shutdownApp(b, app)
	}
}

func shutdownApp(b *testing.B, app *App) {
	b.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		b.Fatal(err)
	}
}

type benchClock struct {
	mu     sync.Mutex
	now    time.Duration
	timers []*benchTimer
}

type benchTimer struct {
	clock   *benchClock
	at      time.Duration
	f       func()
	stopped bool
	fired   bool
}

func (c *benchClock) AfterFunc(d time.Duration, f func()) session.Timer {
	c.mu.Lock()
	defer c.mu.Unlock()
	t := &benchTimer{clock: c, at: c.now + d, f: f}
	c.timers = append(c.timers, t)
	return t
}

func (t *benchTimer) Stop() bool {
	t.clock.mu.Lock()
	defer t.clock.mu.Unlock()
	was := !t.stopped && !t.fired
	t.stopped = true
	return was
}

func (c *benchClock) Advance(d time.Duration) {
	c.mu.Lock()
	c.now += d
	var due []*benchTimer
	kept := c.timers[:0]
	for _, t := range c.timers {
		switch {
		case t.stopped || t.fired:
		case t.at <= c.now:
			t.fired = true
			due = append(due, t)
		default:
			kept = append(kept, t)
		}
	}
	clear(c.timers[len(kept):])
	c.timers = kept
	c.mu.Unlock()
	for _, t := range due {
		t.f()
	}
}

func BenchmarkAbandonedSSRAfterTTL(b *testing.B) {
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		clock := &benchClock{}
		app, err := New(Config{Component: benchPage, SessionConfig: &session.Config{Clock: clock}})
		if err != nil {
			b.Fatal(err)
		}
		handler := app.Handler()
		warm := httptest.NewRecorder()
		handler.ServeHTTP(warm, httptest.NewRequest(http.MethodGet, "/", nil))
		if warm.Code != http.StatusOK {
			b.Fatalf("warmup status %d", warm.Code)
		}
		clock.Advance(DefaultTokenTTL)
		waitRegistryEmpty(b, app)
		before, stackBefore := benchHeapInUse()
		b.StartTimer()
		for j := 0; j < benchAbandonedRequests; j++ {
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
			if rec.Code != http.StatusOK {
				b.Fatalf("status %d", rec.Code)
			}
		}
		b.StopTimer()
		if got := app.registry.Len(); got != benchAbandonedRequests {
			b.Fatalf("%d sessions registered before expiry, want %d", got, benchAbandonedRequests)
		}
		clock.Advance(DefaultTokenTTL)
		waitRegistryEmpty(b, app)
		clock.Advance(0)
		after, stackAfter := benchHeapInUse()
		retained := int64(after) - int64(before) + int64(stackAfter) - int64(stackBefore)
		b.ReportMetric(float64(retained)/float64(benchAbandonedRequests), "heap+stack-B/req-after-ttl")
		shutdownApp(b, app)
	}
}

func waitRegistryEmpty(b *testing.B, app *App) {
	b.Helper()
	deadline := time.Now().Add(30 * time.Second)
	for app.registry.Len() != 0 {
		if time.Now().After(deadline) {
			b.Fatalf("%d sessions still registered after the TTL", app.registry.Len())
		}
		time.Sleep(time.Millisecond)
	}
}
