package session

import (
	"time"
)

func defaultConfig() Config {
	return Config{
		TTL:            90 * time.Second,
		Clock:          time.Now,
		DOMGetTimeout:  defaultDOMRequestTimeout,
		DOMCallTimeout: defaultDOMRequestTimeout,
	}
}
