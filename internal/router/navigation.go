package router

import (
	"net/http"
	"net/url"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
)

func Navigate(ctx *runtime.Ctx, href string) {
	navigate(ctx, href, false)
}

func Replace(ctx *runtime.Ctx, href string) {
	navigate(ctx, href, true)
}

func NavigateWith(ctx *runtime.Ctx, fn func(Location) Location) {
	currentLoc := UseLocation(ctx)
	cloned := cloneLocation(currentLoc)
	target := fn(cloned)
	href := buildHref(target.Path, target.Query, target.Hash)
	Navigate(ctx, href)
}

func ReplaceWith(ctx *runtime.Ctx, fn func(Location) Location) {
	currentLoc := UseLocation(ctx)
	cloned := cloneLocation(currentLoc)
	target := fn(cloned)
	href := buildHref(target.Path, target.Query, target.Hash)
	Replace(ctx, href)
}

func NavigateToHash(ctx *runtime.Ctx, hash string) {
	Navigate(ctx, "#"+hash)
}

func Back(ctx *runtime.Ctx) {
	bus := runtime.GetBus(ctx)
	if bus == nil {
		return
	}
	bus.PublishRouterBack()
}

func Forward(ctx *runtime.Ctx) {
	bus := runtime.GetBus(ctx)
	if bus == nil {
		return
	}
	bus.PublishRouterForward()
}

func navigate(ctx *runtime.Ctx, href string, replace bool) {
	requestState := headers.UseRequestState(ctx)
	currentLoc, setLocation := locationCtx.UseContext(ctx)
	bus := runtime.GetBus(ctx)

	if bus == nil || requestState == nil || !requestState.IsLive() {
		if requestState != nil {
			current := Location{
				Path:  requestState.Path(),
				Query: requestState.Query(),
				Hash:  requestState.Hash(),
			}

			target := resolveHref(current, href)
			redirectURL := buildHref(target.Path, target.Query, target.Hash)
			status := http.StatusFound
			if replace {
				status = http.StatusSeeOther
			}
			requestState.SetRedirect(redirectURL, status)
		}

		return
	}

	if currentLoc.Path == "" {
		currentLoc = Location{Path: "/", Query: url.Values{}}
	}

	target := resolveHref(currentLoc, href)
	target = canonicalizeLocation(target)

	if setLocation != nil {
		setLocation(target)
	}

	payload := protocol.RouterNavPayload{
		Path:    target.Path,
		Query:   target.Query.Encode(),
		Hash:    target.Hash,
		Replace: replace,
	}

	if replace {
		bus.PublishRouterReplace(payload)
	} else {
		bus.PublishRouterPush(payload)
	}
}
