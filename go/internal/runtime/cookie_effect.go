package runtime

import "net/http"

// CookieEffect instructs the client to perform an HTTP handshake so the server can set HttpOnly cookies.
type CookieEffect struct {
	Type     string `json:"type"`
	Endpoint string `json:"endpoint"`
	SID      string `json:"sid"`
	Token    string `json:"token"`
	Method   string `json:"method,omitempty"`
}

// newCookieEffect builds a CookieEffect using the provided endpoint, session identifier, and token.
func newCookieEffect(endpoint, sid, token string) *CookieEffect {
	if stringsTrim(endpoint) == "" || stringsTrim(sid) == "" || stringsTrim(token) == "" {
		return nil
	}
	return &CookieEffect{
		Type:     "cookies",
		Endpoint: endpoint,
		SID:      sid,
		Token:    token,
		Method:   http.MethodPost,
	}
}
