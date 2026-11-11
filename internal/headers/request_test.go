package headers_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

type probe struct{ req *headers.Request }

func newState(live bool, cookie string) *headers.RequestState {
	h := http.Header{}
	h.Set("User-Agent", "probe/1")
	h.Set("X-Custom", "ignored")
	if cookie != "" {
		h.Set("Cookie", cookie)
	}
	st := headers.NewRequestState(headers.NewRequestInfoFromHeaders(h))
	st.SetIsLive(live)
	return st
}

func start(t *testing.T, st *headers.RequestState, p *probe) *runtimetest.App {
	t.Helper()
	root := func(*runtime.Ctx) work.Node {
		return headers.Provider(headers.ProviderProps{State: st, Children: work.BuildElement("html",
			work.BuildElement("head", headers.Render()),
			work.BuildElement("body", runtime.Component(func(ctx *runtime.Ctx) work.Node {
				p.req = headers.UseRequest(ctx)
				return work.BuildElement("p", work.TextOf(p.req.Cookie("theme")))
			})()),
		)})
	}
	return runtimetest.New(t, root)
}

func TestRequestHeadersAreSignals(t *testing.T) {
	p := &probe{}
	app := start(t, newState(false, ""), p)
	var ua, custom string
	var all http.Header
	app.On(func() {
		ua = p.req.Header("user-agent").Peek()
		custom = p.req.Header("X-Custom").Peek()
		all = p.req.Headers().Peek()
	})
	if ua != "probe/1" || custom != "" {
		t.Fatalf("ua = %q custom = %q", ua, custom)
	}
	all.Set("User-Agent", "mutated")
	app.On(func() {
		if got := p.req.Headers().Peek().Get("User-Agent"); got != "probe/1" {
			t.Errorf("Headers() exposed live storage: %q", got)
		}
	})
}

type headerProbe struct {
	req   *headers.Request
	notes int
}

func startHeaders(t *testing.T, st *headers.RequestState, p *headerProbe) *runtimetest.App {
	t.Helper()
	root := func(*runtime.Ctx) work.Node {
		return headers.Provider(headers.ProviderProps{State: st, Children: work.BuildElement("html",
			work.BuildElement("head", headers.Render()),
			work.BuildElement("body", runtime.Component(func(ctx *runtime.Ctx) work.Node {
				p.req = headers.UseRequest(ctx)
				tenant := p.req.Header("X-Tenant")
				all := p.req.Headers()
				count := runtime.UseComputed(ctx, func() string {
					p.notes++
					return strconv.Itoa(len(all.Get()))
				})
				return work.BuildElement("div",
					work.BuildElement("p", work.TextOf(tenant)),
					work.BuildElement("i", work.TextOf(count)),
				)
			})()),
		)})
	}
	return runtimetest.New(t, root)
}

func tenantState(live bool, tenant string) *headers.RequestState {
	h := http.Header{}
	if tenant != "" {
		h.Set("X-Tenant", tenant)
	}
	st := headers.NewRequestState(headers.NewRequestInfoFromHeaders(h, "X-Tenant"))
	st.SetIsLive(live)
	return st
}

func TestHeaderSignalUpdatesWhenTheUpgradeMergesADifferentValue(t *testing.T) {
	p := &headerProbe{}
	st := tenantState(false, "acme")
	app := startHeaders(t, st, p)
	if !strings.Contains(app.SSR, ">acme<") {
		t.Fatalf("ssr = %s", app.SSR)
	}
	edits := app.Do(func() { st.MergeHeaders(http.Header{"X-Tenant": {"globex"}}) })
	if len(edits) != 1 || edits[0].Op != "t" || edits[0].Value != "globex" {
		t.Fatalf("edits = %+v", edits)
	}
	if !strings.Contains(app.Normalized(), ">globex<") {
		t.Fatalf("dom = %s", app.Normalized())
	}
}

func TestUnchangedHeadersDoNotNotify(t *testing.T) {
	p := &headerProbe{}
	st := tenantState(false, "acme")
	app := startHeaders(t, st, p)
	before := p.notes
	edits := app.Do(func() { st.MergeHeaders(http.Header{"X-Tenant": {"acme"}}) })
	if len(edits) != 0 {
		t.Fatalf("edits = %+v", edits)
	}
	if p.notes != before {
		t.Fatalf("the Headers() computed re-ran for an identical header set: %d -> %d", before, p.notes)
	}
}

func TestHeaderSignalFollowsEveryReconnect(t *testing.T) {
	p := &headerProbe{}
	st := tenantState(false, "")
	app := startHeaders(t, st, p)
	for i, want := range []string{"one", "two", "two", "three"} {
		app.Do(func() { st.MergeHeaders(http.Header{"X-Tenant": {want}}) })
		if !strings.Contains(app.Normalized(), ">"+want+"<") {
			t.Fatalf("reconnect %d: dom = %s", i, app.Normalized())
		}
	}
}

func TestHeaderSignalsRaceWithOffActorMerges(t *testing.T) {
	p := &headerProbe{}
	st := tenantState(false, "")
	app := startHeaders(t, st, p)
	var wg sync.WaitGroup
	for g := 0; g < 4; g++ {
		wg.Add(1)
		go func(g int) {
			defer wg.Done()
			for i := 0; i < 200; i++ {
				st.MergeHeaders(http.Header{"X-Tenant": {strconv.Itoa(g*1000 + i)}})
			}
		}(g)
	}
	wg.Wait()
	app.Sync()
	var got string
	app.On(func() { got = p.req.Header("X-Tenant").Peek() })
	want, _ := st.Get("X-Tenant")
	if got != want {
		t.Fatalf("signal = %q, state = %q", got, want)
	}
	if !strings.Contains(app.Normalized(), ">"+want+"<") {
		t.Fatalf("dom = %s", app.Normalized())
	}
}

func TestCookieIsASignalSeededFromTheRequest(t *testing.T) {
	p := &probe{}
	app := start(t, newState(false, "theme=dark"), p)
	if !strings.Contains(app.SSR, ">dark<") {
		t.Fatalf("ssr = %s", app.SSR)
	}
	var v string
	app.On(func() { v = p.req.Cookie("theme").Peek() })
	if v != "dark" {
		t.Fatalf("cookie = %q", v)
	}
}

func TestCookieSignalFollowsAckedChanges(t *testing.T) {
	p := &probe{}
	st := newState(true, "theme=dark")
	app := start(t, st, p)
	edits := app.Do(func() {
		st.ApplyCookie(&http.Cookie{Name: "theme", Value: "light"}, false)
		st.NotifyChange()
	})
	if len(edits) != 1 || !strings.Contains(app.Normalized(), ">light<") {
		t.Fatalf("edits = %v dom = %s", edits, app.Normalized())
	}
	app.Do(func() {
		st.ApplyCookie(&http.Cookie{Name: "theme"}, true)
		st.NotifyChange()
	})
	if strings.Contains(app.Normalized(), "light") {
		t.Fatalf("dom = %s", app.Normalized())
	}
}

func TestSetCookieLiveUpdatesTheSignalAndSendsTheScriptMessage(t *testing.T) {
	p := &probe{}
	st := newState(true, "")
	app := start(t, st, p)
	var sent []protocol.ScriptPayload
	app.S.Bus.SubscribeAll(func(topic protocol.Topic, event string, data interface{}) {
		if strings.HasPrefix(string(topic), "script:") && event == string(protocol.ScriptSendAction) {
			if pl, ok := data.(protocol.ScriptPayload); ok {
				sent = append(sent, pl)
			}
		}
	})
	app.Do(func() { p.req.SetCookie("theme", "light", &headers.CookieOptions{Path: "/"}) })
	if !strings.Contains(app.Normalized(), ">light<") {
		t.Fatalf("dom = %s", app.Normalized())
	}
	if len(sent) != 1 || sent[0].Event != "setCookie" {
		t.Fatalf("script messages = %+v", sent)
	}
	if hs := st.ResponseHeaders(); len(hs.Values("Set-Cookie")) != 0 {
		t.Fatalf("live SetCookie wrote a response header: %v", hs)
	}
}

func TestSetCookieBeforeLiveWritesTheResponseHeader(t *testing.T) {
	p := &probe{}
	st := newState(false, "")
	app := start(t, st, p)
	app.Do(func() { p.req.SetCookie("theme", "light", nil) })
	got := st.ResponseHeaders().Values("Set-Cookie")
	if len(got) != 1 || !strings.HasPrefix(got[0], "theme=light") {
		t.Fatalf("Set-Cookie = %v", got)
	}
	if !strings.Contains(app.Normalized(), ">light<") {
		t.Fatalf("dom = %s", app.Normalized())
	}
}

func TestSetCookieRoundTripThroughTheTokenHandlers(t *testing.T) {
	p := &probe{}
	st := newState(true, "")
	app := start(t, st, p)
	var payload protocol.ScriptPayload
	app.S.Bus.SubscribeAll(func(topic protocol.Topic, event string, data interface{}) {
		if pl, ok := data.(protocol.ScriptPayload); ok && event == string(protocol.ScriptSendAction) {
			payload = pl
		}
	})
	app.Do(func() { p.req.SetCookie("theme", "light", nil) })
	data, _ := payload.Data.(map[string]any)
	token, _ := data["token"].(string)
	setURL, _ := data["url"].(string)
	ackURL, _ := data["ackUrl"].(string)
	if token == "" || setURL == "" || ackURL == "" {
		t.Fatalf("payload = %+v", payload)
	}
	post := func(path string) *httptest.ResponseRecorder {
		u, _ := url.Parse(path)
		req := httptest.NewRequest(http.MethodPost, u.Path, strings.NewReader(`{"token":"`+token+`"}`))
		rec := httptest.NewRecorder()
		app.S.ServeHTTP(rec, req)
		return rec
	}
	set := post(setURL)
	if set.Code != http.StatusNoContent || !strings.Contains(strings.Join(set.Result().Header.Values("Set-Cookie"), ";"), "theme=light") {
		t.Fatalf("set response = %d %v", set.Code, set.Result().Header)
	}
	if ack := post(ackURL); ack.Code != http.StatusNoContent {
		t.Fatalf("ack = %d", ack.Code)
	}
	if again := post(setURL); again.Code != http.StatusForbidden {
		t.Fatalf("token reuse = %d", again.Code)
	}
}

func TestMissingProviderIsAnError(t *testing.T) {
	root := func(*runtime.Ctx) work.Node {
		return runtime.Component(func(ctx *runtime.Ctx) work.Node {
			headers.UseRequest(ctx)
			return work.BuildElement("p")
		})()
	}
	s := runtime.NewSession(runtime.Config{SessionID: "x", Root: root, DevMode: true})
	runtimetest.Start(t, s)
	_ = s.Ask(5e9, func() error {
		snap, err := s.RenderSnapshot()
		if err == nil && !snap.Crashed {
			t.Error("expected a crash")
		}
		return nil
	})
}
