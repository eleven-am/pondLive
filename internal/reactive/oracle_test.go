package reactive

import (
	"math/rand"
	"testing"
)

type byteSrc struct {
	data []byte
	pos  int
}

func (b *byteSrc) next(n int) int {
	if len(b.data) == 0 {
		return 0
	}
	v := int(b.data[b.pos%len(b.data)])
	b.pos++
	return v % n
}

type nodeSpec struct {
	isSignal bool
	a, b, c  int
	k        int
}

func evalSpec(specs []nodeSpec, vals []int, i int) int {
	s := specs[i]
	if s.isSignal {
		return vals[i]
	}
	va := evalSpec(specs, vals, s.a)
	if va%2 == 0 {
		return (evalSpec(specs, vals, s.b) + s.k) % 5
	}
	return (evalSpec(specs, vals, s.c)*2 + s.k) % 5
}

type effectSpec struct {
	reads    []int
	runs     int
	observed []int
	last     []int
	handle   *Effect
	dead     bool
}

func runGraphProgram(t *testing.T, data []byte) {
	src := &byteSrc{data: data}
	g, root := setup()
	nsig := 1 + src.next(5)
	ncomp := src.next(10)
	specs := make([]nodeSpec, 0, nsig+ncomp)
	vals := make([]int, nsig+ncomp)
	sigs := make([]*Signal[int], nsig)
	nodes := make([]Readable[int], nsig+ncomp)
	for i := 0; i < nsig; i++ {
		specs = append(specs, nodeSpec{isSignal: true})
		vals[i] = src.next(5)
		sigs[i] = NewSignal(g, vals[i])
		nodes[i] = sigs[i]
	}
	for j := 0; j < ncomp; j++ {
		i := nsig + j
		s := nodeSpec{a: src.next(i), b: src.next(i), c: src.next(i), k: src.next(5)}
		specs = append(specs, s)
		nodes[i] = NewComputed(root, func() int {
			if nodes[s.a].Get()%2 == 0 {
				return (nodes[s.b].Get() + s.k) % 5
			}
			return (nodes[s.c].Get()*2 + s.k) % 5
		})
	}
	oracle := func(reads []int) []int {
		out := make([]int, len(reads))
		for i, r := range reads {
			out[i] = evalSpec(specs, vals, r)
		}
		return out
	}
	total := nsig + ncomp
	neff := 1 + src.next(6)
	effects := make([]*effectSpec, neff)
	for e := range effects {
		es := &effectSpec{}
		n := 1 + src.next(3)
		for i := 0; i < n; i++ {
			es.reads = append(es.reads, src.next(total))
		}
		kind := KindRender
		if src.next(2) == 1 {
			kind = KindWatch
		}
		es.handle = NewEffect(root, kind, func() {
			es.runs++
			es.observed = es.observed[:0]
			for _, r := range es.reads {
				es.observed = append(es.observed, nodes[r].Get())
			}
		})
		es.last = oracle(es.reads)
		effects[e] = es
	}
	steps := 1 + src.next(12)
	for step := 0; step < steps; step++ {
		for _, es := range effects {
			es.runs = 0
		}
		for w := src.next(4); w >= 0; w-- {
			i := src.next(nsig)
			v := src.next(5)
			vals[i] = v
			sigs[i].Set(v)
		}
		if src.next(7) == 0 {
			es := effects[src.next(neff)]
			es.handle.Dispose()
			es.dead = true
		}
		if src.next(5) == 0 {
			for i := 0; i < total; i++ {
				if got, want := nodes[i].Peek(), evalSpec(specs, vals, i); got != want {
					t.Fatalf("step %d node %d: peek %d, oracle %d", step, i, got, want)
				}
			}
		}
		flush(t, g)
		for e, es := range effects {
			want := oracle(es.reads)
			if es.dead {
				if es.runs != 0 {
					t.Fatalf("step %d effect %d ran after dispose", step, e)
				}
				continue
			}
			if es.runs > 1 {
				t.Fatalf("step %d effect %d ran %d times in one flush", step, e, es.runs)
			}
			changed := false
			for i := range want {
				if want[i] != es.last[i] {
					changed = true
				}
			}
			if changed && es.runs != 1 {
				t.Fatalf("step %d effect %d missed an update: want %v last %v", step, e, want, es.last)
			}
			for i := range want {
				if es.observed[i] != want[i] {
					t.Fatalf("step %d effect %d observed %v, oracle %v (glitch or stale read)", step, e, es.observed, want)
				}
			}
			es.last = want
		}
	}
	root.Dispose()
	for i, s := range sigs {
		if s.targets != nil {
			t.Fatalf("signal %d retained links after dispose", i)
		}
	}
}

func FuzzGraphMatchesOracle(f *testing.F) {
	f.Add([]byte{3, 5, 1, 2, 3, 4, 0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 10})
	f.Add([]byte{1, 1, 0, 0, 0, 0, 0, 0})
	f.Add([]byte{4, 9, 4, 3, 2, 1, 9, 8, 7, 6, 5, 4, 3, 2, 1, 0, 3, 3, 3, 1, 2, 1, 2})
	f.Add([]byte{200, 100, 50, 25, 12, 6, 3, 1, 255, 254, 253, 99, 98, 97})
	rng := rand.New(rand.NewSource(7))
	for i := 0; i < 24; i++ {
		b := make([]byte, 80)
		rng.Read(b)
		f.Add(b)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		runGraphProgram(t, data)
	})
}

func TestRandomGraphsMatchOracle(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 3000; i++ {
		b := make([]byte, 150)
		rng.Read(b)
		runGraphProgram(t, b)
	}
}
