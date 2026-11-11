package runtime_test

import (
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

type kv struct{ K, V int }

type pn struct {
	kind  string
	tag   string
	n     int
	text  string
	kids  []*pn
	arms  []*pn
	exit  bool
	attrs []string
}

func genNode(rng *rand.Rand, depth int, root bool) *pn {
	tags := []string{"div", "span", "section", "b"}
	if root || depth <= 0 {
		return genElem(rng, depth, tags)
	}
	switch rng.Intn(8) {
	case 0, 1:
		return genElem(rng, depth, tags)
	case 2:
		return &pn{kind: "text", text: []string{"x", "y z", "<&>", ""}[rng.Intn(4)]}
	case 3:
		return &pn{kind: "bound", n: rng.Intn(4)}
	case 4:
		return &pn{kind: "show", n: rng.Intn(4), exit: rng.Intn(3) == 0, arms: []*pn{genNode(rng, depth-1, true), genOptional(rng, depth-1)}}
	case 5:
		return &pn{kind: "switch", n: rng.Intn(4), exit: rng.Intn(3) == 0,
			arms: []*pn{genNode(rng, depth-1, true), genNode(rng, depth-1, true), genNode(rng, depth-1, true)}}
	case 6:
		return &pn{kind: "items", n: rng.Intn(2), exit: rng.Intn(2) == 0, arms: []*pn{genNode(rng, depth-1, true)}}
	default:
		return &pn{kind: "comp", kids: []*pn{genNode(rng, depth-1, rng.Intn(2) == 0)}}
	}
}

func genOptional(rng *rand.Rand, depth int) *pn {
	if rng.Intn(2) == 0 {
		return nil
	}
	return genNode(rng, depth, true)
}

func genElem(rng *rand.Rand, depth int, tags []string) *pn {
	e := &pn{kind: "elem", tag: tags[rng.Intn(len(tags))]}
	for _, a := range []string{"attr", "presence", "class", "style", "handler"} {
		if rng.Intn(4) == 0 {
			e.attrs = append(e.attrs, a)
		}
	}
	e.n = rng.Intn(4)
	for i, count := 0, rng.Intn(4); i < count && depth > 0; i++ {
		e.kids = append(e.kids, genNode(rng, depth-1, false))
	}
	return e
}

type fenv struct {
	bools [4]runtime.ReadSignal[bool]
	ints  [4]runtime.ReadSignal[int]
	lists [2]runtime.ReadSignal[[]kv]
	feed  *runtime.Feed[kv, int]
	load  func() []kv
}

func (p *pn) build(e *fenv) Node {
	switch p.kind {
	case "text":
		return txt(p.text)
	case "bound":
		return bound(e.ints[p.n])
	case "elem":
		items := []Item{}
		for _, a := range p.attrs {
			switch a {
			case "attr":
				items = append(items, attr("data-a", e.ints[p.n]))
			case "presence":
				items = append(items, attr("hidden", e.bools[p.n]))
			case "class":
				items = append(items, cls("c", classWhen{"on", e.bools[(p.n+1)%4]}))
			case "style":
				items = append(items, work.Style("order", e.ints[(p.n+2)%4]))
			case "handler":
				items = append(items, on("click", noopHandler))
			}
		}
		for _, k := range p.kids {
			items = append(items, k.build(e))
		}
		return el(p.tag, items...)
	case "show":
		opts := []runtime.ShowOpt{}
		if p.exit {
			opts = append(opts, runtime.Exit(50*time.Millisecond))
		}
		if p.arms[1] != nil {
			els := p.arms[1]
			opts = append(opts, runtime.Else(func() Node { return els.build(e) }))
		}
		then := p.arms[0]
		return runtime.Show(e.bools[p.n], func() Node { return then.build(e) }, opts...)
	case "switch":
		args := []runtime.SwitchArg{}
		if p.exit {
			args = append(args, runtime.Exit(50*time.Millisecond))
		}
		for i := 0; i < 2; i++ {
			arm := p.arms[i]
			args = append(args, runtime.Case(i, func() Node { return arm.build(e) }))
		}
		def := p.arms[2]
		args = append(args, runtime.Default(func() Node { return def.build(e) }))
		sel := runtime.ReadSignal[int](e.ints[p.n])
		return runtime.Switch(selMod{sel}, args...)
	case "items":
		arm := p.arms[0]
		var opts []runtime.ForOpt
		if p.exit {
			opts = append(opts, runtime.Exit(50*time.Millisecond))
		}
		return runtime.For(runtime.Items(e.lists[p.n], func(r kv) int { return r.K }),
			runtime.PropsComponent(func(ctx *Ctx, item runtime.ReadSignal[kv]) Node {
				v := runtime.UseComputed(ctx, func() int { return item.Get().V })
				return el("div", attr("data-k", item.Peek().K), bound(v), arm.build(e))
			}), opts...)
	case "feed":
		arm := p.arms[0]
		var opts []runtime.ForOpt
		if p.exit {
			opts = append(opts, runtime.Exit(50*time.Millisecond))
		}
		return runtime.For(e.feed, runtime.PropsComponent(func(ctx *Ctx, item runtime.ReadSignal[kv]) Node {
			r := item.Peek()
			return el("div", txt(fmt.Sprintf("%d:%d", r.K, r.V)), arm.build(e))
		}), opts...)
	case "comp":
		inner := p.kids[0]
		c := runtime.Component(func(ctx *Ctx) Node {
			runtime.UseSignal(ctx, 1)
			return inner.build(e)
		})
		return c()
	}
	panic("unknown node kind " + p.kind)
}

type selMod struct{ runtime.ReadSignal[int] }

func (s selMod) Get() int  { return s.ReadSignal.Get() % 3 }
func (s selMod) Peek() int { return s.ReadSignal.Peek() % 3 }

type liveState struct {
	bools [4]*runtime.Signal[bool]
	ints  [4]*runtime.Signal[int]
	lists [2]*runtime.Signal[[]kv]
	feed  *runtime.Feed[kv, int]
	feedM []kv
}

type snapshot struct {
	bools [4]bool
	ints  [4]int
	lists [2][]kv
	feedM []kv
}

func (s *liveState) snap() snapshot {
	var out snapshot
	for i := range s.bools {
		out.bools[i] = s.bools[i].Peek()
		out.ints[i] = s.ints[i].Peek()
	}
	for i := range s.lists {
		out.lists[i] = append([]kv(nil), s.lists[i].Peek()...)
	}
	out.feedM = append([]kv(nil), s.feedM...)
	return out
}

func freshRender(t testing.TB, prog *pn, snap snapshot) string {
	root := func(ctx *Ctx) Node {
		e := &fenv{load: func() []kv { return snap.feedM }}
		for i := range e.bools {
			e.bools[i] = runtime.Const(snap.bools[i])
			e.ints[i] = runtime.Const(snap.ints[i])
		}
		for i := range e.lists {
			e.lists[i] = runtime.Const(snap.lists[i])
		}
		e.feed = runtime.NewFeed(func(r kv) int { return r.K }, e.load)
		return el("main", prog.build(e))
	}
	s := runtime.NewSession(runtime.Config{Root: root})
	snapShot, err := s.RenderSnapshot()
	if err != nil {
		t.Fatalf("fresh render: %v", err)
	}
	norm, err := runtimetest.NormalizedOf(snapShot.HTML)
	if err != nil {
		t.Fatal(err)
	}
	return norm
}

func randomKVs(rng *rand.Rand, keys []int) []kv {
	n := rng.Intn(6)
	perm := rng.Perm(len(keys))
	out := make([]kv, 0, n)
	for i := 0; i < n && i < len(perm); i++ {
		out = append(out, kv{K: keys[perm[i]], V: rng.Intn(3)})
	}
	return out
}

var opCounts = map[protocol.Op]int{}

func runProgram(t testing.TB, seed int64, steps int) {
	rng := rand.New(rand.NewSource(seed))
	prog := &pn{kind: "elem", tag: "div", kids: []*pn{
		genNode(rng, 3, true),
		{kind: "feed", exit: rng.Intn(2) == 0, arms: []*pn{genNode(rng, 2, true)}},
	}}
	state := &liveState{}
	app := runtimetest.New(t, func(ctx *Ctx) Node {
		e := &fenv{}
		for i := range e.bools {
			sig := runtime.UseSignal(ctx, false)
			state.bools[i], e.bools[i] = sig, sig
			isig := runtime.UseSignal(ctx, 0)
			state.ints[i], e.ints[i] = isig, isig
		}
		for i := range e.lists {
			lsig := runtime.UseSignal(ctx, []kv(nil))
			state.lists[i], e.lists[i] = lsig, lsig
		}
		e.load = func() []kv { return append([]kv(nil), state.feedM...) }
		f := runtime.NewFeed(func(r kv) int { return r.K }, e.load)
		if state.feed == nil {
			state.feed = f
		}
		e.feed = f
		return el("main", prog.build(e))
	})
	keys := []int{1, 2, 3, 4, 5, 6, 7, 8}
	next := 100
	check := func(label string) {
		t.Helper()
		app.Advance(time.Second)
		for _, frame := range app.Frames {
			for _, e := range frame {
				opCounts[e.Op]++
			}
		}
		app.Frames = nil
		got := app.Normalized()
		want := freshRender(t, prog, state.snap())
		if got != want {
			t.Fatalf("seed %d after %s: edits applied to SSR differ from a fresh render\nlive:  %s\nfresh: %s", seed, label, got, want)
		}
	}
	check("mount")
	for step := 0; step < steps; step++ {
		label := ""
		switch rng.Intn(7) {
		case 0, 6:
			i, v := rng.Intn(4), rng.Intn(2) == 0
			label = fmt.Sprintf("bool %d=%v", i, v)
			app.Do(func() { state.bools[i].Set(v) })
		case 1:
			i, v := rng.Intn(4), rng.Intn(5)
			label = fmt.Sprintf("int %d=%d", i, v)
			app.Do(func() { state.ints[i].Set(v) })
		case 2, 3:
			i := rng.Intn(2)
			v := randomKVs(rng, keys)
			label = fmt.Sprintf("list %d=%v", i, v)
			app.Do(func() { state.lists[i].Set(v) })
		default:
			op := rng.Intn(5)
			label = fmt.Sprintf("feed op %d", op)
			if state.feed == nil {
				continue
			}
			app.Do(func() { feedStep(rng, state, op, &next) })
		}
		check(label)
	}
}

func feedStep(rng *rand.Rand, state *liveState, op int, next *int) {
	m := state.feedM
	switch {
	case op == 0 || len(m) == 0:
		*next++
		it := kv{*next, rng.Intn(3)}
		state.feedM = append(m, it)
		state.feed.Append(it)
	case op == 1:
		*next++
		it := kv{*next, rng.Intn(3)}
		state.feedM = append([]kv{it}, m...)
		state.feed.Prepend(it)
	case op == 2:
		i := rng.Intn(len(m))
		state.feedM = append(append([]kv(nil), m[:i]...), m[i+1:]...)
		state.feed.Delete(m[i].K)
	case op == 3:
		i := rng.Intn(len(m))
		it := kv{m[i].K, rng.Intn(3)}
		state.feedM = append([]kv(nil), m...)
		state.feedM[i] = it
		state.feed.Replace(it)
	default:
		i := rng.Intn(len(m))
		*next++
		it := kv{*next, rng.Intn(3)}
		state.feedM = append(append(append([]kv(nil), m[:i]...), it), m[i:]...)
		state.feed.InsertBefore(m[i].K, it)
	}
}

func TestProgramsMatchFreshRender(t *testing.T) {
	for seed := int64(0); seed < 200; seed++ {
		runProgram(t, seed, 12)
	}
	for _, op := range []protocol.Op{protocol.OpText, protocol.OpAttr, protocol.OpRemoveAttr, protocol.OpStyle,
		protocol.OpInsert, protocol.OpMove, protocol.OpRemove, protocol.OpClear} {
		if opCounts[op] == 0 {
			t.Errorf("the generated programs never produced a %q edit: %v", op, opCounts)
		}
	}
	t.Logf("edit counts over 200 programs: %v", opCounts)
}

func FuzzProgramMatchesFreshRender(f *testing.F) {
	for _, seed := range []int64{0, 1, 2, 3, 5, 8, 13, 21, 34, 55, 89, 144} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, seed int64) {
		runProgram(t, seed, 12)
	})
}
