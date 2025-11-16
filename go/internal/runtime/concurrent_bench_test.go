package runtime

import (
	"sync/atomic"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// BenchmarkSequentialRendering measures the performance of sequential (non-concurrent) rendering
func BenchmarkSequentialRendering(b *testing.B) {
	benchmarkRendering(b, 0, 10)
}

// BenchmarkConcurrentRendering2Workers measures concurrent rendering with 2 workers
func BenchmarkConcurrentRendering2Workers(b *testing.B) {
	benchmarkRendering(b, 2, 10)
}

// BenchmarkConcurrentRendering4Workers measures concurrent rendering with 4 workers
func BenchmarkConcurrentRendering4Workers(b *testing.B) {
	benchmarkRendering(b, 4, 10)
}

// BenchmarkConcurrentRendering8Workers measures concurrent rendering with 8 workers
func BenchmarkConcurrentRendering8Workers(b *testing.B) {
	benchmarkRendering(b, 8, 10)
}

// BenchmarkDeepHierarchySequential tests deep component trees sequentially
func BenchmarkDeepHierarchySequential(b *testing.B) {
	benchmarkDeepHierarchy(b, 0, 20)
}

// BenchmarkDeepHierarchyConcurrent tests deep component trees with concurrency
func BenchmarkDeepHierarchyConcurrent(b *testing.B) {
	benchmarkDeepHierarchy(b, 4, 20)
}

// benchmarkRendering is a helper that benchmarks rendering with configurable worker count
func benchmarkRendering(b *testing.B, workers int, componentCount int) {
	type props struct{}
	type childProps struct{ Key string }

	var renderCount atomic.Int32
	child := func(ctx Ctx, props childProps) h.Node {
		renderCount.Add(1)
		get, set := UseState(ctx, 0)
		UseEffect(ctx, func() Cleanup {
			if get() == 0 {
				set(1)
			}
			return nil
		})
		return h.Li(h.Class(props.Key), h.Textf("%s:%d", props.Key, get()))
	}

	parent := func(ctx Ctx, _ props) h.Node {
		items := make([]h.Item, componentCount)
		for i := 0; i < componentCount; i++ {
			key := string(rune('a' + i))
			items[i] = Render(ctx, child, childProps{Key: key}, WithKey(key)).(h.Item)
		}
		return h.Ul(items...)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		renderCount.Store(0)
		sess := NewSession(parent, props{})
		if workers > 0 {
			sess.EnableConcurrentRendering(workers)
		}
		sess.SetPatchSender(func([]diff.Op) error { return nil })
		sess.InitialStructured()
		_ = sess.Flush()
	}
}

// benchmarkDeepHierarchy benchmarks rendering of deep component trees
func benchmarkDeepHierarchy(b *testing.B, workers int, depth int) {
	type levelProps struct{ Level int }

	var level Component[levelProps]
	level = func(ctx Ctx, props levelProps) h.Node {
		get, set := UseState(ctx, 0)
		UseEffect(ctx, func() Cleanup {
			if get() == 0 {
				set(1)
			}
			return nil
		})

		if props.Level < depth {
			return h.Div(
				h.Textf("Level %d", props.Level),
				Render(ctx, level, levelProps{Level: props.Level + 1}),
			)
		}
		return h.Div(h.Textf("Leaf %d", props.Level))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sess := NewSession(level, levelProps{Level: 0})
		if workers > 0 {
			sess.EnableConcurrentRendering(workers)
		}
		sess.SetPatchSender(func([]diff.Op) error { return nil })
		sess.InitialStructured()
		_ = sess.Flush()
	}
}

// BenchmarkEventBurstSequential benchmarks rapid event handling without concurrency
func BenchmarkEventBurstSequential(b *testing.B) {
	benchmarkEventBurst(b, 0, 100)
}

// BenchmarkEventBurstConcurrent benchmarks rapid event handling with concurrency
func BenchmarkEventBurstConcurrent(b *testing.B) {
	benchmarkEventBurst(b, 4, 100)
}

func benchmarkEventBurst(b *testing.B, workers int, eventCount int) {
	type props struct{}

	counter := func(ctx Ctx, _ props) h.Node {
		get, set := UseState(ctx, 0)
		handler := func(h.Event) h.Updates {
			set(get() + 1)
			return nil
		}
		return h.Div(
			h.Button(
				h.On("click", handler),
				h.Textf("%d", get()),
			),
		)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		sess := NewSession(counter, props{})
		if workers > 0 {
			sess.EnableConcurrentRendering(workers)
		}
		sess.SetPatchSender(func([]diff.Op) error { return nil })

		structured := sess.InitialStructured()
		handlerID := findHandlerAttr(structured, "data-onclick")

		for j := 0; j < eventCount; j++ {
			_ = sess.DispatchEvent(handlerID, dom.Event{Name: "click"})
		}
	}
}
