package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// MediaActions provides media playback DOM actions for audio and video elements.
// Embedded in refs for video and audio elements.
type MediaActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
}

func NewMediaActions[T dom.ElementDescriptor](ref *dom.ElementRef[T]) *MediaActions[T] {
	return &MediaActions[T]{ref: ref}
}

// Play starts playback of the media element.
func (a *MediaActions[T]) Play(ctx dom.ActionExecutor) {
	dom.DOMCall[T](ctx, a.ref, "play")
}

// Pause pauses playback of the media element.
func (a *MediaActions[T]) Pause(ctx dom.ActionExecutor) {
	dom.DOMCall[T](ctx, a.ref, "pause")
}

// Load reloads the media element.
func (a *MediaActions[T]) Load(ctx dom.ActionExecutor) {
	dom.DOMCall[T](ctx, a.ref, "load")
}

// SetCurrentTime sets the current playback time in seconds.
func (a *MediaActions[T]) SetCurrentTime(ctx dom.ActionExecutor, seconds float64) {
	dom.DOMSet[T](ctx, a.ref, "currentTime", seconds)
}

// SetVolume sets the volume level (0.0 to 1.0).
func (a *MediaActions[T]) SetVolume(ctx dom.ActionExecutor, volume float64) {
	dom.DOMSet[T](ctx, a.ref, "volume", volume)
}

// SetMuted sets the muted state of the media element.
func (a *MediaActions[T]) SetMuted(ctx dom.ActionExecutor, muted bool) {
	dom.DOMSet[T](ctx, a.ref, "muted", muted)
}

// SetPlaybackRate sets the playback speed (1.0 is normal speed).
func (a *MediaActions[T]) SetPlaybackRate(ctx dom.ActionExecutor, rate float64) {
	dom.DOMSet[T](ctx, a.ref, "playbackRate", rate)
}
