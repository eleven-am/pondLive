package runtime

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

type handlerFixture struct {
	t    *testing.T
	sess *Session
	ctx  *Ctx
}

func newHandlerFixture(t *testing.T) *handlerFixture {
	t.Helper()
	sess := NewSession(Config{SessionID: "sess1", Root: func(*Ctx) work.Node { return work.BuildElement("div") }})
	sess.Start()
	t.Cleanup(sess.Close)
	f := &handlerFixture{t: t, sess: sess}
	onActor(t, sess, func() {
		if _, err := sess.RenderSnapshot(); err != nil {
			t.Fatal(err)
		}
		f.ctx = sess.newCtx(sess.root.NewChild())
	})
	return f
}

func (f *handlerFixture) use(method string, chain ...HandlerFunc) HandlerHandle {
	f.t.Helper()
	var h HandlerHandle
	onActor(f.t, f.sess, func() { h = UseHandler(f.ctx, method, chain...) })
	return h
}

func serve(sess *Session, method, url string, body io.Reader) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, url, body)
	rr := httptest.NewRecorder()
	sess.ServeHTTP(rr, req)
	return rr
}

func TestUseHandlerRegistersAndServes(t *testing.T) {
	f := newHandlerFixture(t)
	h := f.use(http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("ok"))
		return nil
	})

	if !strings.HasPrefix(h.URL(), "/_handlers/sess1/") {
		t.Fatalf("unexpected URL: %s", h.URL())
	}
	rr := serve(f.sess, http.MethodGet, h.URL(), nil)
	if rr.Code != http.StatusTeapot || rr.Body.String() != "ok" {
		t.Fatalf("code=%d body=%q", rr.Code, rr.Body.String())
	}
}

func TestUseHandlerMethodGuard(t *testing.T) {
	f := newHandlerFixture(t)
	h := f.use(http.MethodPost, func(w http.ResponseWriter, r *http.Request) error { return nil })

	rr := serve(f.sess, http.MethodGet, h.URL(), nil)

	if rr.Code != http.StatusMethodNotAllowed || rr.Header().Get("Allow") != http.MethodPost {
		t.Fatalf("code=%d allow=%q", rr.Code, rr.Header().Get("Allow"))
	}
}

func TestHandlerPanicReturns500AndIsReported(t *testing.T) {
	f := newHandlerFixture(t)
	diags := collectDiagnostics(f.sess)
	h := f.use(http.MethodGet, func(w http.ResponseWriter, r *http.Request) error { panic("boom") })

	rr := serve(f.sess, http.MethodGet, h.URL(), nil)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d", rr.Code)
	}
	found := false
	for _, d := range diags() {
		found = found || strings.HasPrefix(d.Phase, "http_handler")
	}
	if !found {
		t.Fatalf("panic not reported: %v", diags())
	}
}

func TestHandlerErrorWithoutWriteIs500AndNilHandlersAreSkipped(t *testing.T) {
	f := newHandlerFixture(t)
	h := f.use(http.MethodGet, nil, func(w http.ResponseWriter, r *http.Request) error { return io.ErrUnexpectedEOF })

	rr := serve(f.sess, http.MethodGet, h.URL(), nil)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d", rr.Code)
	}
}

func TestHandlerDestroyAndOwnerDisposeRemoveTheRoute(t *testing.T) {
	f := newHandlerFixture(t)
	hits := 0
	ok := func(w http.ResponseWriter, r *http.Request) error { hits++; return nil }
	destroyed := f.use(http.MethodGet, ok)
	disposed := f.use(http.MethodGet, ok)

	destroyed.Destroy()
	onActor(t, f.sess, func() {})
	f.ctx.o.Dispose()

	for _, h := range []HandlerHandle{destroyed, disposed} {
		if rr := serve(f.sess, http.MethodGet, "/_handlers/sess1/"+strings.TrimPrefix(h.URL(), "/_handlers/sess1/"), nil); rr.Code != http.StatusNotFound {
			t.Fatalf("route still served: %d", rr.Code)
		}
	}
	if hits != 0 {
		t.Fatalf("hits = %d", hits)
	}
	if len(f.sess.httpHandlers) != 0 {
		t.Fatalf("handler table leaked %d entries", len(f.sess.httpHandlers))
	}
}

func TestServeHTTPRejectsBadPathsAndSessions(t *testing.T) {
	f := newHandlerFixture(t)
	for _, path := range []string{"/", "/_handlers", "/_handlers/sess1", "/_handlers/other/h1", "/other/sess1/h1"} {
		if rr := serve(f.sess, http.MethodGet, path, nil); rr.Code != http.StatusNotFound {
			t.Fatalf("%s: code = %d", path, rr.Code)
		}
	}
	var nilSess *Session
	if rr := serve(nilSess, http.MethodGet, "/_handlers/a/b", nil); rr.Code != http.StatusNotFound {
		t.Fatalf("nil session code = %d", rr.Code)
	}
}

func TestHandlerQueueTimeoutReturns503(t *testing.T) {
	f := newHandlerFixture(t)
	h := f.use(http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		_, _ = w.Write([]byte("ran"))
		return nil
	})
	prev := handlerQueueTimeout
	handlerQueueTimeout = 50 * time.Millisecond
	defer func() { handlerQueueTimeout = prev }()
	release := make(chan struct{})
	blocked := make(chan struct{})
	go func() {
		_ = f.sess.Ask(5*time.Second, func() error {
			close(blocked)
			<-release
			return nil
		})
	}()
	<-blocked

	rr := serve(f.sess, http.MethodGet, h.URL(), nil)
	close(release)

	if rr.Code != http.StatusServiceUnavailable || rr.Header().Get("Retry-After") == "" || strings.Contains(rr.Body.String(), "ran") {
		t.Fatalf("code=%d retry=%q body=%q", rr.Code, rr.Header().Get("Retry-After"), rr.Body.String())
	}
}

func TestHandlerBodyLimits(t *testing.T) {
	f := newHandlerFixture(t)
	var got int
	h := f.use(http.MethodPost, func(w http.ResponseWriter, r *http.Request) error {
		b, _ := io.ReadAll(r.Body)
		got = len(b)
		return nil
	})

	big := bytes.Repeat([]byte("a"), HandlerBodyLimit+1)
	if rr := serve(f.sess, http.MethodPost, h.URL(), bytes.NewReader(big)); rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("known length: %d", rr.Code)
	}
	req := httptest.NewRequest(http.MethodPost, h.URL(), io.MultiReader(bytes.NewReader(big)))
	req.ContentLength = -1
	rr := httptest.NewRecorder()
	f.sess.ServeHTTP(rr, req)
	if rr.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("unknown length: %d", rr.Code)
	}
	if rr := serve(f.sess, http.MethodPost, h.URL(), bytes.NewReader(big[:HandlerBodyLimit])); rr.Code != http.StatusOK || got != HandlerBodyLimit {
		t.Fatalf("at the limit: code=%d got=%d", rr.Code, got)
	}
}

func TestHandlerResponseOverCapReturns500AndDiagnostic(t *testing.T) {
	f := newHandlerFixture(t)
	diags := collectDiagnostics(f.sess)
	h := f.use(http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		chunk := bytes.Repeat([]byte("x"), 1<<20)
		for i := 0; i < 9; i++ {
			if _, err := w.Write(chunk); err != nil {
				return err
			}
		}
		return nil
	})

	rr := serve(f.sess, http.MethodGet, h.URL(), nil)

	if rr.Code != http.StatusInternalServerError {
		t.Fatalf("code = %d", rr.Code)
	}
	if !hasPhase(diags(), "http_handler:response-too-large") {
		t.Fatalf("diagnostics = %v", diags())
	}
}

type stalledWriter struct {
	header  http.Header
	started chan struct{}
	release chan struct{}
	once    sync.Once
}

func (w *stalledWriter) Header() http.Header { return w.header }
func (w *stalledWriter) WriteHeader(int)     {}
func (w *stalledWriter) Write(p []byte) (int, error) {
	w.once.Do(func() { w.started <- struct{}{} })
	<-w.release
	return len(p), nil
}

func TestSlowHandlerClientDoesNotBlockActor(t *testing.T) {
	f := newHandlerFixture(t)
	h := f.use(http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		_, err := w.Write(bytes.Repeat([]byte("x"), 1<<16))
		return err
	})
	sw := &stalledWriter{header: make(http.Header), started: make(chan struct{}, 1), release: make(chan struct{})}
	done := make(chan struct{})
	go func() {
		defer close(done)
		f.sess.ServeHTTP(sw, httptest.NewRequest(http.MethodGet, h.URL(), nil))
	}()
	select {
	case <-sw.started:
	case <-time.After(2 * time.Second):
		t.Fatal("the response never reached the client writer")
	}

	within(t, time.Second, "an actor barrier while the client writer is stalled", func() { onActor(t, f.sess, func() {}) })

	close(sw.release)
	<-done
}

func TestHandlerQueueCapReturns503(t *testing.T) {
	f := newHandlerFixture(t)
	f.sess.SetHandlerQueueCap(1)
	gate := make(chan struct{})
	entered := make(chan struct{}, 1)
	h := f.use(http.MethodGet, func(w http.ResponseWriter, r *http.Request) error {
		entered <- struct{}{}
		<-gate
		return nil
	})
	go serve(f.sess, http.MethodGet, h.URL(), nil)
	<-entered

	rr := serve(f.sess, http.MethodGet, h.URL(), nil)
	close(gate)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("code = %d", rr.Code)
	}
}

func TestSerializePassDoesNotReRegisterHTTPHandlers(t *testing.T) {
	calls := 0
	var seen []string
	sess := NewSession(Config{SessionID: "sess1", Root: func(ctx *Ctx) work.Node {
		h := UseHandler(ctx, http.MethodGet, func(w http.ResponseWriter, r *http.Request) error { calls++; return nil })
		seen = append(seen, h.URL())
		return work.BuildElement("a", work.Attribute("href", h.URL()))
	}})
	sess.Start()
	t.Cleanup(sess.Close)
	onActor(t, sess, func() {
		if _, err := sess.RenderSnapshot(); err != nil {
			t.Fatal(err)
		}
	})
	before := len(sess.httpHandlers)

	onActor(t, sess, func() {
		for i := 0; i < 3; i++ {
			if _, err := sess.ResetStep(); err != nil {
				t.Fatal(err)
			}
		}
	})

	if len(sess.httpHandlers) != before || before != 1 {
		t.Fatalf("handlers %d -> %d", before, len(sess.httpHandlers))
	}
	for _, u := range seen {
		if u != seen[0] {
			t.Fatalf("handler url changed between runs: %v", seen)
		}
	}
	_ = protocol.Diagnostic{}
}
