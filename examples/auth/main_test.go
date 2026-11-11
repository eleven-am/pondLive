package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/pkg"
	"github.com/eleven-am/pondlive/pkg/livetest"
)

func get(t *testing.T, path string, cookie string) *httptest.ResponseRecorder {
	t.Helper()
	app, err := newApp()
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, path, nil)
	if cookie != "" {
		req.Header.Set("Cookie", "session="+cookie)
	}
	rec := httptest.NewRecorder()
	app.Handler().ServeHTTP(rec, req)
	return rec
}

func TestHomeShowsLoginWhenLoggedOut(t *testing.T) {
	rec := get(t, "/", "")

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d", rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Welcome to Auth Example") || !strings.Contains(body, ">Login<") || strings.Contains(body, "Logout") {
		t.Fatalf("logged-out home:\n%s", body)
	}
}

func TestHomeShowsTheUserWhenLoggedIn(t *testing.T) {
	rec := get(t, "/", "ada")

	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "Hello, ada") || !strings.Contains(body, "Logout") {
		t.Fatalf("status = %d, logged-in home:\n%s", rec.Code, body)
	}
	nav := body[strings.Index(body, "<nav"):strings.Index(body, "</nav>")]
	if strings.Count(nav, "<!--r:") != 2 || !strings.Contains(nav, "Hello, ada") || !strings.Contains(nav, "Dashboard") {
		t.Fatalf("the fragment arm must render inside its own region:\n%s", nav)
	}
}

func TestLoginPageRendersTheForm(t *testing.T) {
	rec := get(t, "/login", "")

	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, `placeholder="Enter any username"`) || !strings.Contains(body, "Sign In") {
		t.Fatalf("status = %d, login page:\n%s", rec.Code, body)
	}
}

func TestDashboardRedirectsToLoginWhenLoggedOut(t *testing.T) {
	rec := get(t, "/dashboard", "")

	if rec.Code < 300 || rec.Code > 399 || rec.Header().Get("Location") != "/login" {
		t.Fatalf("status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestDashboardRendersForALoggedInUser(t *testing.T) {
	rec := get(t, "/dashboard", "ada")

	body := rec.Body.String()
	if rec.Code != http.StatusOK || !strings.Contains(body, "protected page") || !strings.Contains(body, ">ada<") {
		t.Fatalf("status = %d, dashboard:\n%s", rec.Code, body)
	}
}

func TestLoginRedirectsToTheDashboardWhenLoggedIn(t *testing.T) {
	rec := get(t, "/login", "ada")

	if rec.Code < 300 || rec.Code > 399 || rec.Header().Get("Location") != "/dashboard" {
		t.Fatalf("status = %d, location = %q", rec.Code, rec.Header().Get("Location"))
	}
}

func TestNothingStartsBeforeTheSocketConnects(t *testing.T) {
	livetest.AssertNothingStartsBeforeConnect(t, root, pkg.WithDevMode())
}
