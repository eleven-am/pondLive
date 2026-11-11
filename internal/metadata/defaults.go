package metadata

var noisyEvents = map[string]struct{}{
	"mousemove": {}, "pointermove": {}, "touchmove": {}, "scroll": {}, "wheel": {},
	"drag": {}, "dragover": {}, "timeupdate": {},
}

var passiveEvents = map[string]struct{}{
	"scroll": {}, "wheel": {}, "touchstart": {}, "touchmove": {},
}

type Resolved struct {
	Passive  bool
	Debounce int
	Throttle int
	Rate     bool
}

func Resolve(event string, o EventOptions) Resolved {
	r := Resolved{Debounce: max(o.Debounce, 0)}
	_, passive := passiveEvents[event]
	r.Passive = o.Passive || (passive && !o.Prevent)
	if r.Debounce > 0 {
		return r
	}
	th := o.Throttle
	if th == RateDefault {
		if _, noisy := noisyEvents[event]; noisy {
			th = RateFrame
		}
	}
	switch {
	case th == RateOff || th == RateDefault:
	case th == RateFrame:
		r.Rate = true
	default:
		r.Rate, r.Throttle = true, th
	}
	return r
}
