package server2

import (
	"errors"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/session"
	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

// Endpoint wires LiveSession instances to a PondSocket server endpoint.
type Endpoint struct {
	registry *SessionRegistry
	endpoint *pond.Endpoint
}

const (
	sessionAssignKey = "live.session"
	headersAssignKey = "live.headers"
)

// Register attaches a LiveSession-aware endpoint to the provided PondSocket server.
func Register(srv *pond.Manager, path string, registry *SessionRegistry) (*Endpoint, error) {
	if srv == nil {
		return nil, errors.New("server: pondsocket server is nil")
	}
	if registry == nil {
		return nil, errors.New("server: session registry is nil")
	}

	endpoint := srv.CreateEndpoint(path, func(ctx *pond.ConnectionContext) error {
		ctx.SetAssigns(headersAssignKey, cloneHeader(ctx.Headers()))
		return ctx.Accept()
	})

	e := &Endpoint{
		registry: registry,
		endpoint: endpoint,
	}

	return e, nil
}

func (e *Endpoint) onAck(ctx *pond.EventContext) error {
	var ack protocol.ClientAck
	if err := ctx.ParsePayload(&ack); err != nil {
		return err
	}

	_, transport, ok := e.getSession(ctx, ack.SID)
	if !ok {
		return nil
	}

	if wsTransport, ok := transport.(*session.WebSocketTransport); ok {
		wsTransport.AckThrough(uint64(ack.Seq))
	}

	return nil
}

func (e *Endpoint) onLeave(ctx *pond.LeaveContext) {
	if ctx == nil || ctx.User == nil {
		return
	}

	if sess, ok := e.registry.Lookup(session.SessionID(ctx.User.UserID)); ok && sess != nil {
		_ = sess.Close()
	}
	e.registry.Detach(ctx.User.UserID)
}

func (e *Endpoint) getSession(ctx *pond.EventContext, sid string) (*session.LiveSession, session.Transport, bool) {
	user := ctx.GetUser()
	if user == nil {
		return nil, nil, false
	}

	sess, t, ok := e.registry.LookupWithConnection(session.SessionID(sid), user.UserID)
	if !ok || sess == nil {
		return nil, nil, false
	}

	return sess, t, true
}

func (e *Endpoint) onEvt(ctx *pond.EventContext) error {

}
