package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

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

// props returns the list of properties this event needs from the client.
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

// MediaAPI provides actions, events, and values for audio and video elements.
type MediaAPI[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.Dispatcher
}

func NewMediaAPI[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.Dispatcher) *MediaAPI[T] {
	return &MediaAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Play starts playback of the media element.
func (a *MediaAPI[T]) Play() {
	dom.DOMCall[T](a.ctx, a.ref, "play")
}

// Pause pauses playback of the media element.
func (a *MediaAPI[T]) Pause() {
	dom.DOMCall[T](a.ctx, a.ref, "pause")
}

// Load reloads the media element.
func (a *MediaAPI[T]) Load() {
	dom.DOMCall[T](a.ctx, a.ref, "load")
}

// SetCurrentTime sets the current playback time in seconds.
func (a *MediaAPI[T]) SetCurrentTime(seconds float64) {
	dom.DOMSet[T](a.ctx, a.ref, "currentTime", seconds)
}

// SetVolume sets the volume level (0.0 to 1.0).
func (a *MediaAPI[T]) SetVolume(volume float64) {
	dom.DOMSet[T](a.ctx, a.ref, "volume", volume)
}

// SetMuted sets the muted state of the media element.
func (a *MediaAPI[T]) SetMuted(muted bool) {
	dom.DOMSet[T](a.ctx, a.ref, "muted", muted)
}

// SetPlaybackRate sets the playback speed (1.0 is normal speed).
func (a *MediaAPI[T]) SetPlaybackRate(rate float64) {
	dom.DOMSet[T](a.ctx, a.ref, "playbackRate", rate)
}

// ============================================================================
// Values
// ============================================================================

// CurrentTime gets the current playback time in seconds.
func (a *MediaAPI[T]) CurrentTime() (float64, error) {
	values, err := a.ctx.DOMGet(a.ref.ID(), "element.currentTime")
	if err != nil {
		return 0, err
	}
	if val, ok := values["element.currentTime"].(float64); ok {
		return val, nil
	}
	return 0, nil
}

// Duration gets the total duration of the media in seconds.
func (a *MediaAPI[T]) Duration() (float64, error) {
	values, err := a.ctx.DOMGet(a.ref.ID(), "element.duration")
	if err != nil {
		return 0, err
	}
	if val, ok := values["element.duration"].(float64); ok {
		return val, nil
	}
	return 0, nil
}

// Volume gets the current volume level (0.0 to 1.0).
func (a *MediaAPI[T]) Volume() (float64, error) {
	values, err := a.ctx.DOMGet(a.ref.ID(), "element.volume")
	if err != nil {
		return 0, err
	}
	if val, ok := values["element.volume"].(float64); ok {
		return val, nil
	}
	return 0, nil
}

// Muted gets the muted state of the media element.
func (a *MediaAPI[T]) Muted() (bool, error) {
	values, err := a.ctx.DOMGet(a.ref.ID(), "element.muted")
	if err != nil {
		return false, err
	}
	if val, ok := values["element.muted"].(bool); ok {
		return val, nil
	}
	return false, nil
}

// Paused gets whether the media is currently paused.
func (a *MediaAPI[T]) Paused() (bool, error) {
	values, err := a.ctx.DOMGet(a.ref.ID(), "element.paused")
	if err != nil {
		return false, err
	}
	if val, ok := values["element.paused"].(bool); ok {
		return val, nil
	}
	return false, nil
}

// PlaybackRate gets the current playback speed (1.0 is normal speed).
func (a *MediaAPI[T]) PlaybackRate() (float64, error) {
	values, err := a.ctx.DOMGet(a.ref.ID(), "element.playbackRate")
	if err != nil {
		return 0, err
	}
	if val, ok := values["element.playbackRate"].(float64); ok {
		return val, nil
	}
	return 0, nil
}

// ============================================================================
// Events
// ============================================================================

// OnPlay registers a handler for the "play" event.
func (a *MediaAPI[T]) OnPlay(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("play", wrapped, MediaEvent{}.props())
}

// OnPause registers a handler for the "pause" event.
func (a *MediaAPI[T]) OnPause(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("pause", wrapped, MediaEvent{}.props())
}

// OnEnded registers a handler for the "ended" event.
func (a *MediaAPI[T]) OnEnded(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("ended", wrapped, MediaEvent{}.props())
}

// OnSeeking registers a handler for the "seeking" event.
func (a *MediaAPI[T]) OnSeeking(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("seeking", wrapped, MediaEvent{}.props())
}

// OnSeeked registers a handler for the "seeked" event.
func (a *MediaAPI[T]) OnSeeked(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("seeked", wrapped, MediaEvent{}.props())
}

// OnRateChange registers a handler for the "ratechange" event.
func (a *MediaAPI[T]) OnRateChange(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("ratechange", wrapped, MediaEvent{}.props())
}

// OnTimeUpdate registers a handler for the "timeupdate" event.
func (a *MediaAPI[T]) OnTimeUpdate(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("timeupdate", wrapped, MediaEvent{}.props())
}

// OnVolumeChange registers a handler for the "volumechange" event.
func (a *MediaAPI[T]) OnVolumeChange(handler func(MediaEvent) Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt Event) Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("volumechange", wrapped, MediaEvent{}.props())
}
