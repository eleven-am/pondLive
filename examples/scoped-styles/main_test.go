package main

import (
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/pkg"
	"github.com/eleven-am/pondlive/pkg/livetest"
)

var scopedLine = regexp.MustCompile(`Scoped classes: \.card → \.([A-Za-z0-9_-]+)`)

func TestSSRScopesTheSameClassNamesPerComponent(t *testing.T) {
	app, err := newApp()
	if err != nil {
		t.Fatal(err)
	}
	rec := httptest.NewRecorder()

	app.Handler().ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{"Red Card Component", "Blue Card Component", ">Scoped Styles Demo</title>"} {
		if !strings.Contains(body, want) {
			t.Fatalf("body is missing %q:\n%s", want, body)
		}
	}
	found := scopedLine.FindAllStringSubmatch(body, -1)
	if len(found) != 2 || found[0][1] == found[1][1] {
		t.Fatalf("scoped .card classes = %v, want two distinct names", found)
	}
	for _, m := range found {
		if !strings.Contains(body, "."+m[1]+"{") && !strings.Contains(body, "."+m[1]+" {") {
			t.Fatalf("no stylesheet rule for the scoped class %q:\n%s", m[1], body)
		}
	}
}

func TestNothingStartsBeforeTheSocketConnects(t *testing.T) {
	livetest.AssertNothingStartsBeforeConnect(t, HomePage, pkg.WithDevMode())
}
