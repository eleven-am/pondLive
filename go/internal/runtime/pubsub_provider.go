package runtime

import (
	"context"
	"errors"
	"strings"
)

// ErrPubsubUnavailable indicates that pub/sub functionality has not been configured.
var ErrPubsubUnavailable = errors.New("runtime: pubsub provider unavailable")

// PubsubHandler processes messages delivered to a subscription.
type PubsubHandler func(topic string, payload []byte, meta map[string]string)

// PubsubProvider integrates the runtime with an external pub/sub transport.
type PubsubProvider interface {
	// Subscribe registers the provided live session for the given topic. The provider
	// must invoke handler for every message fan-out targeting the topic.
	Subscribe(session *LiveSession, topic string, handler PubsubHandler) (token string, err error)
	// Unsubscribe removes a previously registered subscription token.
	Unsubscribe(session *LiveSession, token string) error
	// Publish fan-outs the payload to all subscribers of the topic. The meta map is optional
	// and may be omitted when the transport does not carry metadata.
	Publish(topic string, payload []byte, meta map[string]string) error
}

// WrapPubsubProvider bridges a provider into a context-aware publish function.
func WrapPubsubProvider(provider PubsubProvider) func(context.Context, string, []byte, map[string]string) error {
	return func(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
		if provider == nil {
			return ErrPubsubUnavailable
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		data := append([]byte(nil), payload...)
		metaCopy := cloneStringMap(meta)
		if err := provider.Publish(strings.TrimSpace(topic), data, metaCopy); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			return nil
		}
	}
}
