package reactive

import (
	"fmt"
	"testing"
)

var sinkInt int

func BenchmarkSetFlushBinding(b *testing.B) {
	for _, n := range []int{100, 1000, 5000} {
		b.Run(fmt.Sprintf("bindings=%d", n), func(b *testing.B) {
			g, root := setup()
			sigs := make([]*Signal[int], n)
			for i := range sigs {
				s := NewSignal(g, 0)
				sigs[i] = s
				NewEffect(root.NewUnit(), KindRender, func() { sinkInt += s.Get() })
			}
			target := sigs[n/2]
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				target.Set(i + 1)
				if err := g.RunQueue(QueueRender); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

func BenchmarkDiamond(b *testing.B) {
	g, root := setup()
	a := NewSignal(g, 0)
	l := NewComputed(root, func() int { return a.Get() + 1 })
	r := NewComputed(root, func() int { return a.Get() * 2 })
	d := NewComputed(root, func() int { return l.Get() + r.Get() })
	NewEffect(root, KindRender, func() { sinkInt += d.Get() })
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		a.Set(i + 1)
		if err := g.RunQueue(QueueRender); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkUnlinkHotSignal(b *testing.B) {
	for _, subs := range []int{10, 1000, 20000} {
		b.Run(fmt.Sprintf("subs=%d", subs), func(b *testing.B) {
			g, root := setup()
			hot := NewSignal(g, 0)
			for i := 0; i < subs; i++ {
				NewEffect(root.NewUnit(), KindRender, func() { hot.Get() })
			}
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				u := root.NewUnit()
				NewEffect(u, KindRender, func() { hot.Get() })
				u.Dispose()
			}
		})
	}
}

func BenchmarkGoidCheck(b *testing.B) {
	g := NewGraph()
	g.BindActor()
	b.ReportAllocs()
	n := 0
	for i := 0; i < b.N; i++ {
		if g.OnActor() {
			n++
		}
	}
	sinkInt = n
}
