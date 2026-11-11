package head_test

import (
	"net/http"
	"net/http/httptest"

	"github.com/eleven-am/pondlive/internal/runtime"
)

type httptestRecorder struct {
	code        int
	contentType string
	body        []byte
}

func serve(s *runtime.Session, href string) *httptestRecorder {
	req := httptest.NewRequest(http.MethodGet, href, nil)
	rec := httptest.NewRecorder()
	s.ServeHTTP(rec, req)
	res := rec.Result()
	defer res.Body.Close()
	return &httptestRecorder{code: res.StatusCode, contentType: res.Header.Get("Content-Type"), body: rec.Body.Bytes()}
}
