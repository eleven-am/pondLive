package runtime_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
)

type msg struct {
	ID   int
	Text string
}

type store struct {
	items []msg
}

func (s *store) load() []msg { return append([]msg(nil), s.items...) }

func feedApp(t *testing.T, st *store, exit time.Duration) (*runtimetest.App, *runtime.Feed[msg, int]) {
	var live *runtime.Feed[msg, int]
	var opts []runtime.ForOpt
	if exit > 0 {
		opts = append(opts, runtime.Exit(exit))
	}
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		f := runtime.NewFeed(func(m msg) int { return m.ID }, st.load)
		if live == nil {
			live = f
		}
		return el("ul", runtime.For(f, runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[msg]) Node {
			return el("li", txt(item.Peek().Text))
		}), opts...))
	})
	return app, live
}

func TestFeedOperations(t *testing.T) {
	st := &store{items: []msg{{2, "b"}, {3, "c"}}}
	app, feed := feedApp(t, st, 0)
	requireEqual(t, app.Normalized(), `<ul><li>b</li><li>c</li></ul>`)

	add := func(fn func()) []protocol.Edit {
		return app.Do(fn)
	}
	edits := add(func() { st.items = append(st.items, msg{4, "d"}); feed.Append(msg{4, "d"}) })
	if ops(edits) != "ins" {
		t.Fatalf("append = %v", edits)
	}
	edits = add(func() { st.items = append([]msg{{1, "a"}}, st.items...); feed.Prepend(msg{1, "a"}) })
	if ops(edits) != "ins" {
		t.Fatalf("prepend = %v", edits)
	}
	requireEqual(t, app.Normalized(), `<ul><li>a</li><li>b</li><li>c</li><li>d</li></ul>`)

	edits = add(func() {
		st.items = []msg{{1, "a"}, {5, "x"}, {2, "b"}, {3, "c"}, {4, "d"}}
		feed.InsertBefore(2, msg{5, "x"})
	})
	if ops(edits) != "ins" {
		t.Fatalf("insertBefore = %v", edits)
	}
	edits = add(func() {
		st.items = []msg{{1, "a"}, {5, "x"}, {2, "B"}, {3, "c"}, {4, "d"}}
		feed.Replace(msg{2, "B"})
	})
	if ops(edits) != "ins,rm" {
		t.Fatalf("replace = %v", edits)
	}
	edits = add(func() {
		st.items = []msg{{1, "a"}, {5, "x"}, {2, "B"}, {4, "d"}}
		feed.Delete(3)
	})
	if ops(edits) != "rm" {
		t.Fatalf("delete = %v", edits)
	}
	requireEqual(t, app.Normalized(), `<ul><li>a</li><li>x</li><li>B</li><li>d</li></ul>`)

	edits = add(func() {
		st.items = []msg{{9, "z"}}
		feed.Reload()
	})
	if ops(edits) != "clr,ins" {
		t.Fatalf("reload = %v", edits)
	}
	requireEqual(t, app.Normalized(), `<ul><li>z</li></ul>`)
}

func TestFeedResetReloadsFromLoader(t *testing.T) {
	st := &store{items: []msg{{1, "a"}, {2, "b"}, {3, "c"}}}
	app, _ := feedApp(t, st, time.Second)
	app.Do(func() {})

	st.items = []msg{{1, "a"}, {3, "c"}, {4, "d"}}
	var html string
	var err error
	app.On(func() { html, err = app.S.ResetStep() })
	if err != nil {
		t.Fatal(err)
	}

	norm, _ := runtimetest.NormalizedOf(html)
	requireEqual(t, norm, `<ul><li>a</li><li>c</li><li>d</li></ul>`)
	app.Sync()
	requireEqual(t, app.Normalized(), `<ul><li>a</li><li>c</li><li>d</li></ul>`)
}

func TestFeedResetTruncatesInFlightExits(t *testing.T) {
	st := &store{items: []msg{{1, "a"}, {2, "b"}}}
	app, feed := feedApp(t, st, time.Hour)
	app.On(func() { feed.Delete(2) })
	app.Pump()
	if !contains(app.Model.HTML(), `data-exiting`) {
		t.Fatalf("row should be exiting: %s", app.Model.HTML())
	}

	st.items = st.items[:1]
	app.Sync()

	requireEqual(t, app.Normalized(), `<ul><li>a</li></ul>`)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func TestFeedOffActorOperationsArePosted(t *testing.T) {
	st := &store{}
	app, feed := feedApp(t, st, 0)
	done := make(chan struct{})
	st.items = []msg{{1, "x"}}
	go func() {
		defer close(done)
		feed.Append(msg{1, "x"})
	}()
	<-done
	app.Sync()
	requireEqual(t, app.Normalized(), `<ul><li>x</li></ul>`)
}

func TestFeedMountedTwiceIsAnError(t *testing.T) {
	feed := runtime.NewFeed(func(m msg) int { return m.ID }, func() []msg { return nil })
	s := runtime.NewSession(runtime.Config{Root: func(ctx *Ctx) Node {
		row := runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[msg]) Node { return el("li") })
		return frag(el("ul", runtime.For(feed, row)), el("ol", runtime.For(feed, row)))
	}, DevMode: true})

	snap, err := s.RenderSnapshot()

	if err != nil || !snap.Crashed || !strings.Contains(snap.Crash.Message, "two live regions") {
		t.Fatalf("snapshot = %+v err = %v", snap, err)
	}
}

func TestFeedResetInDevNamesRowsThatLoadForgot(t *testing.T) {
	st := &store{items: []msg{{1, "a"}}}
	app, feed := feedApp(t, st, 0)
	app.On(func() { feed.Append(msg{2, "pushed but never stored"}) })
	app.On(func() { feed.Prepend(msg{0, "also forgotten"}) })
	app.Pump()

	var err error
	app.On(func() { _, err = app.S.ResetStep() })

	var nd *runtime.NondeterministicError
	if !errors.As(err, &nd) {
		t.Fatalf("reset error = %v", err)
	}
	for _, want := range []string{"load omitted 2 row(s)", "keys 0, 2", "Append, Prepend, InsertBefore or Replace"} {
		if !strings.Contains(nd.Error(), want) {
			t.Fatalf("error %q does not contain %q", nd.Error(), want)
		}
	}
}

func TestFeedResetAfterALoadDoesNotBlamePushesTheLoadAlreadySaw(t *testing.T) {
	st := &store{items: []msg{{1, "a"}}}
	app, feed := feedApp(t, st, 0)
	app.Do(func() { st.items = append(st.items, msg{2, "b"}); feed.Append(msg{2, "b"}) })
	if err := resetErr(app); err != nil {
		t.Fatalf("a consistent loader was blamed: %v", err)
	}
	if err := resetErr(app); err != nil {
		t.Fatalf("second reset: %v", err)
	}
	app.Do(func() { feed.Delete(2) })
	st.items = st.items[:1]
	if err := resetErr(app); err != nil {
		t.Fatalf("a deleted push was blamed: %v", err)
	}
}

func TestFeedReplaceAndInsertBeforeCountAsPushes(t *testing.T) {
	st := &store{items: []msg{{1, "a"}, {2, "b"}}}
	app, feed := feedApp(t, st, 0)
	app.On(func() { feed.Replace(msg{1, "A"}); feed.InsertBefore(2, msg{9, "x"}) })
	app.Pump()
	st.items = []msg{{2, "b"}}
	err := resetErr(app)
	if err == nil || !strings.Contains(err.Error(), "keys 1, 9") {
		t.Fatalf("reset error = %v", err)
	}
}

func TestFeedResetInProdKeepsLoadAsTheSourceOfTruth(t *testing.T) {
	st := &store{items: []msg{{1, "a"}}}
	var live *runtime.Feed[msg, int]
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		f := runtime.NewFeed(func(m msg) int { return m.ID }, st.load)
		if live == nil {
			live = f
		}
		return el("ul", runtime.For(f, runtime.PropsComponent(func(_ *Ctx, item runtime.ReadSignal[msg]) Node { return el("li", txt(item.Peek().Text)) })))
	}, runtimetest.Dev(false))
	app.On(func() { live.Append(msg{2, "b"}) })
	app.Pump()
	if err := resetErr(app); err != nil {
		t.Fatalf("prod reset: %v", err)
	}
	requireEqual(t, app.Normalized(), `<ul><li>a</li></ul>`)
}

func resetErr(app *runtimetest.App) (err error) {
	app.On(func() { _, err = app.S.ResetStep() })
	app.Pump()
	return err
}
