package session

import (
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/metatags"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/styles"
	"github.com/eleven-am/pondlive/go/internal/work"
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

func bootComponent(ctx *runtime.Ctx, props bootProps, children []work.Node) work.Node {
	app := wrapComponent(props.component)

	return headers.Provider(ctx, props.requestState,
		metatags.Provider(ctx,
			router.ProvideRouter(ctx,
				styles.Provider(ctx,
					&work.Element{
						Tag: "html",
						Children: []work.Node{
							&work.Element{
								Tag: "head",
								Children: []work.Node{
									metatags.Render(ctx),
									styles.Render(ctx),
									headers.Render(ctx),
								},
							},
							&work.Element{
								Tag: "body",
								Children: []work.Node{
									work.Component(app),
									&work.Element{
										Tag: "script",
										Attrs: map[string][]string{
											"src":   {props.ClientAsset},
											"defer": {""},
										},
									},
								},
							},
						},
					},
				),
			),
		),
	)
}

func loadBootComponent(liveSession *LiveSession, component Component, clientAsset string) func(*runtime.Ctx, any, []work.Node) work.Node {
	var requestInfo *headers.RequestInfo
	if liveSession != nil {
		liveSession.transportMu.RLock()
		t := liveSession.transport
		liveSession.transportMu.RUnlock()

		if t != nil {
			requestInfo = t.RequestInfo()
		}
	}

	return func(ctx *runtime.Ctx, _ any, children []work.Node) work.Node {
		boot := bootProps{
			requestState: headers.NewRequestState(requestInfo),
			component:    component,
			ClientAsset:  clientAsset,
		}

		return bootComponent(ctx, boot, children)
	}
}
