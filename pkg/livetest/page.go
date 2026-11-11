package livetest

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/pkg"
)

type Page struct {
	t   testing.TB
	app *runtimetest.App
}

type Element struct {
	page *Page
	id   uint32
}

type Option func(*config)

type config struct{ url string }

func WithURL(rawURL string) Option { return func(c *config) { c.url = rawURL } }

func Mount(t testing.TB, root func() pkg.Node, opts ...Option) *Page {
	t.Helper()
	return mount(t, root, true, opts)
}

func MountProd(t testing.TB, root func() pkg.Node, opts ...Option) *Page {
	t.Helper()
	return mount(t, root, false, opts)
}

func mount(t testing.TB, root func() pkg.Node, dev bool, opts []Option) *Page {
	t.Helper()
	cfg := config{url: "/"}
	for _, o := range opts {
		o(&cfg)
	}
	state := headers.NewRequestState(headers.NewRequestInfo(httptest.NewRequest(http.MethodGet, cfg.url, nil)))
	app := runtimetest.New(t, func(*pkg.Ctx) pkg.Node {
		return session.Document(session.DocumentConfig{State: state, Body: root(), DevMode: dev})
	}, runtimetest.Dev(dev))
	app.On(func() { app.S.SetConnected(true) })
	app.Sync()
	return &Page{t: t, app: app}
}

func (p *Page) HTML() string { return p.app.Normalized() }

func (p *Page) Do(fn func()) { p.t.Helper(); p.app.Do(fn) }

func (p *Page) Diagnostics(phase string) []pkg.Diagnostic { return p.app.Diagnostics(phase) }

func (p *Page) ByID(id string) *Element {
	p.t.Helper()
	return &Element{page: p, id: p.app.FindByAttr("id", id)}
}

func (e *Element) echoed(edits []protocol.Edit, names ...string) bool {
	for _, edit := range edits {
		if edit.ID != e.id || (edit.Op != protocol.OpAttr && edit.Op != protocol.OpRemoveAttr) {
			continue
		}
		for _, name := range names {
			if edit.Name == name {
				return true
			}
		}
	}
	return false
}

func (e *Element) Type(text string) (echoed bool) {
	e.page.t.Helper()
	return e.echoed(e.page.app.Type(e.id, text), "value")
}

func (e *Element) Check(on bool) (echoed bool) {
	e.page.t.Helper()
	return e.echoed(e.page.app.Check(e.id, on), "checked")
}

func (e *Element) Choose(values ...string) (echoed bool) {
	e.page.t.Helper()
	return e.echoed(e.page.app.Choose(e.id, values...), "value", protocol.AttrSelection)
}

func (e *Element) Toggle(open bool) (echoed bool) {
	e.page.t.Helper()
	return e.echoed(e.page.app.Toggle(e.id, open), "open")
}

func (e *Element) Click() { e.page.t.Helper(); e.dispatch("click", nil) }

func (e *Element) Press(key string) {
	e.page.t.Helper()
	e.dispatch("keydown", map[string]any{"key": key})
}

func (e *Element) dispatch(event string, payload map[string]any) {
	e.page.t.Helper()
	if err := e.page.app.Dispatch(e.id, event, payload); err != nil {
		e.page.t.Errorf("livetest: %s: %v", event, err)
	}
}

func (e *Element) Push(prop string, value any) {
	e.page.t.Helper()
	e.page.app.PushValue(e.id, prop, value)
	e.page.app.Sync()
}

func (e *Element) PushCurrentTime(seconds float64) { e.page.t.Helper(); e.Push("currentTime", seconds) }

func (e *Element) PushRect(r pkg.DOMRect) {
	e.page.t.Helper()
	e.Push("rect", protocol.FloatMap(protocol.RectKeys, r.X, r.Y, r.Width, r.Height, r.Top, r.Right, r.Bottom, r.Left))
}

func (e *Element) PushScroll(m pkg.ScrollMetrics) {
	e.page.t.Helper()
	e.Push("scroll", protocol.FloatMap(protocol.ScrollKeys, m.ScrollTop, m.ScrollLeft, m.ScrollHeight, m.ScrollWidth, m.ClientHeight, m.ClientWidth))
}
