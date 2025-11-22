package meta

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

var metaCtx = runtime.CreateContext[*Controller](&Controller{
	get:    func() map[string]metaEntry { return make(map[string]metaEntry) },
	set:    func(string, metaEntry) {},
	remove: func(string) {},
})

// Provider creates a meta context with state management.
// It uses UseRef to store meta entries that can be mutated during render.
func Provider[P any](ctx runtime.Ctx, asserUrl string, component runtime.Component[P], props P) *dom.StructuredNode {
	entriesRef := runtime.UseRef(ctx, make(map[string]metaEntry))
	manager := headers.UseHeadersManager(ctx)

	controllerRef := runtime.UseRef(ctx, &Controller{
		get: func() map[string]metaEntry {
			return entriesRef.Cur
		},
		set: func(componentID string, entry metaEntry) {
			entriesRef.Cur[componentID] = entry
		},
		remove: func(componentID string) {
			delete(entriesRef.Cur, componentID)
		},
	})

	controller := controllerRef.Cur

	return metaCtx.Provide(ctx, controller, func(ctx runtime.Ctx) *dom.StructuredNode {
		scriptNodes := html.ScriptTags(html.ScriptTag{
			Src:   asserUrl,
			Defer: true,
		})

		body := make([]dom.Item, 0)
		body = append(body, component(ctx, props))
		for _, script := range scriptNodes {
			body = append(body, script)
		}

		bodyEl := dom.El(html.HTMLBodyElement{}, body...)
		if manager != nil {
			manager.AttachTo(bodyEl)
		}

		return dom.El(html.HTMLHtmlElement{},
			Head(ctx),
			bodyEl,
		)
	})
}
