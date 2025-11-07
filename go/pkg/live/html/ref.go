package html

import (
	"fmt"
	"sync"
)

// ElementRef provides a typed handle to a rendered element instance. The type
// parameter links the ref to the concrete descriptor generated for the element
// builder (for example HTMLVideoElement).
type ElementRef[T ElementDescriptor] struct {
	id          string
	descriptor  T
	bindings    map[string]EventBinding
	bindingsMu  sync.Mutex
	attached    bool
	stateGetter func() any
	stateSetter func(any)
	listenersMu sync.RWMutex
	listeners   map[string][]any
}

// NewElementRef constructs a ref bound to the provided descriptor and id. The
// hook runtime is responsible for injecting state accessors via installState.
func NewElementRef[T ElementDescriptor](id string, descriptor T) *ElementRef[T] {
	return &ElementRef[T]{
		id:         id,
		descriptor: descriptor,
		bindings:   make(map[string]EventBinding),
	}
}

// ID exposes the stable identifier associated with the ref.
func (r *ElementRef[T]) ID() string {
	if r == nil {
		return ""
	}
	return r.id
}

// Descriptor reports the descriptor associated with the ref.
func (r *ElementRef[T]) Descriptor() T {
	var zero T
	if r == nil {
		return zero
	}
	return r.descriptor
}

// DescriptorTag returns the HTML tag name associated with the ref.
func (r *ElementRef[T]) DescriptorTag() string {
	if r == nil {
		return ""
	}
	return r.descriptor.TagName()
}

// Bind registers an event binding on the ref. Attach will merge the binding
// into the owning element if that event has not been explicitly overridden by
// the element props.
func (r *ElementRef[T]) Bind(event string, binding EventBinding) {
	if r == nil || event == "" {
		return
	}
	r.bindingsMu.Lock()
	defer r.bindingsMu.Unlock()
	if r.bindings == nil {
		r.bindings = make(map[string]EventBinding)
	}
	r.bindings[event] = binding
}

// BindingSnapshot returns a copy of the event bindings registered on the ref.
func (r *ElementRef[T]) BindingSnapshot() map[string]EventBinding {
	if r == nil {
		return nil
	}
	r.bindingsMu.Lock()
	defer r.bindingsMu.Unlock()
	if len(r.bindings) == 0 {
		return nil
	}
	out := make(map[string]EventBinding, len(r.bindings))
	for event, binding := range r.bindings {
		out[event] = cloneEventBinding(binding)
	}
	return out
}

// CachedState returns the last cached state map for the ref.
func (r *ElementRef[T]) CachedState() any {
	if r == nil || r.stateGetter == nil {
		return nil
	}
	return r.stateGetter()
}

// updateState stores the supplied snapshot in the ref-local cache. The hook
// runtime injects the setter so that UseElement can back refs with a component
// UseState cell.
func (r *ElementRef[T]) updateState(next any) {
	if r == nil || r.stateSetter == nil {
		return
	}
	r.stateSetter(next)
}

// InstallState injects getter and setter callbacks used by the runtime to keep
// the element ref's cached state synchronized with hook state.
func (r *ElementRef[T]) InstallState(get func() any, set func(any)) {
	if r == nil {
		return
	}
	r.stateGetter = get
	r.stateSetter = set
}

func (r *ElementRef[T]) resetAttachment() {
	if r == nil {
		return
	}
	r.attached = false
}

// ResetAttachment clears the internal attachment guard so the ref can be
// attached during the next render pass.
func (r *ElementRef[T]) ResetAttachment() {
	r.resetAttachment()
}

func (r *ElementRef[T]) addListener(event string, listener any) {
	if r == nil || event == "" || listener == nil {
		return
	}
	r.listenersMu.Lock()
	defer r.listenersMu.Unlock()
	if r.listeners == nil {
		r.listeners = make(map[string][]any)
	}
	r.listeners[event] = append(r.listeners[event], listener)
}

func (r *ElementRef[T]) listenersFor(event string) []any {
	if r == nil || event == "" {
		return nil
	}
	r.listenersMu.RLock()
	defer r.listenersMu.RUnlock()
	listeners := r.listeners[event]
	if len(listeners) == 0 {
		return nil
	}
	out := make([]any, len(listeners))
	copy(out, listeners)
	return out
}

// Attach wires the provided ref into the element. The ref may only be attached
// once per render tree; attempting to reuse it will panic to surface the
// misconfiguration immediately.
func Attach[T ElementDescriptor](ref *ElementRef[T]) Prop {
	if ref == nil {
		return nil
	}
	return elementAttachProp[T]{ref: ref}
}

type elementAttachProp[T ElementDescriptor] struct {
	ref *ElementRef[T]
}

func (elementAttachProp[T]) isProp() {}

func (p elementAttachProp[T]) applyTo(e *Element) {
	if p.ref == nil || e == nil {
		return
	}
	if p.ref.attached {
		panic("html: element ref attached multiple times")
	}
	if e.Descriptor == nil {
		panic(fmt.Sprintf("html: element %q has no descriptor; regenerate builders", e.Tag))
	}
	if _, ok := any(e.Descriptor).(T); !ok {
		panic(fmt.Sprintf("html: cannot attach ref for %T to <%s>", p.ref.descriptor, e.Tag))
	}
	p.ref.attached = true
	if e.Events == nil {
		e.Events = map[string]EventBinding{}
	}
	p.ref.bindingsMu.Lock()
	for event, binding := range p.ref.bindings {
		if existing, exists := e.Events[event]; exists {
			e.Events[event] = mergeEventBinding(existing, binding)
			continue
		}
		e.Events[event] = binding
	}
	p.ref.bindingsMu.Unlock()
	if p.ref.id != "" {
		if e.Attrs == nil {
			e.Attrs = map[string]string{}
		}
		e.Attrs["data-live-ref"] = p.ref.id
	}
	e.RefID = p.ref.id
}

var applyRefDefaultsFunc = func(any) {}

// ApplyRefDefaults invokes any generator-provided default bindings for the ref.
func ApplyRefDefaults[T ElementDescriptor](ref *ElementRef[T]) {
	if ref == nil {
		return
	}
	applyRefDefaultsFunc(ref)
}

func cloneEventBinding(binding EventBinding) EventBinding {
	clone := binding
	if len(binding.Listen) > 0 {
		clone.Listen = append([]string(nil), binding.Listen...)
	}
	if len(binding.Props) > 0 {
		clone.Props = append([]string(nil), binding.Props...)
	}
	return clone
}
