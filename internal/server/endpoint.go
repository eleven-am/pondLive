package server

import (
	"errors"
	"net/http"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/session"
	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

type Endpoint struct {
	registry    *SessionRegistry
	endpoint    *pond.Endpoint
	pubsubLobby *PubSubLobby
}

const (
	sessionAssignKey = "live.session"
	headersAssignKey = "live.headers"
)

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
	e.configure()
	e.pubsubLobby = NewPubSubLobby(endpoint, registry)

	return e, nil
}

func (e *Endpoint) configure() {
	lobby := e.endpoint.CreateChannel("live/:sid", e.onJoin)
	lobby.OnMessage("evt", e.onEvt)
	lobby.OnMessage("ack", e.onAck)
	lobby.OnLeave(e.onLeave)
}

type joinPayload struct {
	SID string `json:"sid"`
	Ver int    `json:"ver"`
	Ack int    `json:"ack"`
}

func (e *Endpoint) onJoin(ctx *pond.JoinContext) error {
	var payload joinPayload
	if err := ctx.ParsePayload(&payload); err != nil {
		return ctx.Decline(pond.StatusBadRequest, "invalid join payload")
	}

	sessionID := ""
	if ctx.Route != nil {
		sessionID = ctx.Route.Param("sid")
	}

	if payload.SID != "" {
		sessionID = payload.SID
	}

	if sessionID == "" {
		return ctx.Decline(pond.StatusBadRequest, "missing session identifier")
	}

	if _, ok := e.registry.Lookup(session.SessionID(sessionID)); !ok {
		return ctx.Decline(pond.StatusNotFound, "session not found or expired")
	}

	ctx.SetAssigns(sessionAssignKey, sessionID)

	user := ctx.GetUser()
	ctx.Accept()
	if errStr := ctx.Error(); errStr != "" {
		return errors.New(errStr)
	}

	var headers http.Header
	if h := ctx.GetAssigns(headersAssignKey); h != nil {
		if hdr, ok := h.(http.Header); ok {
			headers = hdr
		}
	}

	transport := session.NewWebSocketTransport(ctx.Channel, user.UserID, headers)
	sess, err := e.registry.Attach(session.SessionID(sessionID), user.UserID, transport)
	if err != nil {
		_ = transport.Close()
		return err
	}

	go func() {
		if err := sess.Flush(); err != nil {
			e.registry.Detach(user.UserID)
		}
	}()

	return nil
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
	var evt protocol.ClientEvt
	if err := ctx.ParsePayload(&evt); err != nil {
		return err
	}

	sess, transport, ok := e.getSession(ctx, evt.SID)
	if !ok || sess == nil {
		return nil
	}

	sess.Touch()
	if bus := sess.Bus(); bus != nil {
		bus.Publish(evt.Type, evt.Action, evt.Payload)
	}

	if transport != nil {
		if wsTransport, isWS := transport.(*session.WebSocketTransport); isWS {
			wsTransport.SendAck(evt.SID)
		}
	}

	return nil
}
