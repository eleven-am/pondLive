package router

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
)

type routeCache struct {
	mu      sync.RWMutex
	entries map[string]*RouteTree
	hits    atomic.Uint64
	misses  atomic.Uint64
}

var globalRouteCache = &routeCache{entries: make(map[string]*RouteTree)}

func (c *routeCache) getOrCompile(key string, base string, defs ...*RouteDef) *RouteTree {
	c.mu.RLock()
	if tree, ok := c.entries[key]; ok {
		c.hits.Add(1)
		c.mu.RUnlock()
		return tree
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if tree, ok := c.entries[key]; ok {
		c.hits.Add(1)
		return tree
	}
	compiled := CompileTree(base, defs...)
	c.entries[key] = compiled
	c.misses.Add(1)
	return compiled
}

// CacheStats reports cache hit/miss totals for observability.
type CacheStats struct {
	Hits   uint64
	Misses uint64
}

func (c *routeCache) stats() CacheStats {
	if c == nil {
		return CacheStats{}
	}
	return CacheStats{Hits: c.hits.Load(), Misses: c.misses.Load()}
}

// RouteCacheStats exposes cache stats for diagnostics and tests.
func RouteCacheStats() CacheStats {
	return globalRouteCache.stats()
}

func routeDefsKey(base string, defs []*RouteDef) string {
	var builder strings.Builder
	builder.WriteString(base)
	for _, def := range defs {
		writeRouteDef(&builder, def)
	}
	return builder.String()
}

func writeRouteDef(b *strings.Builder, def *RouteDef) {
	if def == nil {
		b.WriteString("nil;")
		return
	}
	b.WriteString("pattern:")
	b.WriteString(def.Pattern)
	b.WriteByte(';')
	if def.Identity != "" {
		b.WriteString("identity:")
		b.WriteString(def.Identity)
		b.WriteByte(';')
	}
	b.WriteString("render:")
	if def.Render != nil {
		ptr := runtime.FuncForPC(reflect.ValueOf(def.Render).Pointer())
		if ptr != nil {
			b.WriteString(ptr.Name())
		}
	}
	b.WriteByte(';')
	for _, child := range def.Children {
		writeRouteDef(b, child)
	}
	b.WriteByte('|')
}

func compileRoutesCached(base string, defs ...*RouteDef) *RouteTree {
	key := routeDefsKey(base, defs)
	return globalRouteCache.getOrCompile(key, base, defs...)
}
