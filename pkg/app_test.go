package pkg

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

var page = Component(func(ctx *Ctx) Node { return Div(Text("hi")) })

func dialWithOrigin(t *testing.T, srv *httptest.Server, origin string) (*websocket.Conn, *http.Response, error) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return websocket.Dial(ctx, "ws"+strings.TrimPrefix(srv.URL, "http")+"/live", &websocket.DialOptions{HTTPHeader: http.Header{"Origin": []string{origin}}})
}

func TestNewAppRejectsShortAttachSecret(t *testing.T) {
	if _, err := NewApp(page, WithAttachSecret([]byte("short"))); err == nil {
		t.Fatal("expected an error for a short attach secret")
	}
}

func TestWithAllowedOriginsOpensTheSocketToThoseOrigins(t *testing.T) {
	closed, err := NewApp(page)
	if err != nil {
		t.Fatal(err)
	}
	open, err := NewApp(page, WithAllowedOrigins("https://app.example"), WithAttachSecret(make([]byte, 32)), WithAttachTokenTTL(time.Minute), WithReadLimit(1024), WithOutboundBudget(1<<20), WithPingInterval(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	closedSrv := httptest.NewServer(closed.Handler())
	defer closedSrv.Close()
	openSrv := httptest.NewServer(open.Handler())
	defer openSrv.Close()

	if c, _, err := dialWithOrigin(t, closedSrv, "https://app.example"); err == nil {
		c.CloseNow()
		t.Fatal("default app must be same-origin only")
	}
	c, _, err := dialWithOrigin(t, openSrv, "https://app.example")
	if err != nil {
		t.Fatalf("allow-listed origin rejected: %v", err)
	}
	c.CloseNow()
	if c, _, err := dialWithOrigin(t, openSrv, "https://other.example"); err == nil {
		c.CloseNow()
		t.Fatal("origin outside the allow list accepted")
	}
}

func TestWithMaxSessionsRefusesPageRequestsBeyondCapacity(t *testing.T) {
	app, err := NewApp(page, WithMaxSessions(1))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	srv := httptest.NewServer(app.Handler())
	defer srv.Close()

	for i, want := range []int{http.StatusOK, http.StatusServiceUnavailable} {
		resp, err := http.Get(srv.URL + "/")
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != want {
			t.Fatalf("request %d: status %d, want %d", i, resp.StatusCode, want)
		}
	}
	if st := app.Stats(); st.Capacity != 1 || st.Sessions.Pending != 1 || st.Rejected.Full != 1 {
		t.Fatalf("stats = %+v", st)
	}
}

func TestWithHTTPConfigReachesTheServer(t *testing.T) {
	app, err := NewApp(page, WithHTTPConfig(HTTPConfig{ReadHeaderTimeout: time.Second, MaxHeaderBytes: 2048}), WithSessionLifetime(time.Hour), WithRetainedHeaders("X-Tenant"), WithWatchdog(WatchdogConfig{Interval: time.Second}))
	if err != nil {
		t.Fatal(err)
	}
	defer app.Shutdown(context.Background())
	srv := app.Server(":0")
	if srv.ReadHeaderTimeout != time.Second || srv.MaxHeaderBytes != 2048 || srv.IdleTimeout <= 0 || srv.WriteTimeout <= 0 {
		t.Fatalf("server = %+v", srv)
	}
}
