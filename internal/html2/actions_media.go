package html2

import (
	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

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

// MediaActions provides comprehensive control over HTML5 audio and video elements.
// It offers playback control, state inspection, and media property access.
//
// Example:
//
//	videoRef := ui.UseRef(ctx)
//	actions := html.Media(ctx, videoRef)
//	actions.Play()
//	actions.SetVolume(0.8)
//
//	return html.El("video", html.Attach(videoRef), html.Src("/movie.mp4"))
type MediaActions struct {
	*ElementActions
}

// NewMediaActions creates a MediaActions for the given ref.
func NewMediaActions(ctx *runtime2.Ctx, ref work.Attachment) *MediaActions {
	return &MediaActions{ElementActions: NewElementActions(ctx, ref)}
}

// ============================================================================
// Actions
// ============================================================================

// Play starts or resumes playback of the media element.
//
// Example:
//
//	actions := html.Media(ctx, videoRef)
//	actions.Play()
func (a *MediaActions) Play() {
	a.Call("play")
}

// Pause pauses playback of the media element.
//
// Example:
//
//	actions := html.Media(ctx, videoRef)
//	actions.Pause()
func (a *MediaActions) Pause() {
	a.Call("pause")
}

// Load reloads the media element, resetting it to the initial state.
//
// Example:
//
//	actions := html.Media(ctx, videoRef)
//	actions.Load()
func (a *MediaActions) Load() {
	a.Call("load")
}

// SetCurrentTime seeks to a specific time position in the media (in seconds).
//
// Example:
//
//	actions := html.Media(ctx, videoRef)
//	actions.SetCurrentTime(120.0)  // Skip to 2 minutes
func (a *MediaActions) SetCurrentTime(seconds float64) {
	a.Set("currentTime", seconds)
}

// SetVolume sets the audio volume level from 0.0 (silent) to 1.0 (maximum).
//
// Example:
//
//	actions := html.Media(ctx, videoRef)
//	actions.SetVolume(0.8)  // Set volume to 80%
func (a *MediaActions) SetVolume(volume float64) {
	a.Set("volume", volume)
}

// SetMuted controls whether the media element's audio is muted.
//
// Example:
//
//	actions := html.Media(ctx, videoRef)
//	actions.SetMuted(true)  // Mute video
func (a *MediaActions) SetMuted(muted bool) {
	a.Set("muted", muted)
}

// SetPlaybackRate sets the speed at which media plays (1.0 is normal speed).
//
// Example:
//
//	actions := html.Media(ctx, videoRef)
//	actions.SetPlaybackRate(1.5)  // 1.5x speed
func (a *MediaActions) SetPlaybackRate(rate float64) {
	a.Set("playbackRate", rate)
}

// ============================================================================
// Getters
// ============================================================================

// GetCurrentTime retrieves the current playback time in seconds.
func (a *MediaActions) GetCurrentTime() (float64, error) {
	values, err := a.Query("currentTime")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["currentTime"]), nil
}

// GetDuration retrieves the total duration of the media in seconds.
func (a *MediaActions) GetDuration() (float64, error) {
	values, err := a.Query("duration")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["duration"]), nil
}

// GetVolume retrieves the current volume level (0.0 to 1.0).
func (a *MediaActions) GetVolume() (float64, error) {
	values, err := a.Query("volume")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["volume"]), nil
}

// GetMuted retrieves the muted state of the media element.
func (a *MediaActions) GetMuted() (bool, error) {
	values, err := a.Query("muted")
	if err != nil {
		return false, err
	}
	if b, ok := values["muted"].(bool); ok {
		return b, nil
	}
	return false, nil
}

// GetPaused retrieves whether the media is currently paused.
func (a *MediaActions) GetPaused() (bool, error) {
	values, err := a.Query("paused")
	if err != nil {
		return false, err
	}
	if b, ok := values["paused"].(bool); ok {
		return b, nil
	}
	return false, nil
}

// GetEnded retrieves whether playback has reached the end.
func (a *MediaActions) GetEnded() (bool, error) {
	values, err := a.Query("ended")
	if err != nil {
		return false, err
	}
	if b, ok := values["ended"].(bool); ok {
		return b, nil
	}
	return false, nil
}

// GetPlaybackRate retrieves the current playback speed.
func (a *MediaActions) GetPlaybackRate() (float64, error) {
	values, err := a.Query("playbackRate")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["playbackRate"]), nil
}

// ============================================================================
// Async Methods - Media State
// ============================================================================

// GetBuffered returns the buffered time ranges for the media element.
func (a *MediaActions) GetBuffered() (*TimeRanges, error) {
	result, err := a.AsyncCall("getBuffered")
	if err != nil {
		return nil, err
	}
	return parseTimeRanges(result), nil
}

// GetPlayed returns the time ranges that have been played.
func (a *MediaActions) GetPlayed() (*TimeRanges, error) {
	result, err := a.AsyncCall("getPlayed")
	if err != nil {
		return nil, err
	}
	return parseTimeRanges(result), nil
}

// GetSeekable returns the seekable time ranges for the media element.
func (a *MediaActions) GetSeekable() (*TimeRanges, error) {
	result, err := a.AsyncCall("getSeekable")
	if err != nil {
		return nil, err
	}
	return parseTimeRanges(result), nil
}

// GetNetworkState returns the current network state of the media element.
// Returns: 0=NETWORK_EMPTY, 1=NETWORK_IDLE, 2=NETWORK_LOADING, 3=NETWORK_NO_SOURCE
func (a *MediaActions) GetNetworkState() (int, error) {
	result, err := a.AsyncCall("networkState")
	if err != nil {
		return 0, err
	}
	if n, ok := result.(float64); ok {
		return int(n), nil
	}
	return 0, nil
}

// GetReadyState returns the current ready state of the media element.
// Returns: 0=HAVE_NOTHING, 1=HAVE_METADATA, 2=HAVE_CURRENT_DATA, 3=HAVE_FUTURE_DATA, 4=HAVE_ENOUGH_DATA
func (a *MediaActions) GetReadyState() (int, error) {
	result, err := a.AsyncCall("readyState")
	if err != nil {
		return 0, err
	}
	if n, ok := result.(float64); ok {
		return int(n), nil
	}
	return 0, nil
}

func parseTimeRanges(result any) *TimeRanges {
	if result == nil {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}
	}

	rangesMap, ok := result.(map[string]any)
	if !ok {
		return &TimeRanges{Length: 0, Ranges: []TimeRange{}}
	}

	length := toInt(rangesMap["length"])
	rangesArray, ok := rangesMap["ranges"].([]any)
	if !ok {
		return &TimeRanges{Length: length, Ranges: []TimeRange{}}
	}

	ranges := make([]TimeRange, 0, len(rangesArray))
	for _, r := range rangesArray {
		rangeMap, ok := r.(map[string]any)
		if !ok {
			continue
		}
		ranges = append(ranges, TimeRange{
			Start: toFloat64(rangeMap["start"]),
			End:   toFloat64(rangeMap["end"]),
		})
	}

	return &TimeRanges{Length: length, Ranges: ranges}
}

func (a *MediaActions) OnPlay(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("play", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnPause(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("pause", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnEnded(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("ended", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnTimeUpdate(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("timeupdate", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnVolumeChange(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("volumechange", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnSeeking(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("seeking", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnSeeked(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("seeked", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnLoadedMetadata(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("loadedmetadata", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnLoadedData(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("loadeddata", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnCanPlay(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("canplay", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnCanPlayThrough(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("canplaythrough", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnWaiting(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("waiting", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnPlaying(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("playing", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnStalled(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("stalled", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnRateChange(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("ratechange", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnDurationChange(handler func(MediaEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("durationchange", work.Handler{
		EventOptions: metadata.EventOptions{Props: MediaEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnProgress(handler func(ProgressEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("progress", work.Handler{
		EventOptions: metadata.EventOptions{Props: ProgressEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildProgressEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnError(handler func(ErrorEvent) work.Updates) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("error", work.Handler{
		EventOptions: metadata.EventOptions{Props: ErrorEvent{}.props()},
		Fn:           func(evt work.Event) work.Updates { return handler(buildErrorEvent(evt)) },
	})
	return a
}
