package meta2

import (
	"maps"

	"github.com/eleven-am/pondlive/go/internal/html2"
	"github.com/eleven-am/pondlive/go/internal/runtime2"
	"github.com/eleven-am/pondlive/go/internal/work"
)

// noopController is the default controller when no Provider exists.
// It silently drops meta changes - this is intentional for SSR or when
// meta is not needed.
var noopController = &Controller{
	get:    func() map[string]metaEntry { return make(map[string]metaEntry) },
	set:    func(string, metaEntry) {},
	remove: func(string) {},
}

var metaCtx = runtime2.CreateContext[*Controller](noopController)

// Provider creates a meta context with reactive state so meta changes rerender <head>.
func Provider(ctx *runtime2.Ctx, assetURL string, children []work.Node) work.Node {
	entries, setEntries := runtime2.UseState(ctx, map[string]metaEntry{})

	entriesRef := runtime2.UseRef(ctx, entries)
	entriesRef.Current = entries

	controller := runtime2.UseMemo(ctx, func() *Controller {
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
	scriptNodes := html2.ScriptTags(html2.ScriptTag{
		Src:   assetURL,
		Defer: true,
	})

	bodyChildren := make([]html2.Item, 0, 1+len(scriptNodes))
	for _, child := range children {
		if child != nil {
			bodyChildren = append(bodyChildren, child)
		}
	}

	for _, script := range scriptNodes {
		bodyChildren = append(bodyChildren, script)
	}

	return html2.Html(
		Head(ctx),
		html2.Body(bodyChildren...),
	)
}
