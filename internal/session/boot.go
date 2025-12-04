package session

import (
	"github.com/eleven-am/pondlive/internal/document"
	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/metatags"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/styles"
	"github.com/eleven-am/pondlive/internal/work"
)

type bootProps struct {
	requestState *headers.RequestState
	component    Component
	ClientAsset  string
}

func wrapComponent(component Component) func(*runtime.Ctx, any, []work.Node) work.Node {
	return func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
		return component(ctx)
	}
}

func bootComponent(ctx *runtime.Ctx, props bootProps, _ []work.Node) work.Node {
	app := wrapComponent(props.component)

	return headers.Provider(ctx, props.requestState,
		metatags.Provider(ctx,
			router.Provide(ctx,
				styles.Provider(ctx,
					document.Provider(ctx,
						document.HtmlElement(ctx,
							&work.Element{
								Tag: "head",
								Children: []work.Node{
									metatags.Render(ctx),
									styles.Render(ctx),
									headers.Render(ctx),
								},
							},
							document.BodyElement(ctx,
								work.Component(app),
								&work.Element{
									Tag: "script",
									Attrs: map[string][]string{
										"src":   {props.ClientAsset},
										"defer": {""},
									},
								},
							),
						),
					),
				),
			),
		),
	)
}

func loadBootComponent(liveSession *LiveSession, component Component, clientAsset string) func(*runtime.Ctx, any, []work.Node) work.Node {
	return func(ctx *runtime.Ctx, _ any, children []work.Node) work.Node {
		var requestState *headers.RequestState
		if liveSession != nil {
			liveSession.transportMu.RLock()
			t := liveSession.transport
			liveSession.transportMu.RUnlock()

			if t != nil {
				requestState = t.RequestState()
			}
		}

		if requestState == nil {
			requestState = headers.NewRequestState(nil)
		}

		boot := bootProps{
			requestState: requestState,
			component:    component,
			ClientAsset:  clientAsset,
		}

		return bootComponent(ctx, boot, children)
	}
}
