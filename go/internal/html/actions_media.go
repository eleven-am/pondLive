package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
)

// MediaActions provides media playback DOM actions for audio and video elements.
// Embedded in refs for video and audio elements.
type MediaActions[T dom.ElementDescriptor] struct {
	ref *dom.ElementRef[T]
	ctx dom.ActionExecutor
}

func NewMediaActions[T dom.ElementDescriptor](ref *dom.ElementRef[T], ctx dom.ActionExecutor) *MediaActions[T] {
	return &MediaActions[T]{ref: ref, ctx: ctx}
}

// Play starts playback of the media element.
func (a *MediaActions[T]) Play() {
	dom.DOMCall[T](a.ctx, a.ref, "play")
}

// Pause pauses playback of the media element.
func (a *MediaActions[T]) Pause() {
	dom.DOMCall[T](a.ctx, a.ref, "pause")
}

// Load reloads the media element.
func (a *MediaActions[T]) Load() {
	dom.DOMCall[T](a.ctx, a.ref, "load")
}

// SetCurrentTime sets the current playback time in seconds.
func (a *MediaActions[T]) SetCurrentTime(seconds float64) {
	dom.DOMSet[T](a.ctx, a.ref, "currentTime", seconds)
}

// SetVolume sets the volume level (0.0 to 1.0).
func (a *MediaActions[T]) SetVolume(volume float64) {
	dom.DOMSet[T](a.ctx, a.ref, "volume", volume)
}

// SetMuted sets the muted state of the media element.
func (a *MediaActions[T]) SetMuted(muted bool) {
	dom.DOMSet[T](a.ctx, a.ref, "muted", muted)
}

// SetPlaybackRate sets the playback speed (1.0 is normal speed).
func (a *MediaActions[T]) SetPlaybackRate(rate float64) {
	dom.DOMSet[T](a.ctx, a.ref, "playbackRate", rate)
}
