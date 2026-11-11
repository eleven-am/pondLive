package runtime

import (
	"errors"

	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

var (
	ErrStaleHandler   = errors.New("handler:stale")
	ErrExitingHandler = errors.New("handler:exiting")
	ErrNotOnActor     = errors.New("runtime: DispatchHandler must run on the session actor")
)

type handlerEntry struct {
	o      *reactive.Owner
	events []handlerFn
}

func (s *Session) registerHandlers(o *reactive.Owner, id uint32, events []handlerSlot) {
	compact := make([]handlerFn, len(events))
	for i, ev := range events {
		compact[i] = handlerFn{event: ev.event, fn: ev.fns[0]}
		if len(ev.fns) > 1 {
			compact[i].more = ev.fns[1:]
		}
	}
	s.handlers[id] = &handlerEntry{o: o, events: compact}
	o.OnCleanup(func() { delete(s.handlers, id) })
}

func (s *Session) HandlerCount() int { return len(s.handlers) }

func (s *Session) isExiting(o *reactive.Owner) bool {
	if len(s.exiting) == 0 {
		return false
	}
	for p := o; p != nil; p = p.Parent() {
		if _, ok := s.exiting[p]; ok {
			return true
		}
	}
	return false
}

func (s *Session) DispatchHandler(id uint32, event string, payload any) error {
	entry := s.handlers[id]
	if entry == nil {
		return ErrStaleHandler
	}
	var slot *handlerFn
	for i := range entry.events {
		if entry.events[i].event == event {
			slot = &entry.events[i]
			break
		}
	}
	if slot == nil {
		return ErrStaleHandler
	}
	if s.isExiting(entry.o) {
		return ErrExitingHandler
	}
	evt := work.Event{Name: event}
	if m, ok := payload.(map[string]any); ok {
		evt.Payload = m
	}
	first, more := slot.fn, slot.more
	s.invokeHandler(entry.o, func() {
		first(evt)
		for _, fn := range more {
			fn(evt)
		}
	})
	return nil
}
