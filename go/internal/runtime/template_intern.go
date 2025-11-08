package runtime

import (
	"strconv"
	"strings"
	"sync"
)

type templateInternCache struct {
	mu      sync.RWMutex
	statics map[string][]string
}

func newTemplateInternCache() *templateInternCache {
	return &templateInternCache{statics: make(map[string][]string)}
}

func (c *templateInternCache) InternStatics(statics []string) []string {
	if len(statics) == 0 {
		return nil
	}
	key := encodeStaticsKey(statics)
	c.mu.RLock()
	interned, ok := c.statics[key]
	c.mu.RUnlock()
	if ok {
		return interned
	}
	clone := append([]string(nil), statics...)
	c.mu.Lock()
	defer c.mu.Unlock()
	if existing, ok := c.statics[key]; ok {
		return existing
	}
	c.statics[key] = clone
	return clone
}

func encodeStaticsKey(statics []string) string {
	if len(statics) == 0 {
		return ""
	}
	var b strings.Builder
	for _, segment := range statics {
		b.WriteString(strconv.Itoa(len(segment)))
		b.WriteByte(':')
		b.WriteString(segment)
		b.WriteByte(';')
	}
	return b.String()
}

var globalTemplateIntern = newTemplateInternCache()
