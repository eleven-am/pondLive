package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/pkg"
	"github.com/eleven-am/pondlive/pkg/livetest"
)

func get(t *testing.T, path string) *httptest.ResponseRecorder {
	t.Helper()
	app, err := newApp()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestSSRRendersTheCounter(t *testing.T) {
	rec := get(t, "/")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		">LiveUI Tailwind Counter: 0</title>",
		"/assets/tailwind.css",
		"LiveUI Tailwind Counter",
		`tabular-nums`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body is missing %q:\n%s", want, body)
		}
	}
	if !strings.Contains(body, ">0<") {
		t.Fatalf("the count is not rendered:\n%s", body)
	}
}

func TestStylesheetIsServed(t *testing.T) {
	rec := get(t, "/assets/tailwind.css")

	if rec.Code != http.StatusOK || rec.Body.Len() == 0 {
		t.Fatalf("status = %d, bytes = %d", rec.Code, rec.Body.Len())
	}
}

func TestNothingStartsBeforeTheSocketConnects(t *testing.T) {
	livetest.AssertNothingStartsBeforeConnect(t, counter, pkg.WithDevMode())
}
