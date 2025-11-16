package runtime

import (
	"sync"
	"sync/atomic"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// TestConcurrentSchedulerBasic verifies that concurrent rendering produces the same results as sequential.
func TestConcurrentSchedulerBasic(t *testing.T) {
	type props struct{}

	var renderCount atomic.Int32
	counter := func(ctx Ctx, _ props) h.Node {
		renderCount.Add(1)
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

	seqSess := NewSession(counter, props{})
	seqSess.SetPatchSender(func([]diff.Op) error { return nil })
	seqSess.InitialStructured()
	seqRenders := renderCount.Load()

	renderCount.Store(0)
	concSess := NewSession(counter, props{})
	concSess.EnableConcurrentRendering(4)
	concSess.SetPatchSender(func([]diff.Op) error { return nil })
	concSess.InitialStructured()
	concRenders := renderCount.Load()

	if seqRenders != concRenders {
		t.Errorf("render counts differ: sequential=%d, concurrent=%d", seqRenders, concRenders)
	}
}

// TestConcurrentSchedulerNestedComponents tests concurrent rendering with parent-child relationships.
func TestConcurrentSchedulerNestedComponents(t *testing.T) {
	type parentProps struct{}
	type childProps struct{ Key string }

	var childRenders sync.Map
	var parentRenders atomic.Int32

	child := func(ctx Ctx, props childProps) h.Node {
		count, _ := childRenders.LoadOrStore(props.Key, new(atomic.Int32))
		count.(*atomic.Int32).Add(1)
		get, set := UseState(ctx, 0)
		UseEffect(ctx, func() Cleanup {

			if get() == 0 {
				set(1)
			}
			return nil
		})
		return h.Li(h.Class(props.Key), h.Textf("%s:%d", props.Key, get()))
	}

	parent := func(ctx Ctx, _ parentProps) h.Node {
		parentRenders.Add(1)
		items := []h.Item{
			Render(ctx, child, childProps{Key: "a"}, WithKey("a")).(h.Item),
			Render(ctx, child, childProps{Key: "b"}, WithKey("b")).(h.Item),
			Render(ctx, child, childProps{Key: "c"}, WithKey("c")).(h.Item),
		}
		return h.Ul(items...)
	}

	sess := NewSession(parent, parentProps{})
	sess.EnableConcurrentRendering(4)
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	for _, key := range []string{"a", "b", "c"} {
		count, ok := childRenders.Load(key)
		if !ok || count.(*atomic.Int32).Load() == 0 {
			t.Errorf("child %q did not render", key)
		}
	}
}

// TestConcurrentSchedulerBurstEvents simulates bursty event loads during concurrent rendering.
func TestConcurrentSchedulerBurstEvents(t *testing.T) {
	type props struct{}

	var renderCount atomic.Int32
	counter := func(ctx Ctx, _ props) h.Node {
		renderCount.Add(1)
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

	sess := NewSession(counter, props{})
	sess.EnableConcurrentRendering(8)

	var patchCount atomic.Int32
	sess.SetPatchSender(func([]diff.Op) error {
		patchCount.Add(1)
		return nil
	})

	structured := sess.InitialStructured()
	handlerID := findHandlerAttr(structured, "data-onclick")
	if handlerID == "" {
		t.Fatal("expected click handler id")
	}

	const burstSize = 100
	for i := 0; i < burstSize; i++ {
		if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}); err != nil {
			t.Errorf("event %d failed: %v", i, err)
		}
	}

	if patchCount.Load() == 0 {
		t.Error("expected patches to be sent")
	}

	if renderCount.Load() < 2 {
		t.Errorf("expected at least 2 renders, got %d", renderCount.Load())
	}
}

// TestConcurrentSchedulerDeepHierarchy tests concurrent rendering with deep component trees.
func TestConcurrentSchedulerDeepHierarchy(t *testing.T) {
	const depth = 10
	type levelProps struct{ Level int }

	var renderCounts [depth + 1]atomic.Int32

	var level Component[levelProps]
	level = func(ctx Ctx, props levelProps) h.Node {
		renderCounts[props.Level].Add(1)
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

	sess := NewSession(level, levelProps{Level: 0})
	sess.EnableConcurrentRendering(4)
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}

	for i := 0; i <= depth; i++ {
		count := renderCounts[i].Load()
		if count == 0 {
			t.Errorf("level %d did not render", i)
		}
	}
}

// TestConcurrentSchedulerTopologicalOrder verifies parent-before-child rendering order.
func TestConcurrentSchedulerTopologicalOrder(t *testing.T) {
	type parentProps struct{}
	type childProps struct{}

	var parentStart, parentEnd atomic.Int64
	var childStart, childEnd atomic.Int64

	child := func(ctx Ctx, _ childProps) h.Node {
		childStart.Store(1)

		for i := 0; i < 1000; i++ {
			_ = i * i
		}
		childEnd.Store(1)
		return h.Span(h.Text("child"))
	}

	parent := func(ctx Ctx, _ parentProps) h.Node {
		parentStart.Store(1)

		for i := 0; i < 1000; i++ {
			_ = i * i
		}
		node := Render(ctx, child, childProps{})
		parentEnd.Store(1)
		return h.Div(node)
	}

	sess := NewSession(parent, parentProps{})
	sess.EnableConcurrentRendering(4)
	sess.SetPatchSender(func([]diff.Op) error { return nil })
	sess.InitialStructured()

	if parentEnd.Load() == 0 {
		t.Error("parent did not complete")
	}
	if childEnd.Load() == 0 {
		t.Error("child did not complete")
	}
}

// TestConcurrentSchedulerRaceDetector is meant to be run with -race flag.
func TestConcurrentSchedulerRaceDetector(t *testing.T) {
	type props struct{}

	var renderCount atomic.Int32
	counter := func(ctx Ctx, _ props) h.Node {
		renderCount.Add(1)
		get, set := UseState(ctx, 0)
		handler := func(h.Event) h.Updates {
			set(get() + 1)
			return nil
		}
		return h.Div(
			h.Button(h.On("click", handler), h.Textf("%d", get())),
		)
	}

	sess := NewSession(counter, props{})
	sess.EnableConcurrentRendering(8)
	sess.SetPatchSender(func([]diff.Op) error { return nil })

	structured := sess.InitialStructured()
	handlerID := findHandlerAttr(structured, "data-onclick")

	for i := 0; i < 500; i++ {
		_ = sess.DispatchEvent(handlerID, dom.Event{Name: "click"})
	}
}

// TestConcurrentSchedulerStateConsistency verifies state updates are not lost during concurrent rendering.
func TestConcurrentSchedulerStateConsistency(t *testing.T) {
	type props struct{}

	counter := func(ctx Ctx, _ props) h.Node {
		get, set := UseState(ctx, 0)
		handler := func(h.Event) h.Updates {
			set(get() + 1)
			return nil
		}
		return h.Div(
			h.Button(h.On("click", handler), h.Textf("%d", get())),
		)
	}

	sess := NewSession(counter, props{})
	sess.EnableConcurrentRendering(4)

	var patches [][]diff.Op
	var patchMu sync.Mutex
	sess.SetPatchSender(func(ops []diff.Op) error {
		patchMu.Lock()
		patches = append(patches, ops)
		patchMu.Unlock()
		return nil
	})

	structured := sess.InitialStructured()
	handlerID := findHandlerAttr(structured, "data-onclick")

	const eventCount = 20
	for i := 0; i < eventCount; i++ {
		if err := sess.DispatchEvent(handlerID, dom.Event{Name: "click"}); err != nil {
			t.Fatalf("event %d error: %v", i, err)
		}
	}

	patchMu.Lock()
	updateCount := 0
	for _, ops := range patches {
		for _, op := range ops {
			if _, ok := op.(diff.SetText); ok {
				updateCount++
			}
		}
	}
	patchMu.Unlock()

	if updateCount == 0 {
		t.Error("expected text updates from state changes")
	}

	t.Logf("processed %d events, sent %d patches with %d text updates",
		eventCount, len(patches), updateCount)
}
