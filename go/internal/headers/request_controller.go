package headers

import (
	"net/http"
	"net/url"
	"sync"
)

// RequestController manages HTTP request state and response headers.
// It stores initial request data (headers, location), response headers to set,
// and manages SSR redirects.
type RequestController struct {
	// Request state from initial HTTP request
	requestHeaders http.Header
	initialPath    string
	initialQuery   url.Values
	initialHash    string

	// Response state
	responseHeaders http.Header
	statusCode      int
	redirectURL     string

	// Connection state
	isLive bool
	mu     sync.RWMutex
}

// NewRequestController creates a new RequestController.
func NewRequestController() *RequestController {
	return &RequestController{
		requestHeaders:  make(http.Header),
		responseHeaders: make(http.Header),
	}
}

// Get retrieves a header value from the initial request headers.
// Returns the value and true if found, empty string and false otherwise.
// This is safe for concurrent access.
func (c *RequestController) Get(name string) (string, bool) {
	if c == nil {
		return "", false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	val := c.requestHeaders.Get(name)
	return val, val != ""
}

// Set sets a response header to be sent in the HTTP response.
// In SSR mode, the handler reads these after render.
// In websocket mode, the UseHeader hook handles this differently.
func (c *RequestController) Set(name, value string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.responseHeaders.Set(name, value)
}

// GetResponseHeaders returns all response headers that were set.
// Used by SSR handler to apply headers to the HTTP response.
func (c *RequestController) GetResponseHeaders() http.Header {
	if c == nil {
		return nil
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	headers := make(http.Header, len(c.responseHeaders))
	for k, v := range c.responseHeaders {
		headers[k] = append([]string(nil), v...)
	}
	return headers
}

// SetInitialHeaders populates the initial request headers from an http.Header.
// This should only be called once during session initialization (in MergeRequest).
// Safe for concurrent access.
func (c *RequestController) SetInitialHeaders(headers http.Header) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.requestHeaders = headers.Clone()
}

// SetRedirect sets a redirect for SSR mode. Internal API used by router.
func (c *RequestController) SetRedirect(url string, statusCode int) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.redirectURL = url
	c.statusCode = statusCode
}

// GetRedirect returns the redirect URL and status code if a redirect was set.
// Returns (url, code, true) if redirect is set, ("", 0, false) otherwise.
// Used by SSR handler to check for redirects after render.
func (c *RequestController) GetRedirect() (url string, code int, hasRedirect bool) {
	if c == nil {
		return "", 0, false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	if c.redirectURL != "" {
		return c.redirectURL, c.statusCode, true
	}
	return "", 0, false
}

// IsLive returns whether the session is currently live (connected).
func (c *RequestController) IsLive() bool {
	if c == nil {
		return false
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.isLive
}

// SetIsLive sets the live status of the session.
func (c *RequestController) SetIsLive(isLive bool) {
	if c == nil {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	c.isLive = isLive
}

// SetInitialLocation sets the initial location from the HTTP request.
// This should only be called once during session initialization (in MergeRequest).
func (c *RequestController) SetInitialLocation(path string, query url.Values, hash string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.initialPath = path
	c.initialQuery = query
	c.initialHash = hash
}

// GetInitialLocation returns the initial location from the HTTP request.
// Returns (path, query, hash).
func (c *RequestController) GetInitialLocation() (path string, query url.Values, hash string) {
	if c == nil {
		return "", nil, ""
	}
	c.mu.RLock()
	defer c.mu.RUnlock()

	queryCopy := make(url.Values)
	for k, v := range c.initialQuery {
		queryCopy[k] = append([]string(nil), v...)
	}
	return c.initialPath, queryCopy, c.initialHash
}

// UpdateCookie optimistically updates a cookie in the request headers.
// This ensures subsequent GetCookie calls see the updated value.
func (c *RequestController) UpdateCookie(name, value string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	req := &http.Request{Header: c.requestHeaders}

	cookies := req.Cookies()
	var newCookies []*http.Cookie
	for _, cookie := range cookies {
		if cookie.Name != name {
			newCookies = append(newCookies, cookie)
		}
	}

	newCookies = append(newCookies, &http.Cookie{
		Name:  name,
		Value: value,
	})

	c.requestHeaders.Del("Cookie")
	for _, cookie := range newCookies {
		if c.requestHeaders.Get("Cookie") == "" {
			c.requestHeaders.Set("Cookie", cookie.String())
		} else {
			c.requestHeaders.Set("Cookie", c.requestHeaders.Get("Cookie")+"; "+cookie.Name+"="+cookie.Value)
		}
	}
}

// DeleteCookieFromRequest removes a cookie from the request headers.
// This ensures subsequent GetCookie calls won't find the deleted cookie.
func (c *RequestController) DeleteCookieFromRequest(name string) {
	if c == nil {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	req := &http.Request{Header: c.requestHeaders}

	cookies := req.Cookies()
	var newCookies []*http.Cookie
	for _, cookie := range cookies {
		if cookie.Name != name {
			newCookies = append(newCookies, cookie)
		}
	}

	c.requestHeaders.Del("Cookie")
	for _, cookie := range newCookies {
		if c.requestHeaders.Get("Cookie") == "" {
			c.requestHeaders.Set("Cookie", cookie.String())
		} else {
			c.requestHeaders.Set("Cookie", c.requestHeaders.Get("Cookie")+"; "+cookie.Name+"="+cookie.Value)
		}
	}
}
