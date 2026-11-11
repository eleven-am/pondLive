package session

import (
	"encoding/json"
	"errors"
	"sync"

	"github.com/gammazero/deque"

	"github.com/eleven-am/pondlive/internal/protocol"
)

const (
	DefaultReplayFrames = 1024
	DefaultReplayBytes  = 256 << 10
)

var (
	ErrResetTooLarge = errors.New("session: reset frame exceeds the reset limit")
	ErrAckAhead      = errors.New("session: ack beyond the sent head")
)

type entry struct {
	seq   uint64
	frame []byte
	reset bool
}

type ReplayPlan int

const (
	PlanCurrent ReplayPlan = iota
	PlanReplay
	PlanReset
)

type outbound struct {
	mu         sync.Mutex
	tr         Transport
	ring       deque.Deque[entry]
	bytes      int
	head       uint64
	detachedAt uint64
	lastReset  uint64
	discarding bool
	suppress   bool
	closed     bool
	maxFrames  int
	maxBytes   int
	maxReset   int
}

func (o *outbound) send(topic, event string, data any) bool {
	o.mu.Lock()
	defer o.mu.Unlock()
	if o.closed || (o.suppress && topic == string(protocol.TopicFrame)) {
		return true
	}
	o.head++
	if o.discarding && o.tr == nil {
		return true
	}
	frame, err := json.Marshal(protocol.Message{Seq: o.head, Topic: topic, Event: event, Data: data})
	if err != nil {
		return false
	}
	if len(frame) > o.maxBytes {
		o.clearLocked()
		if o.tr != nil {
			_ = o.tr.SendFrame(frame)
		}
		return true
	}
	o.ring.PushBack(entry{seq: o.head, frame: frame})
	o.bytes += len(frame)
	for o.ring.Len() > o.maxFrames || o.bytes > o.maxBytes {
		if o.tr == nil {
			o.clearLocked()
			o.discarding = true
			return true
		}
		if o.ring.Len() <= 1 {
			break
		}
		o.bytes -= len(o.ring.PopFront().frame)
	}
	if o.tr != nil {
		_ = o.tr.SendFrame(frame)
	}
	return true
}

func (o *outbound) clearLocked() {
	o.ring.Clear()
	o.bytes = 0
}

func (o *outbound) ack(seq uint64) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	if seq > o.head {
		return ErrAckAhead
	}
	for o.ring.Len() > 0 && o.ring.Front().seq <= seq {
		o.bytes -= len(o.ring.PopFront().frame)
	}
	return nil
}

func (o *outbound) plan(bs uint64, wantReset bool) (ReplayPlan, string) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.planLocked(bs, wantReset)
}

func (o *outbound) planLocked(bs uint64, wantReset bool) (ReplayPlan, string) {
	switch {
	case bs > o.head:
		return PlanReset, protocol.CauseClientAhead
	case wantReset:
		return PlanReset, protocol.CauseRejoinRequest
	case bs == o.head:
		return PlanCurrent, ""
	case o.ring.Len() == 0:
		return PlanReset, protocol.CauseRejoinGap
	case o.ring.Front().seq <= bs+1, o.ring.Front().reset:
		return PlanReplay, ""
	default:
		return PlanReset, protocol.CauseRejoinGap
	}
}

func (o *outbound) appendReset(data any) error {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.appendResetLocked(data)
}

func (o *outbound) appendResetLocked(data any) error {
	if o.closed {
		return nil
	}
	seq := o.head + 1
	frame, err := json.Marshal(protocol.Message{Seq: seq, Topic: string(protocol.TopicFrame), Event: string(protocol.FrameResetAction), Data: data})
	if err != nil {
		return err
	}
	if len(frame) > o.maxReset {
		return ErrResetTooLarge
	}
	o.head = seq
	o.lastReset = seq
	o.clearLocked()
	o.ring.PushBack(entry{seq: seq, frame: frame, reset: true})
	o.bytes = len(frame)
	o.discarding = false
	if o.tr != nil {
		_ = o.tr.SendFrame(frame)
	}
	return nil
}

func (o *outbound) attachPlanned(tr Transport, bs uint64, wantReset bool, build func(cause string) (any, error)) (mode, cause string, err error) {
	o.mu.Lock()
	if o.closed {
		o.mu.Unlock()
		return protocol.ModeCurrent, "", nil
	}
	plan, cause := o.planLocked(bs, wantReset)
	switch plan {
	case PlanCurrent:
		o.attachLocked(tr, bs)
		o.mu.Unlock()
		return protocol.ModeCurrent, "", nil
	case PlanReplay:
		o.attachLocked(tr, bs)
		o.mu.Unlock()
		return protocol.ModeReplay, "", nil
	}
	o.suppress = true
	o.mu.Unlock()

	data, err := build(cause)

	o.mu.Lock()
	defer o.mu.Unlock()
	o.suppress = false
	if err != nil {
		return "", cause, err
	}
	if err := o.appendResetLocked(data); err != nil {
		return "", cause, err
	}
	o.attachLocked(tr, 0)
	return protocol.ModeReset, cause, nil
}

func (o *outbound) attach(tr Transport, from uint64) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.attachLocked(tr, from)
}

func (o *outbound) attachLocked(tr Transport, from uint64) {
	if o.closed {
		return
	}
	o.tr = tr
	o.discarding = false
	all := o.ring.Len() > 0 && o.ring.Front().reset && o.ring.Front().seq > from
	for i := 0; i < o.ring.Len(); i++ {
		e := o.ring.At(i)
		if all || e.seq > from {
			_ = tr.SendFrame(e.frame)
		}
	}
}

func (o *outbound) detach() {
	o.mu.Lock()
	o.tr = nil
	o.detachedAt = o.head
	o.mu.Unlock()
}

func (o *outbound) detachedFrom() uint64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.detachedAt
}

func (o *outbound) suppressFrames(on bool) {
	o.mu.Lock()
	o.suppress = on
	o.mu.Unlock()
}

func (o *outbound) shut() {
	o.mu.Lock()
	o.closed = true
	o.tr = nil
	o.clearLocked()
	o.mu.Unlock()
}

func (o *outbound) current() Transport {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.tr
}

func (o *outbound) lastResetSeq() uint64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.lastReset
}

func (o *outbound) headSeq() uint64 {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.head
}

func (o *outbound) retained() (int, int) {
	o.mu.Lock()
	defer o.mu.Unlock()
	return o.ring.Len(), o.bytes
}
