package session

import (
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/runtime"
)

// HeaderContext provides access to HTTP headers, cookies, and request metadata.
var HeaderContext = runtime.CreateContext[HeaderState](noopHeaderState{})

type noopHeaderState struct{}

func (noopHeaderState) GetHeader(string) (string, bool)       { return "", false }
func (noopHeaderState) SetHeader(string, string)              {}
func (noopHeaderState) DeleteHeader(string)                   {}
func (noopHeaderState) AllHeaders() http.Header               { return http.Header{} }
func (noopHeaderState) GetCookie(string) (*http.Cookie, bool) { return nil, false }
func (noopHeaderState) SetCookie(*http.Cookie)                {}
func (noopHeaderState) DeleteCookie(string)                   {}
func (noopHeaderState) AllCookies() []*http.Cookie            { return nil }
func (noopHeaderState) RequestID() string                     { return "" }
func (noopHeaderState) SetMeta(string, any)                   {}
func (noopHeaderState) Meta(string) (any, bool)               { return nil, false }
