package html

// MediaEvent represents media playback events (play, pause, timeupdate, volumechange, etc).
type MediaEvent struct {
	Event
	CurrentTime  float64 // Current playback position
	Duration     float64 // Total duration
	Volume       float64 // Volume level (0.0 to 1.0)
	PlaybackRate float64 // Playback speed
	Paused       bool    // Is paused
	Ended        bool    // Has ended
	Muted        bool    // Is muted
	Seeking      bool    // Is seeking
}

// Props returns the list of properties this event needs from the client.
func (MediaEvent) props() []string {
	return []string{
		"target.currentTime",
		"target.duration",
		"target.volume",
		"target.playbackRate",
		"target.paused",
		"target.ended",
		"target.muted",
		"target.seeking",
	}
}

// MediaHandler provides media event handlers.
type MediaHandler struct {
	ref RefListener
}

// NewMediaHandler creates a new MediaHandler.
func NewMediaHandler(ref RefListener) *MediaHandler {
	return &MediaHandler{ref: ref}
}

// OnPlay registers a handler for the "play" event.
func (h *MediaHandler) OnPlay(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("play", wrapped, MediaEvent{}.props())
}

// OnPause registers a handler for the "pause" event.
func (h *MediaHandler) OnPause(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("pause", wrapped, MediaEvent{}.props())
}

// OnEnded registers a handler for the "ended" event.
func (h *MediaHandler) OnEnded(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("ended", wrapped, MediaEvent{}.props())
}

// OnSeeking registers a handler for the "seeking" event.
func (h *MediaHandler) OnSeeking(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("seeking", wrapped, MediaEvent{}.props())
}

// OnSeeked registers a handler for the "seeked" event.
func (h *MediaHandler) OnSeeked(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("seeked", wrapped, MediaEvent{}.props())
}

// OnRateChange registers a handler for the "ratechange" event.
func (h *MediaHandler) OnRateChange(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("ratechange", wrapped, MediaEvent{}.props())
}

// OnTimeUpdate registers a handler for the "timeupdate" event.
func (h *MediaHandler) OnTimeUpdate(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("timeupdate", wrapped, MediaEvent{}.props())
}

// OnVolumeChange registers a handler for the "volumechange" event.
func (h *MediaHandler) OnVolumeChange(handler func(MediaEvent) Updates) {
	if h.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	h.ref.AddListener("volumechange", wrapped, MediaEvent{}.props())
}

func buildMediaEvent(evt Event) MediaEvent {
	return MediaEvent{
		Event:        evt,
		CurrentTime:  payloadFloat(evt.Payload, "target.currentTime", 0),
		Duration:     payloadFloat(evt.Payload, "target.duration", 0),
		Volume:       payloadFloat(evt.Payload, "target.volume", 0),
		PlaybackRate: payloadFloat(evt.Payload, "target.playbackRate", 0),
		Paused:       payloadBool(evt.Payload, "target.paused", false),
		Ended:        payloadBool(evt.Payload, "target.ended", false),
		Muted:        payloadBool(evt.Payload, "target.muted", false),
		Seeking:      payloadBool(evt.Payload, "target.seeking", false),
	}
}
