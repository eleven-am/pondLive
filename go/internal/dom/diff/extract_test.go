package diff

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

func TestExtractMetadata_NilNode(t *testing.T) {
	patches := ExtractMetadata(nil)
	if patches != nil {
		t.Errorf("Expected nil patches for nil node, got %v", patches)
	}
}

func TestExtractMetadata_EmptyNode(t *testing.T) {
	node := &dom.StructuredNode{
		Tag: "div",
	}
	patches := ExtractMetadata(node)
	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for empty node, got %d", len(patches))
	}
}

func TestExtractMetadata_Handlers(t *testing.T) {
	node := &dom.StructuredNode{
		Tag: "button",
		Handlers: []dom.HandlerMeta{
			{Event: "click", Handler: "h1", Props: []string{"value"}},
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
	node := &dom.StructuredNode{
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

func TestExtractMetadata_Router(t *testing.T) {
	node := &dom.StructuredNode{
		Tag: "a",
		Router: &dom.RouterMeta{
			PathValue: "/home",
			Query:     "foo=bar",
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}

	p := patches[0]
	if p.Op != OpSetRouter {
		t.Errorf("Expected OpSetRouter, got %v", p.Op)
	}
}

func TestExtractMetadata_Upload(t *testing.T) {
	node := &dom.StructuredNode{
		Tag: "input",
		Upload: &dom.UploadMeta{
			UploadID: "upload1",
			Multiple: true,
			MaxSize:  1024,
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}

	p := patches[0]
	if p.Op != OpSetUpload {
		t.Errorf("Expected OpSetUpload, got %v", p.Op)
	}
}

func TestExtractMetadata_MultipleMetadata(t *testing.T) {
	node := &dom.StructuredNode{
		Tag:   "button",
		RefID: "btnRef",
		Handlers: []dom.HandlerMeta{
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
	node := &dom.StructuredNode{
		Tag: "div",
		Children: []*dom.StructuredNode{
			{
				Tag:   "button",
				RefID: "btn1",
			},
			{
				Tag: "span",
				Children: []*dom.StructuredNode{
					{
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
	node := &dom.StructuredNode{
		Fragment: true,
		Children: []*dom.StructuredNode{
			{
				Tag:   "button",
				RefID: "btn1",
			},
			{
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

func TestExtractMetadata_Component(t *testing.T) {
	node := &dom.StructuredNode{
		ComponentID: "MyComponent",
		Children: []*dom.StructuredNode{
			{
				Tag:   "div",
				RefID: "div1",
			},
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 1 {
		t.Fatalf("Expected 1 patch, got %d", len(patches))
	}

	if len(patches[0].Path) != 0 {
		t.Errorf("Expected patch path [], got %v", patches[0].Path)
	}
}

func TestExtractMetadata_UnsafeHTML(t *testing.T) {
	node := &dom.StructuredNode{
		Tag:        "div",
		UnsafeHTML: "<p>HTML content</p>",
		Children: []*dom.StructuredNode{
			{
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
	node := &dom.StructuredNode{
		Text: "Hello world",
	}
	patches := ExtractMetadata(node)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for text node, got %d", len(patches))
	}
}

func TestExtractMetadata_CommentNode(t *testing.T) {
	node := &dom.StructuredNode{
		Comment: "This is a comment",
	}
	patches := ExtractMetadata(node)

	if len(patches) != 0 {
		t.Errorf("Expected 0 patches for comment node, got %d", len(patches))
	}
}

func TestExtractMetadata_AllMetadataTypes(t *testing.T) {
	node := &dom.StructuredNode{
		Tag:   "button",
		RefID: "myBtn",
		Handlers: []dom.HandlerMeta{
			{Event: "click", Handler: "h1"},
		},
		Router: &dom.RouterMeta{
			PathValue: "/page",
		},
		Upload: &dom.UploadMeta{
			UploadID: "u1",
		},
	}
	patches := ExtractMetadata(node)

	if len(patches) != 4 {
		t.Fatalf("Expected 4 patches, got %d", len(patches))
	}

	expectedOps := []OpKind{OpSetHandlers, OpSetRef, OpSetRouter, OpSetUpload}
	for i, expectedOp := range expectedOps {
		if patches[i].Op != expectedOp {
			t.Errorf("Expected patch %d to be %v, got %v", i, expectedOp, patches[i].Op)
		}
		if patches[i].Seq != i {
			t.Errorf("Expected patch %d seq=%d, got %d", i, i, patches[i].Seq)
		}
	}
}
