package metatags

import (
	"maps"

	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type Ctx = runtime.Ctx

// noopController is the default controller when no Provider exists.
// It silently drops meta changes - this is intentional for SSR or when
// meta is not needed.
var noopController = &Controller{
	get:    func() map[string]metaEntry { return make(map[string]metaEntry) },
	set:    func(string, metaEntry) {},
	remove: func(string) {},
}

var metaCtx = runtime.CreateContext[*Controller](noopController)

var Provider = html.Component(func(ctx *Ctx, children []work.Node) work.Node {
	entries, setEntries := runtime.UseState(ctx, map[string]metaEntry{})

	entriesRef := runtime.UseRef(ctx, entries)
	entriesRef.Current = entries

	controller := runtime.UseMemo(ctx, func() *Controller {
		return &Controller{
			get: func() map[string]metaEntry {
				return entriesRef.Current
			},
			set: func(componentID string, entry metaEntry) {
				next := maps.Clone(entriesRef.Current)
				next[componentID] = entry
				setEntries(next)
			},
			remove: func(componentID string) {
				next := maps.Clone(entriesRef.Current)
				delete(next, componentID)
				setEntries(next)
			},
		}
	})

	metaCtx.UseProvider(ctx, controller)

	return &work.Fragment{Children: children}
})
