package html

// AnimationEvent represents CSS animation events.
type AnimationEvent struct {
	Event
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
	ref RefListener
}

// NewAnimationHandler creates a new AnimationHandler.
func NewAnimationHandler(ref RefListener) *AnimationHandler {
	return &AnimationHandler{ref: ref}
}

// OnAnimationStart registers a handler for the "animationstart" event.
func (h *AnimationHandler) OnAnimationStart(handler func(AnimationEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationstart", wrapped, AnimationEvent{}.props())
}

// OnAnimationEnd registers a handler for the "animationend" event.
func (h *AnimationHandler) OnAnimationEnd(handler func(AnimationEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationend", wrapped, AnimationEvent{}.props())
}

// OnAnimationIteration registers a handler for the "animationiteration" event.
func (h *AnimationHandler) OnAnimationIteration(handler func(AnimationEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationiteration", wrapped, AnimationEvent{}.props())
}

// OnAnimationCancel registers a handler for the "animationcancel" event.
func (h *AnimationHandler) OnAnimationCancel(handler func(AnimationEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildAnimationEvent(evt)) }
	h.ref.AddListener("animationcancel", wrapped, AnimationEvent{}.props())
}

func buildAnimationEvent(evt Event) AnimationEvent {
	return AnimationEvent{
		Event:         evt,
		AnimationName: PayloadString(evt.Payload, "event.animationName", ""),
		ElapsedTime:   payloadFloat(evt.Payload, "event.elapsedTime", 0),
		PseudoElement: PayloadString(evt.Payload, "event.pseudoElement", ""),
	}
}
