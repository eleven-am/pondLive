package html

import "github.com/eleven-am/pondlive/go/internal/dom2"

// TransitionEvent represents CSS transition events.
type TransitionEvent struct {
	dom2.Event
	PropertyName  string  // Name of the CSS property transitioning
	ElapsedTime   float64 // Time elapsed since transition started
	PseudoElement string  // Pseudo-element on which transition runs
}

// Props returns the list of properties this event needs from the client.
func (TransitionEvent) props() []string {
	return []string{
		"event.propertyName",
		"event.elapsedTime",
		"event.pseudoElement",
	}
}

// TransitionHandler provides transition event handlers.
type TransitionHandler struct {
	ref dom2.RefListener
}

// NewTransitionHandler creates a new TransitionHandler.
func NewTransitionHandler(ref dom2.RefListener) *TransitionHandler {
	return &TransitionHandler{ref: ref}
}

// OnTransitionStart registers a handler for the "transitionstart" event.
func (h *TransitionHandler) OnTransitionStart(handler func(TransitionEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildTransitionEvent(evt)) }
	h.ref.AddListener("transitionstart", wrapped, TransitionEvent{}.props())
}

// OnTransitionEnd registers a handler for the "transitionend" event.
func (h *TransitionHandler) OnTransitionEnd(handler func(TransitionEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildTransitionEvent(evt)) }
	h.ref.AddListener("transitionend", wrapped, TransitionEvent{}.props())
}

// OnTransitionRun registers a handler for the "transitionrun" event.
func (h *TransitionHandler) OnTransitionRun(handler func(TransitionEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildTransitionEvent(evt)) }
	h.ref.AddListener("transitionrun", wrapped, TransitionEvent{}.props())
}

// OnTransitionCancel registers a handler for the "transitioncancel" event.
func (h *TransitionHandler) OnTransitionCancel(handler func(TransitionEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildTransitionEvent(evt)) }
	h.ref.AddListener("transitioncancel", wrapped, TransitionEvent{}.props())
}

func buildTransitionEvent(evt dom2.Event) TransitionEvent {
	detail := extractDetail(evt.Payload)
	return TransitionEvent{
		Event:         evt,
		PropertyName:  PayloadString(detail, "event.propertyName", ""),
		ElapsedTime:   payloadFloat(detail, "event.elapsedTime", 0),
		PseudoElement: PayloadString(detail, "event.pseudoElement", ""),
	}
}
