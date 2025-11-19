package diff

import (
	"reflect"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom2"
)

func TestDiffTextChange(t *testing.T) {
	prev := dom2.TextNode("hello")
	next := dom2.TextNode("world")
	patches := Diff(prev, next)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText || patches[0].Value.(string) != "world" {
		t.Fatalf("unexpected patch %#v", patches[0])
	}
}

func TestDiffElementAttr(t *testing.T) {
	prev := dom2.ElementNode("div").WithAttr("class", "foo")
	next := dom2.ElementNode("div").WithAttr("class", "foo", "bar").WithAttr("id", "x")
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
	prev := dom2.ElementNode("div")
	next := dom2.ElementNode("div").WithChildren(dom2.TextNode("hi"))
	patches := Diff(prev, next)
	if len(patches) != 1 || patches[0].Op != OpAddChild {
		t.Fatalf("expected add child patch, got %#v", patches)
	}
}

func TestDiffStyle(t *testing.T) {
	prev := dom2.ElementNode("div").WithStyle("color", "red").WithStyle("font-size", "14px")
	next := dom2.ElementNode("div").WithStyle("color", "blue").WithStyle("font-size", "14px").WithStyle("background", "white")
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
	prev := dom2.ElementNode("div").WithStyle("color", "red").WithStyle("font-size", "14px")
	next := dom2.ElementNode("div").WithStyle("color", "red")
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
	prev := dom2.ElementNode("div")
	next := dom2.ElementNode("div").WithRef("myref")
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
	prev := dom2.ElementNode("div").WithRef("myref")
	next := dom2.ElementNode("div")
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
	prev := dom2.ComponentNode("comp1").WithChildren(
		dom2.TextNode("hello"),
	)
	next := dom2.ComponentNode("comp1").WithChildren(
		dom2.TextNode("world"),
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
	prev := dom2.FragmentNode(dom2.TextNode("hello"))
	next := dom2.FragmentNode(dom2.TextNode("world"))
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText {
		t.Fatalf("expected setText op, got %s", patches[0].Op)
	}
}

func TestDiffTagMismatch(t *testing.T) {
	prev := dom2.ElementNode("div")
	next := dom2.ElementNode("span")
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpReplaceNode {
		t.Fatalf("expected replaceNode op, got %s", patches[0].Op)
	}
}

func TestDiffChildDeletion(t *testing.T) {
	prev := dom2.ElementNode("div").WithChildren(
		dom2.TextNode("child1"),
		dom2.TextNode("child2"),
	)
	next := dom2.ElementNode("div").WithChildren(
		dom2.TextNode("child1"),
	)
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpDelChild {
			found = true
			if p.Index != 1 {
				t.Fatalf("expected delChild at index 1, got %d", p.Index)
			}
		}
	}
	if !found {
		t.Fatalf("expected delChild op")
	}
}

func TestDiffAddChildIndex(t *testing.T) {
	prev := dom2.ElementNode("div").WithChildren(
		dom2.TextNode("child1"),
	)
	next := dom2.ElementNode("div").WithChildren(
		dom2.TextNode("child1"),
		dom2.TextNode("child2"),
	)
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpAddChild {
			found = true
			if p.Index != 1 {
				t.Fatalf("expected addChild at index 1, got %d", p.Index)
			}
		}
	}
	if !found {
		t.Fatalf("expected addChild op with Index field")
	}
}

func TestDiffKeyedReorder(t *testing.T) {
	prev := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Item A")),
		dom2.ElementNode("li").WithKey("b").WithChildren(dom2.TextNode("Item B")),
		dom2.ElementNode("li").WithKey("c").WithChildren(dom2.TextNode("Item C")),
	)
	next := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("c").WithChildren(dom2.TextNode("Item C")),
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Item A")),
		dom2.ElementNode("li").WithKey("b").WithChildren(dom2.TextNode("Item B")),
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
	prev := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Item A")),
		dom2.ElementNode("li").WithKey("b").WithChildren(dom2.TextNode("Item B")),
	)
	next := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Item A")),
		dom2.ElementNode("li").WithKey("b").WithChildren(dom2.TextNode("Item B")),
		dom2.ElementNode("li").WithKey("c").WithChildren(dom2.TextNode("Item C")),
	)
	patches := Diff(prev, next)

	foundAdd := false
	for _, p := range patches {
		if p.Op == OpAddChild {
			foundAdd = true
			if p.Index != 2 {
				t.Fatalf("expected addChild at index 2, got %d", p.Index)
			}
		}
	}
	if !foundAdd {
		t.Fatalf("expected addChild operation")
	}
}

func TestDiffKeyedDeletion(t *testing.T) {
	prev := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Item A")),
		dom2.ElementNode("li").WithKey("b").WithChildren(dom2.TextNode("Item B")),
		dom2.ElementNode("li").WithKey("c").WithChildren(dom2.TextNode("Item C")),
	)
	next := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Item A")),
		dom2.ElementNode("li").WithKey("c").WithChildren(dom2.TextNode("Item C")),
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
	prev := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Old text")),
	)
	next := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("New text")),
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
	prev := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Keyed A")),
		dom2.ElementNode("li").WithChildren(dom2.TextNode("Unkeyed 1")),
		dom2.ElementNode("li").WithKey("b").WithChildren(dom2.TextNode("Keyed B")),
	)
	next := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithKey("b").WithChildren(dom2.TextNode("Keyed B")),
		dom2.ElementNode("li").WithChildren(dom2.TextNode("Unkeyed 2")),
		dom2.ElementNode("li").WithKey("a").WithChildren(dom2.TextNode("Keyed A")),
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
	prev := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithChildren(dom2.TextNode("Item 1")),
		dom2.ElementNode("li").WithChildren(dom2.TextNode("Item 2")),
	)
	next := dom2.ElementNode("ul").WithChildren(
		dom2.ElementNode("li").WithChildren(dom2.TextNode("Item 2")),
		dom2.ElementNode("li").WithChildren(dom2.TextNode("Item 1")),
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
	prev := dom2.ElementNode("button")
	prev.Handlers = []dom2.HandlerMeta{
		{Event: "click", Handler: "h1"},
	}
	next := dom2.ElementNode("button")
	next.Handlers = []dom2.HandlerMeta{
		{Event: "click", Handler: "h1"},
		{Event: "mouseover", Handler: "h2"},
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetHandlers {
			found = true
			handlers := p.Value.([]dom2.HandlerMeta)
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
	prev := dom2.ElementNode("a")
	next := dom2.ElementNode("a")
	next.Router = &dom2.RouterMeta{PathValue: "/home"}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetRouter {
			found = true
			router := p.Value.(*dom2.RouterMeta)
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
	prev := dom2.ElementNode("a")
	prev.Router = &dom2.RouterMeta{PathValue: "/home"}
	next := dom2.ElementNode("a")
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
	prev := dom2.ElementNode("input")
	next := dom2.ElementNode("input")
	next.Upload = &dom2.UploadMeta{
		UploadID: "upload1",
		Accept:   []string{".jpg", ".png"},
		Multiple: true,
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetUpload {
			found = true
			upload := p.Value.(*dom2.UploadMeta)
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
	prev := dom2.CommentNode("old comment")
	next := dom2.CommentNode("new comment")
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
	prev := dom2.ComponentNode("comp1")
	next := dom2.ComponentNode("comp2")
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
	prev := dom2.ElementNode("div")
	next := dom2.ElementNode("div").WithStyle("color", "red")
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
	prev := dom2.ElementNode("style")
	next := dom2.ElementNode("style")
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
