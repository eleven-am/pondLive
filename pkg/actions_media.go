package pkg

import (
	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type TimeRange struct {
	Start float64
	End   float64
}

type TimeRanges struct {
	Length int
	Ranges []TimeRange
}

type MediaActions struct {
	*ElementActions
}

func NewMediaActions(ctx *runtime.Ctx, ref *runtime.ElementRef) *MediaActions {
	return &MediaActions{ElementActions: NewElementActions(ctx, ref)}
}

func (a *MediaActions) Play() {
	a.Call("play")
}

func (a *MediaActions) Pause() {
	a.Call("pause")
}

func (a *MediaActions) Load() {
	a.Call("load")
}

func (a *MediaActions) SetCurrentTime(seconds float64) {
	a.Set("currentTime", seconds)
}

func (a *MediaActions) SetVolume(volume float64) {
	a.Set("volume", volume)
}

func (a *MediaActions) SetMuted(muted bool) {
	a.Set("muted", muted)
}

func (a *MediaActions) SetPlaybackRate(rate float64) {
	a.Set("playbackRate", rate)
}

func (a *MediaActions) GetCurrentTime() (float64, error) {
	values, err := a.Query("currentTime")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["currentTime"]), nil
}

func (a *MediaActions) GetDuration() (float64, error) {
	values, err := a.Query("duration")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["duration"]), nil
}

func (a *MediaActions) GetVolume() (float64, error) {
	values, err := a.Query("volume")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["volume"]), nil
}

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

func (a *MediaActions) GetPlaybackRate() (float64, error) {
	values, err := a.Query("playbackRate")
	if err != nil {
		return 0, err
	}
	return toFloat64(values["playbackRate"]), nil
}

func (a *MediaActions) GetBuffered() (*TimeRanges, error) {
	result, err := a.AsyncCall("getBuffered")
	if err != nil {
		return nil, err
	}
	return parseTimeRanges(result), nil
}

func (a *MediaActions) GetPlayed() (*TimeRanges, error) {
	result, err := a.AsyncCall("getPlayed")
	if err != nil {
		return nil, err
	}
	return parseTimeRanges(result), nil
}

func (a *MediaActions) GetSeekable() (*TimeRanges, error) {
	result, err := a.AsyncCall("getSeekable")
	if err != nil {
		return nil, err
	}
	return parseTimeRanges(result), nil
}

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

func (a *MediaActions) OnPlay(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("play", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnPause(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("pause", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnEnded(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("ended", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnTimeUpdate(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("timeupdate", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnVolumeChange(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("volumechange", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnSeeking(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("seeking", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnSeeked(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("seeked", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnLoadedMetadata(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("loadedmetadata", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnLoadedData(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("loadeddata", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnCanPlay(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("canplay", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnCanPlayThrough(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("canplaythrough", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnWaiting(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("waiting", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnPlaying(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("playing", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnStalled(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("stalled", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnRateChange(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("ratechange", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnDurationChange(handler func(MediaEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("durationchange", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: MediaEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildMediaEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnProgress(handler func(ProgressEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("progress", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ProgressEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildProgressEvent(evt)) },
	})
	return a
}

func (a *MediaActions) OnError(handler func(ErrorEvent) work.Updates, opts ...metadata.EventOptions) *MediaActions {
	if handler == nil {
		return a
	}
	a.addHandler("error", work.Handler{
		EventOptions: mergeOpts(metadata.EventOptions{Props: ErrorEvent{}.props()}, opts...),
		Fn:           func(evt work.Event) work.Updates { return handler(buildErrorEvent(evt)) },
	})
	return a
}
