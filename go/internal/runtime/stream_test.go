package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/work"
)

func makeStreamCtx() (*Ctx, *Instance, *Session) {
	inst := &Instance{
		ID:        "test-comp",
		HookFrame: []HookSlot{},
	}
	sess := &Session{
		DirtyQueue: []*Instance{},
		DirtySet:   make(map[*Instance]struct{}),
	}
	ctx := &Ctx{instance: inst, session: sess, hookIndex: 0}
	return ctx, inst, sess
}

// TestUseStreamBasic verifies basic UseStream functionality.
func TestUseStreamBasic(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	items := []StreamItem[string]{
		{Key: "a", Value: "Item A"},
		{Key: "b", Value: "Item B"},
	}

	frag, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, items...)

	if frag == nil {
		t.Fatal("UseStream returned nil fragment")
	}

	result := handle.Items()
	if len(result) != 2 {
		t.Errorf("expected 2 items, got %d", len(result))
	}
	if result[0].Key != "a" || result[0].Value != "Item A" {
		t.Errorf("unexpected first item: %+v", result[0])
	}
}

// TestUseStreamAppend verifies Append adds items to end.
func TestUseStreamAppend(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	_, handle := UseStream(ctx, func(item StreamItem[int]) work.Node {
		return &work.Element{Tag: "div"}
	})

	changed := handle.Append(StreamItem[int]{Key: "first", Value: 1})
	if !changed {
		t.Error("expected append to return true")
	}

	items := handle.Items()
	if len(items) != 1 {
		t.Errorf("expected 1 item, got %d", len(items))
	}
	if items[0].Key != "first" || items[0].Value != 1 {
		t.Errorf("unexpected item: %+v", items[0])
	}
}

// TestUseStreamPrepend verifies Prepend adds items to beginning.
func TestUseStreamPrepend(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{{Key: "b", Value: "B"}}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	handle.Prepend(StreamItem[string]{Key: "a", Value: "A"})

	items := handle.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "a" {
		t.Errorf("expected first item key 'a', got %q", items[0].Key)
	}
}

// TestUseStreamDelete verifies Delete removes items by key.
func TestUseStreamDelete(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{
		{Key: "a", Value: "A"},
		{Key: "b", Value: "B"},
		{Key: "c", Value: "C"},
	}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	changed := handle.Delete("b")
	if !changed {
		t.Error("expected delete to return true")
	}

	items := handle.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "a" || items[1].Key != "c" {
		t.Errorf("unexpected items after delete: %v", items)
	}
}

// TestUseStreamDeleteNonexistent verifies Delete returns false for missing keys.
func TestUseStreamDeleteNonexistent(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	})

	changed := handle.Delete("nonexistent")
	if changed {
		t.Error("expected delete to return false for nonexistent key")
	}
}

// TestUseStreamReplace verifies Replace updates existing items.
func TestUseStreamReplace(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{{Key: "a", Value: "old"}}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	changed := handle.Replace(StreamItem[string]{Key: "a", Value: "new"})
	if !changed {
		t.Error("expected replace to return true")
	}

	items := handle.Items()
	if items[0].Value != "new" {
		t.Errorf("expected value 'new', got %q", items[0].Value)
	}
}

// TestUseStreamReplaceNonexistent verifies Replace returns false for missing keys.
func TestUseStreamReplaceNonexistent(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	})

	changed := handle.Replace(StreamItem[string]{Key: "nonexistent", Value: "value"})
	if changed {
		t.Error("expected replace to return false for nonexistent key")
	}
}

// TestUseStreamUpsert verifies Upsert inserts or updates items.
func TestUseStreamUpsert(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	_, handle := UseStream(ctx, func(item StreamItem[int]) work.Node {
		return &work.Element{Tag: "div"}
	})

	handle.Upsert(StreamItem[int]{Key: "a", Value: 1})
	items := handle.Items()
	if len(items) != 1 || items[0].Value != 1 {
		t.Errorf("expected 1 item with value 1, got %+v", items)
	}

	handle.Upsert(StreamItem[int]{Key: "a", Value: 2})
	items = handle.Items()
	if len(items) != 1 || items[0].Value != 2 {
		t.Errorf("expected 1 item with value 2, got %+v", items)
	}
}

// TestUseStreamClear verifies Clear removes all items.
func TestUseStreamClear(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{{Key: "a", Value: "A"}}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	changed := handle.Clear()
	if !changed {
		t.Error("expected clear to return true")
	}

	items := handle.Items()
	if len(items) != 0 {
		t.Errorf("expected 0 items after clear, got %d", len(items))
	}
}

// TestUseStreamClearEmpty verifies Clear returns false when already empty.
func TestUseStreamClearEmpty(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	})

	changed := handle.Clear()
	if changed {
		t.Error("expected clear to return false when empty")
	}
}

// TestUseStreamReset verifies Reset replaces all items.
func TestUseStreamReset(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{{Key: "a", Value: "A"}}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	newItems := []StreamItem[string]{
		{Key: "x", Value: "X"},
		{Key: "y", Value: "Y"},
	}
	changed := handle.Reset(newItems)
	if !changed {
		t.Error("expected reset to return true")
	}

	items := handle.Items()
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
	if items[0].Key != "x" || items[1].Key != "y" {
		t.Errorf("unexpected items: %+v", items)
	}
}

// TestUseStreamResetSame verifies Reset returns false when items unchanged.
func TestUseStreamResetSame(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{{Key: "a", Value: "A"}}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	changed := handle.Reset([]StreamItem[string]{{Key: "a", Value: "A"}})
	if changed {
		t.Error("expected reset to return false when items unchanged")
	}
}

// TestUseStreamInsertBefore verifies InsertBefore inserts at correct position.
func TestUseStreamInsertBefore(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{
		{Key: "a", Value: "A"},
		{Key: "c", Value: "C"},
	}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	handle.InsertBefore("c", StreamItem[string]{Key: "b", Value: "B"})

	items := handle.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[1].Key != "b" {
		t.Errorf("expected middle item key 'b', got %q", items[1].Key)
	}
}

// TestUseStreamInsertAfter verifies InsertAfter inserts at correct position.
func TestUseStreamInsertAfter(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{
		{Key: "a", Value: "A"},
		{Key: "c", Value: "C"},
	}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	handle.InsertAfter("a", StreamItem[string]{Key: "b", Value: "B"})

	items := handle.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}
	if items[1].Key != "b" {
		t.Errorf("expected middle item key 'b', got %q", items[1].Key)
	}
}

// TestUseStreamKeyAutoAssign verifies keys are applied to rendered elements.
func TestUseStreamKeyAutoAssign(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	items := []StreamItem[string]{{Key: "test", Value: "value"}}
	frag, _ := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, items...)

	fragment, ok := frag.(*work.Fragment)
	if !ok {
		t.Fatal("expected Fragment node")
	}
	if len(fragment.Children) != 1 {
		t.Fatalf("expected 1 child, got %d", len(fragment.Children))
	}
	elem, ok := fragment.Children[0].(*work.Element)
	if !ok {
		t.Fatal("expected Element child")
	}
	if elem.Key != "test" {
		t.Errorf("expected child key 'test', got %q", elem.Key)
	}
}

// TestUseStreamEmptyFragment verifies empty stream returns empty fragment.
func TestUseStreamEmptyFragment(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	frag, _ := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	})

	fragment, ok := frag.(*work.Fragment)
	if !ok {
		t.Fatal("expected Fragment node")
	}
	if len(fragment.Children) != 0 {
		t.Errorf("expected 0 children, got %d", len(fragment.Children))
	}
}

// TestUseStreamPanicNilRenderer verifies panic when renderer is nil.
func TestUseStreamPanicNilRenderer(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil renderer")
		}
	}()

	UseStream[string](ctx, nil)
}

// TestUseStreamPanicEmptyKey verifies panic when item has empty key.
func TestUseStreamPanicEmptyKey(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for empty key")
		}
	}()

	items := []StreamItem[string]{{Key: "", Value: "value"}}
	UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, items...)
}

// TestUseStreamPanicDuplicateKey verifies panic when items have duplicate keys.
func TestUseStreamPanicDuplicateKey(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate keys")
		}
	}()

	items := []StreamItem[string]{
		{Key: "a", Value: "A"},
		{Key: "a", Value: "B"},
	}
	UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, items...)
}

// TestUseStreamPanicNilRendererResult verifies panic when renderer returns nil.
func TestUseStreamPanicNilRendererResult(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil renderer result")
		}
	}()

	items := []StreamItem[string]{{Key: "a", Value: "A"}}
	UseStream(ctx, func(item StreamItem[string]) work.Node {
		return nil
	}, items...)
}

// TestUseStreamPanicConflictingKey verifies panic when renderer sets conflicting key.
func TestUseStreamPanicConflictingKey(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for conflicting key")
		}
	}()

	items := []StreamItem[string]{{Key: "a", Value: "A"}}
	UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div", Key: "different"}
	}, items...)
}

// TestUseStreamPanicOutsideRender verifies panic when called with nil ctx.
func TestUseStreamPanicOutsideRender(t *testing.T) {
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for nil ctx")
		}
	}()

	UseStream[string](nil, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	})
}

// TestUseStreamMoveExistingKey verifies inserting existing key moves it.
func TestUseStreamMoveExistingKey(t *testing.T) {
	ctx, _, _ := makeStreamCtx()

	initial := []StreamItem[string]{
		{Key: "a", Value: "A"},
		{Key: "b", Value: "B"},
		{Key: "c", Value: "C"},
	}
	_, handle := UseStream(ctx, func(item StreamItem[string]) work.Node {
		return &work.Element{Tag: "div"}
	}, initial...)

	handle.InsertAfter("c", StreamItem[string]{Key: "a", Value: "A"})

	items := handle.Items()
	if len(items) != 3 {
		t.Fatalf("expected 3 items, got %d", len(items))
	}

	if items[0].Key != "b" || items[1].Key != "c" || items[2].Key != "a" {
		t.Errorf("unexpected order: %v", items)
	}
}
