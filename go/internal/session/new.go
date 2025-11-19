package session

import (
	"time"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// New creates a new LiveSession with the given ID, version, root component, and configuration.
func New(id SessionID, version int, app Component, cfg *Config) *LiveSession {
	effectiveConfig := defaultConfig()
	if cfg != nil {
		if cfg.Transport != nil {
			effectiveConfig.Transport = cfg.Transport
		}
		if cfg.TTL > 0 {
			effectiveConfig.TTL = cfg.TTL
		}
		if cfg.Clock != nil {
			effectiveConfig.Clock = cfg.Clock
		}
		effectiveConfig.DevMode = cfg.DevMode
	}

	sess := &LiveSession{
		id:        id,
		version:   version,
		header:    newHeaderState(),
		lifecycle: NewLifecycle(effectiveConfig.Clock, effectiveConfig.TTL),
		transport: effectiveConfig.Transport,
		devMode:   effectiveConfig.DevMode,
		nextSeq:   1,
	}

	wrapped := documentRoot(sess, app)

	sess.component = runtime.NewSession(wrapped, struct{}{})
	sess.configureRuntime(effectiveConfig)

	return sess
}

func defaultConfig() Config {
	return Config{
		TTL:            90 * time.Second,
		Clock:          time.Now,
		DOMGetTimeout:  defaultDOMRequestTimeout,
		DOMCallTimeout: defaultDOMRequestTimeout,
	}
}
