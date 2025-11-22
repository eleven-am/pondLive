package server

import (
	"errors"
	"sync/atomic"

	"github.com/eleven-am/pondlive/go/internal/protocol"
)

var errTransportClosed = errors.New("server: transport closed")

type channelSender interface {
	BroadcastTo(event string, payload any, userIDs ...string) error
}

type transport struct {
	channel channelSender
	target  string

	closed atomic.Bool
}

func newTransport(ch channelSender, target string) *transport {
	return &transport{
		channel: ch,
		target:  target,
	}
}

func (t *transport) IsLive() bool {
	return true
}

func (t *transport) Close() error {
	t.closed.Store(true)
	return nil
}

func (t *transport) SendBoot(boot protocol.Boot) error {
	if boot.T == "" {
		boot.T = "boot"
	}
	return t.send("boot", boot)
}

func (t *transport) SendInit(init protocol.Init) error {
	if init.T == "" {
		init.T = "init"
	}
	return t.send("init", init)
}

func (t *transport) SendResume(res protocol.ResumeOK) error {
	if res.T == "" {
		res.T = "resume"
	}
	return t.send("resume", res)
}

func (t *transport) SendFrame(frame protocol.Frame) error {
	if frame.T == "" {
		frame.T = "frame"
	}
	return t.send("frame", frame)
}

func (t *transport) SendEventAck(ack protocol.EventAck) error {
	if ack.T == "" {
		ack.T = "evt-ack"
	}
	return t.send("evt-ack", ack)
}

func (t *transport) SendServerError(err protocol.ServerError) error {
	if err.T == "" {
		err.T = "error"
	}
	return t.send("error", err)
}

func (t *transport) SendDiagnostic(diag protocol.Diagnostic) error {
	if diag.T == "" {
		diag.T = "diagnostic"
	}
	return t.send("diagnostic", diag)
}

func (t *transport) SendPubsubControl(ctrl protocol.PubsubControl) error {
	if ctrl.T == "" {
		ctrl.T = "pubsub"
	}
	return t.send("pubsub", ctrl)
}

func (t *transport) SendUploadControl(ctrl protocol.UploadControl) error {
	if ctrl.T == "" {
		ctrl.T = "upload"
	}
	return t.send("upload", ctrl)
}

func (t *transport) SendDOMRequest(req protocol.DOMRequest) error {
	if req.T == "" {
		req.T = "dom_req"
	}
	return t.send("dom_req", req)
}

func (t *transport) SendScriptEvent(event protocol.ScriptEvent) error {
	if event.T == "" {
		event.T = "script:event"
	}
	return t.send("script:event", event)
}

func (t *transport) send(event string, payload any) error {
	if t.closed.Load() {
		return errTransportClosed
	}
	ch := t.channel
	target := t.target
	if ch == nil || target == "" {
		return errors.New("server: transport missing channel or recipient")
	}
	return ch.BroadcastTo(event, payload, target)
}
