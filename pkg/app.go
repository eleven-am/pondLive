package pkg

import (
	"context"
	"net/http"
	"time"

	"github.com/eleven-am/pondlive/internal/server"
	"github.com/eleven-am/pondlive/internal/session"
	pond "github.com/eleven-am/pondsocket/go/pondsocket"
)

type App = server.App

type appConfig struct {
	clientAsset   string
	sessionConfig *session.Config
	idGenerator   func(*http.Request) (session.SessionID, error)
	ctx           context.Context
	pubsub        pond.PubSub
}

type AppOption func(*appConfig)

func WithDevMode() AppOption {
	return func(c *appConfig) {
		if c.sessionConfig == nil {
			c.sessionConfig = &session.Config{}
		}
		c.sessionConfig.DevMode = true
	}
}

func WithDOMTimeout(timeout time.Duration) AppOption {
	return func(c *appConfig) {
		if c.sessionConfig == nil {
			c.sessionConfig = &session.Config{}
		}
		c.sessionConfig.DOMTimeout = timeout
	}
}

func WithIDGenerator(gen func(*http.Request) (session.SessionID, error)) AppOption {
	return func(c *appConfig) {
		c.idGenerator = gen
	}
}

func WithContext(ctx context.Context) AppOption {
	return func(c *appConfig) {
		c.ctx = ctx
	}
}

func WithPubSub(pubsub PubSub) AppOption {
	return func(c *appConfig) {
		c.pubsub = pubsub
	}
}

func NewApp(component func(*Ctx) Node, opts ...AppOption) (*App, error) {
	cfg := &appConfig{}

	for _, opt := range opts {
		opt(cfg)
	}

	serverCfg := server.Config{
		Component:     component,
		ClientAsset:   cfg.clientAsset,
		SessionConfig: cfg.sessionConfig,
		IDGenerator:   cfg.idGenerator,
		Context:       cfg.ctx,
		PubSub:        cfg.pubsub,
	}

	return server.New(serverCfg)
}
