package server

import (
	"errors"
	"fmt"
	"net/http"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/session"
)

const (
	sessionAssignKey         = "live.session"
	connectionStateAssignKey = "live.connection_state"
)

// Endpoint wires LiveSession instances to a PondSocket server endpoint.
type Endpoint struct {
	registry *SessionRegistry
	endpoint *pond.Endpoint
	pubsub   *pubsubProvider
}

// Register attaches a LiveSession-aware endpoint to the provided PondSocket server.
// The registry must contain sessions rendered during SSR so they can be resumed
// when the websocket connection is established.
func Register(srv *pond.Manager, path string, registry *SessionRegistry) (*Endpoint, error) {
	if srv == nil {
		return nil, errors.New("server: pondsocket server is nil")
	}
	if registry == nil {
		return nil, errors.New("server: session registry is nil")
	}

	endpoint := srv.CreateEndpoint(path, func(ctx *pond.ConnectionContext) error {
		state := &connectionState{
			Headers: cloneHeader(ctx.Headers()),
			Cookies: cloneCookies(ctx.Cookies()),
		}
		ctx.SetAssigns(connectionStateAssignKey, state)
		return ctx.Accept()
	})

	e := &Endpoint{
		registry: registry,
		endpoint: endpoint,
	}
	e.pubsub = newPubsubProvider(e, registry)
	e.configure()
	return e, nil
}

func (e *Endpoint) configure() {
	lobby := e.endpoint.CreateChannel("live/:sid", e.onJoin)
	lobby.OnMessage("evt", e.onClientEvent)
	lobby.OnMessage("ack", e.onAck)
	lobby.OnMessage("nav", e.onNavigate)
	lobby.OnMessage("pop", e.onPopState)
	lobby.OnMessage("routerReset", e.onRouterReset)
	lobby.OnMessage("recover", e.onRecover)
	lobby.OnMessage("dom_res", e.onDOMResponse)
	lobby.OnMessage("script:message", e.onScriptMessage)
	lobby.OnLeave(e.onLeave)

	if e.pubsub != nil {
		pub := e.endpoint.CreateChannel("pubsub/:topic", e.pubsub.handleJoin)
		pub.OnOutgoing("pub", e.pubsub.handleOutgoing)
		pub.OnLeave(e.pubsub.handleLeave)
	}
}

type joinPayload struct {
	SID string            `json:"sid"`
	Ver int               `json:"ver"`
	Ack int               `json:"ack"`
	Loc protocol.Location `json:"loc"`
}

type navPayload struct {
	SID   string `json:"sid"`
	Path  string `json:"path"`
	Query string `json:"q"`
	Hash  string `json:"hash"`
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

	transport := newTransport(ctx.Channel, user.UserID)
	sess, err := e.registry.Attach(session.SessionID(sessionID), user.UserID, transport)
	if err != nil {
		_ = transport.Close()
		return err
	}

	if raw := ctx.GetAssigns(connectionStateAssignKey); raw != nil {
		switch snapshot := raw.(type) {
		case *connectionState:
			mergeConnectionState(sess, snapshot)
		case connectionState:
			mergeConnectionState(sess, &snapshot)
		}
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
	if !ok || sess == nil || transport == nil {
		return nil
	}

	domEvent := payloadToDOMEvent(evt.Payload)

	ack := protocol.EventAck{
		T:    "evt_ack",
		SID:  evt.SID,
		CSeq: evt.CSeq,
	}

	if err := transport.SendEventAck(ack); err != nil {
		return err
	}

	go func() {
		if err := sess.HandleEvent(evt.HID, domEvent); err != nil {
			_ = transport.SendServerError(serverError(session.SessionID(evt.SID), "event_failed", err))
			return
		}
		if err := sess.Flush(); err != nil {
			_ = transport.SendServerError(serverError(session.SessionID(evt.SID), "flush_failed", err))
		}
	}()

	return nil
}

func (e *Endpoint) onAck(ctx *pond.EventContext) error {
	var ack protocol.ClientAck
	if err := ctx.ParsePayload(&ack); err != nil {
		return err
	}

	sess, ok := e.registry.Lookup(session.SessionID(ack.SID))
	if !ok || sess == nil {
		return nil
	}

	sess.Ack(ack.Seq)
	return nil
}

func (e *Endpoint) onNavigate(ctx *pond.EventContext) error {
	var nav navPayload
	if err := ctx.ParsePayload(&nav); err != nil {
		return nil
	}

	sess, transport, ok := e.getSession(ctx, nav.SID)
	if !ok || sess == nil {
		return nil
	}

	if err := sess.HandleNavigation(nav.Path, nav.Query, nav.Hash); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.SessionID(nav.SID), "nav_failed", err))
		}
		return err
	}

	if err := sess.Flush(); err != nil {
		fmt.Println("Flush error:", err)
		if transport != nil {
			_ = transport.SendServerError(serverError(session.SessionID(nav.SID), "flush_failed", err))
		}
		return err
	}

	return nil
}

func (e *Endpoint) onDOMResponse(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	var resp protocol.DOMResponse
	if err := ctx.ParsePayload(&resp); err != nil {
		return err
	}

	sessionID := resp.SID
	if sessionID == "" {
		return nil
	}

	sess, ok := e.registry.Lookup(session.SessionID(sessionID))
	if !ok || sess == nil {
		return nil
	}

	sess.HandleDOMResponse(resp)
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

func (e *Endpoint) onPopState(ctx *pond.EventContext) error {

	var nav navPayload
	if err := ctx.ParsePayload(&nav); err != nil {
		return nil
	}

	sess, transport, ok := e.getSession(ctx, nav.SID)
	if !ok || sess == nil {
		return nil
	}

	if err := sess.HandlePopState(nav.Path, nav.Query, nav.Hash); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.SessionID(nav.SID), "pop_failed", err))
		}
		return err
	}

	if err := sess.Flush(); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.SessionID(nav.SID), "flush_failed", err))
		}
		return err
	}
	return nil
}

func (e *Endpoint) onRouterReset(ctx *pond.EventContext) error {
	var payload protocol.RouterReset
	if err := ctx.ParsePayload(&payload); err != nil {
		return nil
	}

	sess, transport, ok := e.getSession(ctx, payload.SID)
	if !ok || sess == nil {
		return nil
	}

	if err := sess.HandleRouterReset(payload.ComponentID); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.SessionID(payload.SID), "router_reset_failed", err))
		}
		return err
	}

	if err := sess.Flush(); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.SessionID(payload.SID), "flush_failed", err))
		}
		return err
	}

	return nil
}

func (e *Endpoint) onRecover(ctx *pond.EventContext) error {
	var payload protocol.Recover
	if err := ctx.ParsePayload(&payload); err != nil {
		return nil
	}

	sess, transport, ok := e.getSession(ctx, payload.SID)
	if !ok || sess == nil {
		return nil
	}

	if err := sess.Recover(); err != nil {
		if transport != nil {
			return transport.SendServerError(serverError(session.SessionID(payload.SID), "recover_failed", err))
		}
		return err
	}

	if err := sess.Flush(); err != nil {
		if transport != nil {
			return transport.SendServerError(serverError(session.SessionID(payload.SID), "flush_failed", err))
		}
		return err
	}

	return nil
}

func (e *Endpoint) onScriptMessage(ctx *pond.EventContext) error {
	var payload protocol.ScriptMessage
	if err := ctx.ParsePayload(&payload); err != nil {
		return nil
	}

	sess, transport, ok := e.getSession(ctx, payload.SID)
	if !ok || sess == nil {
		return nil
	}

	if err := sess.HandleScriptMessage(payload); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.SessionID(payload.SID), "script_message_failed", err))
		}
		return err
	}

	return nil
}

// PubsubProvider exposes the pubsub provider for session configuration.
func (e *Endpoint) PubsubProvider() runtime.PubsubProvider {
	if e == nil {
		return nil
	}
	return e.pubsub
}

// Helper types and functions

type connectionState struct {
	Headers http.Header
	Cookies []*http.Cookie
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

func cloneCookies(cookies []*http.Cookie) []*http.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	clone := make([]*http.Cookie, len(cookies))
	for i, c := range cookies {
		if c != nil {
			cp := *c
			clone[i] = &cp
		}
	}
	return clone
}

func mergeConnectionState(sess *session.LiveSession, state *connectionState) {

}

func payloadToDOMEvent(payload map[string]interface{}) dom.Event {
	event := dom.Event{
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
