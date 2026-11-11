package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/pkg"
	"github.com/eleven-am/pondlive/pkg/livetest"
)

func TestSSRRendersTheEmptyChat(t *testing.T) {
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
	for _, want := range []string{
		">LiveUI Stream Chat</title>",
		"Stream Chat",
		"No messages yet. Start typing!",
		"Prepend",
		"Clear",
		`placeholder="Type a message..."`,
		`disabled=""`,
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("body is missing %q:\n%s", want, body)
		}
	}
}

func TestNothingStartsBeforeTheSocketConnects(t *testing.T) {
	livetest.AssertNothingStartsBeforeConnect(t, chat, pkg.WithDevMode())
}

func TestPressingEnterSendsTheTypedMessageAndClearsTheField(t *testing.T) {
	page := livetest.Mount(t, chat)
	field := page.ByID("draft")

	field.Type("hello there")
	field.Press("a")
	if strings.Contains(page.HTML(), "hello there</") {
		t.Fatalf("a key other than Enter must not send: %s", page.HTML())
	}
	field.Press("Enter")

	got := page.HTML()
	if !strings.Contains(got, "hello there") || strings.Contains(got, "No messages yet") {
		t.Fatalf("Enter must send the typed message: %s", got)
	}
}
