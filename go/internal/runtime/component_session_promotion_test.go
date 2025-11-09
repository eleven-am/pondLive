package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/diff"
	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestComponentSessionPromotesStaticText(t *testing.T) {
	type parentProps struct{}
	type childProps struct{}
	var setValue func(string)
	child := func(ctx Ctx, _ childProps) h.Node {
		value, set := UseState(ctx, "alpha")
		setValue = func(next string) { set(next) }
		return h.Div(h.Text(value()))
	}
	parent := func(ctx Ctx, _ parentProps) h.Node {
		return h.Div(Render(ctx, child, childProps{}))
	}

	sess := NewSession(parent, parentProps{})
	structured := sess.InitialStructured()
	if len(structured.Components) == 0 {
		t.Fatalf("expected structured components to be populated")
	}
	t.Logf("initial components: %v", mapKeys(structured.Components))
	if setValue == nil {
		t.Fatalf("expected child setter to be initialized")
	}
	for _, dyn := range structured.D {
		if dyn.Kind == render.DynText {
			t.Fatalf("expected initial render to avoid dynamic text slots, got %+v", structured.D)
		}
	}

	var lastOps []diff.Op
	sess.SetPatchSender(func(ops []diff.Op) error {
		lastOps = append([]diff.Op(nil), ops...)
		return nil
	})

	setValue("bravo")
	lastOps = nil
	if err := sess.Flush(); err != nil {
		t.Fatalf("promotion flush error: %v", err)
	}
	boots := sess.consumeComponentBoots()
	t.Logf("component boots: %d", len(boots))
	if len(boots) != 1 {
		update := sess.consumeTemplateUpdate()
		t.Fatalf("expected exactly one component boot, got %d (template update=%v)", len(boots), update != nil)
	}
	update := sess.consumeTemplateUpdate()
	if update == nil {
		t.Fatal("expected template update during promotion")
	}
	sawPromotedText := false
	for _, dyn := range update.structured.D {
		if dyn.Kind == render.DynText && dyn.Text == "bravo" {
			sawPromotedText = true
			break
		}
	}
	if !sawPromotedText {
		t.Fatalf("expected template update to include promoted text slot, got %+v", update.structured.D)
	}
	boot := boots[0]
	hasTextSlot := false
	for _, slot := range boot.dynamics {
		if slot.Kind == "text" {
			hasTextSlot = true
			if slot.Text != "bravo" {
				t.Fatalf("expected promoted slot text 'bravo', got %q", slot.Text)
			}
		}
	}
	if !hasTextSlot {
		t.Fatalf("expected promoted template to include text slot, got %+v", boot.dynamics)
	}
	if sess.consumeTemplateUpdate() != nil {
		t.Fatalf("expected template update to be consumed")
	}
	foundPrevText := false
	for _, dyn := range sess.prev.D {
		if dyn.Kind == render.DynText && dyn.Text == "bravo" {
			foundPrevText = true
			break
		}
	}
	if !foundPrevText {
		t.Fatalf("expected session state to retain promoted text slot, got %+v", sess.prev.D)
	}

	setValue("charlie")
	lastOps = nil
	if err := sess.Flush(); err != nil {
		t.Fatalf("second update flush error: %v", err)
	}
	boots = sess.consumeComponentBoots()
	if len(boots) != 0 {
		t.Fatalf("expected no component boot after promotion, got %d", len(boots))
	}
	if len(lastOps) == 0 {
		t.Fatalf("expected diff operations for dynamic update")
	}
	sawSetText := false
	for _, op := range lastOps {
		if setText, ok := op.(diff.SetText); ok && setText.Text == "charlie" {
			sawSetText = true
			break
		}
	}
	if !sawSetText {
		t.Fatalf("expected SetText operation for 'charlie', got %+v", lastOps)
	}
}

func mapKeys[T any](m map[string]T) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func TestComponentSessionPromotesStaticAttrs(t *testing.T) {
	type parentProps struct{}
	type childProps struct{}
	var setClass func(string)
	child := func(ctx Ctx, _ childProps) h.Node {
		value, set := UseState(ctx, "alpha")
		setClass = func(next string) { set(next) }
		return h.Div(h.Attr("class", value()))
	}
	parent := func(ctx Ctx, _ parentProps) h.Node {
		return h.Div(Render(ctx, child, childProps{}))
	}

	sess := NewSession(parent, parentProps{})
	structured := sess.InitialStructured()
	for _, dyn := range structured.D {
		if dyn.Kind == render.DynAttrs {
			t.Fatalf("expected initial render to avoid dynamic attr slots, got %+v", structured.D)
		}
	}
	if setClass == nil {
		t.Fatalf("expected class setter to be captured")
	}

	var lastOps []diff.Op
	sess.SetPatchSender(func(ops []diff.Op) error {
		lastOps = append([]diff.Op(nil), ops...)
		return nil
	})

	setClass("bravo")
	lastOps = nil
	if err := sess.Flush(); err != nil {
		t.Fatalf("promotion flush error: %v", err)
	}
	boots := sess.consumeComponentBoots()
	if len(boots) != 1 {
		update := sess.consumeTemplateUpdate()
		t.Fatalf("expected exactly one component boot, got %d (template update=%v)", len(boots), update != nil)
	}
	update := sess.consumeTemplateUpdate()
	if update == nil {
		t.Fatal("expected template update during attr promotion")
	}
	sawPromotedAttrs := false
	for _, dyn := range update.structured.D {
		if dyn.Kind != render.DynAttrs {
			continue
		}
		if dyn.Attrs["class"] == "bravo" {
			sawPromotedAttrs = true
			break
		}
	}
	if !sawPromotedAttrs {
		t.Fatalf("expected template update to include promoted attrs, got %+v", update.structured.D)
	}
	boot := boots[0]
	hasAttrSlot := false
	for _, slot := range boot.dynamics {
		if slot.Kind != "attrs" {
			continue
		}
		hasAttrSlot = true
		if slot.Attrs["class"] != "bravo" {
			t.Fatalf("expected promoted attrs to include class 'bravo', got %+v", slot.Attrs)
		}
	}
	if !hasAttrSlot {
		t.Fatalf("expected promoted template to include attr slot, got %+v", boot.dynamics)
	}
	if sess.consumeTemplateUpdate() != nil {
		t.Fatalf("expected template update to be consumed")
	}
	foundPrevAttrs := false
	for _, dyn := range sess.prev.D {
		if dyn.Kind == render.DynAttrs && dyn.Attrs["class"] == "bravo" {
			foundPrevAttrs = true
			break
		}
	}
	if !foundPrevAttrs {
		t.Fatalf("expected session state to retain promoted attrs, got %+v", sess.prev.D)
	}

	setClass("charlie")
	lastOps = nil
	if err := sess.Flush(); err != nil {
		t.Fatalf("second update flush error: %v", err)
	}
	boots = sess.consumeComponentBoots()
	if len(boots) != 0 {
		t.Fatalf("expected no component boot after attr promotion, got %d", len(boots))
	}
	sawSetAttrs := false
	for _, op := range lastOps {
		if setAttrs, ok := op.(diff.SetAttrs); ok {
			if setAttrs.Upsert != nil && setAttrs.Upsert["class"] == "charlie" {
				sawSetAttrs = true
				break
			}
		}
	}
	if !sawSetAttrs {
		t.Fatalf("expected SetAttrs operation for 'charlie', got %+v", lastOps)
	}
}
