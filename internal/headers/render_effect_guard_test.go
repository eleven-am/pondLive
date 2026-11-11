package headers_test

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

type navCase struct {
	op   string
	call func(*headers.Request)
}

func navCases() []navCase {
	return []navCase{
		{"req.Push", func(r *headers.Request) { r.Push("/next") }},
		{"req.Replace", func(r *headers.Request) { r.Replace("/next") }},
		{"req.Back", func(r *headers.Request) { r.Back() }},
		{"req.Forward", func(r *headers.Request) { r.Forward() }},
	}
}

func guardApp(t *testing.T, dev bool, call func(*headers.Request)) (*runtimetest.App, *headers.RequestState, *runtime.Boundary) {
	t.Helper()
	st := headers.NewRequestState(&headers.RequestInfo{Path: "/", Headers: http.Header{}})
	var boundary *runtime.Boundary
	var show *runtime.Signal[bool]
	app := runtimetest.New(t, func(ctx *runtime.Ctx) work.Node {
		boundary = runtime.UseErrorBoundary(ctx)
		show = runtime.UseSignal(ctx, false)
		return headers.Provider(headers.ProviderProps{State: st, Children: work.BuildElement("div",
			runtime.Show(show, func() work.Node {
				return runtime.Component(func(ctx *runtime.Ctx) work.Node {
					req := headers.UseRequest(ctx)
					runtime.UseRenderEffect(ctx, func() func() { call(req); return nil })
					return work.BuildElement("p", work.NewText("x"))
				})()
			}))})
	}, runtimetest.Dev(dev))
	app.Do(func() {
		app.S.SetConnected(true)
		show.Set(true)
	})
	return app, st, boundary
}

func TestRenderEffectCannotNavigateInDev(t *testing.T) {
	for _, tc := range navCases() {
		t.Run(tc.op, func(t *testing.T) {
			app, st, boundary := guardApp(t, true, tc.call)
			var msgs []string
			app.On(func() {
				for _, e := range boundary.Errors().Peek().All() {
					msgs = append(msgs, e.Message)
				}
			})
			if len(msgs) != 1 || !strings.Contains(msgs[0], tc.op) || !strings.Contains(msgs[0], "UseRenderEffect") {
				t.Fatalf("errors = %v", msgs)
			}
			if _, _, has := st.Redirect(); has {
				t.Fatal("the refused navigation recorded a redirect")
			}
		})
	}
}

func TestRenderEffectNavigationIsDroppedWithADiagnosticInProd(t *testing.T) {
	for _, tc := range navCases() {
		t.Run(tc.op, func(t *testing.T) {
			app, st, boundary := guardApp(t, false, tc.call)
			var n int
			app.On(func() { n = boundary.Errors().Peek().Count() })
			if n != 0 {
				t.Fatalf("prod raised %d errors", n)
			}
			if _, _, has := st.Redirect(); has {
				t.Fatal("the dropped navigation recorded a redirect")
			}
			diags := app.Diagnostics("warning:RENDER_EFFECT_FORBIDDEN_CALL")
			if len(diags) != 1 || diags[0].Metadata["call"] != tc.op {
				t.Fatalf("diagnostics = %v", diags)
			}
		})
	}
}

func TestNavigationOutsideARenderEffectIsUnchanged(t *testing.T) {
	var p urlProbe
	app, _ := startURL(t, "/", &p)
	app.Do(func() { p.req.Push("/next") })
	if got := app.Normalized(); !strings.Contains(got, "/next") {
		t.Fatalf("html = %s", got)
	}
}

func ssrApp(t *testing.T, dev bool, call func(*headers.Request)) (*headers.RequestState, *runtime.Session) {
	t.Helper()
	st := headers.NewRequestState(&headers.RequestInfo{Path: "/", Headers: http.Header{}})
	root := func(*runtime.Ctx) work.Node {
		return headers.Provider(headers.ProviderProps{State: st, Children: work.BuildElement("div",
			runtime.Component(func(ctx *runtime.Ctx) work.Node {
				req := headers.UseRequest(ctx)
				runtime.UseRenderEffect(ctx, func() func() { call(req); return nil })
				return work.BuildElement("p", work.NewText("x"))
			})())})
	}
	s := runtime.NewSession(runtime.Config{SessionID: "ssr", Root: root, DevMode: dev})
	s.DeferEffects()
	runtimetest.Start(t, s)
	if err := s.Ask(5*time.Second, func() error {
		_, err := s.RenderSnapshot()
		return err
	}); err != nil {
		t.Fatal(err)
	}
	return st, s
}

func TestRenderEffectNavigationBeforeConnectIsAnHTTPRedirect(t *testing.T) {
	for _, dev := range []bool{true, false} {
		for _, tc := range navCases()[:2] {
			name := tc.op + "/prod"
			if dev {
				name = tc.op + "/dev"
			}
			t.Run(name, func(t *testing.T) {
				st, _ := ssrApp(t, dev, tc.call)
				url, code, has := st.Redirect()
				if !has || url != "/next" || code != map[string]int{"req.Push": http.StatusFound, "req.Replace": http.StatusSeeOther}[tc.op] {
					t.Fatalf("redirect = %q %d %v", url, code, has)
				}
			})
		}
	}
}

func TestHistoryCommandsBeforeConnectPublishNothing(t *testing.T) {
	for _, tc := range navCases()[2:] {
		t.Run(tc.op, func(t *testing.T) {
			var frames int
			st := headers.NewRequestState(&headers.RequestInfo{Path: "/", Headers: http.Header{}})
			root := func(*runtime.Ctx) work.Node {
				return headers.Provider(headers.ProviderProps{State: st, Children: runtime.Component(func(ctx *runtime.Ctx) work.Node {
					req := headers.UseRequest(ctx)
					runtime.UseRenderEffect(ctx, func() func() { tc.call(req); return nil })
					return work.BuildElement("p")
				})()})
			}
			s := runtime.NewSession(runtime.Config{SessionID: "ssr", Root: root})
			s.Bus.SubscribeToRouterCommands(func(protocol.RouterServerAction, any) { frames++ })
			s.DeferEffects()
			runtimetest.Start(t, s)
			if err := s.Ask(5*time.Second, func() error { _, err := s.RenderSnapshot(); return err }); err != nil {
				t.Fatal(err)
			}
			if frames != 0 {
				t.Fatalf("%d router commands were published before the socket connected", frames)
			}
		})
	}
}
