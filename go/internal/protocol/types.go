package protocol

import (
	"github.com/eleven-am/pondlive/go/internal/route"
	"github.com/eleven-am/pondlive/go/internal/view/diff"
)

type Topic string

const (
	RouteHandler Topic = "router"
	DOMHandler   Topic = "dom"
	TopicFrame   Topic = "frame"

	AckTopic Topic = "ack"
)

type Event struct {
	Type Topic  `json:"t"`
	SID  string `json:"sid"`
}

type ClientEvt struct {
	Event
	Action  string      `json:"a"`
	Payload interface{} `json:"p,omitempty"`
}

type ServerEvt struct {
	Event
	Action  string      `json:"a"`
	Payload interface{} `json:"p,omitempty"`
	Seq     uint64      `json:"seq,omitempty"`
}

type ClientAck struct {
	Event
	Seq uint64 `json:"seq"`
}

type ServerAck struct {
	Event
	Seq uint64 `json:"seq"`
}

type ClientConfig struct {
	Debug *bool `json:"debug,omitempty"`
}

type Boot struct {
	T        string         `json:"t"`
	SID      string         `json:"sid"`
	Ver      int            `json:"ver"`
	Seq      int            `json:"seq"`
	Patch    []diff.Patch   `json:"patch"`
	Location route.Location `json:"location"`
	Client   *ClientConfig  `json:"client,omitempty"`
}

type ServerError struct {
	T       string `json:"t"`
	SID     string `json:"sid"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
