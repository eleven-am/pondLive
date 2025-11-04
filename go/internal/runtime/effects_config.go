package runtime

import (
	"log"
	"sync"
	"time"
)

var (
	effectWarnMu           sync.RWMutex
	effectWarningThreshold time.Duration
	effectWarningHandler   = defaultEffectWarningHandler
)

func defaultEffectWarningHandler(component string, duration time.Duration) {
	if component == "" {
		log.Printf("liveui: effect exceeded threshold: duration=%s", duration)
		return
	}
	log.Printf("liveui: effect exceeded threshold: component=%s duration=%s", component, duration)
}

// SetEffectWarningThreshold updates the duration after which effect execution
// is considered slow. A non-positive duration disables warnings entirely.
func SetEffectWarningThreshold(d time.Duration) {
	effectWarnMu.Lock()
	effectWarningThreshold = d
	effectWarnMu.Unlock()
}

// SetEffectWarningHandler installs a callback invoked whenever an effect
// exceeds the configured threshold. Passing nil restores the default logger.
func SetEffectWarningHandler(handler func(component string, duration time.Duration)) {
	effectWarnMu.Lock()
	if handler == nil {
		effectWarningHandler = defaultEffectWarningHandler
	} else {
		effectWarningHandler = handler
	}
	effectWarnMu.Unlock()
}

func observeEffectDuration(comp *component, duration time.Duration) bool {
	effectWarnMu.RLock()
	threshold := effectWarningThreshold
	handler := effectWarningHandler
	effectWarnMu.RUnlock()
	if threshold <= 0 || duration <= threshold {
		return false
	}
	var id string
	if comp != nil {
		id = comp.id
	}
	handler(id, duration)
	return true
}
