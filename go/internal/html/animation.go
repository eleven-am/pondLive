package html

import "github.com/eleven-am/pondlive/go/internal/dom2"

// AnimationEvent represents CSS animation events.
type AnimationEvent struct {
	dom2.Event
	AnimationName string  // Name of the animation
	ElapsedTime   float64 // Time elapsed since animation started
	PseudoElement string  // Pseudo-element on which animation runs
}

// Props returns the list of properties this event needs from the client.
func (AnimationEvent) props() []string {
	return []string{
		"event.animationName",
		"event.elapsedTime",
		"event.pseudoElement",
	}
}

// AnimationHandler provides animation event handlers.
type AnimationHandler struct {
	ref dom2.RefListener
}

// NewAnimationHandler creates a new AnimationHandler.
func NewAnimationHandler(ref dom2.RefListener) *AnimationHandler {
	return &AnimationHandler{ref: ref}
}

// OnAnimationStart registers a handler for the "animationstart" event.
func (h *AnimationHandler) OnAnimationStart(handler func(AnimationEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationstart", wrapped, AnimationEvent{}.props())
}

// OnAnimationEnd registers a handler for the "animationend" event.
func (h *AnimationHandler) OnAnimationEnd(handler func(AnimationEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationend", wrapped, AnimationEvent{}.props())
}

// OnAnimationIteration registers a handler for the "animationiteration" event.
func (h *AnimationHandler) OnAnimationIteration(handler func(AnimationEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationiteration", wrapped, AnimationEvent{}.props())
}

// OnAnimationCancel registers a handler for the "animationcancel" event.
func (h *AnimationHandler) OnAnimationCancel(handler func(AnimationEvent) dom2.Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationcancel", wrapped, AnimationEvent{}.props())
}

func buildAnimationEvent(evt dom2.Event) AnimationEvent {
	detail := extractDetail(evt.Payload)
	return AnimationEvent{
		Event:         evt,
		AnimationName: PayloadString(detail, "event.animationName", ""),
		ElapsedTime:   payloadFloat(detail, "event.elapsedTime", 0),
		PseudoElement: PayloadString(detail, "event.pseudoElement", ""),
	}
}
