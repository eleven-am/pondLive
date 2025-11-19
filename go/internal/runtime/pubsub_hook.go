package runtime

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

var ErrPubsubUnavailable = errors.New("runtime2: pubsub unavailable")

// PubsubProvider defines the interface for pubsub backends.
// LiveSession provides this via context.
type PubsubProvider interface {
	Subscribe(ctx context.Context, topic string, handler func(payload []byte, meta map[string]string)) (token string, err error)
	Unsubscribe(ctx context.Context, token string) error
	Publish(ctx context.Context, topic string, payload []byte, meta map[string]string) error
}

// PubsubMessage captures a decoded message delivered through UsePubsub.
type PubsubMessage[T any] struct {
	Payload    T
	Meta       map[string]string
	ReceivedAt time.Time
}

type pubsubConfig[T any] struct {
	encode   func(T) ([]byte, error)
	decode   func([]byte) (T, error)
	provider PubsubProvider
}

type PubsubOption[T any] interface{ applyPubsubOption(*pubsubConfig[T]) }

type pubsubOptionFunc[T any] func(*pubsubConfig[T])

func (f pubsubOptionFunc[T]) applyPubsubOption(c *pubsubConfig[T]) { f(c) }

// WithPubsubCodec allows a custom encoder/decoder for the hook.
func WithPubsubCodec[T any](encode func(T) ([]byte, error), decode func([]byte) (T, error)) PubsubOption[T] {
	return pubsubOptionFunc[T](func(c *pubsubConfig[T]) {
		if encode != nil {
			c.encode = encode
		}
		if decode != nil {
			c.decode = decode
		}
	})
}

// WithPubsubProvider overrides the provider used for subscriptions and publishes.
func WithPubsubProvider[T any](provider PubsubProvider) PubsubOption[T] {
	return pubsubOptionFunc[T](func(c *pubsubConfig[T]) {
		c.provider = provider
	})
}

// PubsubHandle exposes helpers returned by UsePubsub.
type PubsubHandle[T any] struct {
	Latest    func() (PubsubMessage[T], bool)
	Messages  func() []PubsubMessage[T]
	Publish   func(context.Context, T) error
	Connected func() bool
}

// UsePubsub subscribes the component to a topic and returns helpers for publishing and retrieving messages.
// The provider can be specified via WithPubsubProvider option or will come from context (provided by LiveSession).
func UsePubsub[T any](ctx Ctx, topic string, opts ...PubsubOption[T]) PubsubHandle[T] {
	if ctx.sess == nil || topic == "" {
		return PubsubHandle[T]{
			Latest:    func() (PubsubMessage[T], bool) { return PubsubMessage[T]{}, false },
			Messages:  func() []PubsubMessage[T] { return nil },
			Publish:   func(context.Context, T) error { return ErrPubsubUnavailable },
			Connected: func() bool { return false },
		}
	}

	config := pubsubConfig[T]{
		encode: func(value T) ([]byte, error) { return json.Marshal(value) },
		decode: func(data []byte) (T, error) {
			var out T
			err := json.Unmarshal(data, &out)
			return out, err
		},
	}

	for _, opt := range opts {
		if opt != nil {
			opt.applyPubsubOption(&config)
		}
	}

	getMessages, setMessages := UseState(ctx, []PubsubMessage[T]{})

	type latestState struct {
		msg PubsubMessage[T]
		ok  bool
	}
	getLatest, setLatest := UseState(ctx, latestState{})

	connectedRef := UseRef(ctx, false)
	tokenRef := UseRef(ctx, "")

	session := ctx.sess

	updateState := func(payload []byte, meta map[string]string) {
		decoded, err := config.decode(payload)
		if err != nil {
			return
		}

		msg := PubsubMessage[T]{
			Payload:    decoded,
			Meta:       cloneStringMap(meta),
			ReceivedAt: time.Now(),
		}

		current := getMessages()
		next := append(append([]PubsubMessage[T](nil), current...), msg)
		setMessages(next)
		setLatest(latestState{msg: msg, ok: true})
	}

	UseEffect(ctx, func() Cleanup {
		token, err := session.subscribePubsub(topic, config.provider, updateState)
		if err != nil {
			return nil
		}

		tokenRef.Cur = token
		connectedRef.Cur = true

		return func() {
			connectedRef.Cur = false
			saved := tokenRef.Cur
			tokenRef.Cur = ""
			if saved != "" {
				_ = session.unsubscribePubsub(saved)
			}
		}
	}, topic)

	return PubsubHandle[T]{
		Latest: func() (PubsubMessage[T], bool) {
			state := getLatest()
			return state.msg, state.ok
		},
		Messages: func() []PubsubMessage[T] {
			cur := getMessages()
			out := make([]PubsubMessage[T], len(cur))
			copy(out, cur)
			return out
		},
		Publish: func(pctx context.Context, value T) error {
			if err := pctx.Err(); err != nil {
				return err
			}

			payload, err := config.encode(value)
			if err != nil {
				return err
			}

			return session.publishPubsub(topic, config.provider, payload, nil)
		},
		Connected: func() bool { return connectedRef.Cur },
	}
}

func cloneStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// ComponentSession pubsub methods

func (s *ComponentSession) subscribePubsub(
	topic string,
	provider PubsubProvider,
	handler func([]byte, map[string]string),
) (string, error) {
	if s == nil || provider == nil {
		return "", ErrPubsubUnavailable
	}

	token, err := provider.Subscribe(context.Background(), topic, handler)
	if err != nil {
		return "", err
	}

	s.pubsubMu.Lock()
	s.pubsubSubs[token] = pubsubSubscription{
		token:    token,
		topic:    topic,
		handler:  handler,
		provider: provider,
	}
	s.pubsubMu.Unlock()

	return token, nil
}

func (s *ComponentSession) unsubscribePubsub(token string) error {
	if s == nil || token == "" {
		return nil
	}

	s.pubsubMu.Lock()
	sub, ok := s.pubsubSubs[token]
	delete(s.pubsubSubs, token)
	s.pubsubMu.Unlock()

	if !ok {
		return nil
	}

	if sub.provider != nil {
		return sub.provider.Unsubscribe(context.Background(), token)
	}

	return nil
}

func (s *ComponentSession) publishPubsub(
	topic string,
	provider PubsubProvider,
	payload []byte,
	meta map[string]string,
) error {
	if s == nil || provider == nil {
		return ErrPubsubUnavailable
	}

	return provider.Publish(context.Background(), topic, payload, meta)
}

// DeliverPubsub delivers a pubsub message to all subscribed handlers for the topic.
// This is called by the pubsub provider when a message arrives.
func (s *ComponentSession) DeliverPubsub(topic string, payload []byte, meta map[string]string) {
	if s == nil {
		return
	}

	s.pubsubMu.RLock()
	if len(s.pubsubSubs) == 0 {
		s.pubsubMu.RUnlock()
		return
	}

	handlers := make([]func([]byte, map[string]string), 0)
	for _, sub := range s.pubsubSubs {
		if sub.topic == topic && sub.handler != nil {
			handlers = append(handlers, sub.handler)
		}
	}
	s.pubsubMu.RUnlock()

	if len(handlers) == 0 {
		return
	}

	for _, h := range handlers {
		handler := h
		payloadCopy := append([]byte(nil), payload...)
		metaCopy := cloneStringMap(meta)

		handler(payloadCopy, metaCopy)
	}
}
