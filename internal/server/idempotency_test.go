package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/coder/websocket"
	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/session"
)

type frameStream struct {
	ch chan map[string]any
}

func streamFrames(c *websocket.Conn) *frameStream {
	s := &frameStream{ch: make(chan map[string]any, 256)}
	go func() {
		defer close(s.ch)
		for {
			_, raw, err := c.Read(context.Background())
			if err != nil {
				return
			}
			var m map[string]any
			if json.Unmarshal(raw, &m) != nil {
				return
			}
			s.ch <- m
		}
	}()
	return s
}

func (s *frameStream) next(t *testing.T) map[string]any {
	t.Helper()
	select {
	case m, ok := <-s.ch:
		if !ok {
			t.Fatal("connection closed while waiting for a frame")
		}
		return m
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for a frame")
	}
	return nil
}

func (s *frameStream) nextMessage(t *testing.T) map[string]any {
	t.Helper()
	for {
		if m := s.next(t); m["seq"] != nil {
			return m
		}
	}
}

func (s *frameStream) messageAndEvack(t *testing.T, evack float64) (map[string]any, float64) {
	t.Helper()
	var msg map[string]any
	last := float64(0)
	for msg == nil || last < evack {
		m := s.next(t)
		switch {
		case m["seq"] != nil:
			msg = m
		case m["t"] == "evack":
			got := m["cs"].(float64)
			if got < last {
				t.Fatalf("evack went backwards: %v after %v", got, last)
			}
			last = got
		}
	}
	return msg, last
}

func (s *frameStream) noMessageWithin(t *testing.T, d time.Duration) {
	t.Helper()
	deadline := time.After(d)
	for {
		select {
		case m, ok := <-s.ch:
			if !ok {
				return
			}
			if m["seq"] != nil {
				t.Fatalf("unexpected message: %v", m)
			}
		case <-deadline:
			return
		}
	}
}

func sendClick(h *liveHarness, c *websocket.Conn, b bootDoc, cs uint64) {
	h.t.Helper()
	writeJSON(h.t, c, b.clickFrame(h.t, int(cs)))
}

func sendResync(h *liveHarness, c *websocket.Conn, b bootDoc, seq uint64) {
	h.t.Helper()
	writeJSON(h.t, c, map[string]any{"t": "resync", "sid": b.SID, "seq": seq, "reason": "unresolved"})
}

func patchText(m map[string]any) string {
	return fmt.Sprint(m["data"])
}

func TestEventWithoutClientSeqTerminates(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	for name, cs := range map[string]any{"missing": nil, "zero": 0} {
		t.Run(name, func(t *testing.T) {
			h := newHarness(t, TransportConfig{})
			defer h.close()
			b := h.boot()
			c := h.dialOK()
			defer c.CloseNow()
			h.join(c, b)
			evt := map[string]any{"t": "h", "i": b.handlerID(t), "sid": b.SID, "a": "click", "p": map[string]any{}}
			if cs != nil {
				evt["cs"] = cs
			}
			writeJSON(t, c, evt)
			if code := readCloseCode(t, c); code != websocket.StatusPolicyViolation {
				t.Fatalf("close code = %d, want 1008", code)
			}
			settled(t, h, 0)
		})
	}
}

func TestDuplicateClientSeqAcrossReconnectRunsOnce(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()
	b := h.boot()

	c := h.dialOK()
	h.join(c, b)
	sendClick(h, c, b, 1)
	first := streamFrames(c)
	m := first.nextMessage(t)
	if !strings.Contains(patchText(m), "count=1") {
		t.Fatalf("first patch = %v", m)
	}
	seq := uint64(m["seq"].(float64))
	h.drop(c, b.SID)

	again := h.dialOK()
	defer again.CloseNow()
	joined := h.joinWith(again, b, seq, false)
	if joined["cs"] != float64(1) || joined["mode"] != "current" {
		t.Fatalf("joined = %v", joined)
	}
	stream := streamFrames(again)
	sendClick(h, again, b, 1)
	sendClick(h, again, b, 2)
	m = stream.nextMessage(t)
	if !strings.Contains(patchText(m), "count=2") || uint64(m["seq"].(float64)) != seq+1 {
		t.Fatalf("the duplicate ran again: %v", m)
	}
	stream.noMessageWithin(t, 150*time.Millisecond)
}

func TestClientSeqGapAcceptedWithDiagnostic(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()
	b := h.boot()

	var mu sync.Mutex
	var gaps []protocol.Diagnostic
	sub := h.session(b.SID).Session().Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		if d.Phase == "protocol:client-seq-gap" {
			mu.Lock()
			gaps = append(gaps, d)
			mu.Unlock()
		}
	})
	defer sub.Unsubscribe()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	stream := streamFrames(c)
	sendClick(h, c, b, 5)
	m, evack := stream.messageAndEvack(t, 5)
	if !strings.Contains(patchText(m), "count=1") || evack != 5 {
		t.Fatalf("gap event not applied or not acknowledged: %v evack=%v", m, evack)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(gaps) != 1 || gaps[0].Metadata["from"] != uint64(1) || gaps[0].Metadata["to"] != uint64(4) {
		t.Fatalf("gap diagnostics = %v", gaps)
	}
}

func TestStaleReaderEventRejectedAfterReplace(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()
	b := h.boot()

	a := h.dialOK()
	defer a.CloseNow()
	h.join(a, b)
	oldGen := h.generation(b.SID)

	second := h.dialOK()
	defer second.CloseNow()
	joined := h.join(second, b)
	if joined["gen"] != float64(oldGen+1) {
		t.Fatalf("joined = %v", joined)
	}

	evt := protocol.ClientEvt{Event: protocol.Event{Type: protocol.HandlerTopic, SID: b.SID}, Action: "click", I: b.handlerID(t), CS: 1}
	accepted, err := h.session(b.SID).Deliver(oldGen, evt)
	if accepted || err != session.ErrStaleConnection {
		t.Fatalf("stale reader accepted=%v err=%v", accepted, err)
	}
	stream := streamFrames(second)
	stream.noMessageWithin(t, 100*time.Millisecond)
	sendClick(h, second, b, 1)
	if m := stream.nextMessage(t); !strings.Contains(patchText(m), "count=1") {
		t.Fatalf("the replacement reader's cs=1 was consumed by the stale reader: %v", m)
	}
}

func TestEvackReflectsAcceptedSeq(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()
	b := h.boot()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	stream := streamFrames(c)

	for cs := uint64(1); cs <= 3; cs++ {
		sendClick(h, c, b, cs)
		if _, got := stream.messageAndEvack(t, float64(cs)); got != float64(cs) {
			t.Fatalf("evack cs = %v after sending %d", got, cs)
		}
	}
	sendClick(h, c, b, 3)
	stream.noMessageWithin(t, 100*time.Millisecond)
}

func resyncHarness(t *testing.T, interval time.Duration) *liveHarness {
	return newReplayHarness(t, counterComponent, func(c *Config) {
		c.SessionConfig = &session.Config{ResyncInterval: interval}
	})
}

func TestResyncRequestsCoalesceAndRateLimit(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := resyncHarness(t, 400*time.Millisecond)
	defer h.close()
	b := h.boot()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	stream := streamFrames(c)

	sendResync(h, c, b, 0)
	r1 := stream.nextMessage(t)
	if r1["event"] != "reset" || r1["data"].(map[string]any)["cause"] != "client-request" {
		t.Fatalf("reset = %v", r1)
	}
	sendResync(h, c, b, uint64(r1["seq"].(float64)))
	r2 := stream.nextMessage(t)
	if r2["event"] != "reset" {
		t.Fatalf("second reset = %v", r2)
	}

	at := uint64(r2["seq"].(float64))
	started := time.Now()
	sendResync(h, c, b, at)
	sendResync(h, c, b, at)
	sendResync(h, c, b, at)
	stream.noMessageWithin(t, 150*time.Millisecond)
	r3 := stream.nextMessage(t)
	if r3["event"] != "reset" || time.Since(started) < 200*time.Millisecond {
		t.Fatalf("deferred reset = %v after %v", r3, time.Since(started))
	}
	stream.noMessageWithin(t, 600*time.Millisecond)

	if got := h.app.Stats().Resets["client-request"]; got != 3 {
		t.Fatalf("Stats.Resets = %v", h.app.Stats().Resets)
	}
}

func TestResyncBurstIsExactlyTwoBeforeLimiting(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := resyncHarness(t, time.Hour)
	defer h.close()
	b := h.boot()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	stream := streamFrames(c)

	sendResync(h, c, b, 0)
	r1 := stream.nextMessage(t)
	sendResync(h, c, b, uint64(r1["seq"].(float64)))
	r2 := stream.nextMessage(t)
	if r1["event"] != "reset" || r2["event"] != "reset" {
		t.Fatalf("burst resets = %v %v", r1, r2)
	}
	at := uint64(r2["seq"].(float64))
	sendResync(h, c, b, at)
	sendResync(h, c, b, at)
	stream.noMessageWithin(t, 200*time.Millisecond)
	if got := h.app.Stats().Resets["client-request"]; got != 2 {
		t.Fatalf("Stats.Resets = %v, want exactly 2", h.app.Stats().Resets)
	}
}

func TestResyncIgnoredWhileResetInFlight(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := resyncHarness(t, time.Hour)
	defer h.close()
	b := h.boot()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	stream := streamFrames(c)

	sendResync(h, c, b, 0)
	r := stream.nextMessage(t)
	sendResync(h, c, b, 0)
	sendResync(h, c, b, 0)
	sendClick(h, c, b, 1)
	next := stream.nextMessage(t)
	if next["event"] != "patch" || next["seq"] != r["seq"].(float64)+1 {
		t.Fatalf("a reset was duplicated or a seq skipped: reset=%v next=%v", r["seq"], next)
	}
	if got := h.app.Stats().Resets["client-request"]; got != 1 {
		t.Fatalf("Stats.Resets = %v", h.app.Stats().Resets)
	}
}

func TestClientOriginatedFrameRouterAndDomEventsAreNotEchoed(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()
	b := h.boot()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)
	stream := streamFrames(c)

	writeJSON(t, c, map[string]any{"t": "frame", "sid": b.SID, "a": "reset", "p": map[string]any{"html": "<html></html>"}, "cs": 1})
	writeJSON(t, c, map[string]any{"t": "router", "sid": b.SID, "a": "popstate", "p": map[string]any{"path": "/x", "query": "", "hash": ""}, "cs": 2})
	writeJSON(t, c, map[string]any{"t": "dom", "sid": b.SID, "a": "other", "p": map[string]any{}, "cs": 3})
	sendClick(h, c, b, 4)
	m := stream.nextMessage(t)
	if m["event"] != "patch" || m["seq"] != b.seqFloat()+1 {
		t.Fatalf("a client event was echoed into the ring: %v", m)
	}
}

func (b bootDoc) seqFloat() float64 { return float64(b.Seq) }

func TestAttachTokenOutlivesTheDisconnectGrace(t *testing.T) {
	ep, err := NewEndpoint(context.Background(), NewSessionRegistry(1, 1), TransportConfig{TokenTTL: 4 * time.Second})
	if err != nil {
		t.Fatal(err)
	}
	defer ep.cancel()
	if ep.disconnectTTL != 4*time.Second || ep.refreshEvery != 2*time.Second {
		t.Fatalf("disconnectTTL=%v refreshEvery=%v", ep.disconnectTTL, ep.refreshEvery)
	}
	now := time.Unix(1_000_000, 0)
	ep.tokens.now = func() time.Time { return now }
	tok := ep.tokens.issue("s")
	now = now.Add(ep.disconnectTTL + ep.refreshEvery - time.Second)
	if err := ep.tokens.verify("s", tok); err != nil {
		t.Fatalf("a token held at disconnect expired before the grace ended: %v", err)
	}
	now = now.Add(2 * time.Second)
	if err := ep.tokens.verify("s", tok); err == nil {
		t.Fatal("the token never expires")
	}
}

func TestAppDisconnectTTLMatchesTheConfiguredTokenTTL(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	app, err := New(Config{Component: counterComponent, Context: ctx, Transport: TransportConfig{TokenTTL: 8 * time.Second}})
	if err != nil {
		t.Fatal(err)
	}
	if app.sessionConfig.DisconnectTTL != 8*time.Second || app.endpoint.tokens.ttl != 12*time.Second {
		t.Fatalf("disconnect=%v token=%v", app.sessionConfig.DisconnectTTL, app.endpoint.tokens.ttl)
	}
}

func TestTeardownOfADeadPeerIsPrompt(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{WebSocketOptions: session.WebSocketOptions{CloseTimeout: 200 * time.Millisecond}})
	defer h.close()
	b := h.boot()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)

	started := time.Now()
	writeJSON(t, c, map[string]any{"t": "h", "i": b.handlerID(t), "sid": b.SID, "a": "click", "p": map[string]any{}})
	waitUntil(t, "the endpoint handler of the silent peer to finish", func() bool { return liveHandlers(h.app.endpoint) == 0 })
	if took := time.Since(started); took > 2*time.Second {
		t.Fatalf("teardown of a peer that never answers the close took %v", took)
	}
	if session.DefaultCloseTimeout >= 5*time.Second {
		t.Fatalf("default close bound %v does not beat the library's 5s", session.DefaultCloseTimeout)
	}
}

var bootScript = regexp.MustCompile(`<script id="live-boot" type="application/json">.*?</script>`)

func stripBootScript(document string) string { return bootScript.ReplaceAllString(document, "") }

var componentIDPattern = regexp.MustCompile(`c[0-9a-f]{16}:`)

const protocolFixturesEnv = "PROTOCOL_FIXTURES_DIR"

type transcriptStep struct {
	Dir   string         `json:"dir"`
	Frame map[string]any `json:"frame,omitempty"`
}

type transcript struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	SID         string           `json:"sid"`
	Ver         int              `json:"ver"`
	BootSeq     uint64           `json:"bootSeq"`
	HTML        string           `json:"html"`
	Steps       []transcriptStep `json:"steps"`
}

type recorder struct {
	t     *testing.T
	h     *liveHarness
	b     bootDoc
	steps []transcriptStep
	conn  *websocket.Conn
	in    *frameStream
}

func normalize(frame map[string]any) map[string]any {
	out := make(map[string]any, len(frame))
	for k, v := range frame {
		out[k] = v
	}
	if _, ok := out["tok"]; ok {
		out["tok"] = "<tok>"
	}
	return out
}

func (r *recorder) connect() {
	r.steps = append(r.steps, transcriptStep{Dir: "connect"})
	r.conn = r.h.dialOK()
	r.in = streamFrames(r.conn)
}

func (r *recorder) disconnect() {
	r.h.drop(r.conn, r.b.SID)
	r.steps = append(r.steps, transcriptStep{Dir: "disconnect"})
}

func (r *recorder) send(frame map[string]any) {
	writeJSON(r.t, r.conn, frame)
	r.steps = append(r.steps, transcriptStep{Dir: "client", Frame: normalize(frame)})
}

func (r *recorder) join(bs uint64, resync bool) {
	r.send(map[string]any{"t": "join", "sid": r.b.SID, "tok": r.b.Tok, "ver": r.b.Ver, "bs": bs, "resync": resync})
}

func (r *recorder) click(cs uint64) {
	r.send(r.b.clickFrame(r.t, int(cs)))
}

func (r *recorder) recv(n int) {
	frames := make([]map[string]any, 0, n)
	for i := 0; i < n; i++ {
		frames = append(frames, r.in.next(r.t))
	}
	rank := func(f map[string]any) int {
		switch {
		case f["t"] == "joined":
			return 0
		case f["seq"] != nil:
			return 1
		}
		return 2
	}
	sort.SliceStable(frames, func(i, j int) bool { return rank(frames[i]) < rank(frames[j]) })
	for _, f := range frames {
		r.steps = append(r.steps, transcriptStep{Dir: "server", Frame: normalize(f)})
	}
}

func (r *recorder) finish(name, description string) transcript {
	r.conn.CloseNow()
	return transcript{Name: name, Description: description, SID: r.b.SID, Ver: r.b.Ver, BootSeq: r.b.Seq, HTML: stripBootScript(r.b.HTML), Steps: r.steps}
}

func transcriptHarness(t *testing.T, replayFrames int) (*liveHarness, bootDoc) {
	h := newReplayHarness(t, counterComponent, func(c *Config) {
		c.IDGenerator = func(*http.Request) (session.SessionID, error) { return "transcript-sid", nil }
		c.SessionConfig = &session.Config{ReplayFrames: replayFrames}
	})
	return h, h.boot()
}

func generateTranscripts(t *testing.T) map[string][]byte {
	t.Helper()
	build := map[string]func() transcript{
		"join-replay": func() transcript {
			h, b := transcriptHarness(t, 0)
			defer h.close()
			r := &recorder{t: t, h: h, b: b}
			r.connect()
			r.join(b.Seq, false)
			r.recv(1)
			r.click(1)
			r.recv(2)
			r.click(2)
			r.recv(2)
			r.disconnect()
			r.connect()
			r.join(b.Seq, false)
			r.recv(3)
			return r.finish("join-replay", "A client that applied nothing rejoins with bs=0 and receives the retained frames in order after joined.")
		},
		"join-reset": func() transcript {
			h, b := transcriptHarness(t, 1)
			defer h.close()
			r := &recorder{t: t, h: h, b: b}
			r.connect()
			r.join(b.Seq, false)
			r.recv(1)
			r.click(1)
			r.recv(2)
			r.click(2)
			r.recv(2)
			r.click(3)
			r.recv(2)
			r.disconnect()
			r.connect()
			r.join(b.Seq, false)
			r.recv(2)
			return r.finish("join-reset", "The ring no longer reaches back to bs, so the server answers the join with a reset.")
		},
		"resync-reset": func() transcript {
			h, b := transcriptHarness(t, 0)
			defer h.close()
			r := &recorder{t: t, h: h, b: b}
			r.connect()
			r.join(b.Seq, false)
			r.recv(1)
			r.send(map[string]any{"t": "resync", "sid": b.SID, "seq": b.Seq, "reason": "unresolved"})
			r.recv(1)
			return r.finish("resync-reset", "A joined client reports an unresolved patch and the server replaces its document with a reset.")
		},
		"duplicate-cs": func() transcript {
			h, b := transcriptHarness(t, 0)
			defer h.close()
			r := &recorder{t: t, h: h, b: b}
			r.connect()
			r.join(b.Seq, false)
			r.recv(1)
			r.click(1)
			r.recv(2)
			r.click(1)
			r.click(2)
			r.recv(2)
			return r.finish("duplicate-cs", "A resent event with an already accepted cs is dropped and produces neither a frame nor an evack.")
		},
	}
	out := make(map[string][]byte, len(build))
	for name, fn := range build {
		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetEscapeHTML(false)
		enc.SetIndent("", "  ")
		if err := enc.Encode(fn()); err != nil {
			t.Fatal(err)
		}
		out[name+".json"] = componentIDPattern.ReplaceAll(buf.Bytes(), []byte("cX:"))
	}
	return out
}

func protocolFixtureDir() string {
	return filepath.Join("..", "..", "client", "test", "fixtures", "protocol")
}

func TestGenerateProtocolTranscripts(t *testing.T) {
	dir := os.Getenv(protocolFixturesEnv)
	if dir == "" {
		t.Skipf("%s not set; run `make protocol-fixtures` to write transcripts", protocolFixturesEnv)
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	existing, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	for _, f := range existing {
		if err := os.Remove(f); err != nil {
			t.Fatal(err)
		}
	}
	for name, data := range generateTranscripts(t) {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestProtocolTranscriptsAreDeterministicAndCommitted(t *testing.T) {
	first := generateTranscripts(t)
	second := generateTranscripts(t)
	for name, data := range first {
		if !bytes.Equal(data, second[name]) {
			t.Fatalf("transcript %s differs between two generations", name)
		}
	}
	for name, data := range first {
		committed, err := os.ReadFile(filepath.Join(protocolFixtureDir(), name))
		if err != nil {
			t.Fatalf("missing transcript %s; run `make protocol-fixtures`: %v", name, err)
		}
		if !bytes.Equal(committed, data) {
			t.Fatalf("transcript %s is stale; run `make protocol-fixtures`", name)
		}
	}
	files, _ := filepath.Glob(filepath.Join(protocolFixtureDir(), "*.json"))
	if len(files) != len(first) {
		t.Fatalf("committed transcript count %d differs from generated %d", len(files), len(first))
	}
}
