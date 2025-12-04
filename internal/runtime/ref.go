package runtime

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/work"
)

type ElementRef struct {
	id         string
	handlers   map[string][]work.Handler
	attached   bool
	generation uint64
}

func (r *ElementRef) RefID() string {
	if r == nil {
		return ""
	}
	return r.id
}

func (r *ElementRef) AddHandler(event string, handler work.Handler) {
	if r == nil || event == "" {
		return
	}

	if r.handlers == nil {
		r.handlers = make(map[string][]work.Handler)
	}

	r.handlers[event] = append(r.handlers[event], handler)
}

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

func (r *ElementRef) AttachTo(el *work.Element) {
	if r == nil || el == nil {
		return
	}

	el.RefID = r.id

	for _, event := range r.Events() {
		handler := r.ProxyHandler(event)
		if el.Handlers == nil {
			el.Handlers = make(map[string]work.Handler)
		}
		el.Handlers[event] = handler
	}
}
