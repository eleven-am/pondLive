package runtime2

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// ElementRef represents a stable reference to a rendered element.
// It can store event handlers that will be attached to the element.
// Multiple handlers for the same event are accumulated and all executed.
type ElementRef struct {
	id         string
	handlers   map[string][]work.Handler
	attached   bool
	generation uint64
}

// RefID returns the unique identifier for this element reference.
func (r *ElementRef) RefID() string {
	if r == nil {
		return ""
	}
	return r.id
}

// AddHandler registers an event handler on the ref.
// Multiple handlers for the same event are accumulated.
// The handler will be applied to the element when attached.
func (r *ElementRef) AddHandler(event string, handler work.Handler) {
	if r == nil || event == "" {
		return
	}

	if r.handlers == nil {
		r.handlers = make(map[string][]work.Handler)
	}

	r.handlers[event] = append(r.handlers[event], handler)
}

// Events returns all event names that have handlers registered.
// Implements work.HandlerProvider.
func (r *ElementRef) Events() []string {
	if r == nil || len(r.handlers) == 0 {
		return nil
	}

	events := make([]string, 0, len(r.handlers))
	for event, handlers := range r.handlers {
		if len(handlers) > 0 {
			events = append(events, event)
		}
	}
	return events
}

// ProxyHandler returns a single handler that dispatches to all accumulated handlers
// for the given event. This is used when attaching the ref to an element.
// Implements work.HandlerProvider.
func (r *ElementRef) ProxyHandler(event string) work.Handler {
	if r == nil {
		return work.Handler{}
	}

	generation := r.generation
	return work.Handler{
		EventOptions: r.MergedOptions(event),
		Fn: func(evt work.Event) work.Updates {
			return r.dispatchEvent(event, evt, generation)
		},
	}
}

// dispatchEvent executes all handlers for the given event.
// Only handlers matching the generation are executed.
func (r *ElementRef) dispatchEvent(event string, evt work.Event, generation uint64) work.Updates {
	if r == nil || generation != r.generation {
		return nil
	}

	handlers := r.handlers[event]
	if len(handlers) == 0 {
		return nil
	}

	var result work.Updates
	for _, handler := range handlers {
		if handler.Fn == nil {
			continue
		}
		if out := handler.Fn(evt); out != nil {
			result = out
		}
	}
	return result
}

// MergedOptions returns the merged EventOptions for all handlers of an event.
// Props and Listen are combined from all handlers.
func (r *ElementRef) MergedOptions(event string) metadata.EventOptions {
	if r == nil || len(r.handlers[event]) == 0 {
		return metadata.EventOptions{}
	}

	var merged metadata.EventOptions
	for _, handler := range r.handlers[event] {
		merged = work.MergeEventOptions(merged, handler.EventOptions)
	}
	return merged
}

// ResetAttachment clears the attachment flag and increments generation.
// This invalidates old proxy handlers and should be called between renders.
func (r *ElementRef) ResetAttachment() {
	if r == nil {
		return
	}
	r.attached = false
	r.generation++

	for event := range r.handlers {
		r.handlers[event] = r.handlers[event][:0]
	}
}
