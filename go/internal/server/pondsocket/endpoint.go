package pondsocket

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/server"
)

const (
	sessionAssignKey         = "live.session"
	connectionStateAssignKey = "live.connection_state"
)

// Endpoint wires LiveUI sessions to a PondSocket server endpoint.
type Endpoint struct {
	registry *server.SessionRegistry
	endpoint *pond.Endpoint
	pubsub   *pubsubProvider
}

// Register attaches a LiveUI-aware endpoint to the provided PondSocket server.
// The registry must contain sessions rendered during SSR so they can be resumed
// when the websocket connection is established.
func Register(srv *pond.Manager, path string, registry *server.SessionRegistry) (*Endpoint, error) {
	if srv == nil {
		return nil, errors.New("live: pondsocket server is nil")
	}
	if registry == nil {
		return nil, errors.New("live: session registry is nil")
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
	lobby.OnMessage("upload", e.onUpload)
	lobby.OnMessage("domres", e.onDOMResponse)
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

	if _, ok := e.registry.Lookup(runtime.SessionID(sessionID)); !ok {
		return ctx.Decline(pond.StatusNotFound, "session not found or expired")
	}

	ctx.SetAssigns(sessionAssignKey, sessionID)

	user := ctx.GetUser()
	ctx.Accept()
	if errStr := ctx.Error(); errStr != "" {
		return errors.New(errStr)
	}

	transport := newTransport(ctx.Channel, user.UserID)
	sess, err := e.registry.Attach(runtime.SessionID(sessionID), user.UserID, transport)
	if err != nil {
		_ = transport.Close()
		return err
	}

	if raw := ctx.GetAssigns(connectionStateAssignKey); raw != nil {
		switch snapshot := raw.(type) {
		case *connectionState:
			sess.MergeConnectionState(snapshot.Headers, snapshot.Cookies)
		case connectionState:
			sess.MergeConnectionState(snapshot.Headers, snapshot.Cookies)
		}
	}

	if payload.Loc.Path != "" || payload.Loc.Query != "" {
		sess.SetLocation(payload.Loc.Path, payload.Loc.Query)
	}

	join := sess.Join(payload.Ver, payload.Ack)

	if join.Init != nil {
		if err := transport.SendInit(*join.Init); err != nil {
			e.registry.DetachConnection(user.UserID)
			return err
		}
	}

	if join.Resume != nil {
		if err := transport.SendResume(*join.Resume); err != nil {
			e.registry.DetachConnection(user.UserID)
			return err
		}
	}

	for _, frame := range join.Templates {
		if err := transport.SendTemplate(frame); err != nil {
			e.registry.DetachConnection(user.UserID)
			return err
		}
	}

	for _, frame := range join.Frames {
		if err := transport.SendFrame(frame); err != nil {
			e.registry.DetachConnection(user.UserID)
			return err
		}
	}

	return nil
}

func (e *Endpoint) onClientEvent(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, transport, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil || transport == nil {
		return nil
	}

	var envelope protocol.ClientEvent
	if err := ctx.ParsePayload(&envelope); err != nil {
		diagErr := badPayloadDiagnostic("transport:parse", "failed to decode client event envelope", err, nil)
		return transport.SendServerError(serverError(session.ID(), "bad_payload", diagErr))
	}

	wire, err := decodeWireEvent(envelope.Payload)
	if err != nil {
		return transport.SendServerError(serverError(session.ID(), "bad_payload", err))
	}

	go func() {
		if err := session.DispatchEvent(envelope.HID, wire.ToEvent(), envelope.CSeq); err != nil {
			_ = transport.SendServerError(serverError(session.ID(), "dispatch_failed", err))
		}
	}()

	ack := protocol.EventAck{
		SID:  string(session.ID()),
		CSeq: envelope.CSeq,
	}
	return transport.SendEventAck(ack)
}

func (e *Endpoint) onUpload(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, transport, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil {
		return nil
	}

	var payload protocol.UploadClient
	if err := ctx.ParsePayload(&payload); err != nil {
		diagErr := badPayloadDiagnostic("transport:parse", "failed to decode upload envelope", err, nil)
		if transport != nil {
			_ = transport.SendServerError(serverError(session.ID(), "bad_payload", diagErr))
		}
		return nil
	}

	if err := session.HandleUploadMessage(payload); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.ID(), "upload_error", err))
		}
		return err
	}
	return nil
}

func (e *Endpoint) onAck(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, _, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil {
		return nil
	}

	var ack protocol.ClientAck
	if err := ctx.ParsePayload(&ack); err != nil {
		return nil
	}

	session.Ack(ack.Seq)
	return nil
}

func (e *Endpoint) onDOMResponse(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, _, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil {
		return nil
	}

	var payload protocol.DOMResponse
	if err := ctx.ParsePayload(&payload); err != nil {
		return nil
	}

	session.HandleDOMResponse(payload)
	return nil
}

type navPayload struct {
	Path  string `json:"path"`
	Query string `json:"q"`
	Hash  string `json:"hash"`
}

func (e *Endpoint) onNavigate(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, transport, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil {
		return nil
	}

	var payload navPayload
	if err := ctx.ParsePayload(&payload); err != nil {
		return nil
	}

	if comp := session.ComponentSession(); comp != nil {
		runtime.InternalHandleNav(comp, runtime.NavMsg{T: "nav", Path: payload.Path, Q: payload.Query, Hash: payload.Hash})
	}
	session.SetLocation(payload.Path, payload.Query)
	if err := session.Flush(); err != nil {
		if transport != nil {
			return transport.SendServerError(serverError(session.ID(), "flush_failed", err))
		}
		return err
	}
	return nil
}

func (e *Endpoint) onPopState(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, transport, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil {
		return nil
	}

	var payload navPayload
	if err := ctx.ParsePayload(&payload); err != nil {
		return nil
	}

	if comp := session.ComponentSession(); comp != nil {
		runtime.InternalHandlePop(comp, runtime.PopMsg{T: "pop", Path: payload.Path, Q: payload.Query, Hash: payload.Hash})
	}
	session.SetLocation(payload.Path, payload.Query)
	if err := session.Flush(); err != nil {
		if transport != nil {
			return transport.SendServerError(serverError(session.ID(), "flush_failed", err))
		}
		return err
	}
	return nil
}

func (e *Endpoint) onRouterReset(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, transport, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil {
		return nil
	}

	var payload protocol.RouterReset
	if err := ctx.ParsePayload(&payload); err != nil {
		diagErr := badPayloadDiagnostic("transport:parse", "failed to decode router reset payload", err, nil)
		if transport != nil {
			_ = transport.SendServerError(serverError(session.ID(), "bad_payload", diagErr))
		}
		return nil
	}

	if err := session.HandleRouterReset(payload.ComponentID); err != nil {
		if transport != nil {
			_ = transport.SendServerError(serverError(session.ID(), "router_reset_failed", err))
		}
		return err
	}
	return nil
}

// PubsubProvider exposes the runtime pub/sub provider configured for this endpoint.
func (e *Endpoint) PubsubProvider() runtime.PubsubProvider {
	if e == nil {
		return nil
	}
	return e.pubsub
}

func (e *Endpoint) onLeave(ctx *pond.LeaveContext) {
	if ctx == nil || ctx.User == nil {
		return
	}
	e.registry.DetachConnection(ctx.User.UserID)
}

type connectionState struct {
	Headers http.Header
	Cookies []*http.Cookie
}

func cloneHeader(headers http.Header) http.Header {
	if len(headers) == 0 {
		return nil
	}
	copy := make(http.Header, len(headers))
	for key, values := range headers {
		copy[key] = append([]string(nil), values...)
	}
	return copy
}

func cloneCookies(cookies []*http.Cookie) []*http.Cookie {
	if len(cookies) == 0 {
		return nil
	}
	out := make([]*http.Cookie, 0, len(cookies))
	for _, ck := range cookies {
		if ck == nil || ck.Name == "" {
			continue
		}
		clone := *ck
		if len(ck.Unparsed) > 0 {
			clone.Unparsed = append([]string(nil), ck.Unparsed...)
		}
		out = append(out, &clone)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func (e *Endpoint) onRecover(ctx *pond.EventContext) error {
	user := ctx.GetUser()
	if user == nil {
		return nil
	}

	session, transport, ok := e.registry.LookupByConnection(user.UserID)
	if !ok || session == nil {
		return nil
	}

	if err := session.Recover(); err != nil {
		if transport != nil {
			return transport.SendServerError(serverError(session.ID(), "recover_failed", err))
		}
		return err
	}
	return nil
}

func decodeWireEvent(payload any) (runtime.WireEvent, error) {
	switch v := payload.(type) {
	case runtime.WireEvent:
		return v, nil
	case *runtime.WireEvent:
		if v == nil {
			return runtime.WireEvent{}, nil
		}
		return *v, nil
	case map[string]any:
		return mapToWireEvent(v)
	default:
		var wire runtime.WireEvent
		data, err := json.Marshal(payload)
		if err != nil {
			return wire, badPayloadDiagnostic("transport:encode", "failed to encode client event payload", err, map[string]any{"payloadType": fmt.Sprintf("%T", payload)})
		}
		if err := json.Unmarshal(data, &wire); err != nil {
			return wire, badPayloadDiagnostic("transport:decode", "failed to decode client event payload", err, map[string]any{"payloadType": fmt.Sprintf("%T", payload)})
		}
		return wire, nil
	}
}

func mapToWireEvent(m map[string]any) (runtime.WireEvent, error) {
	var wire runtime.WireEvent
	if name, ok := m["name"].(string); ok {
		wire.Name = name
	} else if eventType, ok := m["type"].(string); ok {
		wire.Name = eventType
	}

	if value, ok := m["value"].(string); ok {
		wire.Value = value
	}

	if payload, ok := m["payload"].(map[string]any); ok {
		wire.Payload = cloneAnyMap(payload)
	} else {
		wire.Payload = cloneAnyMap(m)
	}

	if form, ok := m["form"].(map[string]any); ok {
		wire.Form = cloneStringMap(form)
	} else if formMap, ok := m["form"].(map[string]string); ok {
		wire.Form = cloneStringMapString(formMap)
	}
	if mods, ok := m["mods"].(map[string]any); ok {
		wire.Mods = convertModifiers(mods)
	}
	return wire, nil
}

func cloneAnyMap(src map[string]any) map[string]any {
	if len(src) == 0 {
		return map[string]any{}
	}
	out := make(map[string]any, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func cloneStringMap(src map[string]any) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		if str, ok := v.(string); ok {
			out[k] = str
		}
	}
	return out
}

func cloneStringMapString(src map[string]string) map[string]string {
	if len(src) == 0 {
		return map[string]string{}
	}
	out := make(map[string]string, len(src))
	for k, v := range src {
		out[k] = v
	}
	return out
}

func convertModifiers(src map[string]any) runtime.WireModifiers {
	mods := runtime.WireModifiers{}
	if v, ok := src["ctrl"].(bool); ok {
		mods.Ctrl = v
	}
	if v, ok := src["meta"].(bool); ok {
		mods.Meta = v
	}
	if v, ok := src["shift"].(bool); ok {
		mods.Shift = v
	}
	if v, ok := src["alt"].(bool); ok {
		mods.Alt = v
	}
	if v, ok := src["button"].(float64); ok {
		mods.Button = int(v)
	} else if vInt, ok := src["button"].(int); ok {
		mods.Button = vInt
	}
	return mods
}

func serverError(id runtime.SessionID, code string, err error) protocol.ServerError {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	if diag, ok := runtime.AsDiagnosticError(err); ok {
		payload := diag.ToServerError(id)
		if code != "" {
			payload.Code = code
		}
		if msg != "" {
			payload.Message = msg
		}
		return payload
	}
	if code == "" {
		code = "runtime_error"
	}
	return protocol.ServerError{
		T:       "error",
		SID:     string(id),
		Code:    code,
		Message: msg,
	}
}

func badPayloadDiagnostic(phase, message string, err error, extras map[string]any) error {
	meta := map[string]any{"error": err.Error()}
	for k, v := range extras {
		meta[k] = v
	}
	diag := runtime.Diagnostic{
		Code:       "bad_payload",
		Phase:      phase,
		Message:    message,
		Suggestion: "Ensure the client event payload matches the expected structure.",
		Metadata:   meta,
	}
	return diag.AsError()
}
