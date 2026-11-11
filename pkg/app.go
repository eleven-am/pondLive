package pkg

import (
	"context"
	"net/http"
	"time"

	"github.com/eleven-am/pondlive/internal/server"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/upload"
)

type App = server.App

type UploadConfig = upload.Config

type HTTPConfig = server.HTTPConfig

type WatchdogConfig = server.WatchdogConfig

type Stats = server.Stats

type StateCounts = server.StateCounts

type RejectedCounts = server.RejectedCounts

type SessionID = session.SessionID

type appConfig struct {
	clientAsset   string
	sessionConfig *session.Config
	idGenerator   func(*http.Request) (session.SessionID, error)
	ctx           context.Context
	uploadConfig  *upload.Config
	transport     server.TransportConfig

	retainedHeaders []string
	maxSessions     int
	maxPending      int
	sessionLifetime time.Duration
	http            server.HTTPConfig
	watchdog        server.WatchdogConfig
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

func WithHandlerQueueCap(n int) AppOption {
	return func(c *appConfig) {
		if c.sessionConfig == nil {
			c.sessionConfig = &session.Config{}
		}
		c.sessionConfig.HandlerQueueCap = n
	}
}

func WithRenderTimeout(d time.Duration) AppOption {
	return func(c *appConfig) {
		if c.sessionConfig == nil {
			c.sessionConfig = &session.Config{}
		}
		c.sessionConfig.RenderTimeout = d
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

func WithAllowedOrigins(origins ...string) AppOption {
	return func(c *appConfig) {
		c.transport.AllowedOrigins = append([]string(nil), origins...)
	}
}

func WithAttachSecret(secret []byte) AppOption {
	return func(c *appConfig) {
		c.transport.AttachSecret = append([]byte(nil), secret...)
	}
}

func WithAttachTokenTTL(ttl time.Duration) AppOption {
	return func(c *appConfig) {
		c.transport.TokenTTL = ttl
	}
}

func WithReadLimit(bytes int64) AppOption {
	return func(c *appConfig) {
		c.transport.ReadLimit = bytes
	}
}

func WithOutboundBudget(bytes int) AppOption {
	return func(c *appConfig) {
		c.transport.OutboundBudget = bytes
	}
}

func WithPingInterval(interval time.Duration) AppOption {
	return func(c *appConfig) {
		c.transport.PingInterval = interval
	}
}

func WithUploadConfig(cfg UploadConfig) AppOption {
	return func(c *appConfig) {
		c.uploadConfig = &cfg
	}
}

func WithRetainedHeaders(names ...string) AppOption {
	return func(c *appConfig) {
		c.retainedHeaders = append([]string(nil), names...)
	}
}

func WithMaxSessions(n int) AppOption {
	return func(c *appConfig) {
		c.maxSessions = n
	}
}

func WithMaxPendingSessions(n int) AppOption {
	return func(c *appConfig) {
		c.maxPending = n
	}
}

func WithSessionLifetime(d time.Duration) AppOption {
	return func(c *appConfig) {
		c.sessionLifetime = d
	}
}

func WithHTTPConfig(cfg HTTPConfig) AppOption {
	return func(c *appConfig) {
		c.http = cfg
	}
}

func WithWatchdog(cfg WatchdogConfig) AppOption {
	return func(c *appConfig) {
		c.watchdog = cfg
	}
}

func NewApp(root func() Node, opts ...AppOption) (*App, error) {
	cfg := &appConfig{}

	for _, opt := range opts {
		opt(cfg)
	}

	serverCfg := server.Config{
		Component:     func(*Ctx) Node { return root() },
		ClientAsset:   cfg.clientAsset,
		SessionConfig: cfg.sessionConfig,
		IDGenerator:   cfg.idGenerator,
		Context:       cfg.ctx,
		UploadConfig:  cfg.uploadConfig,
		Transport:     cfg.transport,

		RetainedHeaders:    cfg.retainedHeaders,
		MaxSessions:        cfg.maxSessions,
		MaxPendingSessions: cfg.maxPending,
		SessionLifetime:    cfg.sessionLifetime,
		HTTP:               cfg.http,
		Watchdog:           cfg.watchdog,
	}

	return server.New(serverCfg)
}
