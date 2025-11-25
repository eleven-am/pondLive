package session

import (
	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/metatags"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/work"
)

type bootProps struct {
	requestState *headers.RequestState
	component    Component
	ClientAsset  string
}

// wrapComponent wraps a session.Component into a runtime.ComponentNode.
func wrapComponent(component Component) runtime.ComponentNode[struct{}] {
	return func(ctx *runtime.Ctx, props struct{}, children []work.Node) work.Node {
		return component(ctx)
	}
}

// bootComponent is the root component that sets up all providers:
func bootComponent(ctx *html.Ctx, props bootProps, children []work.Node) work.Node {
	app := wrapComponent(props.component)
	headers.UseProvideRequestState(ctx, props.requestState)
	appNode := app(ctx, struct{}{}, children)
	routerWrapped := router.ProvideRouter(ctx, []work.Node{appNode})
	return metatags.Provider(ctx, props.ClientAsset, []work.Node{routerWrapped})
}

// loadBootComponent loads the boot component with the given props.
func loadBootComponent(liveSession *LiveSession, component Component, clientAsset string) func(*runtime.Ctx, any, []work.Node) work.Node {
	var requestInfo *headers.RequestInfo
	if liveSession != nil {
		liveSession.mu.Lock()
		t := liveSession.transport
		liveSession.mu.Unlock()

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
