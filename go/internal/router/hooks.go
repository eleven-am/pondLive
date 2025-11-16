package router

import (
	"net/url"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"

	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
)

func requireStore(ctx runtime.Ctx) *RouterStore {
	store := routerStoreCtx.Use(ctx)
	if store != nil {
		return store
	}

	sess := ctx.Session()
	if sess != nil {

		loc := runtime.InternalCurrentLocation(sess)
		if loc.Path != "" {

			tempStore := NewStore(fromRuntimeLocation(loc))
			return tempStore
		}
	}
	panic("router: missing Router context; wrap component in router.Router")
}

// UseLocation returns the current router location.
func UseLocation(ctx runtime.Ctx) Location {
	return requireStore(ctx).Location()
}

// UseParams returns the active route params.
func UseParams(ctx runtime.Ctx) map[string]string {
	return requireStore(ctx).Params()
}

// UseParam returns a single route parameter by key.
func UseParam(ctx runtime.Ctx, key string) string {
	return UseParams(ctx)[key]
}

// UseSearch returns a copy of the query parameters.
func UseSearch(ctx runtime.Ctx) url.Values {
	loc := UseLocation(ctx)
	return cloneValues(loc.Query)
}

// UseSearchParam returns getter/setter functions for a specific query key.
func UseSearchParam(ctx runtime.Ctx, key string) (func() []string, func([]string)) {
	store := requireStore(ctx)
	getter := func() []string {
		loc := store.Location()
		values := loc.Query[key]
		if len(values) == 0 {
			return nil
		}
		out := make([]string, len(values))
		copy(out, values)
		return out
	}
	setter := func(values []string) {
		loc := store.Location()
		next := loc
		next.Query = setSearchParam(loc.Query, key, values)
		store.RecordNavigation(NavKindReplace, next)
	}
	return getter, setter
}

// Navigate pushes a new href onto the history stack.
func Navigate(ctx runtime.Ctx, href string) {
	store := requireStore(ctx)
	target := resolveHref(store.Location(), href)
	store.RecordNavigation(NavKindPush, target)
}

// Replace swaps the current history entry with the provided href.
func Replace(ctx runtime.Ctx, href string) {
	store := requireStore(ctx)
	target := resolveHref(store.Location(), href)
	store.RecordNavigation(NavKindReplace, target)
}

// NavigateWithSearch applies a patch to the query parameters and pushes the result.
func NavigateWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
	store := requireStore(ctx)
	loc := store.Location()
	query := cloneValues(loc.Query)
	if patch != nil {
		query = patch(query)
	}
	loc.Query = canonicalizeValues(query)
	store.RecordNavigation(NavKindPush, loc)
}

// ReplaceWithSearch applies a patch to the query parameters and replaces the result.
func ReplaceWithSearch(ctx runtime.Ctx, patch func(url.Values) url.Values) {
	store := requireStore(ctx)
	loc := store.Location()
	query := cloneValues(loc.Query)
	if patch != nil {
		query = patch(query)
	}
	loc.Query = canonicalizeValues(query)
	store.RecordNavigation(NavKindReplace, loc)
}

// Back records a history back event.
func Back(ctx runtime.Ctx) {
	store := requireStore(ctx)
	store.RecordBack()
}

// Redirect triggers a replace navigation during the next effect pass.
func Redirect(ctx runtime.Ctx, to string) h.Node {
	target := to
	runtime.UseEffect(ctx, func() runtime.Cleanup {
		Replace(ctx, target)
		return nil
	}, target)
	return h.Fragment()
}

func setSearchParam(q url.Values, key string, values []string) url.Values {
	out := cloneValues(q)
	if len(values) == 0 {
		delete(out, key)
		return out
	}
	out[key] = canonicalizeList(values)
	return out
}
