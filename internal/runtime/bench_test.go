package runtime

import (
	"fmt"
	goruntime "runtime"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type benchPage struct {
	s     *Session
	list  *reactive.Signal[[]int]
	items []int
}

func el(tag string, items ...work.Item) work.Node { return work.BuildElement(tag, items...) }

const domNodesPerRow = 8

func buildBenchPage(nodes int, feed bool) *benchPage {
	rows := max(nodes/domNodesPerRow, 1)
	p := &benchPage{}
	p.items = make([]int, rows)
	for i := range p.items {
		p.items[i] = i + 1
	}
	p.s = NewSession(Config{SessionID: "bench", Root: func(ctx *Ctx) work.Node {
		p.list = UseSignal(ctx, p.items)
		row := PropsComponent(func(ctx *Ctx, item ReadSignal[int]) work.Node {
			n := item.Peek()
			var button work.Node = el("button", work.NewText("go"))
			if !feed {
				button = el("button", work.On("click", func(work.Event) {}), work.NewText("go"))
			}
			return el("div", work.Class("row", "item"),
				el("span", work.TextOf(n)),
				el("p", work.NewText("static text for the row")),
				button,
				el("i", work.Attribute("data-n", n)),
			)
		})
		var src Source[int]
		if feed {
			items := p.items
			src = NewFeed(func(n int) int { return n }, func() []int { return items })
		} else {
			src = Items(p.list, func(n int) int { return n })
		}
		return el("main", For(src, row))
	}})
	if _, err := p.s.RenderSnapshot(); err != nil {
		panic(err)
	}
	return p
}

func buildBindingPage(bindings int) (*Session, []*reactive.Signal[int]) {
	sigs := make([]*reactive.Signal[int], bindings)
	s := NewSession(Config{SessionID: "bench", Root: func(ctx *Ctx) work.Node {
		kids := make([]work.Item, bindings)
		for i := range kids {
			sigs[i] = UseSignal(ctx, 0)
			kids[i] = el("span", work.TextOf(sigs[i]))
		}
		return el("div", kids...)
	}})
	if _, err := s.RenderSnapshot(); err != nil {
		panic(err)
	}
	return s, sigs
}

func BenchmarkBindingUpdate(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			s, sigs := buildBindingPage(n)
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sigs[i%n].Set(i + 1)
				if err := s.graph.RunQueue(reactive.QueueRender); err != nil {
					b.Fatal(err)
				}
				s.edits = s.edits[:0]
			}
		})
	}
}

func BenchmarkLeafUpdateToFlush(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprint(n), func(b *testing.B) {
			s, sigs := buildBindingPage(n)
			var sink int
			s.Bus.SubscribeToFrameEdits(func(edits []protocol.Edit) {
				for _, e := range edits {
					raw, _ := e.MarshalJSON()
					sink += len(raw)
				}
			})
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				sigs[i%n].Set(i + 1)
				s.flush()
			}
			_ = sink
		})
	}
}

func BenchmarkMount(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		for _, feed := range []bool{false, true} {
			name := fmt.Sprintf("%d/items", n)
			if feed {
				name = fmt.Sprintf("%d/feed", n)
			}
			b.Run(name, func(b *testing.B) {
				b.ReportAllocs()
				for i := 0; i < b.N; i++ {
					buildBenchPage(n, feed)
				}
			})
		}
	}
}

func BenchmarkSerializeReset(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		for _, feed := range []bool{false, true} {
			name := fmt.Sprintf("%d/items", n)
			if feed {
				name = fmt.Sprintf("%d/feed", n)
			}
			b.Run(name, func(b *testing.B) {
				p := buildBenchPage(n, feed)
				b.ReportAllocs()
				b.ResetTimer()
				for i := 0; i < b.N; i++ {
					if _, err := p.s.DocumentHTML(); err != nil {
						b.Fatal(err)
					}
				}
			})
		}
	}
}

func BenchmarkListOperations(b *testing.B) {
	const n = 5000
	ops := map[string]func(p *benchPage, i int) []int{
		"insert-front": func(p *benchPage, i int) []int {
			return append([]int{1000000 + i}, p.items...)
		},
		"swap": func(p *benchPage, i int) []int {
			out := append([]int(nil), p.items...)
			out[0], out[len(out)-1] = out[len(out)-1], out[0]
			return out
		},
		"shuffle": func(p *benchPage, i int) []int {
			out := append([]int(nil), p.items...)
			for j := range out {
				k := (j*7919 + i) % len(out)
				out[j], out[k] = out[k], out[j]
			}
			return out
		},
	}
	for name, op := range ops {
		b.Run(name, func(b *testing.B) {
			p := buildBenchPage(n*domNodesPerRow, false)
			var edits, bytes int
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				p.list.Set(p.items)
				p.s.graph.RunQueue(reactive.QueueRender)
				p.s.edits = nil
				next := op(p, i)
				b.StartTimer()
				p.list.Set(next)
				if err := p.s.graph.RunQueue(reactive.QueueRender); err != nil {
					b.Fatal(err)
				}
				edits, bytes = len(p.s.edits), 0
				for _, e := range p.s.edits {
					raw, _ := e.MarshalJSON()
					bytes += len(raw)
				}
				p.s.edits = nil
			}
			b.ReportMetric(float64(edits), "edits")
			b.ReportMetric(float64(bytes), "edit-bytes")
		})
	}
}

func TestRetainedMemoryPerSession(t *testing.T) {
	baseline := retainedBytes(t, func() *Session { return buildBenchPage(domNodesPerRow, true).s })
	t.Logf("fixed part of a session (one row): %d bytes", baseline)
	cases := []struct {
		nodes int
		limit int64
	}{{100, 12 << 10}, {1000, 110 << 10}, {5000, 450 << 10}}
	for _, c := range cases {
		got := retainedBytes(t, func() *Session { return buildBenchPage(c.nodes, false).s })
		tree := got - baseline
		t.Logf("retained at %d DOM nodes (For over Items, handler per row): %d bytes, tree part %d (gate %d)", c.nodes, got, tree, c.limit)
		if tree > c.limit {
			t.Errorf("tree-dependent memory %d exceeds the gate %d at %d nodes", tree, c.limit, c.nodes)
		}
	}
	feed := retainedBytes(t, func() *Session { return buildBenchPage(5000, true).s })
	t.Logf("retained at 5000 DOM nodes (For over a Feed, static rows): %d bytes (gate %d)", feed, 150<<10)
	if feed > int64(150<<10) {
		t.Errorf("feed memory %d exceeds the gate", feed)
	}
	counter := retainedBytes(t, func() *Session {
		s := NewSession(Config{Root: func(ctx *Ctx) work.Node {
			n := UseSignal(ctx, 0)
			rows := make([]work.Item, 20)
			for i := range rows {
				rows[i] = el("div", el("span", work.TextOf(n)), el("button", work.On("click", func(work.Event) {}), work.NewText("+")))
			}
			return el("main", rows...)
		}})
		if _, err := s.RenderSnapshot(); err != nil {
			t.Fatal(err)
		}
		return s
	})
	t.Logf("retained by a 20-row counter session: %d bytes (gate %d)", counter, 30<<10)
	if counter > int64(30<<10) {
		t.Errorf("counter session memory %d exceeds the gate", counter)
	}
	goruntime.GC()
}
