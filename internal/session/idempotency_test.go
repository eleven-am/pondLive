package session

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
)

func startLiveWith(t *testing.T, comp Component, cfg *Config) (*LiveSession, *recordingTransport) {
	t.Helper()
	ls := NewLiveSession("idem", 1, comp, cfg, WithRequest(httptest.NewRequest(http.MethodGet, "/", nil)))
	t.Cleanup(func() { _ = ls.Close() })
	if _, err := ls.Render(context.Background()); err != nil {
		t.Fatal(err)
	}
	tr := newRecordingTransport()
	if _, err := ls.Attach(context.Background(), tr, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	return ls, tr
}

func clickAt(refID uint32, i int, cs uint64) protocol.ClientEvt {
	evt := clickEvt(refID, i)
	evt.CS = cs
	return evt
}

func logLen(t *testing.T, ls *LiveSession, probe *clickProbe) int {
	t.Helper()
	var n int
	onSession(t, ls, func() { n = len(probe.log) })
	return n
}

func TestDeliverDedupesByClientSeq(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, _ := startLiveWith(t, probe.component, nil)
	var refID uint32
	onSession(t, ls, func() { refID = probe.refID() })
	gen := ls.Generation()

	for _, step := range []struct {
		cs       uint64
		accepted bool
	}{{1, true}, {1, false}, {2, true}, {2, false}, {1, false}, {3, true}} {
		accepted, err := ls.Deliver(gen, clickAt(refID, int(step.cs), step.cs))
		if err != nil || accepted != step.accepted {
			t.Fatalf("cs=%d accepted=%v err=%v, want accepted=%v", step.cs, accepted, err, step.accepted)
		}
	}
	if err := ls.Flush(); err != nil {
		t.Fatal(err)
	}
	if got := logLen(t, ls, probe); got != 3 {
		t.Fatalf("handler ran %d times, want 3", got)
	}
}

func TestDeliverAcceptsAGapAndReportsADiagnostic(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, _ := startLiveWith(t, probe.component, nil)
	var refID uint32
	onSession(t, ls, func() { refID = probe.refID() })

	var mu sync.Mutex
	var diags []protocol.Diagnostic
	sub := ls.Session().Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		mu.Lock()
		diags = append(diags, d)
		mu.Unlock()
	})
	defer sub.Unsubscribe()

	gen := ls.Generation()
	if accepted, err := ls.Deliver(gen, clickAt(refID, 0, 5)); err != nil || !accepted {
		t.Fatalf("gap event accepted=%v err=%v", accepted, err)
	}
	if accepted, _ := ls.Deliver(gen, clickAt(refID, 0, 3)); accepted {
		t.Fatal("an event below the advanced cs must be a duplicate")
	}
	mu.Lock()
	defer mu.Unlock()
	var found bool
	for _, d := range diags {
		if d.Phase == "protocol:client-seq-gap" {
			found = true
			if d.Metadata["from"] != uint64(1) || d.Metadata["to"] != uint64(4) {
				t.Fatalf("gap metadata = %v", d.Metadata)
			}
		}
	}
	if !found {
		t.Fatalf("no client-seq-gap diagnostic in %v", diags)
	}
}

func TestDeliverRejectsAStaleGeneration(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, _ := startLiveWith(t, probe.component, nil)
	var refID uint32
	onSession(t, ls, func() { refID = probe.refID() })

	old := ls.Generation()
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	if ls.Generation() == old {
		t.Fatal("re-attach did not advance the generation")
	}
	accepted, err := ls.Deliver(old, clickAt(refID, 0, 1))
	if accepted || !errors.Is(err, ErrStaleConnection) {
		t.Fatalf("stale deliver accepted=%v err=%v", accepted, err)
	}
	if got := logLen(t, ls, probe); got != 0 {
		t.Fatalf("a stale reader ran %d handlers", got)
	}
	if accepted, err := ls.Deliver(ls.Generation(), clickAt(refID, 0, 1)); err != nil || !accepted {
		t.Fatalf("current generation deliver accepted=%v err=%v", accepted, err)
	}
}

func TestAttachReportsTheAcceptedClientSeq(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, _ := startLiveWith(t, probe.component, nil)
	var refID uint32
	onSession(t, ls, func() { refID = probe.refID() })

	gen := ls.Generation()
	for cs := uint64(1); cs <= 4; cs++ {
		if _, err := ls.Deliver(gen, clickAt(refID, int(cs), cs)); err != nil {
			t.Fatal(err)
		}
	}
	res, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if res.CS != 4 {
		t.Fatalf("joined cs = %d, want 4", res.CS)
	}
}

func TestBacklogDoesNotConsumeTheClientSeq(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, _ := startLiveWith(t, probe.component, nil)
	var refID uint32
	onSession(t, ls, func() { refID = probe.refID() })

	release := make(chan struct{})
	if err := ls.Session().Post(func() { <-release }); err != nil {
		t.Fatal(err)
	}
	gen := ls.Generation()
	var failed uint64
	for cs := uint64(1); cs <= 5000; cs++ {
		accepted, err := ls.Deliver(gen, clickAt(refID, int(cs), cs))
		if errors.Is(err, runtime.ErrClientBacklog) {
			failed = cs
			break
		}
		if err != nil || !accepted {
			close(release)
			t.Fatalf("cs=%d accepted=%v err=%v", cs, accepted, err)
		}
	}
	if failed == 0 {
		close(release)
		t.Fatal("the backlog cap was never reached")
	}
	ls.inMu.Lock()
	last := ls.lastCS
	ls.inMu.Unlock()
	if last != failed-1 {
		close(release)
		t.Fatalf("lastCS = %d after the refusal of %d", last, failed)
	}
	close(release)
	if err := ls.Flush(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for {
		accepted, err := ls.Deliver(gen, clickAt(refID, int(failed), failed))
		if err == nil {
			if !accepted {
				t.Fatal("the refused cs was not accepted on resend")
			}
			break
		}
		if !errors.Is(err, runtime.ErrClientBacklog) || time.Now().After(deadline) {
			t.Fatalf("resend failed: %v", err)
		}
		time.Sleep(time.Millisecond)
	}
}

func TestConcurrentReadersDeliverEachClientSeqExactlyOnce(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, _ := startLiveWith(t, probe.component, nil)
	var refID uint32
	onSession(t, ls, func() { refID = probe.refID() })
	gen := ls.Generation()

	const n = 1000
	var accepted atomic.Int64
	var wg sync.WaitGroup
	for r := 0; r < 2; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for cs := uint64(1); cs <= n; cs++ {
				for {
					ok, err := ls.Deliver(gen, clickAt(refID, int(cs), cs))
					if errors.Is(err, runtime.ErrClientBacklog) {
						time.Sleep(100 * time.Microsecond)
						continue
					}
					if err != nil {
						t.Error(err)
						return
					}
					if ok {
						accepted.Add(1)
					}
					break
				}
			}
		}()
	}
	wg.Wait()
	if err := ls.Flush(); err != nil {
		t.Fatal(err)
	}
	deadline := time.Now().Add(5 * time.Second)
	for logLen(t, ls, probe) < n && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if got := logLen(t, ls, probe); got != n {
		t.Fatalf("handler ran %d times, want %d", got, n)
	}
	if accepted.Load() != n {
		t.Fatalf("accepted %d events, want %d", accepted.Load(), n)
	}
}

func TestClientOriginatedRouterFrameAndDomEventsAreNotEchoed(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, tr := startLiveWith(t, probe.component, nil)
	gen := ls.Generation()
	before := ls.out.headSeq()

	cs := uint64(0)
	send := func(topic protocol.Topic, action string, payload any) {
		cs++
		evt := protocol.ClientEvt{Event: protocol.Event{Type: topic, SID: "idem"}, Action: action, Payload: payload, CS: cs}
		if _, err := ls.Deliver(gen, evt); err != nil {
			t.Fatal(err)
		}
	}
	send(protocol.TopicFrame, "reset", map[string]any{"html": "<html></html>"})
	send(protocol.TopicFrame, "patch", []any{})
	send(protocol.RouteHandler, "popstate", map[string]any{"path": "/x", "query": "", "hash": ""})
	send(protocol.RouteHandler, "push", map[string]any{"path": "/y", "query": "", "hash": ""})
	send(protocol.DOMHandler, "anything", map[string]any{})
	if err := ls.Flush(); err != nil {
		t.Fatal(err)
	}
	onSession(t, ls, func() {})

	if after := ls.out.headSeq(); after != before {
		t.Fatalf("client events took seqs: head %d -> %d, frames %v", before, after, tr.seqs())
	}
	var loc string
	onSession(t, ls, func() { loc = ls.requestState.URL().Path })
	if loc != "/x" {
		t.Fatalf("popstate no longer tracked, location = %q", loc)
	}
}

func resetMessages(tr *recordingTransport) []protocol.Message {
	var out []protocol.Message
	for _, m := range tr.messages() {
		if m.Topic == "frame" && m.Event == "reset" {
			out = append(out, m)
		}
	}
	return out
}

func resync(t *testing.T, ls *LiveSession, bs uint64) {
	t.Helper()
	ls.RequestResync(ls.Generation(), bs, "unresolved")
	onSession(t, ls, func() {})
}

func TestRequestResyncEmitsAResetAndNotesIt(t *testing.T) {
	leakCheck(t)
	var mu sync.Mutex
	var causes []string
	cfg := &Config{OnReset: func(_ *LiveSession, cause string) {
		mu.Lock()
		causes = append(causes, cause)
		mu.Unlock()
	}}
	probe := &clickProbe{}
	ls, tr := startLiveWith(t, probe.component, cfg)

	var diag atomic.Int64
	sub := ls.Session().Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		if d.Phase == "protocol:reset" && d.Metadata["cause"] == protocol.CauseClientRequest {
			diag.Add(1)
		}
	})
	defer sub.Unsubscribe()

	resync(t, ls, ls.out.headSeq())
	resets := resetMessages(tr)
	if len(resets) != 1 {
		t.Fatalf("resets = %d, want 1", len(resets))
	}
	mu.Lock()
	defer mu.Unlock()
	if len(causes) != 1 || causes[0] != protocol.CauseClientRequest {
		t.Fatalf("OnReset causes = %v", causes)
	}
	deadline := time.Now().Add(2 * time.Second)
	for diag.Load() == 0 && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	if diag.Load() != 1 {
		t.Fatalf("protocol:reset diagnostics = %d", diag.Load())
	}
}

func TestRequestResyncFromAStaleGenerationIsIgnored(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, tr := startLiveWith(t, probe.component, nil)
	old := ls.Generation()
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	ls.RequestResync(old, 0, "gap")
	onSession(t, ls, func() {})
	if n := len(resetMessages(tr)); n != 0 {
		t.Fatalf("stale resync produced %d resets", n)
	}
}

func TestRequestResyncIsIgnoredWhileAResetIsInFlight(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, tr := startLiveWith(t, probe.component, nil)

	head := ls.out.headSeq()
	resync(t, ls, head)
	resync(t, ls, head)
	resync(t, ls, head)
	if n := len(resetMessages(tr)); n != 1 {
		t.Fatalf("resets = %d, want 1", n)
	}
	reset := resetMessages(tr)[0]
	resync(t, ls, reset.Seq)
	if n := len(resetMessages(tr)); n != 2 {
		t.Fatalf("a resync from after the reset must be served, resets = %d", n)
	}
}

func TestRequestResyncIsRateLimitedAndCoalesced(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, tr := startLiveWith(t, probe.component, &Config{ResyncInterval: 400 * time.Millisecond})

	resync(t, ls, ls.out.headSeq())
	first := resetMessages(tr)[0]
	resync(t, ls, first.Seq)
	second := resetMessages(tr)[1]
	if n := len(resetMessages(tr)); n != 2 {
		t.Fatalf("burst resets = %d, want 2", n)
	}

	resync(t, ls, second.Seq)
	resync(t, ls, second.Seq)
	resync(t, ls, second.Seq)
	if n := len(resetMessages(tr)); n != 2 {
		t.Fatalf("a limited resync was served immediately, resets = %d", n)
	}
	deadline := time.Now().Add(3 * time.Second)
	for len(resetMessages(tr)) < 3 && time.Now().Before(deadline) {
		time.Sleep(5 * time.Millisecond)
	}
	if n := len(resetMessages(tr)); n != 3 {
		t.Fatalf("deferred reset never arrived, resets = %d", n)
	}
	time.Sleep(600 * time.Millisecond)
	if n := len(resetMessages(tr)); n != 3 {
		t.Fatalf("coalescing failed, resets = %d", n)
	}
}

func TestRequestResyncBurstIsExactlyTwoBeforeLimiting(t *testing.T) {
	leakCheck(t)
	probe := &clickProbe{}
	ls, tr := startLiveWith(t, probe.component, &Config{ResyncInterval: time.Hour})

	resync(t, ls, ls.out.headSeq())
	if n := len(resetMessages(tr)); n != 1 {
		t.Fatalf("first resync resets = %d, want 1", n)
	}
	resync(t, ls, resetMessages(tr)[0].Seq)
	if n := len(resetMessages(tr)); n != 2 {
		t.Fatalf("second resync resets = %d, want 2 (burst must be 2)", n)
	}
	resync(t, ls, resetMessages(tr)[1].Seq)
	resync(t, ls, resetMessages(tr)[1].Seq)
	if n := len(resetMessages(tr)); n != 2 {
		t.Fatalf("third resync was served inside the interval, resets = %d", n)
	}
}

func TestAttachResetIsCountedThroughOnReset(t *testing.T) {
	leakCheck(t)
	var mu sync.Mutex
	var causes []string
	cfg := &Config{OnReset: func(_ *LiveSession, cause string) {
		mu.Lock()
		causes = append(causes, cause)
		mu.Unlock()
	}}
	probe := &clickProbe{}
	ls, _ := startLiveWith(t, probe.component, cfg)
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, ls.out.headSeq(), true); err != nil {
		t.Fatal(err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(causes) != 1 || causes[0] != protocol.CauseRejoinRequest {
		t.Fatalf("OnReset causes = %v", causes)
	}
}

func TestRecoveryDiagnosticsAreLoggedAtWarn(t *testing.T) {
	for _, phase := range []string{"protocol:reset", "protocol:client-seq-gap"} {
		if got := diagnosticLevel(phase); got != slog.LevelWarn {
			t.Fatalf("%s level = %v, want WARN", phase, got)
		}
	}
	for _, phase := range []string{"protocol:encode", "actor:panic", "flush:update-depth"} {
		if got := diagnosticLevel(phase); got != slog.LevelError {
			t.Fatalf("%s level = %v, want ERROR", phase, got)
		}
	}
}
