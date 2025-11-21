package diff

import (
	"reflect"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

func TestDiffTextChange(t *testing.T) {
	prev := dom.TextNode("hello")
	next := dom.TextNode("world")
	patches := Diff(prev, next)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText || patches[0].Value.(string) != "world" {
		t.Fatalf("unexpected patch %#v", patches[0])
	}
}

func TestDiffElementAttr(t *testing.T) {
	prev := dom.ElementNode("div").WithAttr("class", "foo")
	next := dom.ElementNode("div").WithAttr("class", "foo", "bar").WithAttr("id", "x")
	patches := Diff(prev, next)
	if len(patches) == 0 {
		t.Fatalf("expected patches")
	}
	foundSet := false
	for _, p := range patches {
		if p.Op == OpSetAttr {
			foundSet = true
			set := p.Value.(map[string][]string)
			if !reflect.DeepEqual(set["class"], []string{"foo", "bar"}) {
				t.Fatalf("unexpected class tokens %#v", set["class"])
			}
		}
	}
	if !foundSet {
		t.Fatalf("expected set attr op")
	}
}

func TestDiffChildAddition(t *testing.T) {
	prev := dom.ElementNode("div")
	next := dom.ElementNode("div").WithChildren(dom.TextNode("hi"))
	patches := Diff(prev, next)
	if len(patches) != 1 || patches[0].Op != OpAddChild {
		t.Fatalf("expected add child patch, got %#v", patches)
	}
}

func TestDiffStyle(t *testing.T) {
	prev := dom.ElementNode("div").WithStyle("color", "red").WithStyle("font-size", "14px")
	next := dom.ElementNode("div").WithStyle("color", "blue").WithStyle("font-size", "14px").WithStyle("background", "white")
	patches := Diff(prev, next)

	foundSet := false
	foundDel := false
	for _, p := range patches {
		if p.Op == OpSetStyle {
			foundSet = true
			set := p.Value.(map[string]string)
			if set["color"] != "blue" {
				t.Fatalf("expected color=blue, got %v", set)
			}
		}
	}
	if !foundSet {
		t.Fatalf("expected setStyle op")
	}
	if foundDel {
		t.Fatalf("did not expect delStyle op")
	}
}

func TestDiffStyleRemoval(t *testing.T) {
	prev := dom.ElementNode("div").WithStyle("color", "red").WithStyle("font-size", "14px")
	next := dom.ElementNode("div").WithStyle("color", "red")
	patches := Diff(prev, next)

	foundDel := false
	for _, p := range patches {
		if p.Op == OpDelStyle {
			foundDel = true
			if p.Name != "font-size" {
				t.Fatalf("expected font-size deleted, got %s", p.Name)
			}
		}
	}
	if !foundDel {
		t.Fatalf("expected delStyle op")
	}
}

func TestDiffRefID(t *testing.T) {
	prev := dom.ElementNode("div")
	next := dom.ElementNode("div").WithRef("myref")
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetRef {
			found = true
			if p.Value.(string) != "myref" {
				t.Fatalf("expected refID=myref, got %v", p.Value)
			}
		}
	}
	if !found {
		t.Fatalf("expected setRef op")
	}
}

func TestDiffRefIDRemoval(t *testing.T) {
	prev := dom.ElementNode("div").WithRef("myref")
	next := dom.ElementNode("div")
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpDelRef {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected delRef op")
	}
}

func TestDiffComponentNode(t *testing.T) {
	prev := dom.ComponentNode("comp1").WithChildren(
		dom.TextNode("hello"),
	)
	next := dom.ComponentNode("comp1").WithChildren(
		dom.TextNode("world"),
	)
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText {
		t.Fatalf("expected setText op, got %s", patches[0].Op)
	}
}

func TestDiffFragmentNode(t *testing.T) {
	prev := dom.FragmentNode(dom.TextNode("hello"))
	next := dom.FragmentNode(dom.TextNode("world"))
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText {
		t.Fatalf("expected setText op, got %s", patches[0].Op)
	}
}

func TestDiffTagMismatch(t *testing.T) {
	prev := dom.ElementNode("div")
	next := dom.ElementNode("span")
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpReplaceNode {
		t.Fatalf("expected replaceNode op, got %s", patches[0].Op)
	}
}

func TestDiffChildDeletion(t *testing.T) {
	prev := dom.ElementNode("div").WithChildren(
		dom.TextNode("child1"),
		dom.TextNode("child2"),
	)
	next := dom.ElementNode("div").WithChildren(
		dom.TextNode("child1"),
	)
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpDelChild {
			found = true
			if p.Index == nil || *p.Index != 1 {
				t.Fatalf("expected delChild at index 1, got %v", p.Index)
			}
		}
	}
	if !found {
		t.Fatalf("expected delChild op")
	}
}

func TestDiffAddChildIndex(t *testing.T) {
	prev := dom.ElementNode("div").WithChildren(
		dom.TextNode("child1"),
	)
	next := dom.ElementNode("div").WithChildren(
		dom.TextNode("child1"),
		dom.TextNode("child2"),
	)
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpAddChild {
			found = true
			if p.Index == nil || *p.Index != 1 {
				t.Fatalf("expected addChild at index 1, got %v", p.Index)
			}
		}
	}
	if !found {
		t.Fatalf("expected addChild op with Index field")
	}
}

func TestDiffKeyedReorder(t *testing.T) {
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
	)
	patches := Diff(prev, next)

	moveCount := 0
	for _, p := range patches {
		if p.Op == OpMoveChild {
			moveCount++
		}
		if p.Op == OpReplaceNode {
			t.Fatalf("should not replace nodes when using keys, got %#v", p)
		}
	}
	if moveCount == 0 {
		t.Fatalf("expected move operations for keyed reordering")
	}
}

func TestDiffKeyedAddition(t *testing.T) {
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	patches := Diff(prev, next)

	foundAdd := false
	for _, p := range patches {
		if p.Op == OpAddChild {
			foundAdd = true
			if p.Index == nil || *p.Index != 2 {
				t.Fatalf("expected addChild at index 2, got %v", p.Index)
			}
		}
	}
	if !foundAdd {
		t.Fatalf("expected addChild operation")
	}
}

func TestDiffKeyedDeletion(t *testing.T) {
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	patches := Diff(prev, next)

	foundDel := false
	for _, p := range patches {
		if p.Op == OpDelChild {
			foundDel = true
			val, ok := p.Value.(map[string]interface{})
			if ok && val["key"] != "b" {
				t.Fatalf("expected deletion of key 'b', got %v", val["key"])
			}
		}
	}
	if !foundDel {
		t.Fatalf("expected delChild operation")
	}
}

func TestDiffKeyedContentChange(t *testing.T) {
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Old text")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("New text")),
	)
	patches := Diff(prev, next)

	foundTextChange := false
	for _, p := range patches {
		if p.Op == OpSetText {
			foundTextChange = true
			if p.Value.(string) != "New text" {
				t.Fatalf("expected 'New text', got %v", p.Value)
			}
		}
		if p.Op == OpReplaceNode || p.Op == OpDelChild || p.Op == OpAddChild {
			t.Fatalf("keyed node should be diffed, not replaced: %#v", p)
		}
	}
	if !foundTextChange {
		t.Fatalf("expected setText operation for keyed node content change")
	}
}

func TestDiffMixedKeyedUnkeyed(t *testing.T) {
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Keyed A")),
		dom.ElementNode("li").WithChildren(dom.TextNode("Unkeyed 1")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Keyed B")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Keyed B")),
		dom.ElementNode("li").WithChildren(dom.TextNode("Unkeyed 2")),
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Keyed A")),
	)
	patches := Diff(prev, next)

	if len(patches) == 0 {
		t.Fatalf("expected patches for mixed keyed/unkeyed reordering")
	}

	foundMove := false
	for _, p := range patches {
		if p.Op == OpMoveChild {
			foundMove = true
		}
	}
	if !foundMove {
		t.Fatalf("expected move operations for keyed items")
	}
}

func TestDiffUnkeyedFallback(t *testing.T) {
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithChildren(dom.TextNode("Item 1")),
		dom.ElementNode("li").WithChildren(dom.TextNode("Item 2")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithChildren(dom.TextNode("Item 2")),
		dom.ElementNode("li").WithChildren(dom.TextNode("Item 1")),
	)
	patches := Diff(prev, next)

	for _, p := range patches {
		if p.Op == OpMoveChild {
			t.Fatalf("unkeyed children should not produce move operations")
		}
	}

	foundTextChange := false
	for _, p := range patches {
		if p.Op == OpSetText {
			foundTextChange = true
		}
	}
	if !foundTextChange {
		t.Fatalf("unkeyed reordering should produce setText operations")
	}
}

func TestDiffHandlers(t *testing.T) {
	prev := dom.ElementNode("button")
	prev.Handlers = []dom.HandlerMeta{
		{Event: "click"},
	}
	next := dom.ElementNode("button")
	next.Handlers = []dom.HandlerMeta{
		{Event: "click"},
		{Event: "mouseover"},
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetHandlers {
			found = true
			handlers := p.Value.([]dom.HandlerMeta)
			if len(handlers) != 2 {
				t.Fatalf("expected 2 handlers, got %d", len(handlers))
			}
		}
	}
	if !found {
		t.Fatalf("expected setHandlers op")
	}
}

func TestDiffRouter(t *testing.T) {
	prev := dom.ElementNode("a")
	next := dom.ElementNode("a")
	next.Router = &dom.RouterMeta{PathValue: "/home"}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetRouter {
			found = true
			router := p.Value.(*dom.RouterMeta)
			if router.PathValue != "/home" {
				t.Fatalf("expected PathValue=/home, got %s", router.PathValue)
			}
		}
	}
	if !found {
		t.Fatalf("expected setRouter op")
	}
}

func TestDiffRouterRemoval(t *testing.T) {
	prev := dom.ElementNode("a")
	prev.Router = &dom.RouterMeta{PathValue: "/home"}
	next := dom.ElementNode("a")
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpDelRouter {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected delRouter op")
	}
}

func TestDiffUpload(t *testing.T) {
	prev := dom.ElementNode("input")
	next := dom.ElementNode("input")
	next.Upload = &dom.UploadMeta{
		UploadID: "upload1",
		Accept:   []string{".jpg", ".png"},
		Multiple: true,
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetUpload {
			found = true
			upload := p.Value.(*dom.UploadMeta)
			if upload.UploadID != "upload1" {
				t.Fatalf("expected uploadID=upload1, got %s", upload.UploadID)
			}
		}
	}
	if !found {
		t.Fatalf("expected setUpload op")
	}
}

func TestDiffComment(t *testing.T) {
	prev := dom.CommentNode("old comment")
	next := dom.CommentNode("new comment")
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetComment {
		t.Fatalf("expected setComment op, got %s", patches[0].Op)
	}
	if patches[0].Value.(string) != "new comment" {
		t.Fatalf("expected 'new comment', got %v", patches[0].Value)
	}
}

func TestDiffComponentIDChange(t *testing.T) {
	prev := dom.ComponentNode("comp1")
	next := dom.ComponentNode("comp2")
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetComponent {
			found = true
			if p.Value.(string) != "comp2" {
				t.Fatalf("expected componentID=comp2, got %v", p.Value)
			}
		}
	}
	if !found {
		t.Fatalf("expected setComponent op for component ID change")
	}
}

func TestDiffNilStyleMaps(t *testing.T) {
	prev := dom.ElementNode("div")
	next := dom.ElementNode("div").WithStyle("color", "red")
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetStyle {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected setStyle op when adding styles to nil map")
	}
}

func TestDiffStylesNilMaps(t *testing.T) {
	prev := dom.ElementNode("style")
	next := dom.ElementNode("style")
	next.Styles = map[string]map[string]string{
		"card": {"color": "blue"},
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetStyleDecl {
			found = true
			if p.Selector != "card" || p.Name != "color" {
				t.Fatalf("unexpected style decl: selector=%s, name=%s", p.Selector, p.Name)
			}
		}
	}
	if !found {
		t.Fatalf("expected setStyleDecl op when adding styles to nil map")
	}
}

// ============================================================
// KEYED DIFFING EDGE CASE TESTS
// These test the specific scenarios that were previously broken
// ============================================================

func TestDiffKeyedSwap(t *testing.T) {
	// Swapping two keyed siblings should produce moves, not replacements
	// Old: [A, B] -> New: [B, A]
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
	)
	patches := Diff(prev, next)

	// Should have move operations, not replacements or add/deletes
	moveCount := 0
	for _, p := range patches {
		if p.Op == OpMoveChild {
			moveCount++
			val := p.Value.(map[string]interface{})
			key := val["key"].(string)
			newIdx := val["newIdx"].(int)
			// Verify the move is key-based with correct target index
			if (key == "b" && newIdx != 0) || (key == "a" && newIdx != 1) {
				t.Fatalf("unexpected move: key=%s newIdx=%d", key, newIdx)
			}
		}
		if p.Op == OpReplaceNode {
			t.Fatalf("swap should not produce replaceNode: %#v", p)
		}
		if p.Op == OpAddChild || p.Op == OpDelChild {
			t.Fatalf("swap should not produce add/del: %#v", p)
		}
	}
	if moveCount == 0 {
		t.Fatalf("expected move operations for keyed swap")
	}
}

func TestDiffKeyedPrepend(t *testing.T) {
	// Prepending a new keyed item before existing ones
	// Old: [A] -> New: [B, A]
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
	)
	patches := Diff(prev, next)

	foundAdd := false
	foundMove := false
	for _, p := range patches {
		if p.Op == OpAddChild {
			foundAdd = true
			if p.Index == nil || *p.Index != 0 {
				t.Fatalf("expected addChild at index 0, got %v", p.Index)
			}
		}
		if p.Op == OpMoveChild {
			foundMove = true
			val := p.Value.(map[string]interface{})
			key := val["key"].(string)
			newIdx := val["newIdx"].(int)
			if key != "a" || newIdx != 1 {
				t.Fatalf("expected move of 'a' to index 1, got key=%s newIdx=%d", key, newIdx)
			}
		}
		if p.Op == OpReplaceNode {
			t.Fatalf("prepend should not produce replaceNode: %#v", p)
		}
	}
	if !foundAdd {
		t.Fatalf("expected addChild operation for prepend")
	}
	if !foundMove {
		t.Fatalf("expected moveChild operation for existing item 'a'")
	}
}

func TestDiffKeyedInsertMiddle(t *testing.T) {
	// Inserting in the middle of a keyed list
	// Old: [A, C] -> New: [A, B, C]
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	patches := Diff(prev, next)

	foundAdd := false
	for _, p := range patches {
		if p.Op == OpAddChild {
			foundAdd = true
			if p.Index == nil || *p.Index != 1 {
				t.Fatalf("expected addChild at index 1, got %v", p.Index)
			}
		}
		if p.Op == OpReplaceNode {
			t.Fatalf("middle insert should not produce replaceNode: %#v", p)
		}
	}
	if !foundAdd {
		t.Fatalf("expected addChild operation for middle insert")
	}
}

func TestDiffKeyedDeleteFirst(t *testing.T) {
	// Deleting the first item from a keyed list
	// Old: [A, B, C] -> New: [B, C]
	// After deleting A at index 0, B and C naturally shift to indices 0 and 1
	// which matches their target positions - no moves needed!
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	patches := Diff(prev, next)

	foundDel := false
	for _, p := range patches {
		if p.Op == OpDelChild {
			foundDel = true
			// Deletion should be at index 0 (the first item)
			if p.Index == nil || *p.Index != 0 {
				t.Fatalf("expected delChild at index 0, got %v", p.Index)
			}
			val, ok := p.Value.(map[string]interface{})
			if ok && val["key"] != "a" {
				t.Fatalf("expected deletion of key 'a', got %v", val["key"])
			}
		}
		// No moves needed - after deletion, items naturally fall into correct positions
		if p.Op == OpReplaceNode || p.Op == OpAddChild {
			t.Fatalf("delete-first should not produce replace/add: %#v", p)
		}
	}
	if !foundDel {
		t.Fatalf("expected delChild operation")
	}
}

func TestDiffKeyedReverseOrder(t *testing.T) {
	// Reversing a list: [A, B, C] -> [C, B, A]
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
	)
	patches := Diff(prev, next)

	moveCount := 0
	for _, p := range patches {
		if p.Op == OpMoveChild {
			moveCount++
		}
		if p.Op == OpReplaceNode || p.Op == OpAddChild || p.Op == OpDelChild {
			t.Fatalf("reverse should only produce moves, got: %#v", p)
		}
	}
	if moveCount == 0 {
		t.Fatalf("expected move operations for list reversal")
	}
}

func TestDiffKeyedComplexReorder(t *testing.T) {
	// Complex reorder with additions and deletions
	// Old: [A, B, C, D] -> New: [D, E, B]
	// Delete: A, C; Add: E; Move: D to 0, B to 2
	prev := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("a").WithChildren(dom.TextNode("Item A")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
		dom.ElementNode("li").WithKey("c").WithChildren(dom.TextNode("Item C")),
		dom.ElementNode("li").WithKey("d").WithChildren(dom.TextNode("Item D")),
	)
	next := dom.ElementNode("ul").WithChildren(
		dom.ElementNode("li").WithKey("d").WithChildren(dom.TextNode("Item D")),
		dom.ElementNode("li").WithKey("e").WithChildren(dom.TextNode("Item E")),
		dom.ElementNode("li").WithKey("b").WithChildren(dom.TextNode("Item B")),
	)
	patches := Diff(prev, next)

	delCount := 0
	addCount := 0
	moveCount := 0
	deletedKeys := make(map[string]bool)

	for _, p := range patches {
		switch p.Op {
		case OpDelChild:
			delCount++
			if val, ok := p.Value.(map[string]interface{}); ok {
				if key, ok := val["key"].(string); ok {
					deletedKeys[key] = true
				}
			}
		case OpAddChild:
			addCount++
		case OpMoveChild:
			moveCount++
		case OpReplaceNode:
			t.Fatalf("complex reorder should not produce replaceNode: %#v", p)
		}
	}

	if delCount != 2 {
		t.Fatalf("expected 2 deletions, got %d", delCount)
	}
	if !deletedKeys["a"] || !deletedKeys["c"] {
		t.Fatalf("expected deletions of 'a' and 'c', got %v", deletedKeys)
	}
	if addCount != 1 {
		t.Fatalf("expected 1 addition, got %d", addCount)
	}
	if moveCount == 0 {
		t.Fatalf("expected move operations")
	}
}
