package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/protocol"
	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/work"
)

func newReplayHarness(t *testing.T, comp session.Component, mutate func(*Config)) *liveHarness {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	cfg := Config{Component: comp, Context: ctx}
	if mutate != nil {
		mutate(&cfg)
	}
	app, err := New(cfg)
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	return &liveHarness{t: t, app: app, srv: httptest.NewServer(app.Handler()), cancel: cancel}
}

func (h *liveHarness) joinWith(c *websocket.Conn, b bootDoc, bs uint64, resync bool) map[string]any {
	h.t.Helper()
	writeJSON(h.t, c, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver, "bs": bs, "resync": resync})
	return readFrame(h.t, c)
}

func (h *liveHarness) click(c *websocket.Conn, b bootDoc, cs int) {
	h.t.Helper()
	writeJSON(h.t, c, b.clickFrame(h.t, int(cs)))
}

func readPatches(t *testing.T, c *websocket.Conn, n int) []uint64 {
	t.Helper()
	var seqs []uint64
	for len(seqs) < n {
		msg := readFrame(t, c)
		if msg["topic"] == "ack" {
			t.Fatalf("server sent an ack topic message: %v", msg)
		}
		if msg["topic"] == "frame" && msg["event"] == "patch" {
			seqs = append(seqs, uint64(msg["seq"].(float64)))
		}
	}
	return seqs
}

func (h *liveHarness) drop(c *websocket.Conn, sid string) {
	h.t.Helper()
	c.CloseNow()
	waitUntil(h.t, "session to detach", func() bool {
		s := h.session(sid)
		return s != nil && s.State() == session.StateDisconnected
	})
}

func (h *liveHarness) joinAndClick(b bootDoc, clicks int) (*websocket.Conn, []uint64) {
	h.t.Helper()
	c := h.dialOK()
	h.join(c, b)
	var seqs []uint64
	for i := 1; i <= clicks; i++ {
		h.click(c, b, i)
		seqs = append(seqs, readPatches(h.t, c, 1)...)
	}
	return c, seqs
}

func assertContiguous(t *testing.T, got []uint64, first uint64) {
	t.Helper()
	for i, s := range got {
		if s != first+uint64(i) {
			t.Fatalf("seqs = %v, want contiguous from %d", got, first)
		}
	}
}

func TestRejoinReplaysUnackedFramesInOrder(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c, seqs := h.joinAndClick(b, 3)
	assertContiguous(t, seqs, seqs[0])
	h.drop(c, b.SID)

	again := h.dialOK()
	defer again.CloseNow()
	joined := h.joinWith(again, b, seqs[0], false)
	if joined["mode"] != "replay" {
		t.Fatalf("joined = %v", joined)
	}
	got := readPatches(t, again, 2)
	assertContiguous(t, got, seqs[1])
}

func TestRejoinAtHeadSendsOnlyJoined(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c, seqs := h.joinAndClick(b, 2)
	h.drop(c, b.SID)

	again := h.dialOK()
	defer again.CloseNow()
	joined := h.joinWith(again, b, seqs[1], false)
	if joined["mode"] != "current" {
		t.Fatalf("joined = %v", joined)
	}
	h.click(again, b, 3)
	next := readPatches(t, again, 1)
	if next[0] != seqs[1]+1 {
		t.Fatalf("first frame after rejoin at head = %d, want %d", next[0], seqs[1]+1)
	}
}

func TestRejoinOutsideRingGetsReset(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newReplayHarness(t, counterComponent, func(c *Config) { c.SessionConfig = &session.Config{ReplayFrames: 4} })
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	h.join(c, b)
	h.drop(c, b.SID)
	bus := h.session(b.SID).Session().Bus
	for i := 0; i < 10; i++ {
		bus.Publish(protocol.TopicFrame, "patch", []any{})
	}
	if err := h.session(b.SID).Flush(); err != nil {
		t.Fatal(err)
	}

	again := h.dialOK()
	defer again.CloseNow()
	joined := h.joinWith(again, b, 0, false)
	if joined["mode"] != "reset" {
		t.Fatalf("joined = %v", joined)
	}
	msg := readFrame(t, again)
	data, _ := msg["data"].(map[string]any)
	if msg["event"] != "reset" || msg["topic"] != "frame" || data["cause"] != "rejoin-gap" {
		t.Fatalf("reset frame = %v", msg)
	}
	if html, _ := data["html"].(string); !strings.Contains(html, "count=0") {
		t.Fatalf("reset html = %q", html)
	}
	if seq := msg["seq"].(float64); seq <= 10 {
		t.Fatalf("reset seq = %v", seq)
	}
}

func TestJoinResyncFlagAlwaysResets(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	joined := h.joinWith(c, b, b.Seq, true)
	if joined["mode"] != "reset" {
		t.Fatalf("joined = %v", joined)
	}
	msg := readFrame(t, c)
	data, _ := msg["data"].(map[string]any)
	if msg["event"] != "reset" || data["cause"] != "rejoin-request" {
		t.Fatalf("reset frame = %v", msg)
	}
}

func TestClientAheadOfServerGetsReset(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	joined := h.joinWith(c, b, 500, false)
	msg := readFrame(t, c)
	data, _ := msg["data"].(map[string]any)
	if joined["mode"] != "reset" || data["cause"] != "client-ahead" {
		t.Fatalf("joined=%v msg=%v", joined, msg)
	}
}

func TestAckTrimsRing(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c, seqs := h.joinAndClick(b, 3)
	defer c.CloseNow()
	sess := h.session(b.SID)
	if n, _ := sess.Retained(); n != 3 {
		t.Fatalf("retained %d before ack", n)
	}
	writeJSON(t, c, map[string]any{"t": "ack", "sid": b.SID, "seq": seqs[1]})
	waitUntil(t, "ring to trim", func() bool { n, _ := sess.Retained(); return n == 1 })
	writeJSON(t, c, map[string]any{"t": "ack", "sid": b.SID, "seq": seqs[2]})
	waitUntil(t, "ring to empty", func() bool { n, bytes := sess.Retained(); return n == 0 && bytes == 0 })
}

func TestAckBeyondHeadTerminates1008(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	writeJSON(t, c, map[string]any{"t": "ack", "sid": b.SID, "seq": 1000})
	if code := readCloseCode(t, c); code != websocket.StatusPolicyViolation {
		t.Fatalf("close code = %d", code)
	}
	settled(t, h, 0)
}

func TestAckFromReplacedConnectionIgnored(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	first, seqs := h.joinAndClick(b, 2)
	defer first.CloseNow()
	second := h.dialOK()
	defer second.CloseNow()
	if joined := h.joinWith(second, b, 0, false); joined["gen"] != float64(2) {
		t.Fatalf("joined = %v", joined)
	}
	sess := h.session(b.SID)
	before, _ := sess.Retained()
	writeJSON(t, first, map[string]any{"t": "ack", "sid": b.SID, "seq": seqs[1]})
	if err := sess.Session().Ask(time.Second, func() error { return nil }); err != nil {
		t.Fatal(err)
	}
	time.Sleep(50 * time.Millisecond)
	if after, _ := sess.Retained(); after != before {
		t.Fatalf("a replaced connection's ack trimmed the ring: %d -> %d", before, after)
	}
}

func TestBootSeqEqualsRingHead(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	sess := h.session(b.SID)
	bus := sess.Session().Bus
	for i := 0; i < 3; i++ {
		bus.Publish(protocol.TopicFrame, "patch", i)
	}
	if b.Seq != 0 {
		t.Fatalf("fresh boot seq = %d", b.Seq)
	}
	if err := sess.Flush(); err != nil {
		t.Fatal(err)
	}
	c := h.dialOK()
	defer c.CloseNow()
	joined := h.joinWith(c, b, b.Seq, false)
	if joined["mode"] != "replay" {
		t.Fatalf("a client at the boot seq must replay frames produced after SSR, got %v", joined)
	}
	assertContiguous(t, []uint64{uint64(readFrame(t, c)["seq"].(float64))}, b.Seq+1)
}

func TestServerSendsNoAckTopic(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	for i := 1; i <= 5; i++ {
		h.click(c, b, i)
		readPatches(t, c, 1)
	}
}

func TestResetLargerThanBudgetDeclines(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	big := func(ctx *rt.Ctx) work.Node {
		return &work.Element{Tag: "p", Children: []work.Node{&work.Text{Value: strings.Repeat("x", 16<<10)}}}
	}
	h := newReplayHarness(t, big, func(c *Config) {
		c.Transport.OutboundBudget = 8 << 10
		c.Transport.CloseTimeout = time.Minute
	})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	writeJSON(t, c, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver, "bs": b.Seq, "resync": true})
	declined := readFrame(t, c)
	if declined["t"] != "declined" || declined["code"] != float64(4404) || declined["reason"] != "session reset too large" {
		t.Fatalf("declined = %v", declined)
	}
	deadline := time.Now().Add(2 * time.Second)
	for h.session(b.SID) != nil {
		if time.Now().After(deadline) {
			t.Fatal("the session was not closed within 2s while CloseTimeout is 1m: RequestClose must precede the decline")
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func TestReplayOverBudgetImpossible(t *testing.T) {
	tests := []struct {
		name       string
		budget     int
		replay     int
		wantReplay int
		wantReset  int
	}{
		{"defaults", 0, 0, session.DefaultReplayBytes, session.DefaultOutboundBudget / 2},
		{"oversized replay is clamped", 1 << 20, 64 << 20, 1 << 18, 1 << 19},
		{"small budget shrinks the default", 100 << 10, 0, 25 << 10, 50 << 10},
		{"tiny budget never yields zero", 3, 0, 1, 1},
		{"smaller configured value is kept", 4 << 20, 1 << 10, 1 << 10, 2 << 20},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			app, err := New(Config{
				Component:     counterComponent,
				Context:       ctx,
				Transport:     TransportConfig{WebSocketOptions: session.WebSocketOptions{OutboundBudget: tc.budget}},
				SessionConfig: &session.Config{ReplayBytes: tc.replay},
			})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = app.Shutdown(context.Background()) }()
			cfg := app.sessionConfig
			budget := app.endpoint.cfg.OutboundBudget
			if cfg.ReplayBytes != tc.wantReplay || cfg.ResetLimit != tc.wantReset {
				t.Fatalf("ReplayBytes=%d ResetLimit=%d, want %d %d", cfg.ReplayBytes, cfg.ResetLimit, tc.wantReplay, tc.wantReset)
			}
			if budget >= 4 && (cfg.ReplayBytes > budget/4 || cfg.ResetLimit > budget/2) {
				t.Fatalf("a full replay could exceed the outbound budget %d", budget)
			}
		})
	}
}
