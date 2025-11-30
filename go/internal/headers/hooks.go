package headers

import (
	"net/http"
	"net/url"
	"time"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

type CookieOptions struct {
	Path     string
	Domain   string
	MaxAge   int
	Expires  time.Time
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

func UseHeaders(ctx *runtime.Ctx) http.Header {
	state := UseRequestState(ctx)
	if state == nil || state.info == nil {
		return nil
	}
	return state.info.Headers.Clone()
}

func UseCookie(ctx *runtime.Ctx, name string) (string, func(value string, opts *CookieOptions)) {
	state := UseRequestState(ctx)
	if state == nil {
		return "", func(string, *CookieOptions) {}
	}

	value, _ := state.GetCookie(name)

	setter := func(newValue string, opts *CookieOptions) {
		if opts != nil && opts.MaxAge < 0 {
			state.DeleteCookieMutation(name)
		} else {
			state.MutateCookie(name, newValue)
		}

		cookie := &http.Cookie{
			Name:  name,
			Value: newValue,
		}

		if opts != nil {
			cookie.Path = opts.Path
			cookie.Domain = opts.Domain
			cookie.MaxAge = opts.MaxAge
			cookie.Expires = opts.Expires
			cookie.Secure = opts.Secure
			cookie.HttpOnly = opts.HttpOnly
			cookie.SameSite = opts.SameSite
		}

		if cookie.Path == "" {
			cookie.Path = "/"
		}

		state.AddResponseHeader("Set-Cookie", cookie.String())
	}

	return value, setter
}

func UseRedirect(ctx *runtime.Ctx) func(url string, code int) {
	state := UseRequestState(ctx)
	if state == nil {
		return func(string, int) {}
	}

	return func(url string, code int) {
		if code == 0 {
			code = http.StatusFound
		}
		state.SetRedirect(url, code)
	}
}

func UsePath(ctx *runtime.Ctx) string {
	state := UseRequestState(ctx)
	if state == nil {
		return ""
	}
	return state.Path()
}

func UseQuery(ctx *runtime.Ctx) map[string]string {
	state := UseRequestState(ctx)
	if state == nil {
		return nil
	}

	query := state.Query()
	if query == nil {
		return nil
	}

	result := make(map[string]string, len(query))
	for k, v := range query {
		if len(v) > 0 {
			result[k] = v[0]
		}
	}
	return result
}

func UseQueryValues(ctx *runtime.Ctx) url.Values {
	state := UseRequestState(ctx)
	if state == nil {
		return nil
	}
	return state.Query()
}

func UseIsLive(ctx *runtime.Ctx) bool {
	state := UseRequestState(ctx)
	if state == nil {
		return false
	}
	return state.IsLive()
}
