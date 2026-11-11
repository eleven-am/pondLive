package runtime_test

import (
	goruntime "runtime"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestScriptIDIsStableAcrossSerializeAndAttachWritesPscript(t *testing.T) {
	var handle runtime.ScriptHandle
	var got []interface{}
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		handle = runtime.UseScript(ctx, "  el.addEventListener('x', () => {}) // c\n")
		handle.On("ping", func(d interface{}) { got = append(got, d) })
		return el("div", runtime.AttachItem(handle))
	})
	id := handle.ID()
	if id == "" || !strings.Contains(app.SSR, `data-pscript="{&quot;i&quot;:&quot;`+id+`&quot;,&quot;s&quot;:&quot;el.addEventListener('x',()=&gt;{})`) {
		t.Fatalf("id=%q ssr=%s", id, app.SSR)
	}

	app.On(func() {
		app.S.Bus.Publish(protocol.Topic("script:"+id+":ping"), string(protocol.ScriptMessageAction), protocol.ScriptPayload{ScriptID: id, Event: "ping", Data: 7.0})
	})
	app.Sync()

	if len(got) != 1 || got[0] != 7.0 {
		t.Fatalf("script message not delivered exactly once (serialize must not subscribe again): %v", got)
	}
}

func TestScriptSubscriptionsAreOwnerCleanups(t *testing.T) {
	var flag *runtime.Signal[bool]
	var id string
	child := runtime.Component(func(ctx *Ctx) Node {
		h := runtime.UseScript(ctx, "x")
		id = h.ID()
		h.On("e", func(interface{}) {})
		return el("p", runtime.AttachItem(h))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		return el("div", runtime.Show(flag, func() Node { return child() }))
	})
	topic := protocol.Topic("script:" + id + ":e")
	if app.S.Bus.SubscriberCount(topic) != 1 {
		t.Fatalf("subscribers = %d", app.S.Bus.SubscriberCount(topic))
	}

	app.Do(func() { flag.Set(false) })

	if app.S.Bus.SubscriberCount(topic) != 0 {
		t.Fatal("script subscription outlived its owner")
	}
}

func TestPortalContentLandsInTheTargetAndLeavesWithItsCreator(t *testing.T) {
	var flag *runtime.Signal[bool]
	modal := runtime.Component(func(ctx *Ctx) Node {
		return frag(el("span", txt("inline")), work.NewPortal(el("dialog", txt("modal"))))
	})
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		flag = runtime.UseSignal(ctx, true)
		return el("div", runtime.Show(flag, func() Node { return el("main", modal()) }), el("div", work.NewPortalTarget()))
	})
	if !strings.Contains(app.Normalized(), "<dialog>modal</dialog>") || !strings.Contains(app.Normalized(), "<span>inline</span>") {
		t.Fatalf("portal content missing: %s", app.Normalized())
	}

	app.Do(func() { flag.Set(false) })

	if strings.Contains(app.Normalized(), "dialog") {
		t.Fatalf("portal entry outlived its creator: %s", app.Normalized())
	}
}

func TestConnectedSignal(t *testing.T) {
	var connected runtime.ReadSignal[bool]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		connected = runtime.SessionOf(ctx).Connected()
		return el("p", bound(connected))
	})
	requireEqual(t, app.SSR, `<p data-pi="1">false</p>`)

	app.Do(func() { app.S.SetConnected(true) })

	requireEqual(t, app.Normalized(), `<p>true</p>`)
}

func TestHeapReturnsAfterMountAndDispose(t *testing.T) {
	measure := func() uint64 {
		var m goruntime.MemStats
		for i := 0; i < 3; i++ {
			goruntime.GC()
		}
		goruntime.ReadMemStats(&m)
		return m.HeapAlloc
	}
	build := func() {
		var list *runtime.Signal[[]int]
		s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
			list = runtime.UseSignal(ctx, []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
			return el("ul", runtime.For(runtime.Items(list, func(n int) int { return n }),
				runtime.PropsComponent(func(ctx *Ctx, item runtime.ReadSignal[int]) Node {
					v := runtime.UseComputed(ctx, func() int { return item.Get() })
					return el("li", on("click", noopHandler), bound(v))
				})))
		}})
		s.Start()
		_ = s.Ask(time.Second, func() error { _, err := s.RenderSnapshot(); return err })
		_ = s.Ask(time.Second, func() error { list.Set([]int{3, 2, 1}); return s.SettleIfNeeded() })
		s.Close()
	}
	build()
	before := measure()
	for i := 0; i < 300; i++ {
		build()
	}
	after := measure()
	if after > before+2<<20 {
		t.Fatalf("heap grew from %d to %d across 300 mounted and closed sessions", before, after)
	}
}

func TestSessionCloseDisposesEverythingOnce(t *testing.T) {
	closed := 0
	s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
		runtime.UseEffect(ctx, func() func() { return func() { closed++ } })
		return el("div", on("click", noopHandler))
	}})
	s.OnClosed(func() { closed += 10 })
	s.Start()
	_ = s.Ask(time.Second, func() error { _, err := s.RenderSnapshot(); s.GoLive(); return err })
	_ = s.Ask(time.Second, func() error { return s.SettleIfNeeded() })

	s.Close()

	if closed != 11 {
		t.Fatalf("closed = %d, want the watch cleanup once and OnClosed once", closed)
	}
	if s.HandlerCount() != 0 {
		t.Fatalf("handlers leaked: %d", s.HandlerCount())
	}
}

func TestInvokeOnADisposedOwnerRunsNothing(t *testing.T) {
	var ctx *Ctx
	var flag *runtime.Signal[bool]
	child := runtime.Component(func(c *Ctx) Node {
		ctx = c
		return el("p", bound(runtime.UseSignal(c, 1)))
	})
	app := runtimetest.New(t, func(c *Ctx) Node {
		flag = runtime.UseSignal(c, true)
		return el("div", runtime.Show(flag, func() Node { return child() }))
	})
	ran := 0
	app.On(func() { runtime.Invoke(ctx, func() { ran++ }) })
	app.Do(func() { flag.Set(false) })

	app.On(func() { runtime.Invoke(ctx, func() { ran++ }) })

	if ran != 1 {
		t.Fatalf("ran = %d, a disposed owner must not run late callbacks", ran)
	}
}
