package pondsocket

import (
	"context"

	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/session"
)

type sessionPubsubAdapter struct {
	session  *session.LiveSession
	provider runtime.PubsubProvider
}

// WrapSessionPubsubProvider returns a runtime2.PubsubProvider that binds
// the given LiveSession to the shared PondSocket pubsub provider.
func WrapSessionPubsubProvider(sess *session.LiveSession, provider runtime.PubsubProvider) runtime.PubsubProvider {
	if sess == nil || provider == nil {
		return nil
	}
	return &sessionPubsubAdapter{
		session:  sess,
		provider: provider,
	}
}

func (a *sessionPubsubAdapter) Subscribe(ctx context.Context, topic string, handler func([]byte, map[string]string)) (string, error) {
	if a == nil || a.provider == nil || a.session == nil {
		return "", runtime.ErrPubsubUnavailable
	}
	ctx = contextWithSession(ctx, a.session)
	return a.provider.Subscribe(ctx, topic, handler)
}

func (a *sessionPubsubAdapter) Unsubscribe(ctx context.Context, token string) error {
	if a == nil || a.provider == nil || a.session == nil {
		return runtime.ErrPubsubUnavailable
	}
	ctx = contextWithSession(ctx, a.session)
	return a.provider.Unsubscribe(ctx, token)
}

func (a *sessionPubsubAdapter) Publish(ctx context.Context, topic string, payload []byte, meta map[string]string) error {
	if a == nil || a.provider == nil || a.session == nil {
		return runtime.ErrPubsubUnavailable
	}
	ctx = contextWithSession(ctx, a.session)
	return a.provider.Publish(ctx, topic, payload, meta)
}
