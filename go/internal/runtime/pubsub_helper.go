package runtime

import (
	"context"
	"encoding/json"
	"strings"
	"sync/atomic"
)

// PubsubPublisher fan-outs serialized payloads to topic subscribers.
type PubsubPublisher interface {
	Publish(ctx context.Context, topic string, payload []byte, meta map[string]string) error
}

// PubsubPublishFunc adapts a function to the PubsubPublisher interface.
type PubsubPublishFunc func(ctx context.Context, topic string, payload []byte, meta map[string]string) error

// Publish implements PubsubPublisher.
func (f PubsubPublishFunc) Publish(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
	if f == nil {
		return ErrPubsubUnavailable
	}
	return f(ctx, topic, payload, meta)
}

// Pubsub couples hook integration with an optional backend publisher for a topic.
type Pubsub[T any] struct {
	topic     string
	encode    func(T) ([]byte, error)
	decode    func([]byte) (T, error)
	publisher PubsubPublisher
	options   []PubsubOption[T]
}

type pubsubPublisherHolder struct {
	publisher PubsubPublisher
}

var defaultPubsubPublisher atomic.Pointer[pubsubPublisherHolder]

// SetDefaultPubsubPublisher registers a global fallback publisher used when helpers are constructed without an explicit backend.
func SetDefaultPubsubPublisher(p PubsubPublisher) {
	if p == nil {
		defaultPubsubPublisher.Store(nil)
		return
	}
	defaultPubsubPublisher.Store(&pubsubPublisherHolder{publisher: p})
}

func loadDefaultPubsubPublisher() PubsubPublisher {
	holder := defaultPubsubPublisher.Load()
	if holder == nil {
		return nil
	}
	return holder.publisher
}

// NewPubsub constructs a helper that wires UsePubsub calls and typed publish convenience.
func NewPubsub[T any](topic string, publisher PubsubPublisher, opts ...PubsubOption[T]) *Pubsub[T] {
	trimmed := strings.TrimSpace(topic)
	helper := &Pubsub[T]{
		topic:  trimmed,
		encode: func(value T) ([]byte, error) { return json.Marshal(value) },
		decode: func(data []byte) (T, error) {
			var out T
			err := json.Unmarshal(data, &out)
			return out, err
		},
		publisher: publisher,
	}

	config := pubsubConfig[T]{
		encode: helper.encode,
		decode: helper.decode,
	}
	for _, opt := range opts {
		if opt != nil {
			opt.applyPubsubOption(&config)
		}
	}
	helper.encode = config.encode
	helper.decode = config.decode
	if len(opts) > 0 {
		helper.options = append(helper.options, opts...)
	}

	return helper
}

// Use subscribes within a component render using the configured topic and options.
func (p *Pubsub[T]) Use(ctx Ctx) PubsubHandle[T] {
	if p == nil {
		return UsePubsub[T](ctx, "")
	}
	return UsePubsub(ctx, p.topic, append([]PubsubOption[T](nil), p.options...)...)
}

// Publish broadcasts a typed payload through the configured publisher.
func (p *Pubsub[T]) Publish(ctx context.Context, value T, meta map[string]string) error {
	if p == nil {
		return ErrPubsubUnavailable
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	publisher := p.publisher
	if publisher == nil {
		publisher = loadDefaultPubsubPublisher()
	}
	if publisher == nil {
		return ErrPubsubUnavailable
	}
	payload, err := p.encode(value)
	if err != nil {
		return err
	}
	data := append([]byte(nil), payload...)
	metaCopy := cloneStringMap(meta)
	if err := publisher.Publish(ctx, p.topic, data, metaCopy); err != nil {
		return err
	}
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
		return nil
	}
}
