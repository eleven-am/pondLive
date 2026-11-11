package shared

import "sync/atomic"

type BroadcastOpt func(*broadcastConfig)

type broadcastConfig struct {
	backend func() Backend
}

func WithBroadcastBackend(f func() Backend) BroadcastOpt {
	return func(c *broadcastConfig) { c.backend = f }
}

type Broadcast[T any] struct {
	b      Backend
	joined atomic.Int64
}

func NewBroadcast[T any](opts ...BroadcastOpt) *Broadcast[T] {
	var cfg broadcastConfig
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.backend != nil {
		return &Broadcast[T]{b: cfg.backend()}
	}
	return &Broadcast[T]{b: NewInProcessStream()}
}

func (b *Broadcast[T]) Send(v T) {
	publish(b.b, "Broadcast.Send", func(Envelope) any { return v })
}

func (b *Broadcast[T]) Subscribers() int { return int(b.joined.Load()) }

func (b *Broadcast[T]) Join(deliver func(T)) (leave func()) {
	unsub := b.b.Subscribe(func(env Envelope) {
		v, _ := env.Value.(T)
		deliver(v)
	})
	b.joined.Add(1)
	var left atomic.Bool
	return func() {
		if left.CompareAndSwap(false, true) {
			unsub()
			b.joined.Add(-1)
		}
	}
}
