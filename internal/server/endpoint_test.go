package server

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
	"testing"
	"time"

	"github.com/coder/websocket"
	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/protocol"
	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/work"
)

type bootDoc struct {
	SID  string `json:"sid"`
	Ver  int    `json:"ver"`
	Seq  uint64 `json:"seq"`
	WS   string `json:"ws"`
	Tok  string `json:"tok"`
	HTML string `json:"-"`
}

type liveHarness struct {
	t      *testing.T
	app    *App
	srv    *httptest.Server
	cancel context.CancelFunc
}

func counterComponent(ctx *rt.Ctx) work.Node {
	count := rt.UseSignal(ctx, 0)
	label := rt.UseComputed(ctx, func() string { return fmt.Sprintf("count=%d", count.Get()) })
	return work.BuildElement("button",
		work.On("click", func(work.Event) { count.Update(func(n int) int { return n + 1 }) }),
		work.TextOf(label),
	)
}

func newHarness(t *testing.T, cfg TransportConfig) *liveHarness {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	app, err := New(Config{Component: counterComponent, Context: ctx, Transport: cfg})
	if err != nil {
		cancel()
		t.Fatal(err)
	}
	h := &liveHarness{t: t, app: app, srv: httptest.NewServer(app.Handler()), cancel: cancel}
	return h
}

func (h *liveHarness) close() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := h.app.Shutdown(ctx); err != nil {
		h.t.Errorf("shutdown: %v", err)
	}
	h.cancel()
	h.srv.Close()
	http.DefaultClient.CloseIdleConnections()
}

func (h *liveHarness) boot() bootDoc {
	h.t.Helper()
	resp, err := http.Get(h.srv.URL + "/")
	if err != nil {
		h.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	html := string(raw)
	marker := `<script id="live-boot" type="application/json">`
	i := strings.Index(html, marker)
	if i < 0 {
		h.t.Fatal("no boot payload in SSR response")
	}
	rest := html[i+len(marker):]
	rest = rest[:strings.Index(rest, "</script>")]
	var b bootDoc
	if err := json.Unmarshal([]byte(strings.ReplaceAll(rest, `<\/`, `</`)), &b); err != nil {
		h.t.Fatal(err)
	}
	b.HTML = html
	return b
}

const (
	dialAttempts = 6
	dialBackoff  = 100 * time.Millisecond
)

func (h *liveHarness) dial(origin string, hdr http.Header) (*websocket.Conn, *http.Response, error) {
	if hdr == nil {
		hdr = http.Header{}
	}
	if origin != "" {
		hdr.Set("Origin", origin)
	}
	var c *websocket.Conn
	var resp *http.Response
	var err error
	for attempt := 1; attempt <= dialAttempts; attempt++ {
		c, resp, err = h.dialOnce(hdr)
		if !errors.Is(err, syscall.EADDRNOTAVAIL) {
			break
		}
		time.Sleep(time.Duration(attempt) * dialBackoff)
	}
	return c, resp, err
}

func (h *liveHarness) dialOnce(hdr http.Header) (*websocket.Conn, *http.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, resp, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(h.srv.URL, "http")+livePath, &websocket.DialOptions{HTTPHeader: hdr})
	if c != nil {
		c.SetReadLimit(1 << 22)
	}
	return c, resp, err
}

func (h *liveHarness) dialOK() *websocket.Conn {
	h.t.Helper()
	c, _, err := h.dial(h.srv.URL, nil)
	if err != nil {
		h.t.Fatalf("dial: %v", err)
	}
	return c
}

func writeJSON(t *testing.T, c *websocket.Conn, v any) {
	t.Helper()
	raw, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := c.Write(ctx, websocket.MessageText, raw); err != nil {
		t.Fatalf("write: %v", err)
	}
}

func readFrame(t *testing.T, c *websocket.Conn) map[string]any {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, raw, err := c.Read(ctx)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	var m map[string]any
	if err := json.Unmarshal(raw, &m); err != nil {
		t.Fatalf("frame is not JSON: %s", raw)
	}
	return m
}

func readCloseCode(t *testing.T, c *websocket.Conn) websocket.StatusCode {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	for {
		_, _, err := c.Read(ctx)
		if err != nil {
			return websocket.CloseStatus(err)
		}
	}
}

func (h *liveHarness) join(c *websocket.Conn, b bootDoc) map[string]any {
	h.t.Helper()
	writeJSON(h.t, c, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver, "bs": b.Seq})
	return readFrame(h.t, c)
}

func (h *liveHarness) session(sid string) *session.LiveSession {
	sess, _ := h.app.registry.Lookup(session.SessionID(sid))
	return sess
}

func (h *liveHarness) attached(sid string) bool {
	sess := h.session(sid)
	return sess != nil && sess.State() == session.StateLive
}

func (h *liveHarness) generation(sid string) uint64 {
	return h.session(sid).Generation()
}

func waitUntil(t *testing.T, what string, cond func() bool) {
	t.Helper()
	waitUntilWithin(t, 5*time.Second, what, cond)
}

func waitUntilWithin(t *testing.T, bound time.Duration, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(bound)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("timed out waiting for %s", what)
		}
		time.Sleep(2 * time.Millisecond)
	}
}

func (c *connTracker) count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.active
}

func liveHandlers(e *Endpoint) int { return e.conns.count() }

var handlerElement = regexp.MustCompile(`<[^>]*\bdata-pi="(\d+)"[^>]*\bdata-pon=|<[^>]*\bdata-pon="[^"]*"[^>]*\bdata-pi="(\d+)"`)

func (b bootDoc) handlerID(t *testing.T) uint32 {
	t.Helper()
	m := handlerElement.FindStringSubmatch(b.HTML)
	if m == nil {
		t.Fatalf("boot document carries no handler: %.300s", b.HTML)
	}
	raw := m[1]
	if raw == "" {
		raw = m[2]
	}
	id, err := strconv.ParseUint(raw, 10, 32)
	if err != nil {
		t.Fatal(err)
	}
	return uint32(id)
}

func (b bootDoc) clickFrame(t *testing.T, cs int) map[string]any {
	t.Helper()
	return map[string]any{"t": "h", "i": b.handlerID(t), "sid": b.SID, "a": "click", "p": map[string]any{}, "cs": cs}
}

func settled(t *testing.T, h *liveHarness, handlers int) {
	t.Helper()
	waitUntil(t, "endpoint handlers to finish", func() bool { return liveHandlers(h.app.endpoint) == handlers })
}

func TestEndToEndSSRJoinEventAndPatch(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	if b.WS != "/live" || b.Tok == "" || b.SID == "" {
		t.Fatalf("boot payload misses transport fields: %+v", b)
	}

	c := h.dialOK()
	defer c.CloseNow()
	joined := h.join(c, b)
	if joined["t"] != "joined" || joined["gen"] != float64(1) {
		t.Fatalf("unexpected join reply: %v", joined)
	}

	writeJSON(t, c, b.clickFrame(t, 1))

	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("no frame patch arrived after the click event")
		default:
		}
		msg := readFrame(t, c)
		if msg["topic"] == "frame" && msg["event"] == "patch" {
			if !strings.Contains(fmt.Sprint(msg["data"]), "count=1") {
				t.Fatalf("patch does not carry the new state: %v", msg["data"])
			}
			return
		}
	}
}

func TestStaleConnectionLeaveKeepsSessionAndReplacementAlive(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	a := h.dialOK()
	defer a.CloseNow()
	if h.join(a, b)["gen"] != float64(1) {
		t.Fatal("first join must get generation 1")
	}

	bc := h.dialOK()
	defer bc.CloseNow()
	if h.join(bc, b)["gen"] != float64(2) {
		t.Fatal("second join must get generation 2")
	}

	if code := readCloseCode(t, a); code != 4409 {
		t.Fatalf("replaced connection must close with 4409, got %d", code)
	}
	settled(t, h, 1)

	if _, ok := h.app.registry.Lookup(session.SessionID(b.SID)); !ok {
		t.Fatal("the stale connection's teardown destroyed the session")
	}
	if !h.attached(b.SID) || h.generation(b.SID) != 2 {
		t.Fatal("the stale connection's teardown detached the live connection")
	}

	writeJSON(t, bc, b.clickFrame(t, 1))
	for {
		msg := readFrame(t, bc)
		if msg["topic"] == "frame" {
			break
		}
	}
}

func TestCleanCloseTearsDownOnce(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	h.join(c, b)

	if err := c.Close(websocket.StatusNormalClosure, "bye"); err != nil {
		t.Fatalf("client close handshake: %v", err)
	}
	settled(t, h, 0)

	if h.attached(b.SID) {
		t.Fatal("clean close did not detach the connection")
	}
	if _, ok := h.app.registry.Lookup(session.SessionID(b.SID)); !ok {
		t.Fatal("clean close must not destroy the session in Phase 1")
	}
}

func TestAbruptDropTearsDownOnce(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	h.join(c, b)

	c.CloseNow()
	settled(t, h, 0)

	if h.attached(b.SID) {
		t.Fatal("abrupt drop did not detach the connection")
	}
	if _, ok := h.app.registry.Lookup(session.SessionID(b.SID)); !ok {
		t.Fatal("abrupt drop must not destroy the session in Phase 1")
	}
}

func TestReattachAfterDropGetsNewGeneration(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	h.join(c, b)
	c.CloseNow()
	settled(t, h, 0)

	again := h.dialOK()
	defer again.CloseNow()
	if got := h.join(again, b)["gen"]; got != float64(2) {
		t.Fatalf("expected generation 2 after re-attach, got %v", got)
	}
}

func TestInboundEventsAreHandedToTheSessionInArrivalOrder(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	const n = 10000
	var mu sync.Mutex
	var got []int
	readerGoroutines := map[string]struct{}{}
	h.app.endpoint.publish = func(_ *session.LiveSession, _ uint64, evt protocol.ClientEvt) (bool, error) {
		buf := make([]byte, 32)
		id := string(buf[:runtime.Stack(buf, false)])
		mu.Lock()
		defer mu.Unlock()
		got = append(got, int(evt.Payload.(float64)))
		readerGoroutines[strings.SplitN(id, " ", 3)[1]] = struct{}{}
		return true, nil
	}

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)

	for i := 0; i < n; i++ {
		writeJSON(t, c, map[string]any{"t": "probe", "sid": b.SID, "a": "n", "p": i, "cs": i + 1})
	}
	waitUntil(t, "every event to reach the session", func() bool {
		mu.Lock()
		defer mu.Unlock()
		return len(got) == n
	})

	mu.Lock()
	defer mu.Unlock()
	if len(got) != n {
		t.Fatalf("expected %d events, got %d", n, len(got))
	}
	for i, v := range got {
		if v != i {
			t.Fatalf("event %d reached the session as %d", i, v)
		}
	}
	if len(readerGoroutines) != 1 {
		t.Fatalf("events were handed over from %d distinct goroutines, expected the single reader", len(readerGoroutines))
	}
}

func TestOriginPolicy(t *testing.T) {
	tests := []struct {
		name    string
		allowed []string
		origin  func(h *liveHarness) string
		ok      bool
	}{
		{"same origin", nil, func(h *liveHarness) string { return h.srv.URL }, true},
		{"cross origin", nil, func(h *liveHarness) string { return "http://evil.example" }, false},
		{"missing origin", nil, func(h *liveHarness) string { return "" }, false},
		{"unparsable origin", nil, func(h *liveHarness) string { return "::" }, false},
		{"allow listed exact origin", []string{"https://app.example"}, func(h *liveHarness) string { return "https://app.example" }, true},
		{"near miss of allow listed origin", []string{"https://app.example"}, func(h *liveHarness) string { return "https://app.example.evil" }, false},
		{"wildcard", []string{"*"}, func(h *liveHarness) string { return "http://anything.example" }, true},
		{"wildcard still needs an origin", []string{"*"}, func(h *liveHarness) string { return "" }, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
			h := newHarness(t, TransportConfig{AllowedOrigins: tc.allowed})
			defer h.close()

			c, resp, err := h.dial(tc.origin(h), nil)
			if tc.ok {
				if err != nil {
					t.Fatalf("expected the upgrade to succeed: %v", err)
				}
				c.CloseNow()
				settled(t, h, 0)
				return
			}
			if err == nil {
				c.CloseNow()
				t.Fatal("expected the upgrade to be rejected")
			}
			if resp == nil || resp.StatusCode != http.StatusForbidden {
				t.Fatalf("expected 403, got %v (%v)", resp, err)
			}
		})
	}
}

func TestJoinDeclines(t *testing.T) {
	forge := func(h *liveHarness, sid string) string { return h.app.endpoint.tokens.issue(sid) }
	tests := []struct {
		name  string
		frame func(h *liveHarness, b bootDoc) any
		code  int
	}{
		{"garbage token", func(h *liveHarness, b bootDoc) any {
			return map[string]any{"t": "join", "sid": b.SID, "tok": "garbage", "ver": b.Ver}
		}, 4401},
		{"missing token", func(h *liveHarness, b bootDoc) any {
			return map[string]any{"t": "join", "sid": b.SID, "ver": b.Ver}
		}, 4401},
		{"token for another sid", func(h *liveHarness, b bootDoc) any {
			return map[string]any{"t": "join", "sid": b.SID, "tok": forge(h, "someone-else"), "ver": b.Ver}
		}, 4401},
		{"expired token", func(h *liveHarness, b bootDoc) any {
			h.app.endpoint.tokens.now = func() time.Time { return time.Now().Add(-time.Hour) }
			tok := forge(h, b.SID)
			h.app.endpoint.tokens.now = time.Now
			return map[string]any{"t": "join", "sid": b.SID, "tok": tok, "ver": b.Ver}
		}, 4401},
		{"unknown session with a valid token", func(h *liveHarness, b bootDoc) any {
			return map[string]any{"t": "join", "sid": "no-such-session", "tok": forge(h, "no-such-session"), "ver": b.Ver}
		}, 4404},
		{"malformed json", func(h *liveHarness, b bootDoc) any { return "not an object" }, 4400},
		{"wrong frame type", func(h *liveHarness, b bootDoc) any {
			return map[string]any{"t": "evt", "sid": b.SID, "tok": b.Tok, "ver": b.Ver}
		}, 4400},
		{"missing sid", func(h *liveHarness, b bootDoc) any { return map[string]any{"t": "join", "tok": b.Tok, "ver": b.Ver} }, 4400},
		{"unsupported version", func(h *liveHarness, b bootDoc) any {
			return map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver + 1}
		}, 4400},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
			h := newHarness(t, TransportConfig{})
			defer h.close()

			b := h.boot()
			c := h.dialOK()
			defer c.CloseNow()
			writeJSON(t, c, tc.frame(h, b))

			reply := readFrame(t, c)
			if reply["t"] != "declined" || reply["code"] != float64(tc.code) {
				t.Fatalf("expected declined %d, got %v", tc.code, reply)
			}
			if code := readCloseCode(t, c); int(code) != tc.code {
				t.Fatalf("expected close %d, got %d", tc.code, code)
			}
			settled(t, h, 0)
			if h.attached(b.SID) || h.generation(b.SID) != 0 {
				t.Fatal("a declined join must not attach")
			}
		})
	}
}

func TestJoinDeadlineClosesWith4408(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{JoinDeadline: 100 * time.Millisecond})
	defer h.close()

	c := h.dialOK()
	defer c.CloseNow()
	started := time.Now()
	if code := readCloseCode(t, c); code != 4408 {
		t.Fatalf("expected 4408, got %d", code)
	}
	if time.Since(started) > 3*time.Second {
		t.Fatal("join deadline took far longer than configured")
	}
	settled(t, h, 0)
}

func TestJoinedIsTheFirstServerFrame(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	sess, _ := h.app.registry.Lookup(session.SessionID(b.SID))
	stop := make(chan struct{})
	flooding := make(chan struct{})
	go func() {
		defer close(flooding)
		for {
			select {
			case <-stop:
				return
			default:
				sess.Session().Bus.Publish(protocol.TopicFrame, "patch", []any{})
				time.Sleep(time.Millisecond)
			}
		}
	}()

	for i := 0; i < 20; i++ {
		c := h.dialOK()
		if first := h.join(c, b); first["t"] != "joined" {
			t.Fatalf("first frame was %v", first)
		}
		c.CloseNow()
	}
	close(stop)
	<-flooding
	settled(t, h, 0)
}

func TestOversizedMessageClosesWith1009(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{ReadLimit: 1024})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)

	writeJSON(t, c, map[string]any{"t": "probe", "sid": b.SID, "a": "n", "p": strings.Repeat("a", 2048), "cs": 1})
	if code := readCloseCode(t, c); code != websocket.StatusMessageTooBig {
		t.Fatalf("expected 1009, got %d", code)
	}
	settled(t, h, 0)
}

func TestProtocolViolationsCloseTheConnection(t *testing.T) {
	tests := []struct {
		name  string
		write func(t *testing.T, c *websocket.Conn, b bootDoc)
		code  websocket.StatusCode
	}{
		{"malformed json", func(t *testing.T, c *websocket.Conn, b bootDoc) {
			_ = c.Write(context.Background(), websocket.MessageText, []byte("{not json"))
		}, websocket.StatusInvalidFramePayloadData},
		{"foreign session id", func(t *testing.T, c *websocket.Conn, b bootDoc) {
			writeJSON(t, c, map[string]any{"t": "probe", "sid": "other", "a": "n"})
		}, websocket.StatusPolicyViolation},
		{"binary frame", func(t *testing.T, c *websocket.Conn, b bootDoc) {
			_ = c.Write(context.Background(), websocket.MessageBinary, []byte{1, 2})
		}, websocket.StatusUnsupportedData},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
			h := newHarness(t, TransportConfig{})
			defer h.close()

			b := h.boot()
			c := h.dialOK()
			defer c.CloseNow()
			h.join(c, b)
			tc.write(t, c, b)
			if code := readCloseCode(t, c); code != tc.code {
				t.Fatalf("expected %d, got %d", tc.code, code)
			}
			settled(t, h, 0)
		})
	}
}

func TestAcksAreAcceptedAndDoNotReachTheBus(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	sess, _ := h.app.registry.Lookup(session.SessionID(b.SID))
	seen := make(chan string, 4)
	sub := sess.Session().Bus.SubscribeAll(func(topic protocol.Topic, event string, _ interface{}) {
		if topic == "ack" {
			seen <- event
		}
	})
	defer sub.Unsubscribe()

	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)

	writeJSON(t, c, map[string]any{"t": "ack", "sid": b.SID, "seq": 0})
	writeJSON(t, c, map[string]any{"t": "probe", "sid": b.SID, "a": "n", "p": 1, "cs": 1})
	sess.Session().Ask(time.Second, func() error { return nil })
	select {
	case ev := <-seen:
		t.Fatalf("a client ack leaked to the bus as %q", ev)
	default:
	}
}

func TestUpgradeRequestHeadersPopulateTheLiveRequestState(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c, _, err := h.dial(h.srv.URL, http.Header{"Cookie": []string{"theme=dark"}, "User-Agent": []string{"probe-agent"}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	h.join(c, b)

	state := h.session(b.SID).RequestState()
	if v, ok := state.GetCookie("theme"); !ok || v != "dark" {
		t.Fatalf("cookie from the upgrade request is missing: %q %v", v, ok)
	}
	if v, ok := state.Get("User-Agent"); !ok || v != "probe-agent" {
		t.Fatalf("header from the upgrade request is missing: %q %v", v, ok)
	}
	if !state.IsLive() {
		t.Fatal("request state must be live after attach")
	}
}

func TestDeclinedJoinsAllocateNoLingeringState(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	sessionsBefore := func() int {
		return h.app.registry.Len()
	}
	before := sessionsBefore()
	goroutinesBefore := runtime.NumGoroutine()

	const attempts = 400
	for i := 0; i < attempts; i++ {
		sid := fmt.Sprintf("random-%d", i)
		c := h.dialOK()
		tok := h.app.endpoint.tokens.issue(sid)
		if i%2 == 0 {
			tok = "garbage"
		}
		writeJSON(t, c, map[string]any{"t": "join", "sid": sid, "tok": tok, "ver": b.Ver})
		reply := readFrame(t, c)
		if reply["t"] != "declined" {
			t.Fatalf("attempt %d was not declined: %v", i, reply)
		}
		readCloseCode(t, c)
		c.CloseNow()
	}
	settled(t, h, 0)

	if after := sessionsBefore(); after != before {
		t.Fatalf("registry grew from %d to %d sessions across declined joins", before, after)
	}
	if grown := runtime.NumGoroutine() - goroutinesBefore; grown > 2 {
		t.Fatalf("%d goroutines survived %d declined joins", grown, attempts)
	}
	t.Logf("declined joins: %d attempts, registry %d -> %d, goroutines %d -> %d", attempts, before, sessionsBefore(), goroutinesBefore, runtime.NumGoroutine())
}

func TestAppContextCancelClosesLiveConnections(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	c := h.dialOK()
	defer c.CloseNow()
	h.join(c, b)

	h.cancel()
	readCloseCode(t, c)
	settled(t, h, 0)
}

func TestJoinedTokenSurvivesExpiryOfTheBootToken(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{TokenTTL: time.Minute})
	defer h.close()

	base := time.Now()
	var offset atomic.Int64
	h.app.endpoint.tokens.now = func() time.Time { return base.Add(time.Duration(offset.Load())) }

	b := h.boot()
	rejoin := func(tok string) map[string]any {
		c := h.dialOK()
		defer c.CloseNow()
		reply := h.join(c, bootDoc{SID: b.SID, Ver: b.Ver, Tok: tok})
		if reply["t"] == "joined" {
		}
		return reply
	}

	first := rejoin(b.Tok)
	fresh, _ := first["tok"].(string)
	if first["t"] != "joined" || fresh == "" {
		t.Fatalf("first join must succeed and carry a token: %v", first)
	}
	settled(t, h, 0)

	offset.Store(int64(50 * time.Second))
	second := rejoin(fresh)
	fresh2, _ := second["tok"].(string)
	if second["t"] != "joined" || fresh2 == "" || fresh2 == fresh {
		t.Fatalf("rejoin with the joined token must succeed and refresh it: %v", second)
	}
	settled(t, h, 0)

	offset.Store(int64(100 * time.Second))
	if reply := rejoin(b.Tok); reply["t"] != "declined" || reply["code"] != float64(4401) {
		t.Fatalf("expired boot token must be declined with 4401, got %v", reply)
	}
	settled(t, h, 0)
	if reply := rejoin(fresh2); reply["t"] != "joined" {
		t.Fatalf("token from the latest joined frame must still be valid: %v", reply)
	}
}

func tokenExpiry(t *testing.T, tok string) time.Time {
	t.Helper()
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil || len(raw) < 8 {
		t.Fatalf("unparsable token %q", tok)
	}
	return time.Unix(int64(binary.BigEndian.Uint64(raw[:8])), 0)
}

func TestLiveConnectionKeepsRefreshingItsToken(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{TokenTTL: time.Minute})
	defer h.close()

	base := time.Now()
	var offset atomic.Int64
	now := func() time.Time { return base.Add(time.Duration(offset.Load())) }
	h.app.endpoint.tokens.now = now
	h.app.endpoint.refreshEvery = 10 * time.Millisecond

	b := h.boot()
	c := h.dialOK()
	h.join(c, b)

	var mu sync.Mutex
	latest := ""
	go func() {
		for {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			_, raw, err := c.Read(ctx)
			cancel()
			if err != nil {
				return
			}
			var f struct{ T, Tok string }
			if json.Unmarshal(raw, &f) == nil && f.T == "tok" {
				mu.Lock()
				latest = f.Tok
				mu.Unlock()
			}
		}
	}()

	for step := 1; step <= 6; step++ {
		offset.Store(int64(time.Duration(step) * 30 * time.Second))
		waitUntil(t, "a token issued after the clock moved", func() bool {
			mu.Lock()
			defer mu.Unlock()
			return latest != "" && tokenExpiry(t, latest).After(now().Add(30*time.Second))
		})
	}
	mu.Lock()
	final := latest
	mu.Unlock()
	c.CloseNow()
	settled(t, h, 0)

	offset.Store(int64(190 * time.Second))
	stale := h.dialOK()
	reply := h.join(stale, b)
	stale.CloseNow()
	if reply["t"] != "declined" || reply["code"] != float64(4401) {
		t.Fatalf("boot token must have expired, got %v", reply)
	}
	settled(t, h, 0)

	again := h.dialOK()
	defer again.CloseNow()
	if reply := h.join(again, bootDoc{SID: b.SID, Ver: b.Ver, Tok: final}); reply["t"] != "joined" {
		t.Fatalf("latest refreshed token rejected after 3x TTL: %v", reply)
	}
}

func TestConcurrentJoinsToOneSessionDoNotRace(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := newHarness(t, TransportConfig{})
	defer h.close()

	b := h.boot()
	const joiners, rounds = 100, 3
	for round := 0; round < rounds; round++ {
		var wg sync.WaitGroup
		for i := 0; i < joiners; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				c, _, err := h.dial(h.srv.URL, nil)
				if err != nil {
					t.Error(err)
					return
				}
				defer c.CloseNow()
				raw, _ := json.Marshal(map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver})
				ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
				defer cancel()
				if err := c.Write(ctx, websocket.MessageText, raw); err != nil {
					t.Error(err)
					return
				}
				c.Read(ctx)
			}()
		}
		wg.Wait()
	}
	settled(t, h, 0)
}
