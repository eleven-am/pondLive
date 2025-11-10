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
	keys     map[string]ID
	idKeys   map[ID]string
}

// NewRegistry constructs a handler registry.
func NewRegistry() Registry {
	return &registry{
		handlers: map[ID]Handler{},
		lookup:   map[uintptr]ID{},
		reverse:  map[ID]uintptr{},
		keys:     map[string]ID{},
		idKeys:   map[ID]string{},
	}
}

func (r *registry) Ensure(fn Handler, key string) ID {
	if fn == nil {
		return ""
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	if key != "" {
		if id, ok := r.keys[key]; ok {
			if r.handlers[id] != nil {
				return id
			}
		}
	}
	var ptr uintptr
	if key == "" {
		ptr = reflect.ValueOf(fn).Pointer()
		if id, ok := r.lookup[ptr]; ok {
			if r.handlers[id] == nil {
				return ""
			}
			return id
		}
	}
	r.next++
	id := ID(fmt.Sprintf("h%d", r.next))
	r.handlers[id] = fn
	if key != "" {
		r.keys[key] = id
		r.idKeys[id] = key
	} else {
		r.lookup[ptr] = id
		r.reverse[id] = ptr
	}
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
	if key, ok := r.idKeys[id]; ok {
		delete(r.keys, key)
		delete(r.idKeys, id)
		return
	}
	if ptr, ok := r.reverse[id]; ok {
		delete(r.lookup, ptr)
		delete(r.reverse, id)
	}
}
