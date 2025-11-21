package session

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/runtime"
)

func TestNew(t *testing.T) {
	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		headers := UseHeader(ctx)
		ua, _ := headers.GetHeader("User-Agent")

		return dom.ElementNode("div").WithChildren(
			dom.ElementNode("p").WithChildren(dom.TextNode("User-Agent: " + ua)),
		)
	}

	transport := &mockTransport{}
	sess := New(
		SessionID("test"),
		1,
		app,
		&Config{Transport: transport},
	)

	if sess.ID() != "test" {
		t.Errorf("expected ID 'test', got %q", sess.ID())
	}

	req := newRequest("/")
	req.Header.Set("User-Agent", "TestBot/3.0")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	ua, ok := sess.Header().GetHeader("User-Agent")
	if !ok || ua != "TestBot/3.0" {
		t.Errorf("expected User-Agent 'TestBot/3.0', got %q (ok=%v)", ua, ok)
	}

	if len(transport.frames) == 0 {
		t.Error("expected patches to be sent")
	}
}

func TestNewWithStateChanges(t *testing.T) {
	var setCount func(int)

	app := func(ctx runtime.Ctx) *dom.StructuredNode {
		headers := UseHeader(ctx)
		count, set := runtime.UseState(ctx, 0)
		setCount = set

		ua, _ := headers.GetHeader("User-Agent")

		return dom.ElementNode("div").WithChildren(
			dom.ElementNode("p").WithChildren(dom.TextNode("UA: "+ua)),
			dom.ElementNode("p").WithChildren(dom.TextNode("Count: "+string(rune('0'+count())))),
		)
	}

	transport := &mockTransport{}
	sess := New(
		SessionID("test"),
		1,
		app,
		&Config{Transport: transport},
	)

	req := newRequest("/")
	req.Header.Set("User-Agent", "TestBot/4.0")
	sess.MergeRequest(req)

	if err := sess.Flush(); err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	initialPatches := len(transport.frames)

	setCount(7)
	if err := sess.Flush(); err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	if len(transport.frames) <= initialPatches {
		t.Errorf("expected more patches after state change, got %d (was %d)",
			len(transport.frames), initialPatches)
	}
}
