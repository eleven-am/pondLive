package session

import (
	"github.com/eleven-am/pondlive/go/internal/protocol"
)

// Transport delivers protocol messages to the client connection backing a session.
// This interface mirrors the pattern from runtime1.Transport but works with runtime2/session.
type Transport interface {
	// IsLive returns true if this is a live WebSocket connection that supports
	// real-time updates. Returns false for SSR or other non-live transports.
	IsLive() bool

	// Session lifecycle messages
	SendBoot(boot protocol.Boot) error
	SendInit(init protocol.Init) error
	SendResume(res protocol.ResumeOK) error

	// Frame updates (DOM patches + effects)
	SendFrame(frame protocol.Frame) error

	// Event acknowledgements
	SendEventAck(ack protocol.EventAck) error

	// Error and diagnostic messages
	SendServerError(err protocol.ServerError) error
	SendDiagnostic(diag protocol.Diagnostic) error

	// Client DOM API requests
	SendDOMRequest(req protocol.DOMRequest) error

	// Optional features (pubsub, scripts)
	SendPubsubControl(ctrl protocol.PubsubControl) error
	SendScriptEvent(event protocol.ScriptEvent) error

	// Connection lifecycle
	Close() error
}
