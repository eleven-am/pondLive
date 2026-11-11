package work

import (
	"time"

	"github.com/eleven-am/pondlive/internal/metadata"
)

type Limit struct{ rate int }

func Throttle(d time.Duration) Limit {
	if d <= 0 {
		return Limit{metadata.RateOff}
	}
	return Limit{max(int(d/time.Millisecond), 1)}
}

func Frame() Limit { return Limit{metadata.RateFrame} }

func Immediate() Limit { return Limit{metadata.RateOff} }

func (l Limit) ApplyEvent(o *metadata.EventOptions) { o.Throttle = l.rate }

func (l Limit) applyBind(b *BindSpec) { b.Throttle = l.rate }

func (l Limit) ObserveRate() int { return l.rate }

type DebounceBy struct{ ms int }

func Debounce(d time.Duration) DebounceBy {
	if d <= 0 {
		return DebounceBy{}
	}
	return DebounceBy{max(int(d/time.Millisecond), 1)}
}

func (d DebounceBy) ApplyEvent(o *metadata.EventOptions) { o.Debounce = d.ms }

func (d DebounceBy) applyBind(b *BindSpec) { b.Debounce = time.Duration(d.ms) * time.Millisecond }
