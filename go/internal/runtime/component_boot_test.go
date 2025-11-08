package runtime

import (
	"strings"
	"testing"

	h "github.com/eleven-am/pondlive/go/internal/html"
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
	if len(transport.frames) == 0 {
		t.Fatal("expected component boot frame to be sent")
	}
	frame := transport.frames[len(transport.frames)-1]
	if len(frame.Patch) != 0 {
		t.Fatalf("expected no diff operations, got %d", len(frame.Patch))
	}
	var bootEffect map[string]any
	for _, eff := range frame.Effects {
		if effMap, ok := eff.(map[string]any); ok {
			if effType, okType := effMap["type"].(string); okType && effType == "componentBoot" {
				bootEffect = effMap
				break
			}
		}
	}
	if bootEffect == nil {
		t.Fatalf("expected componentBoot effect, got %+v", frame.Effects)
	}
	idValue, _ := bootEffect["componentId"].(string)
	if idValue == "" || idValue != componentID {
		t.Fatalf("unexpected component id in effect: %q (expected %q)", idValue, componentID)
	}
	htmlValue, _ := bootEffect["html"].(string)
	if htmlValue == "" || !strings.Contains(htmlValue, "bravo") {
		t.Fatalf("expected html payload to contain updated text, got %q", htmlValue)
	}
	slotsValue, ok := bootEffect["slots"].([]int)
	if !ok || len(slotsValue) == 0 {
		t.Fatalf("expected slot list in effect, got %#v", bootEffect["slots"])
	}
	slotIndex := slotsValue[len(slotsValue)-1]
	if slotIndex < 0 || slotIndex >= len(sess.snapshot.Dynamics) {
		t.Fatalf("expected slot to map into snapshot, got %d", slotIndex)
	}
	dyn := sess.snapshot.Dynamics[slotIndex]
	if dyn.Kind != "text" || dyn.Text != "bravo" {
		t.Fatalf("expected snapshot text slot to update to 'bravo', got %+v", dyn)
	}
}
