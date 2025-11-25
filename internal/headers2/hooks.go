package headers2

import (
	"net/http"
	"time"

	"github.com/eleven-am/pondlive/go/internal/runtime2"
)

// CookieOptions configures cookie attributes.
type CookieOptions struct {
	Path     string
	Domain   string
	MaxAge   int       // MaxAge=0 means no Max-Age attribute. MaxAge<0 means delete cookie.
	Expires  time.Time // Zero time means no Expires attribute.
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// UseHeaders returns the request headers from context.
// Returns nil if not within a RequestState provider.
func UseHeaders(ctx *runtime2.Ctx) http.Header {
	state := UseRequestState(ctx)
	if state == nil || state.info == nil {
		return nil
	}
	return state.info.Headers
}

// UseCookie returns a cookie value and a setter function.
// The getter respects optimistic mutations made during the current render.
// The setter updates the cookie (via response header in SSR, or script in live mode).
func UseCookie(ctx *runtime2.Ctx, name string) (string, func(value string, opts *CookieOptions)) {
	state := UseRequestState(ctx)
	if state == nil {
		return "", func(string, *CookieOptions) {}
	}

	value, _ := state.GetCookie(name)

	setter := func(newValue string, opts *CookieOptions) {
		state.MutateCookie(name, newValue)

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

// UseRedirect returns a function to trigger a redirect.
// In SSR mode, this sets the redirect response.
// In live mode, this should trigger client-side navigation.
func UseRedirect(ctx *runtime2.Ctx) func(url string, code int) {
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

// UsePath returns the current request path.
func UsePath(ctx *runtime2.Ctx) string {
	state := UseRequestState(ctx)
	if state == nil {
		return ""
	}
	return state.Path()
}

// UseQuery returns the current request query parameters.
func UseQuery(ctx *runtime2.Ctx) map[string]string {
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

// UseIsLive returns whether the session is in live (WebSocket) mode.
func UseIsLive(ctx *runtime2.Ctx) bool {
	state := UseRequestState(ctx)
	if state == nil {
		return false
	}
	return state.IsLive()
}
