package server

import (
	"errors"
	"net/http"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/session"
	"github.com/eleven-am/pondlive/go/internal/work"
)

const (
	sessionAssignKey = "live.session"
	headersAssignKey = "live.headers"
)

// Endpoint wires LiveSession instances to a PondSocket server endpoint.
type Endpoint struct {
	registry *SessionRegistry
	endpoint *pond.Endpoint
}

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
	e.configure()
	return e, nil
}

func (e *Endpoint) configure() {
	lobby := e.endpoint.CreateChannel("live/:sid", e.onJoin)
	lobby.OnMessage("evt", e.onClientEvent)
	lobby.OnMessage("ack", e.onAck)
	lobby.OnLeave(e.onLeave)
}

type joinPayload struct {
	SID string            `json:"sid"`
	Ver int               `json:"ver"`
	Ack int               `json:"ack"`
	Loc protocol.Location `json:"loc"`
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

	if err := sess.Flush(); err != nil {
		e.registry.Detach(user.UserID)
		return err
	}

	return nil
}

func (e *Endpoint) onClientEvent(ctx *pond.EventContext) error {
	var evt protocol.ClientEvent
	if err := ctx.ParsePayload(&evt); err != nil {
		return err
	}

	sess, transport, ok := e.getSession(ctx, evt.SID)
	if !ok || sess == nil {
		return nil
	}

	domEvent := payloadToWorkEvent(evt.Payload)

	sess.Receive(evt.HID, "invoke", domEvent)

	if transport != nil {
		ack := map[string]any{
			"t":    "evt_ack",
			"sid":  evt.SID,
			"cseq": evt.CSeq,
		}
		_ = transport.Send("ack", "evt_ack", ack)
	}

	go func() {
		if err := sess.Flush(); err != nil && transport != nil {
			_ = transport.Send("error", "flush_failed", serverError(session.SessionID(evt.SID), "flush_failed", err))
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

// Helper functions

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

func payloadToWorkEvent(payload map[string]interface{}) work.Event {
	event := work.Event{
		Payload: make(map[string]any),
	}

	if name, ok := payload["name"].(string); ok {
		event.Name = name
	} else if eventType, ok := payload["type"].(string); ok {
		event.Name = eventType
	}

	if value, ok := payload["value"].(string); ok {
		event.Value = value
	}

	if form, ok := payload["form"].(map[string]interface{}); ok {
		event.Form = make(map[string]string)
		for k, v := range form {
			if str, ok := v.(string); ok {
				event.Form[k] = str
			}
		}
	}

	if mods, ok := payload["mods"].(map[string]interface{}); ok {
		if ctrl, ok := mods["ctrl"].(bool); ok {
			event.Mods.Ctrl = ctrl
		}
		if meta, ok := mods["meta"].(bool); ok {
			event.Mods.Meta = meta
		}
		if shift, ok := mods["shift"].(bool); ok {
			event.Mods.Shift = shift
		}
		if alt, ok := mods["alt"].(bool); ok {
			event.Mods.Alt = alt
		}
		if button, ok := mods["button"].(float64); ok {
			event.Mods.Button = int(button)
		} else if button, ok := mods["button"].(int); ok {
			event.Mods.Button = button
		}
	}

	for k, v := range payload {
		if k != "name" && k != "type" && k != "value" && k != "form" && k != "mods" {
			event.Payload[k] = v
		}
	}

	return event
}

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return http.Header{}
	}
	clone := make(http.Header, len(h))
	for k, v := range h {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}

func serverError(sid session.SessionID, code string, err error) protocol.ServerError {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return protocol.ServerError{
		T:       "error",
		SID:     string(sid),
		Code:    code,
		Message: msg,
	}
}
