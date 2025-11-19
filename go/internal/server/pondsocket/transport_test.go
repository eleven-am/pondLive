package pondsocket

import (
	"errors"
	"sync"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/protocol"
)

type stubChannel struct {
	mu    sync.Mutex
	calls []struct {
		event   string
		payload any
		targets []string
	}
	err error
}

func (s *stubChannel) BroadcastTo(event string, payload any, userIDs ...string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	call := struct {
		event   string
		payload any
		targets []string
	}{
		event:   event,
		payload: payload,
		targets: append([]string(nil), userIDs...),
	}
	s.calls = append(s.calls, call)
	return nil
}

func TestTransportSendsNormalizedEvents(t *testing.T) {
	ch := &stubChannel{}
	tr := newTransport(ch, "user-1")

	if err := tr.SendBoot(protocol.Boot{}); err != nil {
		t.Fatalf("SendBoot: %v", err)
	}
	if err := tr.SendInit(protocol.Init{}); err != nil {
		t.Fatalf("SendInit: %v", err)
	}
	if err := tr.SendResume(protocol.ResumeOK{}); err != nil {
		t.Fatalf("SendResume: %v", err)
	}
	if err := tr.SendFrame(protocol.Frame{}); err != nil {
		t.Fatalf("SendFrame: %v", err)
	}
	if err := tr.SendEventAck(protocol.EventAck{}); err != nil {
		t.Fatalf("SendEventAck: %v", err)
	}
	if err := tr.SendServerError(protocol.ServerError{}); err != nil {
		t.Fatalf("SendServerError: %v", err)
	}
	if err := tr.SendDiagnostic(protocol.Diagnostic{}); err != nil {
		t.Fatalf("SendDiagnostic: %v", err)
	}
	if err := tr.SendPubsubControl(protocol.PubsubControl{Op: "join", Topic: "news"}); err != nil {
		t.Fatalf("SendPubsubControl: %v", err)
	}
	if err := tr.SendUploadControl(protocol.UploadControl{Op: "cancel", ID: "u1"}); err != nil {
		t.Fatalf("SendUploadControl: %v", err)
	}
	if err := tr.SendDOMRequest(protocol.DOMRequest{ID: "req-1", Ref: "ref:1"}); err != nil {
		t.Fatalf("SendDOMRequest: %v", err)
	}

	ch.mu.Lock()
	defer ch.mu.Unlock()

	if len(ch.calls) != 10 {
		t.Fatalf("expected 10 calls, got %d", len(ch.calls))
	}

	want := []string{"boot", "init", "resume", "frame", "evt-ack", "error", "diagnostic", "pubsub", "upload", "domreq"}
	for i, event := range want {
		if ch.calls[i].event != event {
			t.Fatalf("call %d expecting event %q, got %q", i, event, ch.calls[i].event)
		}
		if len(ch.calls[i].targets) != 1 || ch.calls[i].targets[0] != "user-1" {
			t.Fatalf("call %d targets mismatch: %#v", i, ch.calls[i].targets)
		}

		switch payload := ch.calls[i].payload.(type) {
		case protocol.Boot:
			if payload.T != "boot" {
				t.Fatalf("expected boot payload T, got %q", payload.T)
			}
		case protocol.Init:
			if payload.T != "init" {
				t.Fatalf("expected init payload T, got %q", payload.T)
			}
		case protocol.ResumeOK:
			if payload.T != "resume" {
				t.Fatalf("expected resume payload T, got %q", payload.T)
			}
		case protocol.Frame:
			if payload.T != "frame" {
				t.Fatalf("expected frame payload T, got %q", payload.T)
			}
		case protocol.EventAck:
			if payload.T != "evt-ack" {
				t.Fatalf("expected ack payload T, got %q", payload.T)
			}
		case protocol.ServerError:
			if payload.T != "error" {
				t.Fatalf("expected error payload T, got %q", payload.T)
			}
		case protocol.Diagnostic:
			if payload.T != "diagnostic" {
				t.Fatalf("expected diagnostic payload T, got %q", payload.T)
			}
		case protocol.PubsubControl:
			if payload.T != "pubsub" {
				t.Fatalf("expected pubsub payload T, got %q", payload.T)
			}
			if payload.Op != "join" || payload.Topic != "news" {
				t.Fatalf("unexpected pubsub payload: %#v", payload)
			}
		case protocol.UploadControl:
			if payload.T != "upload" {
				t.Fatalf("expected upload payload T, got %q", payload.T)
			}
			if payload.Op != "cancel" || payload.ID != "u1" {
				t.Fatalf("unexpected upload payload: %#v", payload)
			}
		case protocol.DOMRequest:
			if payload.T != "domreq" {
				t.Fatalf("expected domreq payload T, got %q", payload.T)
			}
			if payload.ID != "req-1" || payload.Ref != "ref:1" {
				t.Fatalf("unexpected dom request payload: %#v", payload)
			}
		default:
			t.Fatalf("unexpected payload type %T", payload)
		}
	}
}

func TestTransportClosePreventsSend(t *testing.T) {
	ch := &stubChannel{}
	tr := newTransport(ch, "user-1")

	if err := tr.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	if err := tr.SendFrame(protocol.Frame{}); !errors.Is(err, errTransportClosed) {
		t.Fatalf("expected errTransportClosed, got %v", err)
	}
}

func TestTransportMissingChannel(t *testing.T) {
	tr := newTransport(nil, "user-1")
	if err := tr.SendFrame(protocol.Frame{}); err == nil {
		t.Fatalf("expected error when channel missing")
	}
}
