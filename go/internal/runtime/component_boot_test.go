package runtime

import (
	"strings"
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestComponentBootEffect(t *testing.T) {
	type props struct{}
	var trigger func(string)
	var componentID string
	component := func(ctx Ctx, _ props) h.Node {
		value, set := UseState(ctx, "alpha")
		componentID = ctx.ComponentID()
		trigger = func(next string) {
			set(next)
			ctx.RequestComponentBoot()
		}
		return h.Div(h.Text(value()))
	}

	transport := &stubTransport{}
	sess := NewLiveSession("comp-boot", 1, component, props{}, &LiveSessionConfig{Transport: transport})
	if trigger == nil {
		t.Fatal("expected trigger to be initialized during initial render")
	}

	trigger("bravo")
	if err := sess.component.Flush(); err != nil {
		t.Fatalf("flush error: %v", err)
	}
	if len(transport.templates) == 0 {
		t.Fatal("expected component template frame to be sent")
	}
	tpl := transport.templates[len(transport.templates)-1]
	if tpl.Scope == nil || tpl.Scope.ComponentID != componentID {
		t.Fatalf("unexpected template scope: %+v", tpl.Scope)
	}
	if tpl.TemplatePayload.HTML == "" || !strings.Contains(tpl.TemplatePayload.HTML, "bravo") {
		t.Fatalf("expected html payload to contain updated text, got %q", tpl.TemplatePayload.HTML)
	}
	if len(tpl.TemplatePayload.Slots) == 0 {
		t.Fatalf("expected slot metadata in template payload, got %+v", tpl.TemplatePayload.Slots)
	}
	slotIndex := tpl.TemplatePayload.Slots[len(tpl.TemplatePayload.Slots)-1].AnchorID
	if slotIndex < 0 || slotIndex >= len(sess.snapshot.Dynamics) {
		t.Fatalf("expected slot to map into snapshot, got %d", slotIndex)
	}
	dyn := sess.snapshot.Dynamics[slotIndex]
	if dyn.Kind != "text" || dyn.Text != "bravo" {
		t.Fatalf("expected snapshot text slot to update to 'bravo', got %+v", dyn)
	}
	if len(tpl.TemplatePayload.SlotPaths) == 0 {
		t.Fatalf("expected slot path manifest in template payload, got %+v", tpl.TemplatePayload.SlotPaths)
	}
	if len(transport.frames) == 0 {
		t.Fatal("expected frame to accompany template")
	}
	frame := transport.frames[len(transport.frames)-1]
	if len(frame.Patch) != 0 {
		t.Fatalf("expected no diff operations, got %d", len(frame.Patch))
	}
}
