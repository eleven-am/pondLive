package pkg

import (
	"time"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/work"
)

type (
	EventOpt   = metadata.EventOpt
	BindOpt    = work.BindOpt
	Limit      = work.Limit
	DebounceBy = work.DebounceBy
)

func Throttle(d time.Duration) Limit { return work.Throttle(d) }

func Frame() Limit { return work.Frame() }

func Immediate() Limit { return work.Immediate() }

func Debounce(d time.Duration) DebounceBy { return work.Debounce(d) }

func Lazy() BindOpt { return work.Lazy{} }

func CapturePointer() EventOpt { return EventOptions{CapturePointer: true} }

func Key(keys ...string) EventOpt {
	return EventOptions{Keys: keys, Fields: []string{"key"}}
}

func Fields(names ...string) EventOpt {
	if err := protocol.ValidateFields(names...); err != nil {
		panic(err)
	}
	return EventOptions{Fields: names}
}

func AllFields() EventOpt { return EventOptions{Fields: []string{metadata.FieldsAll}} }
