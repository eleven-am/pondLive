package session

import (
	"log/slog"
	"time"
)

type SessionID string

const (
	DefaultMaxLifetime    = 24 * time.Hour
	DefaultResyncInterval = time.Second
	resyncBurst           = 2
)

type Config struct {
	DevMode bool

	ClientAsset string

	DOMTimeout time.Duration

	Logger *slog.Logger

	UploadEndpoint string

	UploadScriptURL string

	PendingTTL time.Duration

	DisconnectTTL time.Duration

	RenderTimeout time.Duration

	HandlerQueueCap int

	MaxLifetime time.Duration

	ReplayFrames int

	ReplayBytes int

	ResetLimit int

	ResyncInterval time.Duration

	OnReset func(*LiveSession, string)

	RetainedHeaders []string

	Clock Clock

	OnClose func(*LiveSession, CloseReason)

	OnLeavePending func(*LiveSession)
}

func DefaultConfig() Config {
	return Config{}
}
