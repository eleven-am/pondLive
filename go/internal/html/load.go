package html

// LoadEvent represents resource loading events (load, error, progress, etc).
type LoadEvent struct {
	Event
	Loaded float64 // Bytes loaded (for progress events)
	Total  float64 // Total bytes (for progress events)
}

// Props returns the list of properties this event needs from the client.
func (LoadEvent) props() []string {
	return []string{
		"event.loaded",
		"event.total",
	}
}

// LoadHandler provides resource loading event handlers.
type LoadHandler struct {
	ref RefListener
}

// NewLoadHandler creates a new LoadHandler.
func NewLoadHandler(ref RefListener) *LoadHandler {
	return &LoadHandler{ref: ref}
}

// OnLoad registers a handler for the "load" event.
func (h *LoadHandler) OnLoad(handler func(LoadEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildLoadEvent(evt)) }
	h.ref.AddListener("load", wrapped, LoadEvent{}.props())
}

// OnError registers a handler for the "error" event.
func (h *LoadHandler) OnError(handler func(LoadEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildLoadEvent(evt)) }
	h.ref.AddListener("error", wrapped, LoadEvent{}.props())
}

// OnAbort registers a handler for the "abort" event.
func (h *LoadHandler) OnAbort(handler func(LoadEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildLoadEvent(evt)) }
	h.ref.AddListener("abort", wrapped, LoadEvent{}.props())
}

// OnLoadStart registers a handler for the "loadstart" event.
func (h *LoadHandler) OnLoadStart(handler func(LoadEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildLoadEvent(evt)) }
	h.ref.AddListener("loadstart", wrapped, LoadEvent{}.props())
}

// OnLoadEnd registers a handler for the "loadend" event.
func (h *LoadHandler) OnLoadEnd(handler func(LoadEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildLoadEvent(evt)) }
	h.ref.AddListener("loadend", wrapped, LoadEvent{}.props())
}

// OnProgress registers a handler for the "progress" event.
func (h *LoadHandler) OnProgress(handler func(LoadEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildLoadEvent(evt)) }
	h.ref.AddListener("progress", wrapped, LoadEvent{}.props())
}

func buildLoadEvent(evt Event) LoadEvent {
	return LoadEvent{
		Event:  evt,
		Loaded: payloadFloat(evt.Payload, "event.loaded", 0),
		Total:  payloadFloat(evt.Payload, "event.total", 0),
	}
}
