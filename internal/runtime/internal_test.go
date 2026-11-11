package runtime

import (
	"context"
	goruntime "runtime"
	"runtime/pprof"
	"slices"
	"strconv"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/work"
)

const retainedSamples = 7

func retainedOnce(build func() *Session) int64 {
	var m0, m1 goruntime.MemStats
	for i := 0; i < 3; i++ {
		goruntime.GC()
	}
	goruntime.ReadMemStats(&m0)
	s := build()
	for i := 0; i < 3; i++ {
		goruntime.GC()
	}
	goruntime.ReadMemStats(&m1)
	goruntime.KeepAlive(s)
	return int64(m1.HeapAlloc) - int64(m0.HeapAlloc)
}

func retainedBytes(t *testing.T, build func() *Session) int64 {
	t.Helper()
	samples := make([]int64, retainedSamples)
	for i := range samples {
		samples[i] = retainedOnce(build)
	}
	slices.Sort(samples)
	return samples[len(samples)/2]
}

func feedSession(n int) *Session {
	items := make([]int, n)
	for i := range items {
		items[i] = i + 1
	}
	s := NewSession(Config{Root: func(ctx *Ctx) work.Node {
		f := NewFeed(func(i int) int { return i }, func() []int { return items })
		return work.BuildElement("ul", For(f, PropsComponent(func(_ *Ctx, item ReadSignal[int]) work.Node {
			return work.BuildElement("li", work.NewText("row"))
		})))
	}})
	if _, err := s.RenderSnapshot(); err != nil {
		panic(err)
	}
	items = nil
	return s
}

func TestFeedRetainsNothingPerStaticRow(t *testing.T) {
	small := retainedBytes(t, func() *Session { return feedSession(100) })
	large := retainedBytes(t, func() *Session { return feedSession(5100) })
	perRow := (large - small) / 5000
	t.Logf("retained per static feed row: %d bytes", perRow)
	if perRow > 100 {
		t.Fatalf("retained %d bytes per static feed row, want <= 100", perRow)
	}
}

func TestStaticFeedRowsDropTheirOwners(t *testing.T) {
	s := feedSession(3)
	reg := s.root.Regions[0].(*feedRegion[int, int])
	for _, r := range reg.rows {
		if r.o != nil {
			t.Fatalf("row %d kept its owner", r.key)
		}
	}
}

type pushedMsg struct {
	ID     int
	Author string
	Text   string
	At     time.Time
}

func pushedFeedSession(n int, dev bool) *Session {
	var live *Feed[pushedMsg, int]
	s := NewSession(Config{DevMode: dev, Root: func(ctx *Ctx) work.Node {
		f := NewFeed(func(m pushedMsg) int { return m.ID }, func() []pushedMsg { return nil })
		if live == nil {
			live = f
		}
		return work.BuildElement("ul", For(f, PropsComponent(func(_ *Ctx, item ReadSignal[pushedMsg]) work.Node {
			return work.BuildElement("li", work.NewText("row"))
		})))
	}})
	if _, err := s.RenderSnapshot(); err != nil {
		panic(err)
	}
	for i := 1; i <= n; i++ {
		live.Append(pushedMsg{ID: i, Author: "alice", Text: "hello", At: time.Unix(int64(i), 0)})
	}
	s.edits = nil
	return s
}

func TestFeedRetainsNoItemPerPushedRowInProdAndLittleInDev(t *testing.T) {
	for _, tc := range []struct {
		name  string
		dev   bool
		limit int64
	}{{"prod", false, 100}, {"dev", true, 150}} {
		small := retainedBytes(t, func() *Session { return pushedFeedSession(100, tc.dev) })
		large := retainedBytes(t, func() *Session { return pushedFeedSession(5100, tc.dev) })
		perRow := (large - small) / 5000
		t.Logf("%s: retained per pushed feed row: %d bytes", tc.name, perRow)
		if perRow > tc.limit {
			t.Fatalf("%s: retained %d bytes per pushed feed row, want <= %d", tc.name, perRow, tc.limit)
		}
	}
}

func TestTheRenderEffectScanFindsEveryLabelledGoroutineStartedJustBeforeIt(t *testing.T) {
	const rounds = 1500
	missed := 0
	baseline := goruntime.NumGoroutine()
	for i := 0; i < rounds; i++ {
		for goruntime.NumGoroutine() > baseline {
			goruntime.Gosched()
		}
		block := make(chan struct{})
		session := "scan-probe-" + strconv.Itoa(i)
		labels := pprof.Labels(renderEffectLabel, renderEffectPrefix+"probe", renderEffectSession, session)
		pprof.Do(context.Background(), labels, func(context.Context) { go func() { <-block }() })
		found := scanRenderEffectGoroutines(session)
		close(block)
		if len(found) != 1 || found[0].component != "probe" || found[0].created == "" {
			missed++
		}
	}
	if missed != 0 {
		t.Fatalf("the scan lost or half-described a leaked goroutine in %d of %d rounds", missed, rounds)
	}
}
