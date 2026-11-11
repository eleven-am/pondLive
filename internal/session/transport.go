package session

import "github.com/coder/websocket"

type Transport interface {
	SendFrame(frame []byte) error

	Closed() bool

	Terminate(code websocket.StatusCode, reason string)
}
