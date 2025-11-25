package session2

import (
	"time"
)

// SessionID uniquely identifies a client session.
type SessionID string

// Config captures the optional configuration for a LiveSession.
type Config struct {
	// TTL is the session time-to-live. Sessions expire after this duration of inactivity.
	TTL time.Duration

	// Clock provides the current time. Defaults to time.Now.
	Clock func() time.Time

	// DevMode enables diagnostic reporting and verbose errors.
	DevMode bool

	// ClientAsset is the versioned JS bundle path for cache busting.
	ClientAsset string

	// DOMTimeout is the timeout for blocking DOM operations (Query, AsyncCall).
	// Defaults to 5 seconds if not set.
	DOMTimeout time.Duration
}

// DefaultConfig returns the default session configuration.
func DefaultConfig() Config {
	return Config{
		TTL:   90 * time.Second,
		Clock: time.Now,
	}
}
