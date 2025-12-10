package headers

import (
	"net/http"
	"time"

	"github.com/eleven-am/pondlive/internal/runtime"
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
	pState := useProviderState(ctx)

	setter := func(newValue string, opts *CookieOptions) {
		if opts != nil && opts.MaxAge < 0 {
			state.DeleteCookieMutation(name)
		} else {
			state.MutateCookie(name, newValue)
		}

		if state.IsLive() && pState != nil {
			sendCookieViaScript(pState, name, newValue, opts)
			return
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

func UseIsLive(ctx *runtime.Ctx) bool {
	state := UseRequestState(ctx)
	if state == nil {
		return false
	}
	return state.IsLive()
}
