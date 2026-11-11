package protocol

const (
	ModeCurrent = "current"
	ModeReplay  = "replay"
	ModeReset   = "reset"
)

var JoinedModes = []string{ModeCurrent, ModeReplay, ModeReset}

const (
	CauseRejoinGap     = "rejoin-gap"
	CauseRejoinRequest = "rejoin-request"
	CauseClientAhead   = "client-ahead"
	CauseClientRequest = "client-request"
)

var ResetCauses = []string{CauseRejoinGap, CauseRejoinRequest, CauseClientAhead, CauseClientRequest}

const (
	ResyncUnresolved = "unresolved"
	ResyncApplyError = "apply-error"
	ResyncGap        = "gap"
)

var ResyncReasons = []string{ResyncUnresolved, ResyncApplyError, ResyncGap}

const ServerErrorType = "error"

type Message struct {
	Seq   uint64 `json:"seq"`
	Topic string `json:"topic"`
	Event string `json:"event"`
	Data  any    `json:"data"`
}

type JoinFrame struct {
	T      string `json:"t" ts:"JoinType"`
	SID    string `json:"sid"`
	Tok    string `json:"tok"`
	Ver    int    `json:"ver"`
	BS     uint64 `json:"bs"`
	Resync bool   `json:"resync"`
}

type JoinedFrame struct {
	T    string `json:"t" ts:"JoinedType"`
	Gen  uint64 `json:"gen"`
	Tok  string `json:"tok"`
	CS   uint64 `json:"cs"`
	Mode string `json:"mode" ts:"JoinedMode"`
}

type TokFrame struct {
	T   string `json:"t" ts:"TokType"`
	Tok string `json:"tok"`
}

type DeclinedFrame struct {
	T      string `json:"t" ts:"DeclinedType"`
	Code   int    `json:"code"`
	Reason string `json:"reason"`
}

type EvackFrame struct {
	T  string `json:"t" ts:"EvackType"`
	CS uint64 `json:"cs"`
}

type ClientAck struct {
	T   string `json:"t" ts:"AckType"`
	SID string `json:"sid"`
	Seq uint64 `json:"seq"`
}

type ClientResync struct {
	T      string `json:"t" ts:"ResyncType"`
	SID    string `json:"sid"`
	Seq    uint64 `json:"seq"`
	Reason string `json:"reason" ts:"ResyncReason"`
}
