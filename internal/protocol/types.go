package protocol

import (
	"github.com/eleven-am/pondlive/internal/route"
)

type Topic string

const (
	RouteHandler    Topic = "router"
	DOMHandler      Topic = "dom"
	TopicFrame      Topic = "frame"
	HandlerTopic    Topic = "h"
	TopicDiagnostic Topic = "diagnostic"
)

type Event struct {
	Type Topic  `json:"t" ts:"EventTopic"`
	SID  string `json:"sid"`
}

type ClientEvt struct {
	Event
	Action  string      `json:"a"`
	I       uint32      `json:"i,omitempty"`
	Payload interface{} `json:"p,omitempty"`
	CS      uint64      `json:"cs"`
}

type ClientConfig struct {
	Debug *bool `json:"debug,omitempty"`
}

type Boot struct {
	T        string         `json:"t" ts:"BootType"`
	SID      string         `json:"sid"`
	Ver      int            `json:"ver"`
	Seq      uint64         `json:"seq"`
	Location route.Location `json:"location"`
	Client   *ClientConfig  `json:"client,omitempty"`
	WS       string         `json:"ws"`
	Tok      string         `json:"tok"`
}

type ServerError struct {
	T          string         `json:"t"`
	SID        string         `json:"sid"`
	Code       string         `json:"code"`
	Message    string         `json:"message"`
	StackTrace string         `json:"stack,omitempty"`
	Meta       map[string]any `json:"meta,omitempty"`
}

func (e *ServerError) Error() string {
	if e == nil {
		return ""
	}
	return e.Message
}

type Diagnostic struct {
	Phase      string
	Message    string
	StackTrace string
	Metadata   map[string]any
}

type ResetPayload struct {
	HTML     string         `json:"html"`
	Location route.Location `json:"location"`
	Cause    string         `json:"cause" ts:"ResetCause"`
}
