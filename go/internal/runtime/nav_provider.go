package runtime

import (
	"sync"
)

// NavUpdate captures pending navigation instructions for a session.
type NavUpdate struct {
	Push    string
	Replace string
	Back    bool
}

// Empty reports whether the update carries any navigation changes.
func (u NavUpdate) Empty() bool {
	return u.Push == "" && u.Replace == "" && !u.Back
}

var (
	navMu      sync.RWMutex
	navHandler func(*ComponentSession) NavUpdate
)

// RegisterNavProvider installs the callback used to drain pending navigation updates
// from component sessions. The provider may be nil to disable navigation tracking.
func RegisterNavProvider(fn func(*ComponentSession) NavUpdate) {
	navMu.Lock()
	navHandler = fn
	navMu.Unlock()
}

func drainNavUpdate(sess *ComponentSession) NavUpdate {
	if sess == nil {
		return NavUpdate{}
	}
	navMu.RLock()
	fn := navHandler
	navMu.RUnlock()
	if fn == nil {
		return NavUpdate{}
	}
	return fn(sess)
}
