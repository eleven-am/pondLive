package server

import (
	"net/http"

	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/session"
)

func cloneHeader(h http.Header) http.Header {
	if h == nil {
		return http.Header{}
	}
	clone := make(http.Header, len(h))
	for k, v := range h {
		clone[k] = append([]string(nil), v...)
	}
	return clone
}

func serverError(sid session.SessionID, code string, err error) protocol.ServerError {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	return protocol.ServerError{
		T:       "error",
		SID:     string(sid),
		Code:    code,
		Message: msg,
	}
}
