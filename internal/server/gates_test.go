package server

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	goruntime "runtime"
	"slices"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"

	pl "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/work"
)

const gateSamples = 5

const classUpdateAllocGate = 25

type countingTransport struct {
	frames atomic.Int64
	bytes  atomic.Int64
	closed atomic.Bool
}

func (c *countingTransport) SendFrame(frame []byte) error {
	c.frames.Add(1)
	c.bytes.Add(int64(len(frame)))
	return nil
}

func (c *countingTransport) Closed() bool { return c.closed.Load() }

func (c *countingTransport) Terminate(websocket.StatusCode, string) { c.closed.Store(true) }

func gatePage(rows int) session.Component {
	return func(ctx *pl.Ctx) work.Node {
		list := pl.UseSignal(ctx, func() []int {
			out := make([]int, rows)
			for i := range out {
				out[i] = i + 1
			}
			return out
		}())
		row := pl.PropsComponent(func(ctx *pl.Ctx, item pl.ReadSignal[int]) work.Node {
			n := item.Peek()
			return work.BuildElement("div", work.Class("row", "item"),
				work.BuildElement("span", work.TextOf(n)),
				work.BuildElement("p", work.NewText("static text for the row")),
				work.BuildElement("button", work.On("click", func(work.Event) {}), work.NewText("go")),
				work.BuildElement("i", work.Attribute("data-n", n)),
			)
		})
		return work.BuildElement("main", pl.For(pl.Items(list, func(n int) int { return n }), row))
	}
}

func TestGateInitialPayloadAtFiveThousandNodes(t *testing.T) {
	app, err := New(Config{Component: gatePage(5000 / 8)})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { shutdownTest(t, app) })
	measure := func() (document, boot int) {
		rec := httptest.NewRecorder()
		app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		body := rec.Body.String()
		const open = `<script id="live-boot" type="application/json">`
		start := strings.Index(body, open)
		if start < 0 {
			t.Fatalf("no boot script in the document")
		}
		end := strings.Index(body[start:], "</script>")
		return len(body) - (end + len("</script>")), end - len(open)
	}
	document, boot := measure()
	for i := 0; i < 3; i++ {
		if d, b := measure(); d != document || b != boot {
			t.Fatalf("payload size is not deterministic: %d/%d then %d/%d", document, boot, d, b)
		}
	}
	body := document + boot
	t.Logf("initial payload at 5000 DOM nodes: %d B HTML, %d B boot JSON, %d B total", document, boot, body)
	if document > 115_000 {
		t.Errorf("HTML is %d B, gate 115000", document)
	}
	if boot > 1000 {
		t.Errorf("boot JSON is %d B, gate 1000", boot)
	}
}

func shutdownTest(t testing.TB, app *App) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := app.Shutdown(ctx); err != nil {
		t.Error(err)
	}
}

func bindingPage(n int, sigs *[]*pl.Signal[int]) session.Component {
	return func(ctx *pl.Ctx) work.Node {
		kids := make([]work.Item, n)
		out := make([]*pl.Signal[int], n)
		for i := range kids {
			out[i] = pl.UseSignal(ctx, 0)
			kids[i] = work.BuildElement("span", work.TextOf(out[i]))
		}
		*sigs = out
		return work.BuildElement("div", kids...)
	}
}

func startBindingSession(tb testing.TB, n int) (*session.LiveSession, *countingTransport, []*pl.Signal[int]) {
	tb.Helper()
	var sigs []*pl.Signal[int]
	ls := session.NewLiveSession("gate", 1, bindingPage(n, &sigs), nil, session.WithRequest(httptest.NewRequest(http.MethodGet, "/", nil)))
	tb.Cleanup(func() { _ = ls.Close() })
	if _, err := ls.Render(context.Background()); err != nil {
		tb.Fatal(err)
	}
	tr := &countingTransport{}
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		tb.Fatal(err)
	}
	return ls, tr, sigs
}

func BenchmarkLeafUpdateToFrame(b *testing.B) {
	for _, n := range []int{10, 100, 1000, 5000} {
		b.Run(itoa(n), func(b *testing.B) {
			ls, tr, sigs := startBindingSession(b, n)
			rt := ls.Session()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sig := sigs[i%n]
				v := i + 1
				if err := rt.Ask(5*time.Second, func() error { sig.Set(v); return nil }); err != nil {
					b.Fatal(err)
				}
			}
			b.StopTimer()
			awaitFrames(b, tr, int64(b.N))
		})
	}
}

func awaitFrames(tb testing.TB, tr *countingTransport, updates int64) {
	tb.Helper()
	if updates == 0 {
		return
	}
	deadline := time.Now().Add(5 * time.Second)
	last := int64(-1)
	for {
		got := tr.frames.Load()
		if got == last && got > 0 {
			if got < updates/2 {
				tb.Fatalf("%d frames for %d updates", got, updates)
			}
			return
		}
		if time.Now().After(deadline) {
			tb.Fatalf("frames never settled: %d for %d updates", got, updates)
		}
		last = got
		time.Sleep(5 * time.Millisecond)
	}
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for ; n > 0; n /= 10 {
		digits = append(digits, byte('0'+n%10))
	}
	slices.Reverse(digits)
	return string(digits)
}

func TestGateLeafUpdateToFrame(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation gates are measured without -race, whose instrumentation inflates allocated bytes")
	}
	const iterations = 400
	var counts []int64
	for _, n := range []int{10, 100, 1000, 5000} {
		ls, tr, sigs := startBindingSession(t, n)
		rt := ls.Session()
		step := 0
		update := func() {
			step++
			sig, v := sigs[step%n], step
			if err := rt.Ask(5*time.Second, func() error { sig.Set(v); return nil }); err != nil {
				t.Fatal(err)
			}
		}
		allocSamples := make([]int64, gateSamples)
		byteSamples := make([]int64, gateSamples)
		for s := range allocSamples {
			awaitFrames(t, tr, int64(step))
			goruntime.GC()
			var m0, m1 goruntime.MemStats
			goruntime.ReadMemStats(&m0)
			for i := 0; i < iterations; i++ {
				update()
			}
			awaitFrames(t, tr, int64(step))
			goruntime.ReadMemStats(&m1)
			allocSamples[s] = int64(m1.Mallocs-m0.Mallocs) / iterations
			byteSamples[s] = int64(m1.TotalAlloc-m0.TotalAlloc) / iterations
		}
		slices.Sort(allocSamples)
		slices.Sort(byteSamples)
		allocs, bytes := allocSamples[gateSamples/2], byteSamples[gateSamples/2]
		t.Logf("update to frame at %d bindings: %d allocs/op, %d B/op", n, allocs, bytes)
		if allocs > 50 || bytes > 4096 {
			t.Errorf("update to frame at %d bindings is %d allocs and %d B, gate 50 and 4096", n, allocs, bytes)
		}
		counts = append(counts, allocs)
	}
	if counts[3]-counts[0] > 2 {
		t.Errorf("update to frame allocs at 5000 are more than 2 above the 10 case: %v", counts)
	}
}

func TestGateClientBundleSize(t *testing.T) {
	const gate = 39119
	info, err := os.Stat(filepath.Join("static", "pondlive.js"))
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("client bundle: %d B (gate %d)", info.Size(), gate)
	if info.Size() > gate {
		t.Errorf("client bundle is %d B, gate %d", info.Size(), gate)
	}
}

func classPage(n int, sigs *[]*pl.Signal[string]) session.Component {
	return func(ctx *pl.Ctx) work.Node {
		kids := make([]work.Item, n)
		out := make([]*pl.Signal[string], n)
		for i := range kids {
			out[i] = pl.UseSignal(ctx, "px-4 bg-red-500")
			kids[i] = work.BuildElement("span", work.Class("p-2 text-sm", out[i]), work.NewText("x"))
		}
		*sigs = out
		return work.BuildElement("div", kids...)
	}
}

func TestGateClassUpdateToFrame(t *testing.T) {
	if raceEnabled {
		t.Skip("allocation gates are measured without -race, whose instrumentation inflates allocated bytes")
	}
	const iterations = 400
	states := []string{"px-8 bg-blue-500", "px-4 bg-red-500"}
	for _, n := range []int{100, 1000} {
		var sigs []*pl.Signal[string]
		ls := session.NewLiveSession("gate", 1, classPage(n, &sigs), nil, session.WithRequest(httptest.NewRequest(http.MethodGet, "/", nil)))
		t.Cleanup(func() { _ = ls.Close() })
		if _, err := ls.Render(context.Background()); err != nil {
			t.Fatal(err)
		}
		tr := &countingTransport{}
		if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
			t.Fatal(err)
		}
		rt := ls.Session()
		step := 0
		update := func() {
			step++
			sig, v := sigs[step%n], states[(step/n)%2]
			if err := rt.Ask(5*time.Second, func() error { sig.Set(v); return nil }); err != nil {
				t.Fatal(err)
			}
		}
		for i := 0; i < 2*n; i++ {
			update()
		}
		allocSamples := make([]int64, gateSamples)
		for s := range allocSamples {
			awaitFrames(t, tr, int64(step))
			goruntime.GC()
			var m0, m1 goruntime.MemStats
			goruntime.ReadMemStats(&m0)
			for i := 0; i < iterations; i++ {
				update()
			}
			awaitFrames(t, tr, int64(step))
			goruntime.ReadMemStats(&m1)
			allocSamples[s] = int64(m1.Mallocs-m0.Mallocs) / iterations
		}
		slices.Sort(allocSamples)
		allocs := allocSamples[gateSamples/2]
		t.Logf("merged class update to frame at %d bindings: %d allocs/op", n, allocs)
		if allocs > classUpdateAllocGate {
			t.Errorf("merged class update at %d bindings is %d allocs, gate %d", n, allocs, classUpdateAllocGate)
		}
	}
}
