package server

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"go.uber.org/goleak"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/work"
)

type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type lifecycleApp struct {
	t    *testing.T
	app  *App
	srv  *httptest.Server
	logs *syncBuffer
}

func newLifecycleApp(t *testing.T, comp session.Component, mutate func(*Config)) *lifecycleApp {
	t.Helper()
	logs := &syncBuffer{}
	cfg := Config{
		Component:     comp,
		SessionConfig: &session.Config{Logger: slog.New(slog.NewTextHandler(logs, nil))},
	}
	if mutate != nil {
		mutate(&cfg)
	}
	app, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	la := &lifecycleApp{t: t, app: app, srv: httptest.NewServer(app.Handler()), logs: logs}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = app.Shutdown(ctx)
		la.srv.Close()
		http.DefaultClient.CloseIdleConnections()
	})
	return la
}

var noRedirect = &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}

func (la *lifecycleApp) get(client *http.Client, hdr http.Header) (*http.Response, string) {
	la.t.Helper()
	req, _ := http.NewRequest(http.MethodGet, la.srv.URL+"/", nil)
	for k, v := range hdr {
		req.Header[k] = v
	}
	resp, err := client.Do(req)
	if err != nil {
		la.t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp, string(raw)
}

func (la *lifecycleApp) bootOf(body string) bootDoc {
	la.t.Helper()
	marker := `<script id="live-boot" type="application/json">`
	i := strings.Index(body, marker)
	if i < 0 {
		la.t.Fatalf("no boot payload in: %.200s", body)
	}
	rest := body[i+len(marker):]
	rest = rest[:strings.Index(rest, "</script>")]
	var b bootDoc
	if err := json.Unmarshal([]byte(strings.ReplaceAll(rest, `<\/`, `</`)), &b); err != nil {
		la.t.Fatal(err)
	}
	b.HTML = body
	return b
}

func (la *lifecycleApp) boot(hdr http.Header) bootDoc {
	la.t.Helper()
	resp, body := la.get(http.DefaultClient, hdr)
	if resp.StatusCode != http.StatusOK {
		la.t.Fatalf("SSR status %d: %.200s", resp.StatusCode, body)
	}
	return la.bootOf(body)
}

func (la *lifecycleApp) dial(hdr http.Header) *websocket.Conn {
	la.t.Helper()
	if hdr == nil {
		hdr = http.Header{}
	}
	hdr.Set("Origin", la.srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws"+strings.TrimPrefix(la.srv.URL, "http")+livePath, &websocket.DialOptions{HTTPHeader: hdr})
	if err != nil {
		la.t.Fatal(err)
	}
	return c
}

func (la *lifecycleApp) join(c *websocket.Conn, b bootDoc) map[string]any {
	la.t.Helper()
	writeJSON(la.t, c, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver})
	return readFrame(la.t, c)
}

func (la *lifecycleApp) waitLen(n int) {
	la.t.Helper()
	waitUntil(la.t, fmt.Sprintf("%d registered sessions", n), func() bool { return la.app.registry.Len() == n })
}

func plainComponent(*rt.Ctx) work.Node {
	return &work.Element{Tag: "div", Children: []work.Node{&work.Text{Value: "hello"}}}
}

func TestRegistryFullReturns503(t *testing.T) {
	var renders atomic.Int64
	la := newLifecycleApp(t, func(ctx *rt.Ctx) work.Node {
		renders.Add(1)
		return plainComponent(ctx)
	}, func(c *Config) { c.MaxSessions = 2; c.MaxPendingSessions = 2 })

	la.boot(nil)
	la.boot(nil)
	rendersBefore := renders.Load()

	resp, _ := la.get(http.DefaultClient, nil)
	if resp.StatusCode != http.StatusServiceUnavailable || resp.Header.Get("Retry-After") == "" {
		t.Fatalf("over capacity: status %d Retry-After %q, want 503 with Retry-After", resp.StatusCode, resp.Header.Get("Retry-After"))
	}
	if renders.Load() != rendersBefore {
		t.Fatal("a refused request was rendered before admission")
	}
	la.waitLen(2)
	st := la.app.Stats()
	if st.Rejected.Full != 1 || st.Capacity != 2 || st.Sessions.Pending != 2 {
		t.Fatalf("stats = %+v", st)
	}

	for _, sess := range la.app.registry.Snapshot() {
		la.app.registry.Remove(sess)
		break
	}
	la.waitLen(1)
	la.boot(nil)
}

func TestDuplicateSessionIDIsAnError(t *testing.T) {
	la := newLifecycleApp(t, plainComponent, func(c *Config) {
		c.IDGenerator = func(*http.Request) (session.SessionID, error) { return "dup", nil }
	})

	first := la.boot(nil)
	resp, _ := la.get(http.DefaultClient, nil)
	if resp.StatusCode != http.StatusInternalServerError {
		t.Fatalf("duplicate id: status %d, want 500", resp.StatusCode)
	}
	if !strings.Contains(la.logs.String(), "ssr:duplicate-session") {
		t.Fatalf("no diagnostic for the duplicate id; logs: %s", la.logs.String())
	}
	la.waitLen(1)

	old, ok := la.app.registry.Lookup("dup")
	if !ok || old.State() != session.StatePending {
		t.Fatal("the duplicate disturbed the existing session")
	}
	c := la.dial(nil)
	defer c.CloseNow()
	if joined := la.join(c, first); joined["t"] != "joined" {
		t.Fatalf("the original session cannot be joined after a duplicate attempt: %v", joined)
	}
	if la.app.Stats().Rejected.Duplicate != 1 {
		t.Fatalf("stats = %+v", la.app.Stats())
	}
}

func TestSSRRedirectClosesSessionAndWritesSetCookie(t *testing.T) {
	var effects atomic.Int64
	la := newLifecycleApp(t, func(ctx *rt.Ctx) work.Node {
		rt.UseEffect(ctx, func() func() { effects.Add(1); return nil })
		if rs := headers.UseRequestState(ctx); rs != nil {
			rs.SetRedirect("/login", http.StatusSeeOther)
			rs.AddResponseHeader("Set-Cookie", "flash=1; Path=/; HttpOnly")
		}
		return plainComponent(ctx)
	}, nil)

	for i := 0; i < 20; i++ {
		resp, _ := la.get(noRedirect, nil)
		if resp.StatusCode != http.StatusSeeOther || resp.Header.Get("Location") != "/login" {
			t.Fatalf("status %d location %q", resp.StatusCode, resp.Header.Get("Location"))
		}
		if got := resp.Header.Values("Set-Cookie"); len(got) != 1 || got[0] != "flash=1; Path=/; HttpOnly" {
			t.Fatalf("redirect lost its Set-Cookie: %v", got)
		}
	}
	la.waitLen(0)
	if effects.Load() != 0 {
		t.Fatalf("a redirected render started %d effects", effects.Load())
	}
	if st := la.app.Stats(); st.Closed["redirected"] != 20 {
		t.Fatalf("stats = %+v", st)
	}
}

func TestSSRSetCookieIsWritten(t *testing.T) {
	la := newLifecycleApp(t, func(ctx *rt.Ctx) work.Node {
		if rs := headers.UseRequestState(ctx); rs != nil {
			rs.AddResponseHeader("Set-Cookie", "a=1; Path=/")
			rs.AddResponseHeader("Set-Cookie", "b=2; Path=/")
			rs.SetResponseHeader("X-Frame-Options", "DENY")
		}
		return plainComponent(ctx)
	}, nil)
	resp, body := la.get(http.DefaultClient, nil)
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if got := resp.Header.Values("Set-Cookie"); len(got) != 2 || got[0] != "a=1; Path=/" || got[1] != "b=2; Path=/" {
		t.Fatalf("Set-Cookie = %v", got)
	}
	if resp.Header.Get("X-Frame-Options") != "DENY" || resp.Header.Get("Cache-Control") != "no-store" {
		t.Fatalf("headers = %v", resp.Header)
	}
	la.bootOf(body)
}

func TestSSRRenderErrorClosesSession(t *testing.T) {
	la := newLifecycleApp(t, func(ctx *rt.Ctx) work.Node {
		sig := rt.UseSignal(ctx, 0)
		sig.Set(1)
		return plainComponent(ctx)
	}, func(c *Config) { c.SessionConfig.DevMode = true })
	for i := 0; i < 5; i++ {
		resp, _ := la.get(http.DefaultClient, nil)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status %d, want 500", resp.StatusCode)
		}
	}
	la.waitLen(0)
}

func TestRootCrashReturns500(t *testing.T) {
	crash := func(*rt.Ctx) work.Node { panic("kaboom-secret") }

	t.Run("production hides the detail", func(t *testing.T) {
		la := newLifecycleApp(t, crash, nil)
		resp, body := la.get(http.DefaultClient, nil)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status %d", resp.StatusCode)
		}
		if strings.Contains(body, "kaboom-secret") || strings.Contains(body, "live-boot") {
			t.Fatalf("production crash body leaks detail or boots a session: %.300s", body)
		}
		if !strings.Contains(body, "Something went wrong") {
			t.Fatalf("no crash page: %.300s", body)
		}
		la.waitLen(0)
	})

	t.Run("dev mode shows the crash page", func(t *testing.T) {
		la := newLifecycleApp(t, crash, func(c *Config) { c.SessionConfig.DevMode = true })
		resp, body := la.get(http.DefaultClient, nil)
		if resp.StatusCode != http.StatusInternalServerError {
			t.Fatalf("status %d", resp.StatusCode)
		}
		if !strings.Contains(body, "kaboom-secret") || strings.Contains(body, "live-boot") {
			t.Fatalf("dev crash body: %.300s", body)
		}
		la.waitLen(0)
	})
}

func TestSSRReadsTheViewOnTheActor(t *testing.T) {
	var once sync.Once
	var stop atomic.Bool
	var wg sync.WaitGroup
	defer func() { stop.Store(true); wg.Wait() }()

	comp := func(ctx *rt.Ctx) work.Node {
		sig := rt.UseSignal(ctx, 0)
		once.Do(func() {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for i := 1; i < 2000 && !stop.Load(); i++ {
					sig.Set(i)
					time.Sleep(20 * time.Microsecond)
				}
			}()
		})
		kids := make([]work.Node, 0, 40)
		for i := 0; i < 40; i++ {
			i := i
			label := rt.UseComputed(ctx, func() string { return fmt.Sprintf("n=%d", sig.Get()+i) })
			kids = append(kids, work.BuildElement("span", work.TextOf(label)))
		}
		return &work.Element{Tag: "div", Children: kids}
	}
	la := newLifecycleApp(t, comp, nil)
	for i := 0; i < 40; i++ {
		resp, body := la.get(http.DefaultClient, nil)
		if resp.StatusCode != http.StatusOK || !strings.Contains(body, "n=") {
			t.Fatalf("status %d", resp.StatusCode)
		}
	}
}

type renderGate struct {
	block atomic.Pointer[chan struct{}]
}

func (g *renderGate) component(ctx *rt.Ctx) work.Node {
	_ = headers.UseRequest(ctx).Headers()
	if ch := g.block.Load(); ch != nil {
		<-*ch
	}
	return plainComponent(ctx)
}

func (g *renderGate) hold() (release func()) {
	ch := make(chan struct{})
	g.block.Store(&ch)
	var once sync.Once
	return func() { once.Do(func() { close(ch) }) }
}

func TestRemoveDoesNotBlockBehindRender(t *testing.T) {
	la := newLifecycleApp(t, plainComponent, nil)
	b := la.boot(nil)
	sess, _ := la.app.registry.Lookup(session.SessionID(b.SID))

	gate := make(chan struct{})
	stuck := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	defer release()
	if err := sess.Session().Post(func() { close(stuck); <-gate }); err != nil {
		t.Fatal(err)
	}
	<-stuck

	done := make(chan struct{})
	go func() {
		la.app.registry.Remove(sess)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("Remove blocked behind a step that never returns")
	}
	if _, ok := la.app.registry.Lookup(session.SessionID(b.SID)); ok {
		t.Fatal("the removed session is still reachable")
	}

	release()
	select {
	case <-sess.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the removed session did not close after its step returned")
	}
}

func TestJoinsBehindAStuckRenderTimeOutTogetherAtTheJoinDeadline(t *testing.T) {
	la := newLifecycleApp(t, plainComponent, func(c *Config) { c.Transport.JoinDeadline = 300 * time.Millisecond })
	b := la.boot(nil)
	sess, _ := la.app.registry.Lookup(session.SessionID(b.SID))
	gate := make(chan struct{})
	stuck := make(chan struct{})
	var once sync.Once
	release := func() { once.Do(func() { close(gate) }) }
	defer release()
	if err := sess.Session().Post(func() { close(stuck); <-gate }); err != nil {
		t.Fatal(err)
	}
	<-stuck

	const joiners = 6
	codes := make(chan int, joiners)
	started := time.Now()
	for i := 0; i < joiners; i++ {
		go func() {
			c := la.dial(nil)
			defer c.CloseNow()
			writeJSON(t, c, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver})
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			for {
				_, _, err := c.Read(ctx)
				if err != nil {
					codes <- int(websocket.CloseStatus(err))
					return
				}
			}
		}()
	}
	for i := 0; i < joiners; i++ {
		select {
		case code := <-codes:
			if code != int(session.CloseJoinTimeout) {
				t.Fatalf("joiner closed with %d, want 4408", code)
			}
		case <-time.After(8 * time.Second):
			t.Fatalf("only %d of %d joiners were released; joins drain one per bound", i, joiners)
		}
	}
	if took := time.Since(started); took > 3*time.Second {
		t.Fatalf("%d joiners took %v behind a stuck render, want about the %v join deadline", joiners, took, 300*time.Millisecond)
	}
	handlersFreed := time.Now().Add(2 * time.Second)
	for liveHandlers(la.app.endpoint) != 0 {
		if time.Now().After(handlersFreed) {
			t.Fatalf("%d endpoint handlers are still parked in Attach after the join deadline", liveHandlers(la.app.endpoint))
		}
		time.Sleep(5 * time.Millisecond)
	}

	release()
	waitUntil(t, "the session to settle after the render returns", func() bool {
		return sess.Session().Ask(time.Second, func() error { return nil }) == nil
	})
	if sess.Generation() != 0 {
		t.Fatalf("an abandoned join attached: generation %d", sess.Generation())
	}
	c := la.dial(nil)
	defer c.CloseNow()
	if joined := la.join(c, b); joined["t"] != "joined" {
		t.Fatalf("join after the render was released: %v", joined)
	}
}

func TestShutdownClosesSessionsTransportsAndUploads(t *testing.T) {
	baseline := goleak.IgnoreCurrent()

	dir := t.TempDir()
	la := newLifecycleApp(t, counterComponent, func(c *Config) {
		c.UploadConfig = &upload.Config{StoragePath: dir, SweepInterval: 20 * time.Millisecond}
	})
	defer func() {
		la.srv.Close()
		http.DefaultClient.CloseIdleConnections()
		goleak.VerifyNone(t, baseline, goleak.IgnoreTopFunction("github.com/tus/tusd/v2/pkg/handler.newDelayedContext.func1"))
	}()

	pending := la.boot(nil)
	joinedBoot := la.boot(nil)
	c := la.dial(nil)
	defer c.CloseNow()
	if la.join(c, joinedBoot)["t"] != "joined" {
		t.Fatal("join failed")
	}
	sessions := la.app.registry.Snapshot()
	if len(sessions) != 2 {
		t.Fatalf("%d sessions registered", len(sessions))
	}
	closeCode := make(chan websocket.StatusCode, 1)
	go func() { closeCode <- readCloseCode(t, c) }()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	started := time.Now()
	if err := la.app.Shutdown(ctx); err != nil {
		t.Fatalf("Shutdown: %v", err)
	}
	if took := time.Since(started); took > 2*time.Second {
		t.Fatalf("Shutdown took %v with a responsive client", took)
	}
	for _, sess := range sessions {
		select {
		case <-sess.Done():
		default:
			t.Fatalf("session %s still running after Shutdown returned", sess.ID())
		}
		if sess.State() != session.StateClosed || sess.CloseReason() != session.ReasonShutdown {
			t.Fatalf("session %s state=%v reason=%v", sess.ID(), sess.State(), sess.CloseReason())
		}
	}
	if la.app.registry.Len() != 0 {
		t.Fatalf("%d sessions left registered", la.app.registry.Len())
	}
	if code := <-closeCode; code != websocket.StatusGoingAway {
		t.Fatalf("live connection closed with %d, want 1001", code)
	}
	if liveHandlers(la.app.endpoint) != 0 {
		t.Fatal("endpoint handlers survived Shutdown")
	}

	if err := la.app.Shutdown(ctx); err != nil {
		t.Fatalf("second Shutdown: %v", err)
	}

	resp, _ := la.get(http.DefaultClient, nil)
	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("SSR after Shutdown: %d, want 503", resp.StatusCode)
	}
	hdr := http.Header{"Origin": {la.srv.URL}}
	dialCtx, dialCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dialCancel()
	_, dialResp, err := websocket.Dial(dialCtx, "ws"+strings.TrimPrefix(la.srv.URL, "http")+livePath, &websocket.DialOptions{HTTPHeader: hdr})
	if err == nil || dialResp == nil || dialResp.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("upgrade after Shutdown: err=%v resp=%v, want 503", err, dialResp)
	}
	_ = pending
}

func TestShutdownReturnsTheContextErrorWhileASessionIsStuck(t *testing.T) {
	g := &renderGate{}
	la := newLifecycleApp(t, g.component, nil)
	b := la.boot(nil)
	sess, _ := la.app.registry.Lookup(session.SessionID(b.SID))

	release := g.hold()
	defer release()
	stuck := make(chan struct{})
	_ = sess.Session().Post(func() { close(stuck); <-*g.block.Load() })
	<-stuck

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	err := la.app.Shutdown(ctx)
	var shutdownErr *ShutdownError
	if !errors.Is(err, context.DeadlineExceeded) || !errors.As(err, &shutdownErr) || shutdownErr.Stuck != 1 {
		t.Fatalf("Shutdown returned %v, want a ShutdownError naming 1 stuck session wrapping the context error", err)
	}
	release()
	select {
	case <-sess.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the session never closed after release")
	}
}

func TestServerHasHeaderLimitsAndTimeouts(t *testing.T) {
	app, err := New(Config{Component: plainComponent})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	srv := app.Server(":0")
	if srv.ReadHeaderTimeout != DefaultReadHeaderTimeout || srv.ReadTimeout != DefaultReadTimeout ||
		srv.WriteTimeout != DefaultWriteTimeout || srv.IdleTimeout != DefaultIdleTimeout || srv.MaxHeaderBytes != DefaultMaxHeaderBytes {
		t.Fatalf("defaults not applied: %+v", srv)
	}
	if srv.ReadHeaderTimeout <= 0 || srv.ReadTimeout <= 0 || srv.WriteTimeout <= 0 || srv.IdleTimeout <= 0 || srv.MaxHeaderBytes <= 0 {
		t.Fatalf("a limit is unset: %+v", srv)
	}

	custom, err := New(Config{Component: plainComponent, HTTP: HTTPConfig{
		ReadHeaderTimeout: time.Second, ReadTimeout: 2 * time.Second, WriteTimeout: 3 * time.Second, IdleTimeout: 4 * time.Second, MaxHeaderBytes: 2048,
	}})
	if err != nil {
		t.Fatal(err)
	}
	defer custom.Shutdown(context.Background())
	cs := custom.Server(":0")
	if cs.ReadHeaderTimeout != time.Second || cs.ReadTimeout != 2*time.Second || cs.WriteTimeout != 3*time.Second || cs.IdleTimeout != 4*time.Second || cs.MaxHeaderBytes != 2048 {
		t.Fatalf("configured limits not applied: %+v", cs)
	}
}

func serveOn(t *testing.T, srv *http.Server) string {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	go srv.Serve(ln)
	t.Cleanup(func() { _ = srv.Close() })
	return ln.Addr().String()
}

func TestServerEnforcesTheHeaderLimits(t *testing.T) {
	app, err := New(Config{Component: plainComponent, HTTP: HTTPConfig{ReadHeaderTimeout: 200 * time.Millisecond, MaxHeaderBytes: 4096}})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	addr := serveOn(t, app.Server(""))

	req, _ := http.NewRequest(http.MethodGet, "http://"+addr+"/", nil)
	req.Header.Set("X-Big", strings.Repeat("a", 64<<10))
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusRequestHeaderFieldsTooLarge {
		t.Fatalf("oversized headers: status %d, want 431", resp.StatusCode)
	}

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_, _ = conn.Write([]byte("GET / HTTP/1.1\r\nHost: x\r\nX-Slow: "))
	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	started := time.Now()
	_, err = io.ReadAll(conn)
	if time.Since(started) > 2*time.Second {
		t.Fatalf("a slow header was held open for %v (err %v)", time.Since(started), err)
	}
}

func TestLiveConnectionsSurviveTheServerTimeouts(t *testing.T) {
	app, err := New(Config{Component: counterComponent, HTTP: HTTPConfig{ReadTimeout: 150 * time.Millisecond, WriteTimeout: 150 * time.Millisecond}})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	addr := serveOn(t, app.Server(""))
	base := "http://" + addr

	resp, err := http.Get(base + "/")
	if err != nil {
		t.Fatal(err)
	}
	raw, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	la := &lifecycleApp{t: t}
	b := la.bootOf(string(raw))

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, "ws://"+addr+livePath, &websocket.DialOptions{HTTPHeader: http.Header{"Origin": {base}}})
	if err != nil {
		t.Fatal(err)
	}
	defer c.CloseNow()
	writeJSON(t, c, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver})
	if readFrame(t, c)["t"] != "joined" {
		t.Fatal("join failed")
	}

	time.Sleep(600 * time.Millisecond)
	writeJSON(t, c, b.clickFrame(t, 1))
	for {
		if msg := readFrame(t, c); msg["topic"] == "frame" && msg["event"] == "patch" {
			return
		}
	}
}

func TestWatchdogReportsSlowStepAndRemovesWedgedSession(t *testing.T) {
	var cleaned atomic.Bool
	comp := func(ctx *rt.Ctx) work.Node {
		rt.UseEffect(ctx, func() func() { return func() { cleaned.Store(true) } })
		return plainComponent(ctx)
	}
	la := newLifecycleApp(t, comp, func(c *Config) {
		c.Watchdog = WatchdogConfig{Interval: 10 * time.Millisecond, SlowStep: 60 * time.Millisecond, WedgeStep: 400 * time.Millisecond}
	})
	b := la.boot(nil)
	c := la.dial(nil)
	defer c.CloseNow()
	if la.join(c, b)["t"] != "joined" {
		t.Fatal("join failed")
	}
	sess, _ := la.app.registry.Lookup(session.SessionID(b.SID))

	var mu sync.Mutex
	var phases []string
	sess.Session().Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) {
		mu.Lock()
		phases = append(phases, d.Phase)
		mu.Unlock()
	})

	gate := make(chan struct{})
	stuck := make(chan struct{})
	_ = sess.Session().Post(func() { close(stuck); <-gate })
	<-stuck

	waitUntil(t, "the slow step diagnostic", func() bool {
		mu.Lock()
		defer mu.Unlock()
		for _, p := range phases {
			if p == "actor:slow-step" {
				return true
			}
		}
		return false
	})
	if _, ok := la.app.registry.Lookup(session.SessionID(b.SID)); !ok {
		t.Fatal("the session was removed before the wedge threshold")
	}

	waitUntil(t, "the wedged session to be removed", func() bool {
		_, ok := la.app.registry.Lookup(session.SessionID(b.SID))
		return !ok
	})
	if st := la.app.Stats(); st.Closed["wedged"] != 1 || st.Evicted != 1 || st.Stuck != 1 {
		t.Fatalf("stats = %+v", st)
	}
	if code := readCloseCode(t, c); code != websocket.StatusInternalError {
		t.Fatalf("wedged session's connection closed with %d, want 1011", code)
	}

	again := la.dial(nil)
	defer again.CloseNow()
	writeJSON(t, again, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver})
	if reply := readFrame(t, again); reply["t"] != "declined" || reply["code"] != float64(session.CloseNotFound) {
		t.Fatalf("join of a wedged session: %v", reply)
	}

	close(gate)
	select {
	case <-sess.Done():
	case <-time.After(5 * time.Second):
		t.Fatal("the wedged session never ran its close step")
	}
	if !cleaned.Load() {
		t.Fatal("cleanups did not run after the wedged step returned")
	}
	if st := la.app.Stats(); st.Stuck != 0 {
		t.Fatalf("stuck gauge did not clear after the step returned: %+v", st)
	}
}

func TestRetainedHeadersReachSSRAndTheUpgrade(t *testing.T) {
	var mu sync.Mutex
	seen := map[string]string{}
	comp := func(ctx *rt.Ctx) work.Node {
		state := headers.UseRequestState(ctx)
		mu.Lock()
		tenant, _ := state.Get("X-Tenant")
		auth, _ := state.Get("Authorization")
		seen["tenant"], seen["auth"] = tenant, auth
		mu.Unlock()
		return plainComponent(ctx)
	}

	t.Run("configured", func(t *testing.T) {
		la := newLifecycleApp(t, comp, func(c *Config) { c.RetainedHeaders = []string{"X-Tenant"} })
		b := la.boot(http.Header{"X-Tenant": {"acme"}, "Authorization": {"Bearer secret"}})
		mu.Lock()
		ssrTenant, ssrAuth := seen["tenant"], seen["auth"]
		mu.Unlock()
		if ssrTenant != "acme" || ssrAuth != "" {
			t.Fatalf("SSR saw tenant=%q auth=%q", ssrTenant, ssrAuth)
		}

		c := la.dial(http.Header{"X-Tenant": {"globex"}, "Authorization": {"Bearer other"}})
		defer c.CloseNow()
		la.join(c, b)
		sess, _ := la.app.registry.Lookup(session.SessionID(b.SID))
		state := sess.RequestState()
		if v, _ := state.Get("X-Tenant"); v != "globex" {
			t.Fatalf("upgrade header X-Tenant = %q, want globex", v)
		}
		if v, _ := state.Get("Authorization"); v != "" {
			t.Fatalf("Authorization retained from the upgrade: %q", v)
		}
	})

	t.Run("not configured", func(t *testing.T) {
		la := newLifecycleApp(t, comp, nil)
		la.boot(http.Header{"X-Tenant": {"acme"}})
		mu.Lock()
		tenant := seen["tenant"]
		mu.Unlock()
		if tenant != "" {
			t.Fatalf("a header outside the allowlist was retained: %q", tenant)
		}
	})
}

func TestStatsCountSessionsPerState(t *testing.T) {
	la := newLifecycleApp(t, counterComponent, nil)
	la.boot(nil)
	live := la.boot(nil)
	dropped := la.boot(nil)

	c := la.dial(nil)
	defer c.CloseNow()
	la.join(c, live)
	d := la.dial(nil)
	la.join(d, dropped)
	d.CloseNow()

	waitUntil(t, "per-state counts", func() bool {
		st := la.app.Stats()
		return st.Sessions.Pending == 1 && st.Sessions.Live == 1 && st.Sessions.Disconnected == 1
	})
}

func TestUploadRoutesOutliveTheServerReadAndWriteTimeouts(t *testing.T) {
	app, err := New(Config{
		Component:    plainComponent,
		UploadConfig: &upload.Config{StoragePath: t.TempDir()},
		HTTP:         HTTPConfig{ReadTimeout: 150 * time.Millisecond, WriteTimeout: 150 * time.Millisecond},
	})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	addr := serveOn(t, app.Server(""))

	sess := session.NewLiveSession("upload-sess", 1, plainComponent, nil)
	defer sess.Close()
	if err := app.registry.Put(sess); err != nil {
		t.Fatal(err)
	}
	var completed atomic.Int64
	token := sess.UploadRegistry().Issue("upload-sess", upload.GrantSpec{Files: 1, OnComplete: func(upload.FileInfo) error { completed.Add(1); return nil }})

	create, _ := http.NewRequest(http.MethodPost, "http://"+addr+"/tus/", nil)
	create.Header.Set("Tus-Resumable", "1.0.0")
	create.Header.Set("Upload-Length", "6")
	create.Header.Set(upload.TokenHeader, token)
	resp, err := http.DefaultClient.Do(create)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create: status %d", resp.StatusCode)
	}
	location := resp.Header.Get("Location")

	pr, pw := io.Pipe()
	go func() {
		for i := 0; i < 6; i++ {
			_, _ = pw.Write([]byte{'x'})
			time.Sleep(80 * time.Millisecond)
		}
		_ = pw.Close()
	}()
	patch, _ := http.NewRequest(http.MethodPatch, location, pr)
	patch.Header.Set("Tus-Resumable", "1.0.0")
	patch.Header.Set("Upload-Offset", "0")
	patch.Header.Set("Content-Type", "application/offset+octet-stream")
	patch.Header.Set(upload.TokenHeader, token)
	resp, err = http.DefaultClient.Do(patch)
	if err != nil {
		t.Fatalf("a PATCH that outlives the server timeouts failed: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("PATCH status %d, want 204", resp.StatusCode)
	}
}

func TestNegativeHTTPLimitsDisableThem(t *testing.T) {
	app, err := New(Config{Component: plainComponent, HTTP: HTTPConfig{ReadTimeout: -1, WriteTimeout: -1, IdleTimeout: time.Second}})
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	srv := app.Server(":0")
	if srv.ReadTimeout != 0 || srv.WriteTimeout != 0 || srv.IdleTimeout != time.Second || srv.ReadHeaderTimeout != DefaultReadHeaderTimeout {
		t.Fatalf("server = %+v", srv)
	}
}

func TestNeverReturningRenderAnswersSSRWith503(t *testing.T) {
	gate := make(chan struct{})
	defer close(gate)
	var block atomic.Bool
	block.Store(true)
	la := newLifecycleApp(t, func(ctx *rt.Ctx) work.Node {
		if block.Load() {
			<-gate
		}
		return plainComponent(ctx)
	}, func(c *Config) {
		c.SessionConfig.RenderTimeout = 150 * time.Millisecond
		c.Watchdog = WatchdogConfig{Interval: 20 * time.Millisecond, WedgeStep: 400 * time.Millisecond}
	})

	done := make(chan *http.Response, 1)
	go func() {
		resp, _ := la.get(http.DefaultClient, nil)
		done <- resp
	}()
	select {
	case resp := <-done:
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("status %d, want 503", resp.StatusCode)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("SSR stayed parked behind a render that never returns")
	}
	waitUntil(t, "the stuck session to be removed by the watchdog", func() bool { return la.app.registry.Len() == 0 })
	block.Store(false)
}

func TestPendingFloodDoesNotStarveLiveSessionsOrAttaches(t *testing.T) {
	var renders atomic.Int64
	la := newLifecycleApp(t, func(ctx *rt.Ctx) work.Node {
		renders.Add(1)
		return plainComponent(ctx)
	}, func(c *Config) { c.MaxSessions = 10; c.MaxPendingSessions = 3 })

	var constructed atomic.Int64
	registryHook := la.app.sessionConfig.OnClose
	la.app.sessionConfig.OnClose = func(sess *session.LiveSession, reason session.CloseReason) {
		constructed.Add(1)
		registryHook(sess, reason)
	}

	first := la.boot(nil)
	la.boot(nil)
	la.boot(nil)
	before := renders.Load()
	for i := 0; i < 20; i++ {
		resp, _ := la.get(http.DefaultClient, nil)
		if resp.StatusCode != http.StatusServiceUnavailable {
			t.Fatalf("flood request %d: status %d, want 503", i, resp.StatusCode)
		}
	}
	if renders.Load() != before {
		t.Fatal("refused pending requests rendered a session")
	}
	waitUntil(t, "constructed-session accounting", func() bool { return true })
	time.Sleep(50 * time.Millisecond)
	if constructed.Load() != 0 {
		t.Fatalf("%d sessions were built (and closed) for refused requests; admission must happen before construction", constructed.Load())
	}
	if st := la.app.Stats(); st.Rejected.Pending != 20 || st.Sessions.Pending != 3 {
		t.Fatalf("stats = %+v", st)
	}

	c := la.dial(nil)
	defer c.CloseNow()
	if la.join(c, first)["t"] != "joined" {
		t.Fatal("a pending flood blocked an attach")
	}
	if st := la.app.Stats(); st.Sessions.Pending != 2 || st.Sessions.Live != 1 {
		t.Fatalf("stats after attach = %+v", st)
	}
	la.boot(nil)
}

func TestShutdownReportsAWedgedSessionThatTheRegistryAlreadyDropped(t *testing.T) {
	la := newLifecycleApp(t, plainComponent, func(c *Config) {
		c.Watchdog = WatchdogConfig{Interval: 10 * time.Millisecond, SlowStep: time.Hour, WedgeStep: 100 * time.Millisecond}
	})
	b := la.boot(nil)
	sess, _ := la.app.registry.Lookup(session.SessionID(b.SID))
	gate := make(chan struct{})
	stuck := make(chan struct{})
	_ = sess.Session().Post(func() { close(stuck); <-gate })
	<-stuck
	waitUntil(t, "the watchdog to drop the session", func() bool { return la.app.registry.Len() == 0 })
	if la.app.Stats().Stuck != 1 {
		t.Fatalf("stats = %+v", la.app.Stats())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
	defer cancel()
	var shutdownErr *ShutdownError
	if err := la.app.Shutdown(ctx); !errors.As(err, &shutdownErr) || shutdownErr.Stuck != 1 {
		t.Fatalf("Shutdown returned %v while a wedged actor is still running", err)
	}
	close(gate)
	<-sess.Done()
	if err := la.app.Shutdown(context.Background()); err != nil {
		t.Fatalf("Shutdown after the step returned: %v", err)
	}
}

func TestReplacingJoinerThatTimesOutDoesNotKickTheLiveConnection(t *testing.T) {
	var slow atomic.Bool
	la := newLifecycleApp(t, func(ctx *rt.Ctx) work.Node {
		n := headers.UseRequest(ctx).Cookie("n")
		rt.UseEffect(ctx, func() func() {
			_ = n.Get()
			if slow.Load() {
				time.Sleep(500 * time.Millisecond)
			}
			return nil
		})
		return plainComponent(ctx)
	}, func(c *Config) { c.Transport.JoinDeadline = 150 * time.Millisecond })
	b := la.boot(nil)

	first := la.dial(nil)
	defer first.CloseNow()
	if la.join(first, b)["t"] != "joined" {
		t.Fatal("first join failed")
	}
	firstClosed := make(chan websocket.StatusCode, 1)
	go func() { firstClosed <- readCloseCode(t, first) }()

	slow.Store(true)
	second := la.dial(http.Header{"Cookie": {"n=2"}})
	defer second.CloseNow()
	writeJSON(t, second, map[string]any{"t": "join", "sid": b.SID, "tok": b.Tok, "ver": b.Ver})
	if code := readCloseCode(t, second); code != session.CloseJoinTimeout {
		t.Fatalf("second joiner closed with %d, want 4408", code)
	}
	time.Sleep(800 * time.Millisecond)
	slow.Store(false)

	select {
	case code := <-firstClosed:
		t.Fatalf("the live connection was closed with %d by a joiner that never attached", code)
	default:
	}
	sess, _ := la.app.registry.Lookup(session.SessionID(b.SID))
	if sess.State() != session.StateLive {
		t.Fatalf("state = %v", sess.State())
	}
}
