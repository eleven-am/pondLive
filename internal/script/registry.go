package script

import (
	"errors"
	"fmt"
	"io/fs"
	"sort"
	"sync"
)

type Limits struct {
	MaxMessage int
	Rate       float64
	Burst      int
}

type Feature string

type Entry interface {
	Key() string
	Source() string
	Entry() string
	FS() fs.FS
	Location() string
	Limits() Limits
	Feature() Feature
	DecodeAny(raw []byte) (any, error)
	ValidateAny(msg any) error
}

type Registry struct {
	mu      sync.Mutex
	entries map[string]Entry
	errs    []error
	sealed  bool
	result  error
}

var Default = NewRegistry()

func NewRegistry() *Registry {
	return &Registry{entries: make(map[string]Entry)}
}

func (r *Registry) Register(e Entry) Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sealed {
		panic(fmt.Sprintf("script: %s declared after the app was sealed; declare scripts at package level", e.Location()))
	}
	existing, ok := r.entries[e.Key()]
	if !ok {
		r.entries[e.Key()] = e
		return e
	}
	if existing.Source() == e.Source() && existing.Entry() == e.Entry() {
		return existing
	}
	r.errs = append(r.errs, fmt.Errorf("script key %s collides: %s and %s declare different sources", e.Key(), existing.Location(), e.Location()))
	return e
}

func (r *Registry) fail(err error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.sealed {
		return
	}
	r.errs = append(r.errs, err)
}

func (r *Registry) Seal() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if !r.sealed {
		r.sealed = true
		r.result = errors.Join(r.errs...)
	}
	return r.result
}

func (r *Registry) Sealed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.sealed
}

func (r *Registry) Entries() []Entry {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]Entry, 0, len(r.entries))
	for _, e := range r.entries {
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Key() < out[j].Key() })
	return out
}

func (r *Registry) Lookup(key string) (Entry, bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	e, ok := r.entries[key]
	return e, ok
}

func Register(e Entry) Entry { return Default.Register(e) }

func Seal() error { return Default.Seal() }

func Sealed() bool { return Default.Sealed() }
