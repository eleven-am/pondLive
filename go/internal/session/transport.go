package session

import "github.com/eleven-am/pondlive/go/internal/headers"

type Transport interface {
	Send(topic string, event string, data any) error

	IsLive() bool

	RequestInfo() *headers.RequestInfo

	RequestState() *headers.RequestState

	Close() error
}
