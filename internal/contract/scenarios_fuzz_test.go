package contract

import (
	"fmt"
	"math/rand"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/runtime/runtimetest"
	"github.com/eleven-am/pondlive/internal/work"
)

const (
	fuzzPrograms = 200
	fuzzSteps    = 8
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
	for _, a := range []string{"attr", "presence", "class", "style", "handler", "optional"} {
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
	strs  [4]runtime.ReadSignal[string]
	lists [2]runtime.ReadSignal[[]kv]
	feed  *runtime.Feed[kv, int]
	load  func() []kv
}

type selMod struct{ runtime.ReadSignal[int] }

func (s selMod) Get() int  { return s.ReadSignal.Get() % 3 }
func (s selMod) Peek() int { return s.ReadSignal.Peek() % 3 }

func exitOpt(on bool) []runtime.RegionOpt {
	if !on {
		return nil
	}
	return []runtime.RegionOpt{runtime.Exit(50 * time.Millisecond)}
}

func (p *pn) build(e *fenv) node {
	switch p.kind {
	case "text":
		return txt(p.text)
	case "bound":
		return bound(e.ints[p.n])
	case "elem":
		items := []item{}
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
				items = append(items, on("click", noop))
			case "optional":
				items = append(items, work.Attribute("title", e.strs[p.n], work.OmitEmpty()))
			}
		}
		for _, k := range p.kids {
			items = append(items, k.build(e))
		}
		return el(p.tag, items...)
	case "show":
		opts := []runtime.ShowOpt{}
		for _, o := range exitOpt(p.exit) {
			opts = append(opts, o)
		}
		if p.arms[1] != nil {
			els := p.arms[1]
			opts = append(opts, runtime.Else(func() node { return els.build(e) }))
		}
		then := p.arms[0]
		return runtime.Show(e.bools[p.n], func() node { return then.build(e) }, opts...)
	case "switch":
		args := []runtime.SwitchArg{}
		for _, o := range exitOpt(p.exit) {
			args = append(args, o)
		}
		for i := 0; i < 2; i++ {
			arm := p.arms[i]
			args = append(args, runtime.Case(i, func() node { return arm.build(e) }))
		}
		def := p.arms[2]
		args = append(args, runtime.Default(func() node { return def.build(e) }))
		return runtime.Switch[int](selMod{e.ints[p.n]}, args...)
	case "items":
		arm := p.arms[0]
		var opts []runtime.ForOpt
		for _, o := range exitOpt(p.exit) {
			opts = append(opts, o)
		}
		return runtime.For(runtime.Items(e.lists[p.n], func(r kv) int { return r.K }),
			runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[kv]) node {
				v := runtime.UseComputed(c, func() int { return it.Get().V })
				return el("div", attr("data-k", it.Peek().K), bound(v), arm.build(e))
			}), opts...)
	case "feed":
		arm := p.arms[0]
		var opts []runtime.ForOpt
		for _, o := range exitOpt(p.exit) {
			opts = append(opts, o)
		}
		return runtime.For(e.feed, runtime.PropsComponent(func(c *ctx, it runtime.ReadSignal[kv]) node {
			r := it.Peek()
			return el("div", txt(fmt.Sprintf("%d:%d", r.K, r.V)), arm.build(e))
		}), opts...)
	case "comp":
		inner := p.kids[0]
		comp := runtime.Component(func(c *ctx) node {
			runtime.UseSignal(c, 1)
			return inner.build(e)
		})
		return comp()
	}
	panic("unknown node kind " + p.kind)
}

type liveState struct {
	bools [4]*runtime.Signal[bool]
	ints  [4]*runtime.Signal[int]
	strs  [4]*runtime.Signal[string]
	lists [2]*runtime.Signal[[]kv]
	feed  *runtime.Feed[kv, int]
	feedM []kv
}

type snapshot struct {
	bools [4]bool
	ints  [4]int
	strs  [4]string
	lists [2][]kv
	feedM []kv
}

func (s *liveState) snap() snapshot {
	var out snapshot
	for i := range s.bools {
		out.bools[i] = s.bools[i].Peek()
		out.ints[i] = s.ints[i].Peek()
		out.strs[i] = s.strs[i].Peek()
	}
	for i := range s.lists {
		out.lists[i] = append([]kv(nil), s.lists[i].Peek()...)
	}
	out.feedM = append([]kv(nil), s.feedM...)
	return out
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

func fuzzInstance(seed int64) instance {
	rng := rand.New(rand.NewSource(seed))
	prog := &pn{kind: "elem", tag: "div", kids: []*pn{
		genNode(rng, 3, true),
		{kind: "feed", exit: rng.Intn(2) == 0, arms: []*pn{genNode(rng, 2, true)}},
	}}
	state := &liveState{}
	keys := []int{1, 2, 3, 4, 5, 6, 7, 8}
	next := 100
	inst := instance{
		root: func(c *ctx) node {
			e := &fenv{}
			for i := range e.bools {
				bs := runtime.UseSignal(c, false)
				state.bools[i], e.bools[i] = bs, bs
				is := runtime.UseSignal(c, 0)
				state.ints[i], e.ints[i] = is, is
				ss := runtime.UseSignal(c, "")
				state.strs[i], e.strs[i] = ss, ss
			}
			for i := range e.lists {
				ls := runtime.UseSignal(c, []kv(nil))
				state.lists[i], e.lists[i] = ls, ls
			}
			e.load = func() []kv { return append([]kv(nil), state.feedM...) }
			e.feed = runtime.NewFeed(func(r kv) int { return r.K }, e.load)
			if state.feed == nil {
				state.feed = e.feed
			}
			return doc(el("main", prog.build(e)))
		},
		fresh: func() func(c *ctx) node {
			snap := state.snap()
			return func(c *ctx) node {
				e := &fenv{load: func() []kv { return snap.feedM }}
				for i := range e.bools {
					e.bools[i] = runtime.UseSignal(c, snap.bools[i])
					e.ints[i] = runtime.UseSignal(c, snap.ints[i])
					e.strs[i] = runtime.UseSignal(c, snap.strs[i])
				}
				for i := range e.lists {
					e.lists[i] = runtime.UseSignal(c, snap.lists[i])
				}
				e.feed = runtime.NewFeed(func(r kv) int { return r.K }, e.load)
				return doc(el("main", prog.build(e)))
			}
		},
	}
	for step := 0; step < fuzzSteps; step++ {
		inst.steps = append(inst.steps, func(a *runtimetest.App) {
			switch rng.Intn(8) {
			case 0, 7:
				i, v := rng.Intn(4), rng.Intn(2) == 0
				a.Do(func() { state.bools[i].Set(v) })
			case 1:
				i, v := rng.Intn(4), rng.Intn(5)
				a.Do(func() { state.ints[i].Set(v) })
			case 2:
				i, v := rng.Intn(4), []string{"", "t", "u v"}[rng.Intn(3)]
				a.Do(func() { state.strs[i].Set(v) })
			case 3, 4:
				i := rng.Intn(2)
				v := randomKVs(rng, keys)
				a.Do(func() { state.lists[i].Set(v) })
			default:
				op := rng.Intn(5)
				a.Do(func() { feedStep(rng, state, op, &next) })
			}
			a.Advance(time.Second)
		})
	}
	return inst
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

func fuzzScenario() scenario {
	sc := scenario{file: "fuzz_regions", description: "random nested For (items and feed), Show and Switch with text, attributes and handlers; fixed seeds"}
	for seed := int64(0); seed < fuzzPrograms; seed++ {
		seed := seed
		sc.cases = append(sc.cases, scenarioCase{name: fmt.Sprintf("seed_%03d", seed), build: func() instance { return fuzzInstance(seed) }})
	}
	return sc
}

func allScenarios() []scenario {
	return []scenario{
		forKeyed(), forPermutations(), forDuplicate(), feedScenario(), exitScenario(),
		showSwitch(), componentsScenario(), textAdjacent(), textEmpty(), tableScenario(),
		attrsForm(), attrsBool(), raw(), svg(), refsScenario(), headScenario(), styleScenario(),
		handlersScenario(), portalScenario(), fragmentUnits(), fuzzScenario(),
	}
}
