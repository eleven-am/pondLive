package diff

import (
	"reflect"
	"testing"

	"github.com/eleven-am/pondlive/go/internal/metadata"
	"github.com/eleven-am/pondlive/go/internal/view"
)

func TestExtractMetadata_NilNode(t *testing.T) {
	patches := ExtractMetadata(nil)
	if patches != nil {
		t.Errorf("Expected nil patches for nil node, got %v", patches)
	}
}

func TestExtractMetadata_EmptyNode(t *testing.T) {
	node := &view.Element{
		Tag: "div",
	}
	patches := ExtractMetadata(node)
	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for empty node, got %d", len(patches))
	}
}

func TestExtractMetadata_Handlers(t *testing.T) {
	node := &view.Element{
		Tag: "button",
		Handlers: []metadata.HandlerMeta{
			{Event: "click", Handler: "h1", EventOptions: metadata.EventOptions{Props: []string{"value"}}},
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}

	p := patches[0]
	if p.Op != OpSetHandlers {
		t.Errorf("Expected OpSetHandlers, got %v", p.Op)
	}
	if p.Seq != 0 {
		t.Errorf("Expected seq=0, got %d", p.Seq)
	}
	if len(p.Path) != 0 {
		t.Errorf("Expected empty path, got %v", p.Path)
	}
}

func TestExtractMetadata_Ref(t *testing.T) {
	node := &view.Element{
		Tag:   "input",
		RefID: "myRef",
	}
	patches := ExtractMetadata(node)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}

	p := patches[0]
	if p.Op != OpSetRef {
		t.Errorf("Expected OpSetRef, got %v", p.Op)
	}
	if p.Value != "myRef" {
		t.Errorf("Expected value='myRef', got %v", p.Value)
	}
}

func TestExtractMetadata_MultipleMetadata(t *testing.T) {
	node := &view.Element{
		Tag:   "button",
		RefID: "btnRef",
		Handlers: []metadata.HandlerMeta{
			{Event: "click", Handler: "h1"},
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 2 {
		t.Fatalf("Expected 2 patches, got %d", len(patches))
	}

	if patches[0].Op != OpSetHandlers {
		t.Errorf("Expected first patch to be OpSetHandlers, got %v", patches[0].Op)
	}
	if patches[1].Op != OpSetRef {
		t.Errorf("Expected second patch to be OpSetRef, got %v", patches[1].Op)
	}

	if patches[0].Seq != 0 {
		t.Errorf("Expected first seq=0, got %d", patches[0].Seq)
	}
	if patches[1].Seq != 1 {
		t.Errorf("Expected second seq=1, got %d", patches[1].Seq)
	}
}

func TestExtractMetadata_NestedChildren(t *testing.T) {
	node := &view.Element{
		Tag: "div",
		Children: []view.Node{
			&view.Element{
				Tag:   "button",
				RefID: "btn1",
			},
			&view.Element{
				Tag: "span",
				Children: []view.Node{
					&view.Element{
						Tag:   "input",
						RefID: "input1",
					},
				},
			},
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 2 {
		t.Fatalf("Expected 2 patches, got %d", len(patches))
	}

	if len(patches[0].Path) != 1 || patches[0].Path[0] != 0 {
		t.Errorf("Expected first patch path [0], got %v", patches[0].Path)
	}

	if len(patches[1].Path) != 2 || patches[1].Path[0] != 1 || patches[1].Path[1] != 0 {
		t.Errorf("Expected second patch path [1, 0], got %v", patches[1].Path)
	}
}

func TestExtractMetadata_Fragment(t *testing.T) {
	node := &view.Fragment{
		Fragment: true,
		Children: []view.Node{
			&view.Element{
				Tag:   "button",
				RefID: "btn1",
			},
			&view.Element{
				Tag:   "span",
				RefID: "span1",
			},
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 2 {
		t.Fatalf("Expected 2 patches, got %d", len(patches))
	}

	if len(patches[0].Path) != 1 || patches[0].Path[0] != 0 {
		t.Errorf("Expected first patch path [0], got %v", patches[0].Path)
	}
	if len(patches[1].Path) != 1 || patches[1].Path[0] != 1 {
		t.Errorf("Expected second patch path [1], got %v", patches[1].Path)
	}
}

func TestExtractMetadata_UnsafeHTML(t *testing.T) {
	node := &view.Element{
		Tag:        "div",
		UnsafeHTML: "<p>HTML content</p>",
		Children: []view.Node{
			&view.Element{
				Tag:   "span",
				RefID: "span1",
			},
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for node with UnsafeHTML, got %d", len(patches))
	}
}

func TestExtractMetadata_TextNode(t *testing.T) {
	node := &view.Text{
		Text: "Hello world",
	}
	patches := ExtractMetadata(node)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for text node, got %d", len(patches))
	}
}

func TestExtractMetadata_CommentNode(t *testing.T) {
	node := &view.Comment{
		Comment: "This is a comment",
	}
	patches := ExtractMetadata(node)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for comment node, got %d", len(patches))
	}
}

func TestExtractMetadata_AllMetadataTypes(t *testing.T) {
	node := &view.Element{
		Tag:   "button",
		RefID: "myBtn",
		Handlers: []metadata.HandlerMeta{
			{Event: "click", Handler: "h1"},
		},
		Script: &metadata.ScriptMeta{
			ScriptID: "script1",
			Script:   "() => {}",
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 3 {
		t.Fatalf("Expected 3 patches, got %d", len(patches))
	}

	expectedOps := []OpKind{OpSetHandlers, OpSetRef, OpSetScript}
	for i, expectedOp := range expectedOps {
		if patches[i].Op != expectedOp {
			t.Errorf("Expected patch %d to be %v, got %v", i, expectedOp, patches[i].Op)
		}
		if patches[i].Seq != i {
			t.Errorf("Expected patch %d seq=%d, got %d", i, i, patches[i].Seq)
		}
	}
}

func TestExtractMetadata_Script(t *testing.T) {
	node := &view.Element{
		Tag: "div",
		Script: &metadata.ScriptMeta{
			ScriptID: "timer1",
			Script:   "(el, transport) => { console.log('hello'); }",
		},
	}

	patches := ExtractMetadata(node)

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
			if len(p.Path) != 0 {
				t.Fatalf("expected empty path for root script, got %v", p.Path)
			}
		}
	}
	if !found {
		t.Fatalf("expected setScript op in extraction")
	}
}

func TestExtractMetadata_NestedScript(t *testing.T) {
	child := &view.Element{
		Tag: "button",
		Script: &metadata.ScriptMeta{
			ScriptID: "click1",
			Script:   "(el, transport) => { el.click(); }",
		},
	}

	parent := &view.Element{
		Tag:      "div",
		Children: []view.Node{child},
	}

	patches := ExtractMetadata(parent)

	found := false
	for _, p := range patches {
		if p.Op == OpSetScript {
			found = true
			script := p.Value.(*metadata.ScriptMeta)
			if script.ScriptID != "click1" {
				t.Fatalf("expected scriptID=click1, got %s", script.ScriptID)
			}
			if !reflect.DeepEqual(p.Path, []int{0}) {
				t.Fatalf("expected path [0] for first child, got %v", p.Path)
			}
		}
	}
	if !found {
		t.Fatalf("expected setScript op for nested script")
	}
}

func TestExtractMetadata_MultipleScripts(t *testing.T) {
	child1 := &view.Element{
		Tag: "div",
		Script: &metadata.ScriptMeta{
			ScriptID: "script1",
			Script:   "() => { console.log('1'); }",
		},
	}

	child2 := &view.Element{
		Tag: "div",
		Script: &metadata.ScriptMeta{
			ScriptID: "script2",
			Script:   "() => { console.log('2'); }",
		},
	}

	parent := &view.Element{
		Tag:      "div",
		Children: []view.Node{child1, child2},
		Script: &metadata.ScriptMeta{
			ScriptID: "script0",
			Script:   "() => { console.log('0'); }",
		},
	}

	patches := ExtractMetadata(parent)

	scriptCount := 0
	foundIDs := make(map[string]bool)

	for _, p := range patches {
		if p.Op == OpSetScript {
			scriptCount++
			script := p.Value.(*metadata.ScriptMeta)
			foundIDs[script.ScriptID] = true
		}
	}

	if scriptCount != 3 {
		t.Fatalf("expected 3 scripts, got %d", scriptCount)
	}
	if !foundIDs["script0"] || !foundIDs["script1"] || !foundIDs["script2"] {
		t.Fatalf("expected script0, script1, script2, got %v", foundIDs)
	}
}

func TestExtractMetadata_ScriptWithHandlers(t *testing.T) {
	node := &view.Element{
		Tag: "button",
		Script: &metadata.ScriptMeta{
			ScriptID: "btn1",
			Script:   "(el, transport) => { console.log('script'); }",
		},
		Handlers: []metadata.HandlerMeta{
			{Event: "click"},
		},
	}

	patches := ExtractMetadata(node)

	foundScript := false
	foundHandlers := false

	for _, p := range patches {
		if p.Op == OpSetScript {
			foundScript = true
		}
		if p.Op == OpSetHandlers {
			foundHandlers = true
		}
	}

	if !foundScript {
		t.Fatalf("expected setScript op")
	}
	if !foundHandlers {
		t.Fatalf("expected setHandlers op")
	}
}

func TestExtractMetadata_ScriptInFragment(t *testing.T) {
	child := &view.Element{
		Tag: "div",
		Script: &metadata.ScriptMeta{
			ScriptID: "frag1",
			Script:   "() => {}",
		},
	}

	fragment := &view.Fragment{
		Fragment: true,
		Children: []view.Node{child},
	}

	patches := ExtractMetadata(fragment)

	found := false
	for _, p := range patches {
		if p.Op == OpSetScript {
			found = true
			script := p.Value.(*metadata.ScriptMeta)
			if script.ScriptID != "frag1" {
				t.Fatalf("expected scriptID=frag1, got %s", script.ScriptID)
			}

			if len(p.Path) != 0 {
				t.Fatalf("expected empty path for flattened fragment, got %v", p.Path)
			}
		}
	}
	if !found {
		t.Fatalf("expected setScript op in fragment")
	}
}
