package runtime

import (
	"context"
	"encoding/json"
	"time"
)

// PubsubMessage captures a decoded message delivered through usePubsub.
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

// WithPubsubProvider overrides the provider used for subscriptions and publishes originating from this hook.
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
		token, err := session.subscribePubsub(topic, config.provider, func(payload []byte, meta map[string]string) {
			updateState(payload, meta)
		})
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
		Publish: func(ctx context.Context, value T) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			payload, err := config.encode(value)
			if err != nil {
				return err
			}
			if err := session.publishPubsub(topic, config.provider, payload, nil); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
				return nil
			}
		},
		Connected: func() bool { return connectedRef.Cur },
	}
}
