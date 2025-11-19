package session

import (
	"github.com/eleven-am/pondlive/go/internal/protocol"
)

// Transport delivers protocol messages to the client connection backing a session.
// This interface mirrors the pattern from runtime1.Transport but works with runtime2/session.
type Transport interface {
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

	// Optional features (pubsub, uploads)
	SendPubsubControl(ctrl protocol.PubsubControl) error
	SendUploadControl(ctrl protocol.UploadControl) error

	// Connection lifecycle
	Close() error
}
