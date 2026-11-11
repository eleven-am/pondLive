package pkg

import "github.com/eleven-am/pondlive/internal/runtime"

func (r *MediaRef) CurrentTime(opts ...ObserveOpt) ReadSignal[float64] {
	return live(r.ElementRef, floatSpec("currentTime", 0), opts)
}

func (r *MediaRef) Duration(opts ...ObserveOpt) ReadSignal[float64] {
	return live(r.ElementRef, floatSpec("duration", 0), opts)
}

func (r *MediaRef) Paused(opts ...ObserveOpt) ReadSignal[bool] {
	seed := func(attrs map[string]string) (bool, bool) {
		_, autoplay := attrs["autoplay"]
		return !autoplay, attrs != nil
	}
	return live(r.ElementRef, boolSpec("paused", true, seed), opts)
}

func (r *MediaRef) Ended(opts ...ObserveOpt) ReadSignal[bool] {
	return live(r.ElementRef, boolSpec("ended", false, nil), opts)
}

func (r *MediaRef) Volume(opts ...ObserveOpt) ReadSignal[float64] {
	return live(r.ElementRef, floatSpec("volume", 1), opts)
}

func (r *MediaRef) Muted(opts ...ObserveOpt) ReadSignal[bool] {
	return live(r.ElementRef, runtime.RefSpec[bool]{Prop: "muted", Decode: decodeBool, Seed: hasAttr("muted")}, opts)
}

func (r *MediaRef) Play() { r.ref.Call("play") }

func (r *MediaRef) Pause() { r.ref.Call("pause") }

func (r *MediaRef) Load() { r.ref.Call("load") }

func (r *MediaRef) SetCurrentTime(seconds float64) { r.ref.Set("currentTime", seconds) }

func (r *MediaRef) SetVolume(volume float64) { r.ref.Set("volume", volume) }

func (r *MediaRef) SetMuted(muted bool) { r.ref.Set("muted", muted) }

func (r *MediaRef) SetPlaybackRate(rate float64) { r.ref.Set("playbackRate", rate) }

func (r *MediaRef) PlaybackRate(fn func(float64, error)) {
	property(r.ElementRef, "playbackRate", toFloat64, fn)
}

func (r *MediaRef) ReadyState(fn func(int, error)) { property(r.ElementRef, "readyState", toInt, fn) }

func (r *MediaRef) NetworkState(fn func(int, error)) {
	property(r.ElementRef, "networkState", toInt, fn)
}
