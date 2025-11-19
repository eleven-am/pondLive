package html

import (
	"github.com/eleven-am/pondlive/go/internal/dom2"
)

// MediaEvent represents media playback events (play, pause, timeupdate, volumechange, etc).
type MediaEvent struct {
	dom2.Event
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

func buildMediaEvent(evt dom2.Event) MediaEvent {
	detail := extractDetail(evt.Payload)
	return MediaEvent{
		Event:        evt,
		CurrentTime:  payloadFloat(detail, "target.currentTime", 0),
		Duration:     payloadFloat(detail, "target.duration", 0),
		Volume:       payloadFloat(detail, "target.volume", 0),
		PlaybackRate: payloadFloat(detail, "target.playbackRate", 0),
		Paused:       payloadBool(detail, "target.paused", false),
		Ended:        payloadBool(detail, "target.ended", false),
		Muted:        payloadBool(detail, "target.muted", false),
		Seeking:      payloadBool(detail, "target.seeking", false),
	}
}

// MediaAPI provides comprehensive control over HTML5 audio and video elements.
// It offers playback control, state inspection, event handling, and advanced media features
// like buffering info, network state, and time range queries.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.Play()
//	videoRef.SetVolume(0.8)
//
//	videoRef.OnTimeUpdate(func(evt h.MediaEvent) h.Updates {
//	    // Update progress bar
//	    return nil
//	})
//
//	return h.Video(
//	    h.Attach(videoRef),
//	    h.Src("/movie.mp4"),
//	)
type MediaAPI[T dom2.ElementDescriptor] struct {
	ref *dom2.ElementRef[T]
	ctx dom2.Dispatcher
}

func NewMediaAPI[T dom2.ElementDescriptor](ref *dom2.ElementRef[T], ctx dom2.Dispatcher) *MediaAPI[T] {
	return &MediaAPI[T]{ref: ref, ctx: ctx}
}

// ============================================================================
// Actions
// ============================================================================

// Play starts or resumes playback of the media element.
// This is an asynchronous operation that may fail if autoplay is blocked.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.Play()
//
//	return h.Video(
//	    h.Attach(videoRef),
//	    h.Src("/movie.mp4"),
//	)
//
// Note: Modern browsers may block autoplay with audio. Consider handling OnPlay/OnPause
// events to update UI state based on actual playback state changes.
func (a *MediaAPI[T]) Play() {
	dom2.DOMCall[T](a.ctx, a.ref, "play")
}

// Pause pauses playback of the media element.
// The playback position is preserved and can be resumed with Play().
//
// Example:
//
//	audioRef := ui.UseElement[*h.AudioRef](ctx)
//	audioRef.Pause()
//
//	return h.Audio(
//	    h.Attach(audioRef),
//	    h.Src("/podcast.mp3"),
//	)
//
// Note: Pause is always successful, unlike Play() which may be blocked by browsers.
func (a *MediaAPI[T]) Pause() {
	dom2.DOMCall[T](a.ctx, a.ref, "pause")
}

// Load reloads the media element, resetting it to the initial state and restarting the resource selection.
// This is useful when you've changed the src attribute or want to reset error states.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.Load()
//
//	return h.Video(
//	    h.Attach(videoRef),
//	    h.Src("/movie.mp4"),
//	)
//
// Note: Load() resets the element to HAVE_NOTHING state and begins resource selection from scratch.
// Any playback progress is lost. Use this sparingly as it interrupts the user experience.
func (a *MediaAPI[T]) Load() {
	dom2.DOMCall[T](a.ctx, a.ref, "load")
}

// SetCurrentTime seeks to a specific time position in the media, measured in seconds.
// This allows jumping to any point in the loaded media.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//
//	// Skip to 2 minutes into the video
//	videoRef.SetCurrentTime(120.0)
//
//	// Jump to 50% through the video
//	duration, _ := videoRef.Duration()
//	videoRef.SetCurrentTime(duration * 0.5)
//
//	return h.Video(
//	    h.Attach(videoRef),
//	    h.Src("/movie.mp4"),
//	)
//
// Note: Seeking triggers "seeking" and "seeked" events. The actual seek may take time
// depending on whether the target position is buffered. Check evt.Seeking in OnSeeking events.
func (a *MediaAPI[T]) SetCurrentTime(seconds float64) {
	dom2.DOMSet[T](a.ctx, a.ref, "currentTime", seconds)
}

// SetVolume sets the audio volume level from 0.0 (silent) to 1.0 (maximum).
// Values outside this range are clamped to the valid range.
//
// Example:
//
//	audioRef := ui.UseElement[*h.AudioRef](ctx)
//	audioRef.SetVolume(0.8)  // Set volume to 80%
//
//	return h.Audio(
//	    h.Attach(audioRef),
//	    h.Src("/music.mp3"),
//	)
//
// Note: Volume only affects audio output, not the muted state. A muted element with
// volume 1.0 will still be silent. Triggers "volumechange" event.
func (a *MediaAPI[T]) SetVolume(volume float64) {
	dom2.DOMSet[T](a.ctx, a.ref, "volume", volume)
}

// SetMuted controls whether the media element's audio is muted.
// When muted, audio output is silenced regardless of volume level.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.SetMuted(true)  // Mute video
//
//	// Toggle mute state
//	isMuted, _ := videoRef.Muted()
//	videoRef.SetMuted(!isMuted)
//
//	return h.Video(
//	    h.Attach(videoRef),
//	    h.Src("/movie.mp4"),
//	)
//
// Note: Muting is independent of volume. A muted element with volume 1.0 is silent.
// Unmuting restores the previous volume level. Triggers "volumechange" event.
func (a *MediaAPI[T]) SetMuted(muted bool) {
	dom2.DOMSet[T](a.ctx, a.ref, "muted", muted)
}

// SetPlaybackRate sets the speed at which media plays. 1.0 is normal speed, 2.0 is double speed,
// 0.5 is half speed. Negative values play backwards if supported.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.SetPlaybackRate(1.5)  // Speed up tutorial videos (1.5x)
//
//	return h.Video(
//	    h.Attach(videoRef),
//	    h.Src("/tutorial.mp4"),
//	)
//
// Note: Not all rates are supported on all platforms. Typical range is 0.25 to 4.0.
// Triggers "ratechange" event when changed. Audio pitch is usually preserved.
func (a *MediaAPI[T]) SetPlaybackRate(rate float64) {
	dom2.DOMSet[T](a.ctx, a.ref, "playbackRate", rate)
}

// ============================================================================
// Values
// ============================================================================

// CurrentTime gets the current playback time in seconds.
// This makes a synchronous call to the client (~1-2ms latency).
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	currentTime, err := videoRef.CurrentTime()
//	if err == nil {
//	    updateProgressDisplay(currentTime)
//	}
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
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
// Returns NaN if duration is not yet known. This makes a synchronous call to the client.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	duration, err := videoRef.Duration()
//	if err == nil {
//	    displayTotalTime(duration)
//	}
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
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
// This makes a synchronous call to the client (~1-2ms latency).
//
// Example:
//
//	audioRef := ui.UseElement[*h.AudioRef](ctx)
//	volume, _ := audioRef.Volume()
//	updateVolumeSlider(volume)
//
//	return h.Audio(h.Attach(audioRef), h.Src("/music.mp3"))
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
// This makes a synchronous call to the client (~1-2ms latency).
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	isMuted, _ := videoRef.Muted()
//	updateMuteButton(isMuted)
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
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
// This makes a synchronous call to the client (~1-2ms latency).
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	isPaused, _ := videoRef.Paused()
//	updatePlayPauseButton(isPaused)
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
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
// This makes a synchronous call to the client (~1-2ms latency).
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	rate, _ := videoRef.PlaybackRate()
//	updateSpeedDisplay(rate)
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
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

// OnPlay registers a handler for the "play" event, fired when playback starts or resumes.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.OnPlay(func(evt h.MediaEvent) h.Updates {
//	    showPauseButton()
//	    return nil
//	})
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
func (a *MediaAPI[T]) OnPlay(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("play", wrapped, MediaEvent{}.props())
}

// OnPause registers a handler for the "pause" event, fired when playback is paused.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.OnPause(func(evt h.MediaEvent) h.Updates {
//	    showPlayButton()
//	    return nil
//	})
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
func (a *MediaAPI[T]) OnPause(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("pause", wrapped, MediaEvent{}.props())
}

// OnEnded registers a handler for the "ended" event, fired when playback reaches the end of the media.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.OnEnded(func(evt h.MediaEvent) h.Updates {
//	    showReplayButton()
//	    return nil
//	})
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
func (a *MediaAPI[T]) OnEnded(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("ended", wrapped, MediaEvent{}.props())
}

// OnSeeking registers a handler for the "seeking" event, fired when a seek operation begins.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.OnSeeking(func(evt h.MediaEvent) h.Updates {
//	    showLoadingSpinner()
//	    return nil
//	})
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
func (a *MediaAPI[T]) OnSeeking(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("seeking", wrapped, MediaEvent{}.props())
}

// OnSeeked registers a handler for the "seeked" event, fired when a seek operation completes.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.OnSeeked(func(evt h.MediaEvent) h.Updates {
//	    hideLoadingSpinner()
//	    return nil
//	})
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
func (a *MediaAPI[T]) OnSeeked(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("seeked", wrapped, MediaEvent{}.props())
}

// OnRateChange registers a handler for the "ratechange" event, fired when playback rate changes.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.OnRateChange(func(evt h.MediaEvent) h.Updates {
//	    updateSpeedIndicator(evt.PlaybackRate)
//	    return nil
//	})
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
func (a *MediaAPI[T]) OnRateChange(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("ratechange", wrapped, MediaEvent{}.props())
}

// OnTimeUpdate registers a handler for the "timeupdate" event, fired periodically during playback.
// This event fires approximately every 250ms during playback and is commonly used for progress tracking.
//
// Example:
//
//	videoRef := ui.UseElement[*h.VideoRef](ctx)
//	videoRef.OnTimeUpdate(func(evt h.MediaEvent) h.Updates {
//	    updateProgressBar(evt.CurrentTime, evt.Duration)
//	    return nil
//	})
//
//	return h.Video(h.Attach(videoRef), h.Src("/movie.mp4"))
func (a *MediaAPI[T]) OnTimeUpdate(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("timeupdate", wrapped, MediaEvent{}.props())
}

// OnVolumeChange registers a handler for the "volumechange" event, fired when volume or muted state changes.
//
// Example:
//
//	audioRef := ui.UseElement[*h.AudioRef](ctx)
//	audioRef.OnVolumeChange(func(evt h.MediaEvent) h.Updates {
//	    updateVolumeIndicator(evt.Volume, evt.Muted)
//	    return nil
//	})
//
//	return h.Audio(h.Attach(audioRef), h.Src("/music.mp3"))
func (a *MediaAPI[T]) OnVolumeChange(handler func(MediaEvent) dom2.Updates) {
	if a.ref == nil || handler == nil {
		return
	}
	wrapped := func(evt dom2.Event) dom2.Updates { return handler(buildMediaEvent(evt)) }
	a.ref.AddListener("volumechange", wrapped, MediaEvent{}.props())
}

// ============================================================================
// Async Methods - Media State
// ============================================================================

// TimeRange represents a single time range with start and end times.
type TimeRange struct {
	Start float64 // Start time in seconds
	End   float64 // End time in seconds
}

// TimeRanges represents a collection of time ranges.
type TimeRanges struct {
	Length int         // Number of time ranges
	Ranges []TimeRange // Array of time ranges
}

// GetBuffered returns the buffered time ranges for the media element.
// This shows which portions of the media have been downloaded/buffered.
// This makes a synchronous call to the client and waits for the response.
func (a *MediaAPI[T]) GetBuffered() (*TimeRanges, error) {
	result, err := dom2.DOMAsyncCall[T](a.ctx, a.ref, "getBuffered")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}, nil
	}

	rangesMap, ok := result.(map[string]any)
	if !ok {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}, nil
	}

	length := payloadInt(rangesMap, "length", 0)
	rangesArray, ok := rangesMap["ranges"].([]any)
	if !ok {
		return &TimeRanges{Length: length, Ranges: []TimeRange{}}, nil
	}

	ranges := make([]TimeRange, 0, len(rangesArray))
	for _, r := range rangesArray {
		rangeMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		ranges = append(ranges, TimeRange{
			Start: payloadFloat(rangeMap, "start", 0),
			End:   payloadFloat(rangeMap, "end", 0),
		})
	}

	return &TimeRanges{Length: length, Ranges: ranges}, nil
}

// GetPlayed returns the time ranges that have been played.
// This makes a synchronous call to the client and waits for the response.
func (a *MediaAPI[T]) GetPlayed() (*TimeRanges, error) {
	result, err := dom2.DOMAsyncCall[T](a.ctx, a.ref, "getPlayed")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}, nil
	}

	rangesMap, ok := result.(map[string]any)
	if !ok {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}, nil
	}

	length := payloadInt(rangesMap, "length", 0)
	rangesArray, ok := rangesMap["ranges"].([]any)
	if !ok {
		return &TimeRanges{Length: length, Ranges: []TimeRange{}}, nil
	}

	ranges := make([]TimeRange, 0, len(rangesArray))
	for _, r := range rangesArray {
		rangeMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		ranges = append(ranges, TimeRange{
			Start: payloadFloat(rangeMap, "start", 0),
			End:   payloadFloat(rangeMap, "end", 0),
		})
	}

	return &TimeRanges{Length: length, Ranges: ranges}, nil
}

// GetSeekable returns the seekable time ranges for the media element.
// This shows which portions of the media can be seeked to.
// This makes a synchronous call to the client and waits for the response.
func (a *MediaAPI[T]) GetSeekable() (*TimeRanges, error) {
	result, err := dom2.DOMAsyncCall[T](a.ctx, a.ref, "getSeekable")
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}, nil
	}

	rangesMap, ok := result.(map[string]any)
	if !ok {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}, nil
	}

	length := payloadInt(rangesMap, "length", 0)
	rangesArray, ok := rangesMap["ranges"].([]any)
	if !ok {
		return &TimeRanges{Length: length, Ranges: []TimeRange{}}, nil
	}

	ranges := make([]TimeRange, 0, len(rangesArray))
	for _, r := range rangesArray {
		rangeMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		ranges = append(ranges, TimeRange{
			Start: payloadFloat(rangeMap, "start", 0),
			End:   payloadFloat(rangeMap, "end", 0),
		})
	}

	return &TimeRanges{Length: length, Ranges: ranges}, nil
}

// GetNetworkState returns the current network state of the media element.
// Returns: 0=NETWORK_EMPTY, 1=NETWORK_IDLE, 2=NETWORK_LOADING, 3=NETWORK_NO_SOURCE
// This makes a synchronous call to the client and waits for the response.
func (a *MediaAPI[T]) GetNetworkState() (int, error) {
	result, err := dom2.DOMAsyncCall[T](a.ctx, a.ref, "networkState")
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}

	state, ok := result.(float64)
	if !ok {
		return 0, nil
	}
	return int(state), nil
}

// GetReadyState returns the current ready state of the media element.
// Returns: 0=HAVE_NOTHING, 1=HAVE_METADATA, 2=HAVE_CURRENT_DATA, 3=HAVE_FUTURE_DATA, 4=HAVE_ENOUGH_DATA
// This makes a synchronous call to the client and waits for the response.
func (a *MediaAPI[T]) GetReadyState() (int, error) {
	result, err := dom2.DOMAsyncCall[T](a.ctx, a.ref, "readyState")
	if err != nil {
		return 0, err
	}
	if result == nil {
		return 0, nil
	}

	state, ok := result.(float64)
	if !ok {
		return 0, nil
	}
	return int(state), nil
}
