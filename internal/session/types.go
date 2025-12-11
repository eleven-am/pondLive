package session

import (
	"time"
)

type SessionID string

type Config struct {
	DevMode bool

	ClientAsset string

	DOMTimeout time.Duration
}

func DefaultConfig() Config {
	return Config{}
}
