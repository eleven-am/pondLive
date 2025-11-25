package session2

import "github.com/eleven-am/pondlive/go/internal/headers2"

// Transport is a simplified interface for sending messages to clients.
// Unlike the old Transport, this has no specialized methods - it just sends
// generic messages that session forwards from the Bus.
type Transport interface {
	// Send sends a message to the client.
	// topic identifies the message type (e.g., "frame", "dom", "script").
	// event specifies the action (e.g., "patch", "query", "response").
	// data contains the message payload.
	Send(topic string, event string, data any) error

	// IsLive returns true for WebSocket connections that support real-time updates.
	// Returns false for SSR or other non-live transports.
	IsLive() bool

	// RequestInfo returns the HTTP request information captured when the transport was created.
	// For SSR, this is the request being served.
	// For WebSocket, this is the handshake request (may be updated on reconnect).
	RequestInfo() *headers2.RequestInfo

	// Close closes the transport connection.
	Close() error
}
