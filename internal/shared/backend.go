package shared

import (
	"sync"
	"sync/atomic"
)

type Envelope struct {
	Version uint64
	Value   any
}

type Backend interface {
	Publish(next func(prev Envelope) any) Envelope
	Latest() Envelope
	Subscribe(deliver func(Envelope)) (unsubscribe func())
}

type inProcess struct {
	pubMu   sync.Mutex
	version uint64
	retain  bool
	latest  atomic.Pointer[Envelope]

	subMu sync.Mutex
	subs  atomic.Pointer[[]*subscriber]
}

type subscriber struct {
	deliver func(Envelope)
}

func NewInProcess(initial any) Backend {
	b := &inProcess{retain: true}
	b.latest.Store(&Envelope{Value: initial})
	return b
}

func NewInProcessStream() Backend {
	return &inProcess{}
}

func (b *inProcess) Latest() Envelope {
	if p := b.latest.Load(); p != nil {
		return *p
	}
	return Envelope{}
}

func (b *inProcess) Publish(next func(prev Envelope) any) Envelope {
	b.pubMu.Lock()
	defer b.pubMu.Unlock()
	prev := b.Latest()
	prev.Version = b.version
	env := Envelope{Version: b.version + 1, Value: next(prev)}
	b.version = env.Version
	if b.retain {
		b.latest.Store(&env)
	}
	if subs := b.subs.Load(); subs != nil {
		for _, s := range *subs {
			s.deliver(env)
		}
	}
	return env
}

func (b *inProcess) Subscribe(deliver func(Envelope)) func() {
	sub := &subscriber{deliver: deliver}
	b.subMu.Lock()
	var old []*subscriber
	if p := b.subs.Load(); p != nil {
		old = *p
	}
	grown := make([]*subscriber, len(old), len(old)+1)
	copy(grown, old)
	grown = append(grown, sub)
	b.subs.Store(&grown)
	b.subMu.Unlock()

	var once sync.Once
	return func() {
		once.Do(func() {
			b.subMu.Lock()
			defer b.subMu.Unlock()
			cur := *b.subs.Load()
			kept := make([]*subscriber, 0, len(cur))
			for _, s := range cur {
				if s != sub {
					kept = append(kept, s)
				}
			}
			b.subs.Store(&kept)
		})
	}
}
