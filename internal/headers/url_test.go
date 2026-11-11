package headers_test

import (
	"net/http"
	"net/url"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/route"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

type urlProbe struct {
	req *headers.Request
}

type busLog struct {
	mu     sync.Mutex
	events []string
	data   []any
}

func (l *busLog) record(app *runtimetest.App) {
	app.S.Bus.SubscribeAll(func(topic protocol.Topic, event string, data interface{}) {
		if topic != protocol.RouteHandler {
			return
		}
		l.mu.Lock()
		defer l.mu.Unlock()
		l.events = append(l.events, event)
		l.data = append(l.data, data)
	})
}

func (l *busLog) snapshot() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return append([]string(nil), l.events...)
}

func startURL(t *testing.T, target string, p *urlProbe) (*runtimetest.App, *headers.RequestState) {
	t.Helper()
	return startURLWith(t, target, p, true, "")
}

func startURLWith(t *testing.T, target string, p *urlProbe, connected bool, host string) (*runtimetest.App, *headers.RequestState) {
	t.Helper()
	u, _ := url.Parse(target)
	st := headers.NewRequestState(&headers.RequestInfo{Host: host, Path: u.Path, Query: u.Query(), Headers: http.Header{}})
	root := func(*runtime.Ctx) work.Node {
		return headers.Provider(headers.ProviderProps{State: st, Children: work.BuildElement("html",
			work.BuildElement("head", headers.Render()),
			work.BuildElement("body", runtime.Component(func(ctx *runtime.Ctx) work.Node {
				p.req = headers.UseRequest(ctx)
				path := runtime.UseComputed(ctx, func() string { return p.req.URL().Get().Path })
				return work.BuildElement("p", work.TextOf(path))
			})()),
		)})
	}
	app := runtimetest.New(t, root)
	if connected {
		app.On(func() { app.S.SetConnected(true) })
		app.Sync()
	}
	return app, st
}

func TestRequestURLStartsFromTheHTTPRequestAndIsCanonical(t *testing.T) {
	p := &urlProbe{}
	app, st := startURL(t, "/a//b/?z=2&z=1", p)
	var got route.Location
	app.On(func() { got = p.req.URL().Peek() })
	if got.Path != "/a/b" || len(got.Query["z"]) != 2 || got.Query["z"][0] != "1" {
		t.Fatalf("url = %+v", got)
	}
	if st.URL().Path != "/a/b" {
		t.Fatalf("state url = %+v", st.URL())
	}
}

func TestRequestURLWritesReachTheStateAndTheDOM(t *testing.T) {
	p := &urlProbe{}
	app, st := startURL(t, "/start", p)
	edits := app.Do(func() { p.req.Push("/next//?q=1") })
	if len(edits) == 0 {
		t.Fatal("no edit frame for a URL change")
	}
	if got := st.URL(); got.Path != "/next" || got.Query.Get("q") != "1" {
		t.Fatalf("state url = %+v", got)
	}
	if !strings.Contains(app.Normalized(), ">/next<") {
		t.Fatalf("dom = %s", app.Normalized())
	}
	edits = app.Do(func() { p.req.Replace("/next?q=1") })
	if len(edits) != 0 {
		t.Fatalf("an unchanged URL produced edits: %v", edits)
	}
}

func TestRequestURLIsACopyAndHasNoSetter(t *testing.T) {
	p := &urlProbe{}
	app, st := startURL(t, "/x?a=1", p)
	app.On(func() {
		got := p.req.URL().Peek()
		got.Query.Set("a", "mutated")
		got.Path = "/mutated"
	})
	if got := st.URL(); got.Path != "/x" || got.Query.Get("a") != "1" {
		t.Fatalf("URL() exposed storage: %+v", got)
	}
	typ := reflect.TypeOf(p.req)
	for i := 0; i < typ.NumMethod(); i++ {
		switch name := typ.Method(i).Name; name {
		case "Header", "Headers", "Cookie", "SetCookie", "URL", "Connected", "Push", "Replace", "Back", "Forward":
		default:
			t.Errorf("unexpected Request method %q", name)
		}
	}
}

func TestRequestURLRefreshesOnNotifyChange(t *testing.T) {
	p := &urlProbe{}
	app, st := startURL(t, "/x", p)
	app.Do(func() { st.MergeHeaders(http.Header{"User-Agent": {"u"}}) })
	var got route.Location
	app.On(func() { got = p.req.URL().Peek() })
	if got.Path != "/x" {
		t.Fatalf("url after merge = %+v", got)
	}
}

func TestRequestURLRacesWithOffActorWrites(t *testing.T) {
	p := &urlProbe{}
	app, st := startURL(t, "/start", p)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 100; i++ {
				p.req.Push("/p" + string(rune('a'+g)))
				_ = st.URL()
			}
		}(g)
	}
	wg.Wait()
	app.Sync()
	var got route.Location
	app.On(func() { got = p.req.URL().Peek() })
	if got.Path != st.URL().Path {
		t.Fatalf("signal %q, state %q", got.Path, st.URL().Path)
	}
}

func TestPushAndReplaceBeforeConnectAreHTTPRedirects(t *testing.T) {
	for _, tc := range []struct {
		name string
		href string
		do   func(*headers.Request, string)
		want string
		code int
	}{
		{"push", "/a?x=1", func(r *headers.Request, h string) { r.Push(h) }, "/a?x=1", http.StatusFound},
		{"replace", "/a?x=1", func(r *headers.Request, h string) { r.Replace(h) }, "/a?x=1", http.StatusSeeOther},
		{"replace status", "/a", func(r *headers.Request, h string) { r.Replace(h, headers.Status(http.StatusMovedPermanently)) }, "/a", http.StatusMovedPermanently},
		{"push cross-origin", "https://other.example/z?q=1", func(r *headers.Request, h string) { r.Push(h) }, "https://other.example/z?q=1", http.StatusFound},
		{"replace cross-origin status", "https://other.example/z", func(r *headers.Request, h string) { r.Replace(h, headers.Status(307)) }, "https://other.example/z", 307},
		{"push same-origin absolute", "http://example.com/n#h", func(r *headers.Request, h string) { r.Push(h) }, "/n#h", http.StatusFound},
	} {
		p := &urlProbe{}
		app, st := startURLWith(t, "/x", p, false, "example.com")
		app.On(func() { tc.do(p.req, tc.href) })
		to, code, ok := st.Redirect()
		if !ok || to != tc.want || code != tc.code {
			t.Fatalf("%s: redirect = %q %d %v", tc.name, to, code, ok)
		}
		if st.URL().Path != "/x" {
			t.Fatalf("%s: url moved before connect: %+v", tc.name, st.URL())
		}
	}
}

func TestPushReplaceBackForwardTellTheBrowser(t *testing.T) {
	p := &urlProbe{}
	app, _ := startURL(t, "/x", p)
	var log busLog
	log.record(app)
	app.Do(func() { p.req.Push("/one"); p.req.Replace("/two"); p.req.Back(); p.req.Forward() })
	got := log.snapshot()
	want := []string{"push", "replace", "back", "forward"}
	if len(got) != len(want) {
		t.Fatalf("events = %v", got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v", got)
		}
	}
}

func TestLiveSameOriginPushAndReplaceStayInApp(t *testing.T) {
	for _, replace := range []bool{false, true} {
		for _, target := range []string{"/next?q=1#h", "http://example.com/next?q=1#h", "//example.com/next?q=1#h"} {
			p := &urlProbe{}
			app, st := startURLWith(t, "/x", p, true, "example.com")
			var log busLog
			log.record(app)
			edits := app.Do(func() {
				if replace {
					p.req.Replace(target)
				} else {
					p.req.Push(target)
				}
			})
			if len(edits) == 0 {
				t.Fatalf("%s: no edit frame", target)
			}
			if got := st.URL(); got.Path != "/next" || got.Query.Get("q") != "1" || got.Hash != "h" {
				t.Fatalf("%s: url = %+v", target, got)
			}
			want := "push"
			if replace {
				want = "replace"
			}
			if ev := log.snapshot(); len(ev) != 1 || ev[0] != want {
				t.Fatalf("%s: events = %v want %s", target, ev, want)
			}
			if !strings.Contains(app.Normalized(), ">/next<") {
				t.Fatalf("%s: dom = %s", target, app.Normalized())
			}
		}
	}
}

func TestLiveCrossOriginPushAndReplaceAskTheBrowserToNavigate(t *testing.T) {
	for _, replace := range []bool{false, true} {
		p := &urlProbe{}
		app, st := startURLWith(t, "/x", p, true, "example.com")
		var log busLog
		log.record(app)
		app.Do(func() {
			if replace {
				p.req.Replace("https://other.example/path?a=1")
			} else {
				p.req.Push("https://other.example/path?a=1")
			}
		})
		ev := log.snapshot()
		if len(ev) != 1 || ev[0] != "redirect" {
			t.Fatalf("replace=%v: events = %v", replace, ev)
		}
		payload, ok := log.data[0].(protocol.RouterRedirectPayload)
		if !ok || payload.URL != "https://other.example/path?a=1" {
			t.Fatalf("payload = %#v", log.data[0])
		}
		if st.URL().Path != "/x" {
			t.Fatalf("url moved: %+v", st.URL())
		}
	}
}

func TestUnsafeURLsAreDroppedWithADiagnostic(t *testing.T) {
	bad := []string{"javascript:alert(1)", "JaVaScRiPt:alert(1)", "data:text/html,x", "vbscript:x", "mailto:a@b.c", "file:///etc/passwd", " javascript:alert(1)", "java\nscript:x", "/\\evil.example", "", "http://[::1", "/a\u007fb", "/a\u00a0b", "/a\u2028b", "/a\u3000b", "/a\ufeffb"}
	for _, connected := range []bool{false, true} {
		p := &urlProbe{}
		app, st := startURLWith(t, "/x", p, connected, "example.com")
		var log busLog
		log.record(app)
		var diags []protocol.Diagnostic
		app.S.Bus.SubscribeToDiagnostics(func(d protocol.Diagnostic) { diags = append(diags, d) })
		for _, target := range bad {
			app.Do(func() { p.req.Push(target); p.req.Replace(target) })
		}
		app.Do(func() { p.req.Replace("/ok", headers.Status(200)); p.req.Replace("/ok", headers.Status(404)) })
		want := 2*len(bad) + 2
		count := 0
		for _, d := range diags {
			if d.Phase == "navigation:unsafe-url" {
				count++
			}
		}
		if count != want {
			t.Errorf("connected=%v: %d diagnostics, want %d", connected, count, want)
		}
		if _, _, ok := st.Redirect(); ok {
			t.Errorf("connected=%v: an unsafe target was recorded", connected)
		}
		if ev := log.snapshot(); len(ev) != 0 || st.URL().Path != "/x" {
			t.Errorf("connected=%v: events = %v url = %+v", connected, ev, st.URL())
		}
	}
}
