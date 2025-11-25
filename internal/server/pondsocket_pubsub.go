package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/google/uuid"

	pond "github.com/eleven-am/pondsocket/go/pondsocket"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/session"
)

type pubsubEnvelope struct {
	Data json.RawMessage   `json:"data,omitempty"`
	Meta map[string]string `json:"meta,omitempty"`
}

type pubsubSubscription struct {
	session session.SessionID
	topic   string
}

type pubsubProvider struct {
	endpoint *Endpoint
	registry *SessionRegistry

	deliver func(pubsubSession, Transport, string, pubsubEnvelope) error

	processOutgoing func(session.SessionID, string, string, pubsubEnvelope) error

	mu            sync.Mutex
	subscriptions map[string]pubsubSubscription
	sessionTopics map[session.SessionID]map[string]int
}

var _ runtime.PubsubProvider = (*pubsubProvider)(nil)

type pubsubSession interface {
	DeliverPubsub(topic string, payload []byte, meta map[string]string)
	Flush() error
	ID() session.SessionID
}

type sessionContextKey struct{}

func newPubsubProvider(endpoint *Endpoint, registry *SessionRegistry) *pubsubProvider {
	if endpoint == nil || registry == nil {
		return nil
	}
	provider := &pubsubProvider{
		endpoint:      endpoint,
		registry:      registry,
		subscriptions: make(map[string]pubsubSubscription),
		sessionTopics: make(map[session.SessionID]map[string]int),
	}
	provider.deliver = provider.deliverToSession
	provider.processOutgoing = provider.processOutgoingMessage
	return provider
}

func (p *pubsubProvider) Subscribe(ctx context.Context, topic string, handler func([]byte, map[string]string)) (string, error) {
	if p == nil {
		return "", runtime.ErrPubsubUnavailable
	}
	sess := sessionFromContext(ctx)
	if sess == nil {
		return "", runtime.ErrPubsubUnavailable
	}
	if handler == nil {
		return "", errors.New("server: handler is nil")
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return "", errors.New("server: topic is empty")
	}

	token := uuid.NewString()

	p.mu.Lock()
	p.subscriptions[token] = pubsubSubscription{
		session: sess.ID(),
		topic:   topic,
	}
	if _, ok := p.sessionTopics[sess.ID()]; !ok {
		p.sessionTopics[sess.ID()] = make(map[string]int)
	}
	p.sessionTopics[sess.ID()][topic]++
	p.mu.Unlock()

	return token, nil
}

func (p *pubsubProvider) Unsubscribe(ctx context.Context, token string) error {
	if p == nil {
		return runtime.ErrPubsubUnavailable
	}
	if token == "" {
		return nil
	}

	var sub pubsubSubscription
	var shouldEvict bool

	p.mu.Lock()
	if existing, ok := p.subscriptions[token]; ok {
		sub = existing
		delete(p.subscriptions, token)
		if counts, exists := p.sessionTopics[sub.session]; exists {
			if counts[sub.topic] > 0 {
				counts[sub.topic]--
			}
			if counts[sub.topic] <= 0 {
				delete(counts, sub.topic)
				shouldEvict = true
			}
			if len(counts) == 0 {
				delete(p.sessionTopics, sub.session)
			}
		}
	} else {
		p.mu.Unlock()
		return nil
	}
	p.mu.Unlock()

	if shouldEvict {
		p.evictFromTopic(sub.session, sub.topic)
	}

	return nil
}

func (p *pubsubProvider) Publish(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
	if p == nil {
		return runtime.ErrPubsubUnavailable
	}
	topic = strings.TrimSpace(topic)
	if topic == "" {
		return errors.New("server: topic is empty")
	}

	channelName := fmt.Sprintf("pubsub/%s", topic)
	ch, err := p.endpoint.endpoint.GetChannelByName(channelName)
	if err != nil || ch == nil {
		return runtime.ErrPubsubUnavailable
	}

	envelope := pubsubEnvelope{
		Data: append(json.RawMessage(nil), payload...),
		Meta: copyStringMap(meta),
	}
	return ch.Broadcast("pub", envelope)
}

func (p *pubsubProvider) handleJoin(ctx *pond.JoinContext) error {
	if ctx == nil {
		return nil
	}

	var payload struct {
		SessionID string `json:"sid,omitempty"`
	}
	_ = ctx.ParsePayload(&payload)

	if ctx.Route == nil {
		return ctx.Decline(pond.StatusBadRequest, "missing route context")
	}
	if topic := ctx.Route.Param("topic"); topic == "" {
		return ctx.Decline(pond.StatusBadRequest, "missing pubsub topic")
	}

	assign := ctx.GetAssigns(sessionAssignKey)
	sessionID, _ := assign.(string)
	if sessionID == "" && payload.SessionID != "" {
		sessionID = payload.SessionID
		ctx.SetAssigns(sessionAssignKey, sessionID)
	}
	if sessionID == "" {
		return ctx.Decline(pond.StatusUnauthorized, "session not bound to connection")
	}

	ctx.Accept()
	if errStr := ctx.Error(); errStr != "" {
		return errors.New(errStr)
	}
	return nil
}

func (p *pubsubProvider) handleOutgoing(ctx *pond.OutgoingContext) error {
	if p == nil || ctx == nil || ctx.User == nil {
		return nil
	}

	assignValue, ok := ctx.User.Assigns[sessionAssignKey]
	if !ok {
		return nil
	}
	sessionID, ok := assignValue.(string)
	if !ok || sessionID == "" {
		return nil
	}

	if ctx.Route == nil {
		return nil
	}
	topic := ctx.Route.Param("topic")
	if topic == "" {
		return nil
	}

	if p.registry == nil {
		return nil
	}

	envelope := p.parseOutgoingPayload(ctx)
	if len(envelope.Data) == 0 {
		envelope.Data = json.RawMessage("null")
	}

	process := p.processOutgoing
	if process == nil {
		process = p.processOutgoingMessage
	}

	err := process(session.SessionID(sessionID), ctx.User.UserID, topic, envelope)
	ctx.Block()
	return err
}

func (p *pubsubProvider) processOutgoingMessage(sessionID session.SessionID, connID, topic string, envelope pubsubEnvelope) error {
	if p == nil || sessionID == "" || topic == "" || p.registry == nil {
		return nil
	}

	sess, transport, ok := p.registry.LookupWithConnection(sessionID, connID)
	if sess == nil || !ok {
		return nil
	}

	deliver := p.deliver
	if deliver == nil {
		deliver = p.deliverToSession
	}
	return deliver(sess, transport, topic, envelope)
}

func (p *pubsubProvider) handleLeave(ctx *pond.LeaveContext) {
	_ = ctx
}

func (p *pubsubProvider) deliverToSession(sess pubsubSession, transport Transport, topic string, envelope pubsubEnvelope) error {
	if sess == nil || topic == "" {
		return nil
	}

	payload := append([]byte(nil), envelope.Data...)
	meta := copyStringMap(envelope.Meta)

	sess.DeliverPubsub(topic, payload, meta)

	if err := sess.Flush(); err != nil {
		p.sendPubsubFlushError(sess, transport, err)
		return err
	}
	return nil
}

func (p *pubsubProvider) sendPubsubFlushError(sess pubsubSession, transport Transport, flushErr error) {
	if sess == nil || flushErr == nil {
		return
	}
	if transport == nil && p != nil && p.registry != nil {
		if _, tr, ok := p.registry.ConnectionForSession(sess.ID()); ok {
			transport = tr
		}
	}
	if transport == nil {
		return
	}
	_ = transport.SendServerError(serverError(sess.ID(), "flush_failed", flushErr))
}

func (p *pubsubProvider) parseOutgoingPayload(ctx *pond.OutgoingContext) pubsubEnvelope {
	var envelope pubsubEnvelope
	payload := ctx.GetPayload()
	if payload == nil {
		return envelope
	}

	bytes, err := json.Marshal(payload)
	if err != nil {
		return envelope
	}
	if err := json.Unmarshal(bytes, &envelope); err != nil || len(envelope.Data) == 0 {
		envelope.Data = append(json.RawMessage(nil), bytes...)
	}
	return envelope
}

func copyStringMap(src map[string]string) map[string]string {
	if len(src) == 0 {
		return nil
	}
	dst := make(map[string]string, len(src))
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func (p *pubsubProvider) evictFromTopic(sessionID session.SessionID, topic string) {
	connID, _, ok := p.registry.ConnectionForSession(sessionID)
	if !ok || connID == "" {
		return
	}
	channelName := fmt.Sprintf("pubsub/%s", topic)
	ch, err := p.endpoint.endpoint.GetChannelByName(channelName)
	if err != nil || ch == nil {
		return
	}
	_ = ch.EvictUser(connID, "server:pubsub-unsubscribe")
}

func contextWithSession(ctx context.Context, sess *session.LiveSession) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, sessionContextKey{}, sess)
}

func sessionFromContext(ctx context.Context) *session.LiveSession {
	if ctx == nil {
		return nil
	}
	if sess, ok := ctx.Value(sessionContextKey{}).(*session.LiveSession); ok {
		return sess
	}
	return nil
}

// serverError constructs a protocol error message.
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
