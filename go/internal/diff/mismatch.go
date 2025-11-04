package diff

import "sync"

var (
	mismatchMu      sync.RWMutex
	mismatchHandler func(error)
)

// RegisterMismatchHandler installs a callback invoked whenever a template mismatch is detected.
func RegisterMismatchHandler(fn func(error)) {
	mismatchMu.Lock()
	mismatchHandler = fn
	mismatchMu.Unlock()
}

func callMismatchHandler(err error) {
	mismatchMu.RLock()
	handler := mismatchHandler
	mismatchMu.RUnlock()
	if handler != nil {
		handler(err)
	}
}
