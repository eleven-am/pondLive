package router

import (
	"net/http"
	"net/url"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func Navigate(ctx *runtime.Ctx, href string) {
	navigate(ctx, href, false)
}

func Replace(ctx *runtime.Ctx, href string) {
	navigate(ctx, href, true)
}

func NavigateWithQuery(ctx *runtime.Ctx, path string, query url.Values) {
	href := buildHref(path, query, "")
	Navigate(ctx, href)
}

func ReplaceWithQuery(ctx *runtime.Ctx, path string, query url.Values) {
	href := buildHref(path, query, "")
	Replace(ctx, href)
}

func NavigateToHash(ctx *runtime.Ctx, hash string) {
	Navigate(ctx, "#"+hash)
}

func Back(ctx *runtime.Ctx) {
	bus := getBus(ctx)
	if bus == nil {
		return
	}
	bus.PublishRouterBack()
}

func Forward(ctx *runtime.Ctx) {
	bus := getBus(ctx)
	if bus == nil {
		return
	}
	bus.PublishRouterForward()
}

func navigate(ctx *runtime.Ctx, href string, replace bool) {
	requestState := headers.UseRequestState(ctx)
	currentLoc, setLocation := LocationContext.UseContext(ctx)
	bus := getBus(ctx)

	if bus == nil || requestState == nil || !requestState.IsLive() {
		if requestState != nil {
			currentLoc := &Location{
				Path:  requestState.Path(),
				Query: requestState.Query(),
				Hash:  requestState.Hash(),
			}

			target := resolveHref(currentLoc, href)
			redirectURL := buildHref(target.Path, target.Query, target.Hash)
			status := http.StatusFound
			if replace {
				status = http.StatusSeeOther
			}
			requestState.SetRedirect(redirectURL, status)
		}

		return
	}

	if currentLoc == nil {
		currentLoc = &Location{Path: "/", Query: url.Values{}}
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
