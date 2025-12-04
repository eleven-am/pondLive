package session

import (
	"time"
)

type SessionID string

type Config struct {
	TTL time.Duration

	Clock func() time.Time

	DevMode bool

	ClientAsset string

	DOMTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{
		TTL:   90 * time.Second,
		Clock: time.Now,
	}
}
