package session

import (
	"net/http"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func TestLiveSessionBasic(t *testing.T) {
	root := func(ctx runtime.Ctx, props struct{}) *dom2.StructuredNode {
		return dom2.ElementNode("div").WithChildren(dom2.TextNode("Hello"))
	}

	sess := NewLiveSession(SessionID("test"), 1, root, struct{}{}, nil)

	if sess.ID() != "test" {
		t.Errorf("expected ID 'test', got %q", sess.ID())
	}
	if sess.Version() != 1 {
		t.Errorf("expected version 1, got %d", sess.Version())
	}
}

func TestLiveSessionHeaders(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom2.StructuredNode {
		headers := UseHeader(ctx)
		ua, _ := headers.GetHeader("User-Agent")
		return dom2.ElementNode("div").WithChildren(dom2.TextNode(ua))
	}

	sess := New(SessionID("test"), 1, app, nil)

	req := newRequest("/")
	req.Header.Set("User-Agent", "TestBot/1.0")
	sess.MergeRequest(req)

	ua, ok := sess.Header().GetHeader("User-Agent")
	if !ok || ua != "TestBot/1.0" {
		t.Errorf("expected User-Agent 'TestBot/1.0', got %q (ok=%v)", ua, ok)
	}
}

func TestLiveSessionDocumentRoot(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom2.StructuredNode {
		headers := UseHeader(ctx)
		token, ok := headers.GetCookie("session")
		text := "no session"
		if ok && token != nil {
			text = token.Value
		}
		return dom2.ElementNode("div").WithChildren(dom2.TextNode(text))
	}

	transport := &mockTransport{}
	sess := New(SessionID("test"), 1, app, &Config{
		Transport: transport,
	})

	req := newRequest("/")
	req.AddCookie(&http.Cookie{Name: "session", Value: "abc123"})
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}
}

func TestLiveSessionEventHandling(t *testing.T) {
	var clicked bool

	root := func(ctx runtime.Ctx) *dom2.StructuredNode {
		btn := dom2.ElementNode("button").WithChildren(dom2.TextNode("Click"))
		btn.Events = map[string]dom2.EventBinding{
			"click": {
				Key: "btn:h0",
				Handler: func(ev dom2.Event) dom2.Updates {
					clicked = true
					return dom2.Rerender()
				},
			},
		}
		return btn
	}

	transport := &mockTransport{}
	sess := New(SessionID("test"), 1, root, &Config{
		Transport: transport,
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	if err := sess.HandleEvent("btn:h0", dom2.Event{}); err != nil {
		t.Fatalf("handle event failed: %v", err)
	}

	if !clicked {
		t.Error("expected click handler to be called")
	}
}

type mockTransport struct {
	frames []protocol.Frame
	boots  []protocol.Boot
	inits  []protocol.Init
}

func (m *mockTransport) SendBoot(boot protocol.Boot) error {
	m.boots = append(m.boots, boot)
	return nil
}

func (m *mockTransport) SendInit(init protocol.Init) error {
	m.inits = append(m.inits, init)
	return nil
}

func (m *mockTransport) SendResume(res protocol.ResumeOK) error {
	return nil
}

func (m *mockTransport) SendFrame(frame protocol.Frame) error {
	m.frames = append(m.frames, frame)
	return nil
}

func (m *mockTransport) SendEventAck(ack protocol.EventAck) error {
	return nil
}

func (m *mockTransport) SendServerError(err protocol.ServerError) error {
	return nil
}

func (m *mockTransport) SendDiagnostic(diag protocol.Diagnostic) error {
	return nil
}

func (m *mockTransport) SendDOMRequest(req protocol.DOMRequest) error {
	return nil
}

func (m *mockTransport) SendPubsubControl(ctrl protocol.PubsubControl) error {
	return nil
}

func (m *mockTransport) SendUploadControl(ctrl protocol.UploadControl) error {
	return nil
}

func (m *mockTransport) Close() error {
	return nil
}

func TestLiveSessionTransport(t *testing.T) {
	var setText func(string)

	root := func(ctx runtime.Ctx) *dom2.StructuredNode {
		text, set := runtime.UseState(ctx, "initial")
		setText = set
		return dom2.ElementNode("div").WithChildren(dom2.TextNode(text()))
	}

	transport := &mockTransport{}
	sess := New(SessionID("test"), 1, root, &Config{
		Transport: transport,
	})

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	setText("updated")
	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	if len(transport.frames) < 2 {
		t.Fatalf("expected at least 2 frames, got %d", len(transport.frames))
	}

	if transport.frames[0].Seq != 1 {
		t.Errorf("expected first seq to be 1, got %d", transport.frames[0].Seq)
	}
	if transport.frames[1].Seq != 2 {
		t.Errorf("expected second seq to be 2, got %d", transport.frames[1].Seq)
	}
}

func TestDocumentRootProvidesRouterLocation(t *testing.T) {
	var seen router.Location
	app := func(ctx runtime.Ctx) *dom2.StructuredNode {
		seen = router.UseLocation(ctx)
		return dom2.ElementNode("div")
	}

	transport := &mockTransport{}
	sess := New(SessionID("router-loc"), 1, app, &Config{Transport: transport})
	req := newRequest("/products/details?page=2&filter=active#info")
	sess.MergeRequest(req)
	if req.URL.Path != "/products/details" {
		t.Fatalf("request path parsed incorrectly: %q", req.URL.Path)
	}
	if got := sess.InitialLocation().Path; got != "/products/details" {
		t.Fatalf("initial location path mismatch: %q", got)
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if seen.Path != "/products/details" {
		t.Fatalf("expected path /products/details, got %q", seen.Path)
	}
	if seen.Hash != "info" {
		t.Fatalf("expected hash info, got %q", seen.Hash)
	}
	if got := seen.Query.Get("page"); got != "2" {
		t.Fatalf("expected query page=2, got %q", got)
	}
	if got := seen.Query.Get("filter"); got != "active" {
		t.Fatalf("expected query filter=active, got %q", got)
	}
}

func TestDocumentRootRouterLocationClone(t *testing.T) {
	var (
		captured []router.Location
		trigger  func(bool)
	)

	app := func(ctx runtime.Ctx) *dom2.StructuredNode {
		loc := router.UseLocation(ctx)
		captured = append(captured, loc)
		flag, setFlag := runtime.UseState(ctx, false)
		if trigger == nil {
			trigger = setFlag
		}
		_ = flag
		return dom2.ElementNode("div")
	}

	transport := &mockTransport{}
	sess := New(SessionID("router-clone"), 1, app, &Config{Transport: transport})
	req := newRequest("/orders?status=pending")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}
	if len(captured) != 1 {
		t.Fatalf("expected first capture, got %d", len(captured))
	}

	captured[0].Query.Set("status", "mutated")
	if trigger == nil {
		t.Fatalf("state trigger not initialized")
	}
	trigger(true)

	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}
	if len(captured) != 2 {
		t.Fatalf("expected second capture, got %d", len(captured))
	}
	if got := captured[1].Query.Get("status"); got != "pending" {
		t.Fatalf("expected cloned query to remain pending, got %q", got)
	}
}

func TestLiveSessionCookies(t *testing.T) {
	root := func(ctx runtime.Ctx) *dom2.StructuredNode {
		headers := UseHeader(ctx)
		headers.SetCookie(&http.Cookie{Name: "session", Value: "abc123"})
		return dom2.ElementNode("div")
	}

	transport := &mockTransport{}
	sess := New(SessionID("cookie"), 1, root, &Config{Transport: transport})

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	// Find frame with cookie effects
	var cookieFrame *protocol.Frame
	for i := range transport.frames {
		if len(transport.frames[i].Effects) > 0 {
			cookieFrame = &transport.frames[i]
			break
		}
	}
	if cookieFrame == nil {
		t.Fatalf("expected frame with cookie effects")
	}

	if len(cookieFrame.Effects) != 1 {
		t.Fatalf("expected 1 cookie effect, got %d", len(cookieFrame.Effects))
	}
	effect, ok := cookieFrame.Effects[0].(protocol.CookieEffect)
	if !ok {
		t.Fatalf("expected CookieEffect, got %T", cookieFrame.Effects[0])
	}

	batch, ok := sess.ConsumeCookieBatch(effect.Token)
	if !ok {
		t.Fatalf("expected cookie batch for token %s", effect.Token)
	}
	if len(batch.Set) != 1 {
		t.Fatalf("expected 1 cookie set, got %d", len(batch.Set))
	}
	if batch.Set[0].Name != "session" || batch.Set[0].Value != "abc123" {
		t.Fatalf("unexpected cookie %v", batch.Set[0])
	}

	if _, ok := sess.ConsumeCookieBatch(effect.Token); ok {
		t.Fatalf("expected token %s to be consumed", effect.Token)
	}
}

func TestLiveSessionCookieDeletes(t *testing.T) {
	root := func(ctx runtime.Ctx) *dom2.StructuredNode {
		return dom2.ElementNode("div")
	}

	transport := &mockTransport{}
	sess := New(SessionID("cookie-del"), 1, root, &Config{Transport: transport})

	header := sess.Header()
	header.SetCookie(&http.Cookie{Name: "session", Value: "init"})
	header.DeleteCookie("session")

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	// Find frame with cookie effects
	var cookieFrame *protocol.Frame
	for i := range transport.frames {
		if len(transport.frames[i].Effects) > 0 {
			cookieFrame = &transport.frames[i]
			break
		}
	}
	if cookieFrame == nil {
		t.Fatalf("expected frame with cookie effects")
	}

	effect, ok := cookieFrame.Effects[0].(protocol.CookieEffect)
	if !ok {
		t.Fatalf("expected CookieEffect, got %T", cookieFrame.Effects[0])
	}

	batch, ok := sess.ConsumeCookieBatch(effect.Token)
	if !ok {
		t.Fatalf("expected cookie batch for token %s", effect.Token)
	}
	if len(batch.Delete) != 1 || batch.Delete[0] != "session" {
		t.Fatalf("expected delete for session cookie, got %v", batch.Delete)
	}
}
