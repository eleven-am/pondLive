package metadata

const EditEvent = "~change"

const (
	RateDefault = 0
	RateFrame   = -1
	RateOff     = -2
	FieldsAll   = "*"
)

type EventOptions struct {
	Prevent        bool     `json:"prevent,omitempty"`
	Stop           bool     `json:"stop,omitempty"`
	Passive        bool     `json:"passive,omitempty"`
	Once           bool     `json:"once,omitempty"`
	Capture        bool     `json:"capture,omitempty"`
	CapturePointer bool     `json:"capturePointer,omitempty"`
	Debounce       int      `json:"debounce,omitempty"`
	Throttle       int      `json:"throttle,omitempty"`
	Keys           []string `json:"keys,omitempty"`
	Fields         []string `json:"fields,omitempty"`
	RateMixed      bool     `json:"-"`
}

type EventOpt interface {
	ApplyEvent(*EventOptions)
}

func (o EventOptions) ApplyEvent(dst *EventOptions) { *dst = Merge(*dst, o) }

func Merge(a, b EventOptions) EventOptions {
	return EventOptions{
		Prevent:        a.Prevent || b.Prevent,
		Stop:           a.Stop || b.Stop,
		Passive:        a.Passive || b.Passive,
		Once:           a.Once || b.Once,
		Capture:        a.Capture || b.Capture,
		CapturePointer: a.CapturePointer || b.CapturePointer,
		Debounce:       smallestPositive(a.Debounce, b.Debounce),
		Throttle:       finestRate(a.Throttle, b.Throttle),
		Keys:           union(a.Keys, b.Keys),
		Fields:         mergeFields(a.Fields, b.Fields),
		RateMixed:      a.RateMixed || b.RateMixed,
	}
}

func MergeHandlers(a, b EventOptions) EventOptions {
	out := Merge(a, b)
	if len(a.Keys) == 0 || len(b.Keys) == 0 {
		out.Keys = nil
	}
	out.RateMixed = a.RateMixed || b.RateMixed || (a.Debounce != b.Debounce) || (a.Throttle != b.Throttle)
	return out
}

func smallestPositive(a, b int) int {
	switch {
	case a > 0 && b > 0:
		return min(a, b)
	case a > 0:
		return a
	default:
		return max(b, 0)
	}
}

func finestRate(a, b int) int {
	switch {
	case a == RateDefault:
		return b
	case b == RateDefault:
		return a
	}
	return min(a, b)
}

func union(a, b []string) []string {
	if len(a) == 0 && len(b) == 0 {
		return nil
	}
	out := make([]string, 0, len(a)+len(b))
	for _, list := range [][]string{a, b} {
		for _, s := range list {
			if !contains(out, s) {
				out = append(out, s)
			}
		}
	}
	return out
}

func contains(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func mergeFields(a, b []string) []string {
	if len(a) == 1 && a[0] == FieldsAll {
		return a
	}
	if len(b) == 1 && b[0] == FieldsAll {
		return b
	}
	return union(a, b)
}
