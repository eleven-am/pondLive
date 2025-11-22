package meta

import (
	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

var metaCtx = runtime.CreateContext[*Controller](&Controller{
	get: func() *Meta { return defaultMeta },
	set: func(*Meta) {},
})

// Provider creates a meta context with state management.
// It uses UseState to create reactive meta that triggers re-renders when updated.
func Provider[P any](ctx runtime.Ctx, asserUrl string, component runtime.Component[P], props P) *dom.StructuredNode {
	current, setCurrent := runtime.UseState(ctx, defaultMeta)
	manager := headers.UseHeadersManager(ctx)

	controller := &Controller{
		get: current,
		set: setCurrent,
	}

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
