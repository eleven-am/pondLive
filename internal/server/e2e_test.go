package server

import (
	"fmt"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/coder/websocket"
	"go.uber.org/goleak"
	"golang.org/x/net/html"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/router"
	rt "github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/work"
)

func e2eButton(label string, fn func()) work.Node {
	return work.BuildElement("button", work.Attribute("id", label), work.On("click", func(work.Event) { fn() }), work.NewText(label))
}

func e2ePage(name string) func() work.Node {
	return rt.Component(func(ctx *rt.Ctx) work.Node {
		r := router.UseRouter(ctx)
		label := rt.UseComputed(ctx, func() string { return name + ":" + r.Params.Get()["id"] })
		return work.BuildElement("section", work.Attribute("id", name), work.TextOf(label))
	})
}

func e2eApp(ctx *rt.Ctx) work.Node {
	count := rt.UseSignal(ctx, 0)
	show := rt.UseSignal(ctx, true)
	rows := rt.UseSignal(ctx, []string{"a", "b", "c"})
	r := router.UseRouter(ctx)
	next := 0

	counter := rt.UseComputed(ctx, func() string { return fmt.Sprintf("count=%d", count.Get()) })
	return work.BuildElement("div",
		e2eButton("inc", func() { count.Update(func(n int) int { return n + 1 }) }),
		work.BuildElement("p", work.Attribute("id", "counter"), work.TextOf(counter)),
		e2eButton("toggle", func() { show.Update(func(b bool) bool { return !b }) }),
		rt.Show(show, func() work.Node { return work.BuildElement("span", work.NewText("shown")) }),
		e2eButton("add", func() {
			next++
			rows.Update(func(l []string) []string { return append(append([]string(nil), l...), fmt.Sprintf("r%d", next)) })
		}),
		e2eButton("reverse", func() {
			rows.Update(func(l []string) []string {
				out := make([]string, len(l))
				for i, v := range l {
					out[len(l)-1-i] = v
				}
				return out
			})
		}),
		e2eButton("drop", func() {
			rows.Update(func(l []string) []string {
				if len(l) == 0 {
					return l
				}
				return append([]string(nil), l[1:]...)
			})
		}),
		e2eButton("about", func() { r.Navigate("/about") }),
		e2eButton("query", func() { r.SetQuery("q", fmt.Sprint(count.Peek())) }),
		work.BuildElement("ul", rt.For(rt.Items(rows, func(s string) string { return s }), rt.PropsComponent(func(_ *rt.Ctx, item rt.ReadSignal[string]) work.Node {
			return work.BuildElement("li", work.NewText(item.Peek()))
		}))),
		router.Routes(
			router.Route("/", e2ePage("home")),
			router.Route("/about", e2ePage("about")),
			router.Route("/users/:id", e2ePage("user")),
		),
	)
}

type e2eClient struct {
	t       *testing.T
	h       *liveHarness
	b       bootDoc
	conn    *websocket.Conn
	in      *frameStream
	model   *runtimetest.Model
	cs      int
	acked   int
	seq     uint64
	resets  int
	frames  int
	lastRst string
}

func decodeEdits(t *testing.T, data any) []protocol.Edit {
	t.Helper()
	list, ok := data.([]any)
	if !ok {
		t.Fatalf("frame data is not a list: %v", data)
	}
	u32 := func(v any) uint32 { return uint32(v.(float64)) }
	out := make([]protocol.Edit, 0, len(list))
	for _, raw := range list {
		a := raw.([]any)
		op := protocol.Op(a[0].(string))
		id := u32(a[1])
		switch op {
		case protocol.OpText:
			out = append(out, protocol.TextEdit(id, a[2].(string)))
		case protocol.OpAttr:
			out = append(out, protocol.AttrEdit(id, a[2].(string), a[3].(string)))
		case protocol.OpRemoveAttr:
			out = append(out, protocol.RemoveAttrEdit(id, a[2].(string)))
		case protocol.OpStyle:
			out = append(out, protocol.StyleEdit(id, a[2].(string), a[3].(string)))
		case protocol.OpHTML:
			out = append(out, protocol.HTMLEdit(id, a[2].(string)))
		case protocol.OpInsert:
			out = append(out, protocol.InsertEdit(id, u32(a[2]), a[3].(string)))
		case protocol.OpMove:
			out = append(out, protocol.MoveEdit(id, u32(a[2]), u32(a[3])))
		case protocol.OpRemove:
			out = append(out, protocol.RemoveEdit(id))
		case protocol.OpClear:
			out = append(out, protocol.ClearEdit(id))
		default:
			t.Fatalf("unknown op %q", op)
		}
	}
	return out
}

func newE2EClient(t *testing.T, h *liveHarness) *e2eClient {
	t.Helper()
	b := h.boot()
	model, err := runtimetest.NewModel(strings.TrimPrefix(stripBootScript(b.HTML), "<!DOCTYPE html>"))
	if err != nil {
		t.Fatalf("ssr is not indexable: %v", err)
	}
	c := &e2eClient{t: t, h: h, b: b, model: model, seq: b.Seq}
	c.conn = h.dialOK()
	joined := h.join(c.conn, b)
	if joined["t"] != "joined" {
		t.Fatalf("join: %v", joined)
	}
	c.in = streamFrames(c.conn)
	return c
}

func (c *e2eClient) idOf(tag, attr, val string) uint32 {
	c.t.Helper()
	for _, id := range c.model.IDs() {
		n := c.model.Node(id)
		if n == nil || n.Type != html.ElementNode || n.Data != tag {
			continue
		}
		for _, a := range n.Attr {
			if a.Key == attr && a.Val == val {
				return id
			}
		}
	}
	c.t.Fatalf("no <%s %s=%q> in %s", tag, attr, val, c.model.HTML())
	return 0
}

func (c *e2eClient) click(label string) {
	c.t.Helper()
	c.cs++
	writeJSON(c.t, c.conn, map[string]any{"t": "h", "i": c.idOf("button", "id", label), "sid": c.b.SID, "a": "click", "p": map[string]any{}, "cs": c.cs})
}

func (c *e2eClient) popstate(path, query string) {
	c.t.Helper()
	c.cs++
	writeJSON(c.t, c.conn, map[string]any{"t": "router", "sid": c.b.SID, "a": "popstate", "p": map[string]any{"path": path, "query": query, "hash": ""}, "cs": c.cs})
}

func (c *e2eClient) apply(m map[string]any) (reset string, isReset bool) {
	c.t.Helper()
	if m["seq"] == nil {
		return "", false
	}
	c.seq = uint64(m["seq"].(float64))
	if m["topic"] != "frame" {
		return "", false
	}
	switch m["event"] {
	case "patch":
		if err := c.model.Apply(decodeEdits(c.t, m["data"])); err != nil {
			c.t.Fatalf("client rejected frame %v: %v", m["data"], err)
		}
		c.frames++
	case "reset":
		payload := m["data"].(map[string]any)
		return payload["html"].(string), true
	}
	return "", false
}

func (c *e2eClient) next() map[string]any {
	c.t.Helper()
	m := c.in.next(c.t)
	if m["t"] == "evack" {
		c.acked = int(m["cs"].(float64))
	}
	return m
}

func (c *e2eClient) settle() {
	c.t.Helper()
	for c.acked < c.cs {
		m := c.next()
		if m["t"] == "evack" {
			continue
		}
		if _, isReset := c.apply(m); isReset {
			c.t.Fatalf("unexpected reset while settling")
		}
	}
}

func (c *e2eClient) forceReset() string {
	c.t.Helper()
	c.settle()
	if err := c.h.session(c.b.SID).Flush(); err != nil {
		c.t.Fatalf("flush barrier: %v", err)
	}
	writeJSON(c.t, c.conn, map[string]any{"t": "resync", "sid": c.b.SID, "seq": c.seq, "reason": "test"})
	for {
		m := c.next()
		if resetHTML, ok := c.apply(m); ok {
			c.resets++
			return resetHTML
		}
	}
}

func (c *e2eClient) checkReset() {
	c.t.Helper()
	resetHTML := c.forceReset()
	want, err := runtimetest.Canonical(resetHTML)
	if err != nil {
		c.t.Fatalf("reset html is not parseable: %v", err)
	}
	if got := c.model.HTML(); got != want {
		at := 0
		for at < len(got) && at < len(want) && got[at] == want[at] {
			at++
		}
		lo := max(0, at-80)
		c.t.Fatalf("reset html differs from the live DOM at byte %d\nlive:  ...%s\nreset: ...%s", at, got[lo:min(len(got), at+120)], want[lo:min(len(want), at+120)])
	}
}

func e2eHarness(t *testing.T) *liveHarness {
	return newReplayHarness(t, e2eApp, func(c *Config) {
		c.SessionConfig = &session.Config{ResyncInterval: time.Microsecond}
		var seq atomic.Int64
		c.IDGenerator = func(*http.Request) (session.SessionID, error) {
			return session.SessionID(fmt.Sprintf("e2e-%d", seq.Add(1))), nil
		}
	})
}

func TestEndToEndClickProducesAnEditFrameThatAppliesToTheModel(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := e2eHarness(t)
	defer h.close()
	c := newE2EClient(t, h)
	defer c.conn.CloseNow()

	if !strings.Contains(c.model.HTML(), "count=0") {
		t.Fatalf("ssr = %s", c.model.HTML())
	}
	c.click("inc")
	var m map[string]any
	for m == nil || m["seq"] == nil {
		m = c.next()
	}
	if m["event"] != "patch" {
		t.Fatalf("frame = %v", m)
	}
	edits := decodeEdits(t, m["data"])
	if len(edits) != 1 || edits[0].Op != protocol.OpText || edits[0].Value != "count=1" {
		t.Fatalf("edits = %+v, want one t edit", edits)
	}
	c.apply(m)
	if !strings.Contains(c.model.HTML(), "count=1") {
		t.Fatalf("model = %s", c.model.HTML())
	}
	c.checkReset()
}

func TestForcedResetEqualsTheLiveDOMAfterListRouterAndQueryChanges(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := e2eHarness(t)
	defer h.close()
	c := newE2EClient(t, h)
	defer c.conn.CloseNow()

	for _, label := range []string{"add", "add", "reverse", "toggle", "about", "drop", "query", "inc"} {
		c.click(label)
	}
	c.popstate("/users/7", "")
	c.checkReset()
	if !strings.Contains(c.model.HTML(), "user:7") {
		t.Fatalf("model = %s", c.model.HTML())
	}
	if c.resets != 1 {
		t.Fatalf("resets = %d", c.resets)
	}
}

func TestResetHTMLEqualsLiveAfterRandomSteps(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := e2eHarness(t)
	defer h.close()
	c := newE2EClient(t, h)
	defer c.conn.CloseNow()

	rng := rand.New(rand.NewSource(20260211))
	buttons := []string{"inc", "toggle", "add", "reverse", "drop", "about", "query"}
	paths := []string{"/", "/about", "/users/1", "/users/2", "/users/3", "/nowhere"}
	for step := 0; step < 50; step++ {
		for k := rng.Intn(3) + 1; k > 0; k-- {
			if rng.Intn(4) == 0 {
				c.popstate(paths[rng.Intn(len(paths))], "")
			} else {
				c.click(buttons[rng.Intn(len(buttons))])
			}
		}
		c.checkReset()
	}
	if c.resets != 50 || c.frames == 0 {
		t.Fatalf("resets = %d frames = %d", c.resets, c.frames)
	}
}

func TestConcurrentSessionsKeepIndependentModels(t *testing.T) {
	defer goleak.VerifyNone(t, goleak.IgnoreCurrent())
	h := e2eHarness(t)
	defer h.close()
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			c := newE2EClient(t, h)
			defer c.conn.CloseNow()
			for n := 0; n < i+1; n++ {
				c.click("inc")
			}
			c.checkReset()
			if !strings.Contains(c.model.HTML(), fmt.Sprintf("count=%d", i+1)) {
				t.Errorf("client %d model = %s", i, c.model.HTML())
			}
		}(i)
	}
	wg.Wait()
}
