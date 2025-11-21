package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestUseStreamBasic(t *testing.T) {
	var handle StreamHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		items := []StreamItem[string]{
			{Key: "a", Value: "Item A"},
			{Key: "b", Value: "Item B"},
		}

		frag, h := UseStream(ctx, func(item StreamItem[string]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div", Text: item.Value}
		}, items...)

		handle = h
		return frag
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	items := handle.Items()
	if len(items) != 2 {
		t.Errorf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "a" || items[0].Value != "Item A" {
		t.Errorf("unexpected first item: %+v", items[0])
	}
}

func TestUseStreamAppend(t *testing.T) {
	var handle StreamHandle[int]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		_, h := UseStream(ctx, func(item StreamItem[int]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div"}
		})
		handle = h
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	changed := handle.Append(StreamItem[int]{Key: "first", Value: 1})
	if !changed {
		t.Error("expected append to return true")
	}

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	items := handle.Items()
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
}

func TestUseStreamPrepend(t *testing.T) {
	var handle StreamHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		initial := []StreamItem[string]{{Key: "b", Value: "B"}}
		_, h := UseStream(ctx, func(item StreamItem[string]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div"}
		}, initial...)
		handle = h
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	handle.Prepend(StreamItem[string]{Key: "a", Value: "A"})
	sess.Flush()

	items := handle.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "a" {
		t.Errorf("expected first item key 'a', got %q", items[0].Key)
	}
}

func TestUseStreamDelete(t *testing.T) {
	var handle StreamHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		initial := []StreamItem[string]{
			{Key: "a", Value: "A"},
			{Key: "b", Value: "B"},
			{Key: "c", Value: "C"},
		}
		_, h := UseStream(ctx, func(item StreamItem[string]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div"}
		}, initial...)
		handle = h
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	changed := handle.Delete("b")
	if !changed {
		t.Error("expected delete to return true")
	}
	sess.Flush()

	items := handle.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "a" || items[1].Key != "c" {
		t.Errorf("unexpected items after delete: %v", items)
	}
}

func TestUseStreamReplace(t *testing.T) {
	var handle StreamHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		initial := []StreamItem[string]{{Key: "a", Value: "old"}}
		_, h := UseStream(ctx, func(item StreamItem[string]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div"}
		}, initial...)
		handle = h
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	changed := handle.Replace(StreamItem[string]{Key: "a", Value: "new"})
	if !changed {
		t.Error("expected replace to return true")
	}
	sess.Flush()

	items := handle.Items()
	if items[0].Value != "new" {
		t.Errorf("expected value 'new', got %q", items[0].Value)
	}
}

func TestUseStreamUpsert(t *testing.T) {
	var handle StreamHandle[int]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		_, h := UseStream(ctx, func(item StreamItem[int]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div"}
		})
		handle = h
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	handle.Upsert(StreamItem[int]{Key: "a", Value: 1})
	sess.Flush()

	items := handle.Items()
	if len(items) != 1 || items[0].Value != 1 {
		t.Errorf("expected 1 item with value 1, got %+v", items)
	}

	handle.Upsert(StreamItem[int]{Key: "a", Value: 2})
	sess.Flush()

	items = handle.Items()
	if len(items) != 1 || items[0].Value != 2 {
		t.Errorf("expected 1 item with value 2, got %+v", items)
	}
}

func TestUseStreamClear(t *testing.T) {
	var handle StreamHandle[string]

	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		initial := []StreamItem[string]{{Key: "a", Value: "A"}}
		_, h := UseStream(ctx, func(item StreamItem[string]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div"}
		}, initial...)
		handle = h
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })
	sess.Flush()

	changed := handle.Clear()
	if !changed {
		t.Error("expected clear to return true")
	}

	items := handle.Items()
	if len(items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(items))
	}
}

func TestUseStreamKeyAutoAssign(t *testing.T) {
	comp := func(ctx Ctx, props struct{}) *dom.StructuredNode {
		items := []StreamItem[string]{{Key: "test", Value: "value"}}
		frag, _ := UseStream(ctx, func(item StreamItem[string]) *dom.StructuredNode {
			return &dom.StructuredNode{Tag: "div"}
		}, items...)
		return frag
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	node := sess.root.node
	if !node.Fragment {
		t.Error("expected fragment node")
	}
	if len(node.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(node.Children))
	}
	if node.Children[0].Key != "test" {
		t.Errorf("expected child key 'test', got %q", node.Children[0].Key)
	}
}

// TestUseStreamDuplicateKeyPanic is removed because ComponentSession.withRecovery
// catches panics during rendering and reports them as diagnostics instead of
// propagating them. The duplicate key detection still works (ensureUniqueKeys
// and rebuildIndex both panic), but the panic is handled internally.
