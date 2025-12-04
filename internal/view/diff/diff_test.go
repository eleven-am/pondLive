package diff

import (
	"reflect"
	"testing"

	"github.com/eleven-am/pondlive/internal/metadata"
	"github.com/eleven-am/pondlive/internal/view"
)

func init() {

	OnDuplicateKey = nil
	PanicOnDuplicateKey = false
}

func textNode(text string) *view.Text {
	return &view.Text{Text: text}
}

func commentNode(comment string) *view.Comment {
	return &view.Comment{Comment: comment}
}

func elementNode(tag string) *view.Element {
	return &view.Element{Tag: tag}
}

func withChildren(el *view.Element, children ...view.Node) *view.Element {
	el.Children = children
	return el
}

func withAttr(el *view.Element, name string, values ...string) *view.Element {
	if el.Attrs == nil {
		el.Attrs = make(map[string][]string)
	}
	el.Attrs[name] = values
	return el
}

func withStyle(el *view.Element, name, value string) *view.Element {
	if el.Style == nil {
		el.Style = make(map[string]string)
	}
	el.Style[name] = value
	return el
}

func withKey(el *view.Element, key string) *view.Element {
	el.Key = key
	return el
}

func withRef(el *view.Element, refID string) *view.Element {
	el.RefID = refID
	return el
}

func fragmentNode(children ...view.Node) *view.Fragment {
	return &view.Fragment{Fragment: true, Children: children}
}

func TestDiffTextChange(t *testing.T) {
	prev := textNode("hello")
	next := textNode("world")
	patches := Diff(prev, next)
	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText || patches[0].Value.(string) != "world" {
		t.Fatalf("unexpected patch %#v", patches[0])
	}
}

func TestDiffElementAttr(t *testing.T) {
	prev := withAttr(elementNode("div"), "class", "foo")
	next := withAttr(withAttr(elementNode("div"), "class", "foo", "bar"), "id", "x")
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
	prev := elementNode("div")
	next := withChildren(elementNode("div"), textNode("hi"))
	patches := Diff(prev, next)
	if len(patches) != 1 || patches[0].Op != OpAddChild {
		t.Fatalf("expected add child patch, got %#v", patches)
	}
}

func TestDiffStyle(t *testing.T) {
	prev := withStyle(withStyle(elementNode("div"), "color", "red"), "font-size", "14px")
	next := withStyle(withStyle(withStyle(elementNode("div"), "color", "blue"), "font-size", "14px"), "background", "white")
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
		if p.Op == OpDelStyle {
			foundDel = true
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
	prev := withStyle(withStyle(elementNode("div"), "color", "red"), "font-size", "14px")
	next := withStyle(elementNode("div"), "color", "red")
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
	prev := elementNode("div")
	next := withRef(elementNode("div"), "myref")
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
	prev := withRef(elementNode("div"), "myref")
	next := elementNode("div")
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

func TestDiffFragmentNode(t *testing.T) {
	prev := fragmentNode(textNode("hello"))
	next := fragmentNode(textNode("world"))
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText {
		t.Fatalf("expected setText op, got %s", patches[0].Op)
	}
}

func TestDiffTagMismatch(t *testing.T) {
	prev := elementNode("div")
	next := elementNode("span")
	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpReplaceNode {
		t.Fatalf("expected replaceNode op, got %s", patches[0].Op)
	}
}

func TestDiffChildDeletion(t *testing.T) {
	prev := withChildren(elementNode("div"),
		textNode("child1"),
		textNode("child2"),
	)
	next := withChildren(elementNode("div"),
		textNode("child1"),
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
	prev := withChildren(elementNode("div"),
		textNode("child1"),
	)
	next := withChildren(elementNode("div"),
		textNode("child1"),
		textNode("child2"),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Old text")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("New text")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Keyed A")),
		withChildren(elementNode("li"), textNode("Unkeyed 1")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Keyed B")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "b"), textNode("Keyed B")),
		withChildren(elementNode("li"), textNode("Unkeyed 2")),
		withChildren(withKey(elementNode("li"), "a"), textNode("Keyed A")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(elementNode("li"), textNode("Item 1")),
		withChildren(elementNode("li"), textNode("Item 2")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(elementNode("li"), textNode("Item 2")),
		withChildren(elementNode("li"), textNode("Item 1")),
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
	prev := elementNode("button")
	prev.Handlers = []metadata.HandlerMeta{
		{Event: "click"},
	}
	next := elementNode("button")
	next.Handlers = []metadata.HandlerMeta{
		{Event: "click"},
		{Event: "mouseover"},
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetHandlers {
			found = true
			handlers := p.Value.([]metadata.HandlerMeta)
			if len(handlers) != 2 {
				t.Fatalf("expected 2 handlers, got %d", len(handlers))
			}
		}
	}
	if !found {
		t.Fatalf("expected setHandlers op")
	}
}

func TestDiffHandlersOrderIndependent(t *testing.T) {
	prev := elementNode("button")
	prev.Handlers = []metadata.HandlerMeta{
		{Event: "click", Handler: "h1"},
		{Event: "mouseover", Handler: "h2"},
	}
	next := elementNode("button")
	next.Handlers = []metadata.HandlerMeta{
		{Event: "mouseover", Handler: "h2"},
		{Event: "click", Handler: "h1"},
	}
	patches := Diff(prev, next)

	for _, p := range patches {
		if p.Op == OpSetHandlers {
			t.Fatalf("handlers with same content in different order should not produce patch")
		}
	}
}

func TestDiffComment(t *testing.T) {
	prev := commentNode("old comment")
	next := commentNode("new comment")
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

func TestDiffNilStyleMaps(t *testing.T) {
	prev := elementNode("div")
	next := withStyle(elementNode("div"), "color", "red")
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

func TestDiffStylesheetNilMaps(t *testing.T) {
	prev := elementNode("style")
	next := elementNode("style")
	next.Stylesheet = &metadata.Stylesheet{
		Rules: []metadata.StyleRule{
			{Selector: "card", Props: map[string]string{"color": "blue"}},
		},
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
		t.Fatalf("expected setStyleDecl op when adding stylesheet to nil")
	}
}

func TestDiffKeyedSwap(t *testing.T) {
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
	)
	patches := Diff(prev, next)

	moveCount := 0
	for _, p := range patches {
		if p.Op == OpMoveChild {
			moveCount++
			val := p.Value.(map[string]interface{})
			key := val["key"].(string)
			newIdx := val["newIdx"].(int)

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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
	)
	patches := Diff(prev, next)

	foundDel := false
	for _, p := range patches {
		if p.Op == OpDelChild {
			foundDel = true
			if p.Index == nil || *p.Index != 0 {
				t.Fatalf("expected delChild at index 0, got %v", p.Index)
			}
			val, ok := p.Value.(map[string]interface{})
			if ok && val["key"] != "a" {
				t.Fatalf("expected deletion of key 'a', got %v", val["key"])
			}
		}
		if p.Op == OpReplaceNode || p.Op == OpAddChild {
			t.Fatalf("delete-first should not produce replace/add: %#v", p)
		}
	}
	if !foundDel {
		t.Fatalf("expected delChild operation")
	}
}

func TestDiffKeyedReverseOrder(t *testing.T) {
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
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
	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
		withChildren(withKey(elementNode("li"), "c"), textNode("Item C")),
		withChildren(withKey(elementNode("li"), "d"), textNode("Item D")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "d"), textNode("Item D")),
		withChildren(withKey(elementNode("li"), "e"), textNode("Item E")),
		withChildren(withKey(elementNode("li"), "b"), textNode("Item B")),
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

func TestDiffScript(t *testing.T) {
	prev := elementNode("div")
	next := elementNode("div")
	next.Script = &metadata.ScriptMeta{
		ScriptID: "timer1",
		Script:   "(el, transport) => { console.log('hello'); }",
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetScript {
			found = true
			script := p.Value.(*metadata.ScriptMeta)
			if script.ScriptID != "timer1" {
				t.Fatalf("expected scriptID=timer1, got %s", script.ScriptID)
			}
			if script.Script != "(el, transport) => { console.log('hello'); }" {
				t.Fatalf("unexpected script content: %s", script.Script)
			}
		}
	}
	if !found {
		t.Fatalf("expected setScript op")
	}
}

func TestDiffScriptRemoval(t *testing.T) {
	prev := elementNode("div")
	prev.Script = &metadata.ScriptMeta{
		ScriptID: "timer1",
		Script:   "(el, transport) => { console.log('hello'); }",
	}
	next := elementNode("div")
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpDelScript {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected delScript op")
	}
}

func TestDiffScriptChange(t *testing.T) {
	prev := elementNode("div")
	prev.Script = &metadata.ScriptMeta{
		ScriptID: "timer1",
		Script:   "(el, transport) => { console.log('hello'); }",
	}
	next := elementNode("div")
	next.Script = &metadata.ScriptMeta{
		ScriptID: "timer1",
		Script:   "(el, transport) => { console.log('world'); }",
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetScript {
			found = true
			script := p.Value.(*metadata.ScriptMeta)
			if script.Script != "(el, transport) => { console.log('world'); }" {
				t.Fatalf("expected updated script, got %s", script.Script)
			}
		}
	}
	if !found {
		t.Fatalf("expected setScript op when script content changes")
	}
}

func TestDiffScriptIDChange(t *testing.T) {
	prev := elementNode("div")
	prev.Script = &metadata.ScriptMeta{
		ScriptID: "timer1",
		Script:   "(el, transport) => { console.log('hello'); }",
	}
	next := elementNode("div")
	next.Script = &metadata.ScriptMeta{
		ScriptID: "timer2",
		Script:   "(el, transport) => { console.log('hello'); }",
	}
	patches := Diff(prev, next)

	found := false
	for _, p := range patches {
		if p.Op == OpSetScript {
			found = true
			script := p.Value.(*metadata.ScriptMeta)
			if script.ScriptID != "timer2" {
				t.Fatalf("expected scriptID=timer2, got %s", script.ScriptID)
			}
		}
	}
	if !found {
		t.Fatalf("expected setScript op when script ID changes")
	}
}

func TestDiffScriptUnchanged(t *testing.T) {
	prev := elementNode("div")
	prev.Script = &metadata.ScriptMeta{
		ScriptID: "timer1",
		Script:   "(el, transport) => { console.log('hello'); }",
	}
	next := elementNode("div")
	next.Script = &metadata.ScriptMeta{
		ScriptID: "timer1",
		Script:   "(el, transport) => { console.log('hello'); }",
	}
	patches := Diff(prev, next)

	for _, p := range patches {
		if p.Op == OpSetScript || p.Op == OpDelScript {
			t.Fatalf("should not produce script operations when unchanged, got %#v", p)
		}
	}
}

func TestDiffDuplicateKeyDetection(t *testing.T) {

	origPanic := PanicOnDuplicateKey
	origCallback := OnDuplicateKey

	PanicOnDuplicateKey = true
	defer func() {
		PanicOnDuplicateKey = origPanic
		OnDuplicateKey = origCallback
		if r := recover(); r == nil {
			t.Fatalf("expected panic for duplicate keys")
		}
	}()

	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A duplicate")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
	)
	Diff(prev, next)
}

func TestDiffDuplicateKeyCallback(t *testing.T) {

	origPanic := PanicOnDuplicateKey
	origCallback := OnDuplicateKey
	defer func() {
		PanicOnDuplicateKey = origPanic
		OnDuplicateKey = origCallback
	}()

	PanicOnDuplicateKey = false
	called := false
	OnDuplicateKey = func(tree, key string, path []int) {
		called = true
		if key != "a" {
			t.Errorf("expected key 'a', got %s", key)
		}
	}

	prev := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A duplicate")),
	)
	next := withChildren(elementNode("ul"),
		withChildren(withKey(elementNode("li"), "a"), textNode("Item A")),
	)
	Diff(prev, next)

	if !called {
		t.Fatalf("expected OnDuplicateKey callback to be called")
	}
}

func TestDiffUnsafeHTMLReplacesEntireNode(t *testing.T) {
	prev := elementNode("div")
	prev.UnsafeHTML = "<p>old</p>"
	prev.Attrs = map[string][]string{"class": {"old"}}

	next := elementNode("div")
	next.UnsafeHTML = "<p>new</p>"
	next.Attrs = map[string][]string{"class": {"new"}}

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch (replaceNode), got %d", len(patches))
	}
	if patches[0].Op != OpReplaceNode {
		t.Fatalf("expected replaceNode op, got %s", patches[0].Op)
	}
}

func TestDiffUnsafeHTMLToNormal(t *testing.T) {
	prev := elementNode("div")
	prev.UnsafeHTML = "<p>raw</p>"

	next := withChildren(elementNode("div"), textNode("normal"))

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpReplaceNode {
		t.Fatalf("expected replaceNode op, got %s", patches[0].Op)
	}
}
