package session

import (
	"context"
	"encoding/json"
	"errors"
	"math/rand"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/router"
	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func newRing(frames, bytes int) *outbound {
	return &outbound{maxFrames: frames, maxBytes: bytes, maxReset: 1 << 20}
}

func sendN(o *outbound, n int) {
	for i := 0; i < n; i++ {
		o.send("frame", "patch", i)
	}
}

func TestRingAppendAckTrimAndByteAccounting(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := newRing(100, 1<<20)
	sendN(o, 5)
	frames, bytes := o.retained()
	if frames != 5 || bytes <= 0 || o.headSeq() != 5 {
		t.Fatalf("frames=%d bytes=%d head=%d", frames, bytes, o.headSeq())
	}
	if err := o.ack(3); err != nil {
		t.Fatal(err)
	}
	frames, bytes2 := o.retained()
	if frames != 2 || bytes2 >= bytes || bytes2 <= 0 {
		t.Fatalf("after ack(3) frames=%d bytes=%d (was %d)", frames, bytes2, bytes)
	}
	if err := o.ack(2); err != nil {
		t.Fatal(err)
	}
	if f, b := o.retained(); f != 2 || b != bytes2 {
		t.Fatalf("a stale ack changed the ring: %d %d", f, b)
	}
	if err := o.ack(5); err != nil {
		t.Fatal(err)
	}
	if f, b := o.retained(); f != 0 || b != 0 {
		t.Fatalf("a full ack left %d frames %d bytes", f, b)
	}
	if err := o.ack(6); err != ErrAckAhead {
		t.Fatalf("ack beyond head = %v", err)
	}
}

func TestRingEvictsOldestWhileAttached(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := newRing(3, 1<<20)
	tr := newRecordingTransport()
	o.attach(tr, 0)
	sendN(o, 10)
	if got := tr.seqs(); len(got) != 10 {
		t.Fatalf("the attached client must still receive every frame, got %v", got)
	}
	frames, _ := o.retained()
	if frames != 3 {
		t.Fatalf("retained %d frames, want 3", frames)
	}
	if o.ring.Front().seq != 8 {
		t.Fatalf("oldest retained seq = %d, want 8", o.ring.Front().seq)
	}
	if o.discarding {
		t.Fatal("an attached ring must evict, not discard")
	}
}

func TestRingNeverRetainsAFrameAboveTheByteCapAndNextRejoinResets(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := newRing(100, 64)
	tr := newRecordingTransport()
	o.attach(tr, 0)
	o.send("frame", "patch", "a")
	o.send("frame", "patch", strings.Repeat("x", 200))
	if got := tr.seqs(); len(got) != 2 {
		t.Fatalf("the attached client must still receive the oversize frame, got %v", got)
	}
	if f, b := o.retained(); f != 0 || b != 0 {
		t.Fatalf("an oversize frame (or anything before it) stayed in the ring: %d %d", f, b)
	}
	o.detach()
	if p, cause := o.plan(1, false); p != PlanReset || cause != protocol.CauseRejoinGap {
		t.Fatalf("plan = %v %q", p, cause)
	}
	if p, _ := o.plan(2, false); p != PlanCurrent {
		t.Fatalf("a client that applied the oversize frame must stay current, got %v", p)
	}
	o.attach(tr, 0)
	o.send("frame", "patch", "b")
	if f, _ := o.retained(); f != 1 {
		t.Fatalf("frames after the gap must be retained again, got %d", f)
	}
}

func TestDetachedRetainedBytesStayWithinTheCapForAnyFrameSizes(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	const maxBytes = 2048
	o := newRing(50, maxBytes)
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 2000; i++ {
		o.send("frame", "patch", strings.Repeat("x", rng.Intn(100000)))
		if _, b := o.retained(); b > maxBytes {
			t.Fatalf("iteration %d retained %d bytes, cap %d", i, b, maxBytes)
		}
	}
}

func TestRingDiscardsOnOverflowWhileDetached(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := newRing(3, 1<<20)
	sendN(o, 10)
	if f, b := o.retained(); f != 0 || b != 0 {
		t.Fatalf("retained (%d, %d), want (0, 0)", f, b)
	}
	if o.headSeq() != 10 || !o.discarding {
		t.Fatalf("head=%d discarding=%v", o.headSeq(), o.discarding)
	}
	sendN(o, 2)
	if f, _ := o.retained(); f != 0 || o.headSeq() != 12 {
		t.Fatalf("a discarding ring stored frames: %d head %d", f, o.headSeq())
	}
}

func TestRingPlanMatrix(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	t.Run("current", func(t *testing.T) {
		o := newRing(100, 1<<20)
		sendN(o, 4)
		if p, _ := o.plan(4, false); p != PlanCurrent {
			t.Fatalf("plan = %v", p)
		}
	})
	t.Run("replay", func(t *testing.T) {
		o := newRing(100, 1<<20)
		sendN(o, 4)
		for _, bs := range []uint64{0, 2, 3} {
			if p, _ := o.plan(bs, false); p != PlanReplay {
				t.Fatalf("bs=%d plan = %v", bs, p)
			}
		}
	})
	t.Run("gap resets", func(t *testing.T) {
		o := newRing(100, 1<<20)
		sendN(o, 6)
		if err := o.ack(3); err != nil {
			t.Fatal(err)
		}
		if p, cause := o.plan(1, false); p != PlanReset || cause != protocol.CauseRejoinGap {
			t.Fatalf("plan = %v %q", p, cause)
		}
		if p, _ := o.plan(3, false); p != PlanReplay {
			t.Fatalf("plan = %v", p)
		}
	})
	t.Run("client ahead", func(t *testing.T) {
		o := newRing(100, 1<<20)
		sendN(o, 2)
		if p, cause := o.plan(9, false); p != PlanReset || cause != protocol.CauseClientAhead {
			t.Fatalf("plan = %v %q", p, cause)
		}
	})
	t.Run("want reset", func(t *testing.T) {
		o := newRing(100, 1<<20)
		sendN(o, 2)
		if p, cause := o.plan(2, true); p != PlanReset || cause != protocol.CauseRejoinRequest {
			t.Fatalf("plan = %v %q", p, cause)
		}
	})
	t.Run("discarding resets", func(t *testing.T) {
		o := newRing(2, 1<<20)
		sendN(o, 5)
		if p, cause := o.plan(0, false); p != PlanReset || cause != protocol.CauseRejoinGap {
			t.Fatalf("plan = %v %q", p, cause)
		}
	})
	t.Run("reset at the front replays from any earlier bs", func(t *testing.T) {
		o := newRing(100, 1<<20)
		sendN(o, 3)
		if err := o.appendReset(protocol.ResetPayload{HTML: "<html></html>"}); err != nil {
			t.Fatal(err)
		}
		sendN(o, 2)
		for _, bs := range []uint64{0, 1, 2, 3} {
			if p, _ := o.plan(bs, false); p != PlanReplay {
				t.Fatalf("bs=%d plan = %v", bs, p)
			}
		}
		tr := newRecordingTransport()
		o.attach(tr, 1)
		got := tr.seqs()
		if len(got) != 3 || got[0] != 4 || got[2] != 6 {
			t.Fatalf("replay after a reset sent %v", got)
		}
	})
}

func TestRingLoneResetIsPoppedByAckAtOrAboveItsSeq(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := newRing(100, 1<<20)
	sendN(o, 2)
	if err := o.appendReset(protocol.ResetPayload{HTML: "<html></html>"}); err != nil {
		t.Fatal(err)
	}
	if f, _ := o.retained(); f != 1 {
		t.Fatalf("a reset must replace the ring, retained %d", f)
	}
	if err := o.ack(2); err != nil {
		t.Fatal(err)
	}
	if f, _ := o.retained(); f != 1 {
		t.Fatalf("an ack below the reset popped it: %d", f)
	}
	if err := o.ack(3); err != nil {
		t.Fatal(err)
	}
	if f, b := o.retained(); f != 0 || b != 0 {
		t.Fatalf("an acked reset stayed: %d %d", f, b)
	}
}

func TestRingResetLargerThanTheLimitIsRefusedWithoutBurningASeq(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := &outbound{maxFrames: 10, maxBytes: 1 << 20, maxReset: 128}
	sendN(o, 2)
	err := o.appendReset(protocol.ResetPayload{HTML: strings.Repeat("x", 1024)})
	if err != ErrResetTooLarge {
		t.Fatalf("err = %v", err)
	}
	if o.headSeq() != 2 {
		t.Fatalf("head = %d, a refused reset must not consume a seq", o.headSeq())
	}
	if f, _ := o.retained(); f != 2 {
		t.Fatalf("a refused reset disturbed the ring: %d", f)
	}
}

func TestRingSuppressedFramesTakeNoSeq(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := newRing(10, 1<<20)
	o.suppressFrames(true)
	o.send("frame", "patch", 1)
	o.send("router", "push", 1)
	o.suppressFrames(false)
	if o.headSeq() != 1 {
		t.Fatalf("head = %d, want only the non-frame message", o.headSeq())
	}
}

func TestRingShutFreesEverything(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	o := newRing(10, 1<<20)
	sendN(o, 4)
	o.shut()
	if f, b := o.retained(); f != 0 || b != 0 {
		t.Fatalf("retained (%d, %d) after shut", f, b)
	}
	o.send("frame", "patch", 1)
	if o.headSeq() != 4 {
		t.Fatalf("a closed ring took a seq: %d", o.headSeq())
	}
}

type emitter struct {
	bus   *protocol.Bus
	emit  bool
	count int
	kill  *recordingTransport
}

func (e *emitter) component(ctx *rt.Ctx) work.Node {
	n := headers.UseRequest(ctx).Cookie("n")
	rt.UseEffect(ctx, func() func() {
		_ = n.Get()
		if e.emit {
			e.emit = false
			e.count++
			e.bus.Publish(protocol.TopicFrame, "patch", "marker")
			if e.kill != nil {
				e.kill.closed.Store(true)
			}
		}
		return nil
	})
	return &work.Element{Tag: "div"}
}

func TestAbortedReplacementReattachesTheOldTransportWithFramesProducedMeanwhileExactlyOnce(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	e := &emitter{}
	ls := newClocked(t, e.component, &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	e.bus = ls.Session().Bus
	old, joiner := newRecordingTransport(), newRecordingTransport()
	if _, err := ls.Attach(context.Background(), old, cookieHeader(1), 0, false); err != nil {
		t.Fatal(err)
	}
	before := len(old.messages())

	e.emit, e.kill = true, joiner
	if _, err := ls.Attach(context.Background(), joiner, cookieHeader(2), 0, false); err == nil {
		t.Fatal("the joiner died mid attach and must fail")
	}
	barrier(t, ls)

	if old.Closed() {
		t.Fatal("the old transport must survive a failed replacement")
	}
	msgs := old.messages()
	markers := 0
	for i, m := range msgs {
		if m.Seq != uint64(i+1) {
			t.Fatalf("old transport saw non-contiguous seqs %v", old.seqs())
		}
		if m.Data == "marker" {
			markers++
		}
	}
	if markers != 1 || len(msgs) <= before {
		t.Fatalf("markers=%d messages=%d (before %d): frames produced during the failed attach must arrive exactly once", markers, len(msgs), before)
	}
}

type modelClient struct {
	applied uint64
	resets  int
}

func (m *modelClient) consume(t *testing.T, tr *recordingTransport) {
	t.Helper()
	msgs := tr.messages()
	tr.fmu.Lock()
	tr.frames = nil
	tr.fmu.Unlock()
	for _, msg := range msgs {
		if msg.Event == string(protocol.FrameResetAction) && msg.Seq > m.applied {
			m.applied = msg.Seq
			m.resets++
			continue
		}
		if msg.Seq <= m.applied {
			t.Fatalf("duplicate seq %d delivered after applied %d", msg.Seq, m.applied)
		}
		if msg.Seq != m.applied+1 {
			t.Fatalf("seq gap: applied %d, got %d", m.applied, msg.Seq)
		}
		m.applied = msg.Seq
	}
}

func TestPropertyRandomDisconnectsAndAckLagNeverLoseOrDuplicate(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	for seed := int64(1); seed <= 25; seed++ {
		rng := rand.New(rand.NewSource(seed))
		frames := 2 + rng.Intn(12)
		ls := newClocked(t, dummyComponent, &fakeClock{}, func(c *Config) { c.ReplayFrames = frames })
		bus := ls.Session().Bus
		model := &modelClient{}
		var tr *recordingTransport
		attach := func() {
			tr = newRecordingTransport()
			if _, err := ls.Attach(context.Background(), tr, nil, model.applied, false); err != nil {
				t.Fatalf("seed %d: %v", seed, err)
			}
			model.consume(t, tr)
		}
		attach()
		for step := 0; step < 120; step++ {
			switch rng.Intn(5) {
			case 0, 1:
				for i := rng.Intn(6); i >= 0; i-- {
					bus.Publish(protocol.TopicFrame, "patch", step)
				}
				barrier(t, ls)
				if tr != nil {
					model.consume(t, tr)
				}
			case 2:
				if tr != nil && rng.Intn(2) == 0 {
					if err := ls.Ack(ls.Generation(), model.applied); err != nil {
						t.Fatalf("seed %d: %v", seed, err)
					}
				}
			case 3:
				if tr != nil {
					ls.Detach(tr)
					barrier(t, ls)
					tr = nil
				}
			case 4:
				if tr == nil {
					attach()
				}
			}
		}
		if tr == nil {
			attach()
		}
		barrier(t, ls)
		model.consume(t, tr)
		if head := ls.out.headSeq(); model.applied != head {
			t.Fatalf("seed %d: client applied %d, server head %d", seed, model.applied, head)
		}
		_ = ls.Close()
	}
}

func TestBootSeqEqualsRingHead(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ls := newClocked(t, dummyComponent, &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	for i := 0; i < 3; i++ {
		ls.Session().Bus.Publish(protocol.TopicFrame, "patch", i)
	}
	barrier(t, ls)
	res, err := ls.Render(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if res.Boot.Seq < 3 || res.Boot.Seq != ls.out.headSeq() {
		t.Fatalf("Boot.Seq = %d, want the ring head %d", res.Boot.Seq, ls.out.headSeq())
	}
	tr := newRecordingTransport()
	ar, err := ls.Attach(context.Background(), tr, nil, res.Boot.Seq, false)
	if err != nil || ar.Mode != protocol.ModeCurrent {
		t.Fatalf("attach at the boot seq: %+v %v", ar, err)
	}
	if got := tr.seqs(); len(got) != 0 {
		t.Fatalf("attach at head replayed %v", got)
	}
}

func TestAckIgnoresStaleGenerationAndRejectsBeyondHead(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ls := newClocked(t, dummyComponent, &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	first, second := newRecordingTransport(), newRecordingTransport()
	a1, err := ls.Attach(context.Background(), first, nil, 0, false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ls.Attach(context.Background(), second, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	ls.Session().Bus.Publish(protocol.TopicFrame, "patch", 1)
	barrier(t, ls)
	head := ls.out.headSeq()
	before, _ := ls.Retained()
	if before == 0 {
		t.Fatal("no frame retained")
	}
	if err := ls.Ack(a1.Gen, head); err != nil {
		t.Fatalf("stale ack = %v", err)
	}
	if n, _ := ls.Retained(); n != before {
		t.Fatalf("a stale-generation ack trimmed the ring: %d -> %d", before, n)
	}
	if err := ls.Ack(ls.Generation(), head+99); err != ErrAckAhead {
		t.Fatalf("ack beyond head = %v", err)
	}
	if err := ls.Ack(ls.Generation(), head); err != nil {
		t.Fatal(err)
	}
	if n, _ := ls.Retained(); n != 0 {
		t.Fatalf("ack did not trim: %d", n)
	}
}

func routerApp(out **router.Router) Component {
	return func(ctx *rt.Ctx) work.Node {
		*out = router.UseRouter(ctx)
		return &work.Element{Tag: "div"}
	}
}

func resetLocation(t *testing.T, tr *recordingTransport) (loc route.Location, found bool) {
	t.Helper()
	for _, m := range tr.messages() {
		if m.Event == string(protocol.FrameResetAction) {
			raw, _ := json.Marshal(m.Data)
			var reset protocol.ResetPayload
			if err := json.Unmarshal(raw, &reset); err != nil {
				t.Fatal(err)
			}
			loc, found = reset.Location, true
		}
	}
	return loc, found
}

func TestResetCarriesTheNavigatedLocation(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var r *router.Router
	ls := newClocked(t, routerApp(&r), &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	onSession(t, ls, func() { r.Navigate("/next?a=1#h") })
	barrier(t, ls)
	tr := newRecordingTransport()
	ar, err := ls.Attach(context.Background(), tr, nil, 0, true)
	if err != nil || ar.Mode != protocol.ModeReset {
		t.Fatalf("attach: %+v %v", ar, err)
	}
	loc, found := resetLocation(t, tr)
	if !found || loc.Path != "/next" || loc.Query.Get("a") != "1" || loc.Hash != "h" {
		t.Fatalf("reset location = %+v found=%v", loc, found)
	}
}

func TestRequestURLFollowsNavigationReplaceAndPopstate(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var r *router.Router
	ls := newClocked(t, routerApp(&r), &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}

	onSession(t, ls, func() { r.Navigate("/a//b/?z=2&z=1&y=%20v#frag") })
	loc := locationOf(t, ls)
	if loc.Path != "/a/b" || loc.Hash != "frag" || loc.Query.Get("y") != "v" || strings.Join(loc.Query["z"], ",") != "1,2" {
		t.Fatalf("navigate location = %+v", loc)
	}
	onSession(t, ls, func() { r.Replace("/replaced") })
	if got := locationOf(t, ls); got.Path != "/replaced" {
		t.Fatalf("replace location = %+v", got)
	}

	clientEvt := func(action string, payload map[string]any) protocol.ClientEvt {
		return protocol.ClientEvt{Event: protocol.Event{Type: protocol.RouteHandler, SID: string(ls.ID())}, Action: action, Payload: payload}
	}
	if err := deliver(ls, clientEvt("push", map[string]any{"path": "//evil.example/x", "query": "", "hash": ""})); err != nil {
		t.Fatal(err)
	}
	barrier(t, ls)
	if got := locationOf(t, ls); got.Path != "/replaced" {
		t.Fatalf("a client-sent push moved the URL to %+v", got)
	}

	if err := deliver(ls, clientEvt("popstate", map[string]any{"path": "", "query": "", "hash": "#top"})); err != nil {
		t.Fatal(err)
	}
	barrier(t, ls)
	if got := locationOf(t, ls); got.Path != "/" || got.Hash != "top" {
		t.Fatalf("popstate location = %+v", got)
	}
	var seen route.Location
	onSession(t, ls, func() { seen = r.Location.Peek() })
	if seen.Path != "/" || seen.Hash != "top" {
		t.Fatalf("router location = %+v", seen)
	}
}

func TestResetAfterNavigationAndReconnectKeepsTheNewURL(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var r *router.Router
	ls := newClocked(t, routerApp(&r), &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	onSession(t, ls, func() { r.Navigate("/one") })
	onSession(t, ls, func() { r.Navigate("/two") })
	for i := 0; i < 2; i++ {
		tr := newRecordingTransport()
		if _, err := ls.Attach(context.Background(), tr, nil, 0, true); err != nil {
			t.Fatal(err)
		}
		if loc, found := resetLocation(t, tr); !found || loc.Path != "/two" {
			t.Fatalf("reconnect %d: reset location = %+v found=%v", i, loc, found)
		}
		var seen route.Location
		onSession(t, ls, func() { seen = r.Location.Peek() })
		if seen.Path != "/two" {
			t.Fatalf("reconnect %d: router location = %+v", i, seen)
		}
	}
}

func resetBuilder(size int) func(string) (any, error) {
	return func(cause string) (any, error) {
		return protocol.ResetPayload{HTML: strings.Repeat("r", size), Cause: cause}, nil
	}
}

func TestPlanAndAttachAreOneStepUnderConcurrentPublishersAndAcks(t *testing.T) {
	for _, tc := range []struct {
		name     string
		maxBytes int
		reset    int
	}{
		{"small frames", 1 << 20, 16},
		{"reset larger than the byte cap", 300, 1000},
	} {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
			o := &outbound{maxFrames: 8, maxBytes: tc.maxBytes, maxReset: 1 << 20}
			stop := make(chan struct{})
			var wg sync.WaitGroup
			wg.Add(2)
			go func() {
				defer wg.Done()
				for {
					select {
					case <-stop:
						return
					default:
						o.send("frame", "patch", "p")
					}
				}
			}()
			go func() {
				defer wg.Done()
				rng := rand.New(rand.NewSource(3))
				for {
					select {
					case <-stop:
						return
					default:
						if h := o.headSeq(); h > 0 {
							_ = o.ack(uint64(rng.Int63n(int64(h))) + 1)
						}
					}
				}
			}()

			model := &modelClient{}
			build := resetBuilder(tc.reset)
			for i := 0; i < 10000; i++ {
				tr := newRecordingTransport()
				if _, _, err := o.attachPlanned(tr, model.applied, false, build); err != nil {
					t.Fatal(err)
				}
				runtime.Gosched()
				o.detach()
				model.consume(t, tr)
			}
			close(stop)
			wg.Wait()
		})
	}
}

func TestFailedResetOnReplacementKeepsTheOldTransportFed(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ls := newClocked(t, dummyComponent, &fakeClock{}, func(c *Config) { c.ResetLimit = 16 })
	defer func() { _ = ls.Close() }()
	old, joiner := newRecordingTransport(), newRecordingTransport()
	if _, err := ls.Attach(context.Background(), old, nil, 0, false); err != nil {
		t.Fatal(err)
	}
	_, err := ls.Attach(context.Background(), joiner, nil, 0, true)
	if !errors.Is(err, ErrResetTooLarge) {
		t.Fatalf("err = %v", err)
	}
	barrier(t, ls)
	if old.Closed() || ls.State() != StateLive || ls.Generation() != 1 {
		t.Fatalf("old closed=%v state=%v gen=%d", old.Closed(), ls.State(), ls.Generation())
	}
	if ls.out.current() != Transport(old) {
		t.Fatal("the old transport is no longer attached to the ring")
	}
	before := len(old.seqs())
	ls.Session().Bus.Publish(protocol.TopicFrame, "patch", "after")
	barrier(t, ls)
	if len(old.seqs()) != before+1 {
		t.Fatalf("the old transport stopped receiving frames: %d -> %d", before, len(old.seqs()))
	}
}

func TestFailedResetOnFirstAttachLeavesTheSessionDisconnected(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ls := newClocked(t, dummyComponent, &fakeClock{}, func(c *Config) { c.ResetLimit = 16 })
	defer func() { _ = ls.Close() }()
	_, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, true)
	if !errors.Is(err, ErrResetTooLarge) {
		t.Fatalf("err = %v", err)
	}
	barrier(t, ls)
	if ls.State() != StateDisconnected || ls.out.current() != nil {
		t.Fatalf("state=%v", ls.State())
	}
}

func locationOf(t *testing.T, ls *LiveSession) (loc route.Location) {
	t.Helper()
	if err := ls.Session().Ask(time.Second, func() error { loc = ls.requestState.URL(); return nil }); err != nil {
		t.Fatal(err)
	}
	return loc
}

func TestClientAheadReportsADiagnostic(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	ls := newClocked(t, dummyComponent, &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	got := make(chan protocol.Diagnostic, 1)
	sub := ls.Session().Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		if d.Phase == "protocol:client-ahead" {
			select {
			case got <- d:
			default:
			}
		}
	})
	defer sub.Unsubscribe()
	ar, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 999, false)
	if err != nil || ar.Mode != protocol.ModeReset {
		t.Fatalf("attach: %+v %v", ar, err)
	}
	select {
	case <-got:
	case <-time.After(2 * time.Second):
		t.Fatal("no diagnostic for a client ahead of the head")
	}
}

func TestSessionAndRouterCanonicalizeEveryPopstatePayloadIdentically(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	var seen *router.Router
	app := func(ctx *rt.Ctx) work.Node {
		seen = router.UseRouter(ctx)
		return &work.Element{Tag: "div"}
	}
	ls := newClocked(t, app, &fakeClock{}, nil)
	defer func() { _ = ls.Close() }()
	if _, err := ls.Attach(context.Background(), newRecordingTransport(), nil, 0, false); err != nil {
		t.Fatal(err)
	}
	barrier(t, ls)

	rng := rand.New(rand.NewSource(99))
	alphabet := []string{"a", "b", "%zz", "=", "&", "?", "#", "/", " ", "%20", "x", "//", "..", "é", "%2F", ";"}
	pick := func() string {
		out := ""
		for i := rng.Intn(9); i > 0; i-- {
			out += alphabet[rng.Intn(len(alphabet))]
		}
		return out
	}
	for i := 0; i < 1500; i++ {
		payload := map[string]any{"path": pick(), "query": pick(), "hash": pick()}
		evt := protocol.ClientEvt{Event: protocol.Event{Type: protocol.RouteHandler, SID: string(ls.ID())}, Action: "popstate", Payload: payload}
		if err := deliver(ls, evt); err != nil {
			t.Fatal(err)
		}
		barrier(t, ls)
		barrier(t, ls)
		var tracked, rendered route.Location
		if err := ls.Session().Ask(time.Second, func() error { tracked, rendered = ls.requestState.URL(), seen.Location.Peek(); return nil }); err != nil {
			t.Fatal(err)
		}
		if tracked.Path != rendered.Path || tracked.Hash != rendered.Hash || tracked.Query.Encode() != rendered.Query.Encode() {
			t.Fatalf("payload %v: session tracked %+v, router holds %+v", payload, tracked, rendered)
		}
	}
}
