package handlers

import (
	"fmt"
	"reflect"
	"sync"
)

type registry struct {
	mu       sync.RWMutex
	next     int
	handlers map[ID]Handler
	lookup   map[uintptr]ID
	reverse  map[ID]uintptr
}

// NewRegistry constructs a handler registry.
func NewRegistry() Registry {
	return &registry{
		handlers: map[ID]Handler{},
		lookup:   map[uintptr]ID{},
		reverse:  map[ID]uintptr{},
	}
}

func (r *registry) Ensure(fn Handler) ID {
	if fn == nil {
		return ""
	}
	ptr := reflect.ValueOf(fn).Pointer()
	r.mu.Lock()
	defer r.mu.Unlock()
	if id, ok := r.lookup[ptr]; ok {
		if r.handlers[id] == nil {
			return ""
		}
		return id
	}
	r.next++
	id := ID(fmt.Sprintf("h%d", r.next))
	r.lookup[ptr] = id
	r.handlers[id] = fn
	r.reverse[id] = ptr
	return id
}

func (r *registry) Get(id ID) (Handler, bool) {
	if id == "" {
		return nil, false
	}
	r.mu.RLock()
	defer r.mu.RUnlock()
	h, ok := r.handlers[id]
	return h, ok
}

func (r *registry) Remove(id ID) {
	if id == "" {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.handlers, id)
	if ptr, ok := r.reverse[id]; ok {
		delete(r.lookup, ptr)
	}
	delete(r.reverse, id)
}
