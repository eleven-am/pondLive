package runtime

import (
	"sync"
	"sync/atomic"

	"github.com/gammazero/deque"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/shared"
)

const broadcastInboxCap = 256

type externalSyncer interface {
	syncFromSource()
}

func (s *Session) trackExternal(x externalSyncer) {
	if s.externals == nil {
		s.externals = make(map[externalSyncer]struct{})
	}
	s.externals[x] = struct{}{}
}

func (s *Session) untrackExternal(x externalSyncer) {
	delete(s.externals, x)
}

func (s *Session) syncExternals() {
	for x := range s.externals {
		x.syncFromSource()
	}
}

type sharedCell[T any] struct {
	s       *Session
	o       *reactive.Owner
	sig     *reactive.Signal[T]
	m       *shared.Member[T]
	pending atomic.Bool
	dead    atomic.Bool
}

func (c *sharedCell[T]) Get() T  { return c.sig.Get() }
func (c *sharedCell[T]) Peek() T { return c.sig.Peek() }

func (c *sharedCell[T]) notify() {
	if c.dead.Load() || !c.pending.CompareAndSwap(false, true) {
		return
	}
	if err := c.s.post(message{kind: msgSetter, thunk: c.apply}); err != nil {
		c.pending.Store(false)
	}
}

func (c *sharedCell[T]) apply() {
	c.pending.Store(false)
	c.syncFromSource()
}

func (c *sharedCell[T]) syncFromSource() {
	if c.dead.Load() || c.o.Disposed() {
		return
	}
	c.sig.Set(c.m.Latest())
}

func (c *sharedCell[T]) release() {
	c.dead.Store(true)
	c.m.Leave()
	c.s.untrackExternal(c)
}

func UseShared[T any](ctx *Ctx, sh *shared.Shared[T]) ReadSignal[T] {
	return reactive.Cell(ctx.o, func() *sharedCell[T] {
		c := &sharedCell[T]{s: ctx.s, o: ctx.o}
		c.m = sh.Join(c.notify)
		c.sig = reactive.NewSignal(ctx.s.graph, c.m.Latest())
		ctx.s.trackExternal(c)
		ctx.o.OnCleanup(c.release)
		return c
	})
}

type broadcastCell[T any] struct {
	s       *Session
	o       *reactive.Owner
	handler func(T)
	leave   func()
	dead    atomic.Bool

	mu        sync.Mutex
	q         deque.Deque[T]
	scheduled bool
	dropped   int

	batch []T
}

func (c *broadcastCell[T]) push(v T) {
	if c.dead.Load() {
		return
	}
	c.mu.Lock()
	if c.q.Len() >= broadcastInboxCap {
		c.q.PopFront()
		c.dropped++
	}
	c.q.PushBack(v)
	if c.scheduled {
		c.mu.Unlock()
		return
	}
	c.scheduled = true
	c.mu.Unlock()
	if err := c.s.post(message{kind: msgStep, thunk: c.drain}); err != nil {
		c.mu.Lock()
		c.scheduled = false
		c.q.Clear()
		c.dropped = 0
		c.mu.Unlock()
	}
}

func (c *broadcastCell[T]) drain() {
	if c.s.deferred.Load() {
		c.s.heldDrains = append(c.s.heldDrains, c.drain)
		return
	}
	c.mu.Lock()
	c.scheduled = false
	dropped := c.dropped
	c.dropped = 0
	c.batch = c.batch[:0]
	for c.q.Len() > 0 {
		c.batch = append(c.batch, c.q.PopFront())
	}
	c.mu.Unlock()

	if dropped > 0 && !c.dead.Load() {
		c.s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "broadcast:drop",
			Message: "broadcast inbox full, oldest events dropped",
			Meta: map[string]any{
				"component": c.o.Info.Name,
				"dropped":   dropped,
			},
		})
	}
	for i := range c.batch {
		v := c.batch[i]
		if c.dead.Load() || c.o.Disposed() {
			break
		}
		c.s.invokeHandler(c.o, func() { c.handler(v) })
	}
	clear(c.batch)
}

func (c *broadcastCell[T]) release() {
	c.dead.Store(true)
	c.leave()
	c.mu.Lock()
	c.q.Clear()
	c.mu.Unlock()
}

func UseBroadcast[T any](ctx *Ctx, b *shared.Broadcast[T], handler func(T)) {
	reactive.Cell(ctx.o, func() *broadcastCell[T] {
		c := &broadcastCell[T]{s: ctx.s, o: ctx.o, handler: handler}
		c.leave = b.Join(c.push)
		ctx.o.OnCleanup(c.release)
		return c
	})
}
