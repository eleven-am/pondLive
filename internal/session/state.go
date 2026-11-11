package session

import "time"

type State uint32

const (
	StatePending State = iota
	StateLive
	StateDisconnected
	StateClosing
	StateClosed
)

func (s State) String() string {
	switch s {
	case StatePending:
		return "pending"
	case StateLive:
		return "live"
	case StateDisconnected:
		return "disconnected"
	case StateClosing:
		return "closing"
	case StateClosed:
		return "closed"
	default:
		return "unknown"
	}
}

type CloseReason uint32

const (
	ReasonNone CloseReason = iota
	ReasonRemoved
	ReasonShutdown
	ReasonRedirected
	ReasonRenderFailed
	ReasonPendingExpired
	ReasonDisconnectExpired
	ReasonLifetimeExpired
	ReasonWedged
	ReasonRuntime
	ReasonResetTooLarge
	ReasonResetNondeterministic
	ReasonCount
)

func (r CloseReason) String() string {
	switch r {
	case ReasonRemoved:
		return "removed"
	case ReasonShutdown:
		return "shutdown"
	case ReasonRedirected:
		return "redirected"
	case ReasonRenderFailed:
		return "render-failed"
	case ReasonPendingExpired:
		return "pending-expired"
	case ReasonDisconnectExpired:
		return "disconnect-expired"
	case ReasonLifetimeExpired:
		return "lifetime-expired"
	case ReasonWedged:
		return "wedged"
	case ReasonRuntime:
		return "runtime"
	case ReasonResetTooLarge:
		return "reset-too-large"
	case ReasonResetNondeterministic:
		return "reset-nondeterministic"
	default:
		return "none"
	}
}

func (r CloseReason) Evicted() bool {
	switch r {
	case ReasonPendingExpired, ReasonDisconnectExpired, ReasonLifetimeExpired, ReasonWedged, ReasonRuntime, ReasonResetTooLarge, ReasonResetNondeterministic:
		return true
	default:
		return false
	}
}

type Timer interface {
	Stop() bool
}

type Clock interface {
	AfterFunc(d time.Duration, f func()) Timer
}

type realClock struct{}

func (realClock) AfterFunc(d time.Duration, f func()) Timer {
	return time.AfterFunc(d, f)
}
