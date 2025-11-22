package headers

import (
	"net/http"
	"time"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// CookieOptions represents options for setting a cookie.
type CookieOptions struct {
	Path     string
	Domain   string
	MaxAge   int
	Expires  time.Time
	Secure   bool
	HttpOnly bool
	SameSite http.SameSite
}

// Handle provides access to both request headers and cookie management.
type Handle struct {
	manager    *Manager
	controller *RequestController
}

// Get retrieves a request header value.
func (h *Handle) Get(name string) (string, bool) {
	if h.controller == nil {
		return "", false
	}
	return h.controller.Get(name)
}

// SetCookie sets a cookie with default options in both SSR and websocket modes.
func (h *Handle) SetCookie(name, value string) {
	if h.manager == nil {
		return
	}
	h.manager.SetCookie(name, value)
}

// SetCookieWithOptions sets a cookie with custom options in both SSR and websocket modes.
func (h *Handle) SetCookieWithOptions(name, value string, options CookieOptions) {
	if h.manager == nil {
		return
	}
	h.manager.SetCookieWithOptions(name, value, options)
}

// DeleteCookie deletes a cookie in both SSR and websocket modes.
func (h *Handle) DeleteCookie(name string) {
	if h.manager == nil {
		return
	}
	h.manager.DeleteCookie(name)
}

// GetCookie retrieves a cookie value from request headers.
// Note: This reads from the initial request, not cookies set during render.
func (h *Handle) GetCookie(name string) (string, bool) {
	cookieHeader, ok := h.Get("Cookie")
	if !ok {
		return "", false
	}

	header := http.Header{}
	header.Add("Cookie", cookieHeader)
	request := &http.Request{Header: header}

	cookie, err := request.Cookie(name)
	if err != nil {
		return "", false
	}

	return cookie.Value, true
}

// UseHeaders returns a handle for accessing request headers and managing cookies.
func UseHeaders(ctx runtime.Ctx) Handle {
	manager := UseHeadersManager(ctx)
	controller := UseRequestController(ctx)
	return Handle{
		manager:    manager,
		controller: controller,
	}
}
