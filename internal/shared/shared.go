package shared

import (
	"sync/atomic"

	"github.com/eleven-am/pondlive/internal/effectguard"
)

type BackendFactory func(initial any) Backend

type sharedConfig struct {
	backend BackendFactory
}

// TODO: add WithName and WithCodec options so a multi-server Backend can address one hub across processes
type SharedOpt func(*sharedConfig)

func WithBackend(f BackendFactory) SharedOpt {
	return func(c *sharedConfig) { c.backend = f }
}

func newBackend(initial any, opts []SharedOpt) Backend {
	var cfg sharedConfig
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.backend != nil {
		return cfg.backend(initial)
	}
	return NewInProcess(initial)
}

func publish(b Backend, op string, next func(prev Envelope) any) {
	if effectguard.Refuse(op) {
		return
	}
	b.Publish(next)
}

type keyed[T any] interface {
	live() *Shared[T]
	acquire() (*Shared[T], func())
	initial() T
}

type Shared[T any] struct {
	b      Backend
	k      keyed[T]
	joined atomic.Int64
}

func NewShared[T any](v T, opts ...SharedOpt) *Shared[T] {
	return &Shared[T]{b: newBackend(v, opts)}
}

func (s *Shared[T]) Get() T {
	if s.k != nil {
		if inner := s.k.live(); inner != nil {
			return inner.Get()
		}
		return s.k.initial()
	}
	v, _ := s.b.Latest().Value.(T)
	return v
}

func (s *Shared[T]) Set(v T) {
	s.update("Shared.Set", func(T) T { return v })
}

func (s *Shared[T]) Update(fn func(T) T) {
	s.update("Shared.Update", fn)
}

func (s *Shared[T]) update(op string, fn func(T) T) {
	if s.k != nil {
		if inner := s.k.live(); inner != nil {
			inner.update(op, fn)
		}
		return
	}
	publish(s.b, op, func(prev Envelope) any {
		cur, _ := prev.Value.(T)
		return fn(cur)
	})
}

func (s *Shared[T]) Subscribers() int {
	if s.k != nil {
		if inner := s.k.live(); inner != nil {
			return inner.Subscribers()
		}
		return 0
	}
	return int(s.joined.Load())
}

type Member[T any] struct {
	sh    *Shared[T]
	unsub func()
	free  func()
	left  atomic.Bool
}

func (s *Shared[T]) Join(notify func()) *Member[T] {
	if s.k != nil {
		inner, release := s.k.acquire()
		m := inner.Join(notify)
		m.free = release
		return m
	}
	m := &Member[T]{sh: s}
	m.unsub = s.b.Subscribe(func(Envelope) { notify() })
	s.joined.Add(1)
	return m
}

func (m *Member[T]) Latest() T {
	v, _ := m.sh.b.Latest().Value.(T)
	return v
}

func (m *Member[T]) Leave() {
	if !m.left.CompareAndSwap(false, true) {
		return
	}
	m.unsub()
	m.sh.joined.Add(-1)
	if m.free != nil {
		m.free()
	}
}
