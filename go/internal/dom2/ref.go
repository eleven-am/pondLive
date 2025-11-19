package dom2

import (
	"fmt"
	"strings"
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
	handlerIDs  map[string]string
	attached    bool
	stateGetter func() any
	stateSetter func(any)
	listenersMu sync.RWMutex
	listeners   map[string]*listenerBucket
	listenerGen uint64
}

// RefListener exposes the subset of the ElementRef API required by code that
// only needs to register listeners.
type RefListener interface {
	AddListener(event string, handler any, props []string)
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

// SetHandlerID stores the handler ID for a specific event after finalization.
func (r *ElementRef[T]) SetHandlerID(event string, handlerID string) {
	if r == nil || event == "" || handlerID == "" {
		return
	}
	r.bindingsMu.Lock()
	defer r.bindingsMu.Unlock()
	if r.handlerIDs == nil {
		r.handlerIDs = make(map[string]string)
	}
	r.handlerIDs[event] = handlerID
}

// GetHandlerID returns the handler ID for a specific event.
func (r *ElementRef[T]) GetHandlerID(event string) string {
	if r == nil {
		return ""
	}
	r.bindingsMu.Lock()
	defer r.bindingsMu.Unlock()
	return r.handlerIDs[event]
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
	r.listenersMu.Lock()
	r.listenerGen++
	r.listenersMu.Unlock()
}

// ResetAttachment clears the internal attachment guard so the ref can be
// attached during the next render pass.
func (r *ElementRef[T]) ResetAttachment() {
	r.resetAttachment()
}

type listenerBucket struct {
	handlers   []EventHandler
	props      []string
	generation uint64
}

func (r *ElementRef[T]) AddListener(event string, handler any, props []string) {
	fn := normalizeEventHandler(handler)
	if r == nil || event == "" || fn == nil {
		return
	}

	r.bindingsMu.Lock()
	if r.bindings == nil {
		r.bindings = make(map[string]EventBinding)
	}
	binding, exists := r.bindings[event]
	if !exists {
		binding = EventBinding{
			Handler: func(evt Event) Updates {
				return r.dispatchEvent(event, evt)
			},
		}
	}
	binding.Props = mergeSelectorLists(binding.Props, props)
	r.bindings[event] = binding
	r.bindingsMu.Unlock()

	r.listenersMu.Lock()
	defer r.listenersMu.Unlock()
	if r.listeners == nil {
		r.listeners = make(map[string]*listenerBucket)
	}
	bucket := r.listeners[event]
	if bucket == nil {
		bucket = &listenerBucket{}
		r.listeners[event] = bucket
	}
	if bucket.generation != r.listenerGen {
		if len(bucket.handlers) > 0 {
			for i := range bucket.handlers {
				bucket.handlers[i] = nil
			}
			bucket.handlers = bucket.handlers[:0]
		}
		bucket.props = nil
		bucket.generation = r.listenerGen
	}
	bucket.handlers = append(bucket.handlers, fn)
	bucket.props = mergeSelectorLists(bucket.props, props)
}

func (r *ElementRef[T]) dispatchEvent(event string, evt Event) Updates {
	if r == nil {
		return nil
	}
	r.listenersMu.RLock()
	bucket := r.listeners[event]
	if bucket == nil || len(bucket.handlers) == 0 {
		r.listenersMu.RUnlock()
		return nil
	}
	handlers := append([]EventHandler(nil), bucket.handlers...)
	r.listenersMu.RUnlock()

	var result Updates
	for _, handler := range handlers {
		if handler == nil {
			continue
		}
		if out := handler(evt); out != nil {
			result = out
		}
	}
	return result
}

// Ref returns the ElementRef itself, primarily used for nil-safe chaining and type assertions.
// This method is useful when working with optional refs or when you need to pass the ref explicitly.
//
// Example:
//
//	var optionalRef *dom2.ElementRef[HTMLDivElement]
//	// Safely get ref without panicking on nil
//	if ref := optionalRef.Ref(); ref != nil {
//	    // Use ref safely
//	}
//
// Note: This method returns nil if called on a nil receiver, making it safe for nil-checking patterns.
func (r *ElementRef[T]) Ref() *ElementRef[T] {
	if r == nil {
		return nil
	}
	return r
}

// On registers a generic event handler that automatically captures all serializable event properties.
// The handler receives an Event with all properties in evt.Payload (map[string]any).
// Users must type-assert values from the payload map themselves.
//
// Example:
//
//	buttonRef := dom2.NewElementRef("btn-1", HTMLButtonElement{})
//	buttonRef.On("customEvent", func(evt dom2.Event) dom2.Updates {
//	    if detail, ok := evt.Payload["detail"].(string); ok {
//	        fmt.Println("Detail:", detail)
//	    }
//	    return nil
//	})
//
// This method coexists with typed methods (provided by wrapper layers) which provide type safety.
// Use this for custom events or when you need maximum flexibility.
func (r *ElementRef[T]) On(eventName string, handler func(Event) Updates) {
	if r == nil || handler == nil {
		return
	}
	r.AddListener(eventName, handler, []string{CaptureAllProperties})
}

// AttachElementRef wires the provided ref into the element. Each ref may only
// be attached once per render tree; attempting to reuse it without calling
// ResetAttachment will panic to surface the bug early.
func AttachElementRef[T ElementDescriptor](ref *ElementRef[T], node *StructuredNode) {
	if ref == nil || node == nil {
		return
	}
	if ref.attached {
		panic("dom2: element ref attached multiple times")
	}
	if node.Descriptor == nil {
		panic(fmt.Sprintf("dom2: element <%s> missing descriptor; regenerate builders", node.Tag))
	}
	if _, ok := any(node.Descriptor).(T); !ok {
		panic(fmt.Sprintf("dom2: cannot attach ref for %T to <%s>", ref.descriptor, node.Tag))
	}
	ref.attached = true
	if node.Events == nil {
		node.Events = make(map[string]EventBinding)
	}
	ref.bindingsMu.Lock()
	for event, binding := range ref.bindings {
		if existing, exists := node.Events[event]; exists {
			node.Events[event] = MergeEventBinding(existing, binding)
			continue
		}
		node.Events[event] = binding
	}
	ref.bindingsMu.Unlock()
	node.RefID = ref.id
}

func refEventBindingKey(id, event string) string {
	id = strings.TrimSpace(id)
	event = strings.TrimSpace(event)
	if id == "" || event == "" {
		return ""
	}
	return fmt.Sprintf("%s/%s", id, strings.ToLower(event))
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

func normalizeEventHandler(handler any) EventHandler {
	switch h := handler.(type) {
	case nil:
		return nil
	case EventHandler:
		return h
	case func(Event) Updates:
		return h
	default:
		return nil
	}
}

func mergeSelectorLists(base, add []string) []string {
	if len(base) == 0 && len(add) == 0 {
		return nil
	}
	seen := make(map[string]struct{}, len(base)+len(add))
	out := make([]string, 0, len(base)+len(add))
	appendVals := func(values []string) {
		for _, v := range values {
			if v == "" {
				continue
			}
			if _, ok := seen[v]; ok {
				continue
			}
			seen[v] = struct{}{}
			out = append(out, v)
		}
	}
	appendVals(base)
	appendVals(add)
	if len(out) == 0 {
		return nil
	}
	return out
}

// ElementRefFactory constructs typed refs for descriptors. It is installed by
// the runtime so the html layer can request refs without depending on the
// runtime package directly.
type ElementRefFactory func(ctx any, descriptor ElementDescriptor) any

var (
	elementRefFactory ElementRefFactory
	factoryMu         sync.RWMutex
)

// InstallElementRefFactory registers the global factory used by useElementRef.
// Installing more than once panics to surface initialization issues early.
func InstallElementRefFactory(fn ElementRefFactory) {
	if fn == nil {
		panic("dom2: element ref factory cannot be nil")
	}
	factoryMu.Lock()
	defer factoryMu.Unlock()
	if elementRefFactory != nil {
		panic("dom2: element ref factory already installed")
	}
	elementRefFactory = fn
}

// AcquireElementRef invokes the registered factory to obtain a ref handle for
// the supplied descriptor.
func AcquireElementRef(ctx any, descriptor ElementDescriptor) any {
	factoryMu.RLock()
	fn := elementRefFactory
	factoryMu.RUnlock()
	if fn == nil {
		panic("dom2: element ref factory not installed")
	}
	return fn(ctx, descriptor)
}
