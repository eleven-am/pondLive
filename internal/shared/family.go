package shared

import "sync"

type SharedFamily[K comparable, T any] struct {
	init func(K) T
	opts []SharedOpt

	mu sync.Mutex
	m  map[K]*familyEntry[T]
}

type familyEntry[T any] struct {
	sh   *Shared[T]
	refs int
}

func NewSharedFamily[K comparable, T any](init func(K) T, opts ...SharedOpt) *SharedFamily[K, T] {
	return &SharedFamily[K, T]{init: init, opts: opts, m: make(map[K]*familyEntry[T])}
}

func (f *SharedFamily[K, T]) Key(k K) *Shared[T] {
	return &Shared[T]{k: &keyRef[K, T]{f: f, key: k}}
}

func (f *SharedFamily[K, T]) Len() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.m)
}

type keyRef[K comparable, T any] struct {
	f   *SharedFamily[K, T]
	key K
}

func (r *keyRef[K, T]) live() *Shared[T] {
	r.f.mu.Lock()
	defer r.f.mu.Unlock()
	if e := r.f.m[r.key]; e != nil {
		return e.sh
	}
	return nil
}

func (r *keyRef[K, T]) initial() T { return r.f.init(r.key) }

func (r *keyRef[K, T]) acquire() (*Shared[T], func()) {
	f := r.f
	f.mu.Lock()
	e := f.m[r.key]
	if e == nil {
		e = &familyEntry[T]{sh: NewShared(f.init(r.key), f.opts...)}
		f.m[r.key] = e
	}
	e.refs++
	f.mu.Unlock()
	return e.sh, func() {
		f.mu.Lock()
		defer f.mu.Unlock()
		e.refs--
		if e.refs == 0 && f.m[r.key] == e {
			delete(f.m, r.key)
		}
	}
}
