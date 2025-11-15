package router2

import (
	"reflect"
	"runtime"
	"strings"
	"sync"
)

type routeCache struct {
	mu      sync.RWMutex
	entries map[string]*RouteTree
}

var globalRouteCache = &routeCache{entries: make(map[string]*RouteTree)}

func (c *routeCache) getOrCompile(key string, base string, defs ...*RouteDef) *RouteTree {
	c.mu.RLock()
	if tree, ok := c.entries[key]; ok {
		c.mu.RUnlock()
		return tree
	}
	c.mu.RUnlock()
	c.mu.Lock()
	defer c.mu.Unlock()
	if tree, ok := c.entries[key]; ok {
		return tree
	}
	compiled := CompileTree(base, defs...)
	c.entries[key] = compiled
	return compiled
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
