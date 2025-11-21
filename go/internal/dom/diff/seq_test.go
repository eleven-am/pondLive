package diff

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
)

func TestSeqStartsAtZero(t *testing.T) {
	prev := &dom.StructuredNode{Tag: "div", Children: []*dom.StructuredNode{{Text: "old"}}}
	next := &dom.StructuredNode{Tag: "div", Children: []*dom.StructuredNode{{Text: "new"}}}

	patches := Diff(prev, next)

	if len(patches) == 0 {
		t.Fatal("expected patches")
	}
	if patches[0].Seq != 0 {
		t.Errorf("expected first patch seq to be 0, got %d", patches[0].Seq)
	}
}

func TestSeqIncrementsSequentially(t *testing.T) {
	prev := &dom.StructuredNode{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"old"}},
		Style: map[string]string{"color": "red"},
		Children: []*dom.StructuredNode{
			{Text: "text1"},
		},
	}
	next := &dom.StructuredNode{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"new"}},
		Style: map[string]string{"color": "blue"},
		Children: []*dom.StructuredNode{
			{Text: "text2"},
		},
	}

	patches := Diff(prev, next)

	if len(patches) < 2 {
		t.Fatalf("expected multiple patches, got %d", len(patches))
	}

	for i := 0; i < len(patches); i++ {
		if patches[i].Seq != i {
			t.Errorf("patch %d: expected seq %d, got %d", i, i, patches[i].Seq)
		}
	}
}

func TestSeqWithMultipleChildOperations(t *testing.T) {
	prev := &dom.StructuredNode{
		Tag: "ul",
		Children: []*dom.StructuredNode{
			{Tag: "li", Children: []*dom.StructuredNode{{Text: "a"}}},
			{Tag: "li", Children: []*dom.StructuredNode{{Text: "b"}}},
			{Tag: "li", Children: []*dom.StructuredNode{{Text: "c"}}},
		},
	}
	next := &dom.StructuredNode{
		Tag: "ul",
		Children: []*dom.StructuredNode{
			{Tag: "li", Children: []*dom.StructuredNode{{Text: "x"}}},
		},
	}

	patches := Diff(prev, next)

	for i := 0; i < len(patches); i++ {
		if patches[i].Seq != i {
			t.Errorf("patch %d: expected seq %d, got %d", i, i, patches[i].Seq)
		}
	}
}

func TestSeqWithKeyedReorder(t *testing.T) {
	prev := &dom.StructuredNode{
		Tag: "ul",
		Children: []*dom.StructuredNode{
			{Tag: "li", Key: "a", Children: []*dom.StructuredNode{{Text: "A"}}},
			{Tag: "li", Key: "b", Children: []*dom.StructuredNode{{Text: "B"}}},
			{Tag: "li", Key: "c", Children: []*dom.StructuredNode{{Text: "C"}}},
		},
	}
	next := &dom.StructuredNode{
		Tag: "ul",
		Children: []*dom.StructuredNode{
			{Tag: "li", Key: "c", Children: []*dom.StructuredNode{{Text: "C"}}},
			{Tag: "li", Key: "a", Children: []*dom.StructuredNode{{Text: "A"}}},
			{Tag: "li", Key: "b", Children: []*dom.StructuredNode{{Text: "B"}}},
		},
	}

	patches := Diff(prev, next)

	for i := 0; i < len(patches); i++ {
		if patches[i].Seq != i {
			t.Errorf("patch %d: expected seq %d, got %d", i, i, patches[i].Seq)
		}
	}
}

func TestSeqWithDeletesFirst(t *testing.T) {

	prev := &dom.StructuredNode{
		Tag: "div",
		Children: []*dom.StructuredNode{
			{Tag: "span", Key: "a"},
			{Tag: "span", Key: "b"},
			{Tag: "span", Key: "c"},
		},
	}
	next := &dom.StructuredNode{
		Tag: "div",
		Children: []*dom.StructuredNode{
			{Tag: "span", Key: "a"},
		},
	}

	patches := Diff(prev, next)

	deletes := 0
	for _, p := range patches {
		if p.Op == OpDelChild {
			deletes++
		}
	}
	if deletes != 2 {
		t.Errorf("expected 2 deletes, got %d", deletes)
	}

	for i := 0; i < len(patches); i++ {
		if patches[i].Seq != i {
			t.Errorf("patch %d: expected seq %d, got %d", i, i, patches[i].Seq)
		}
	}
}

func TestSeqWithAddsAfterDeletes(t *testing.T) {
	prev := &dom.StructuredNode{
		Tag: "div",
		Children: []*dom.StructuredNode{
			{Tag: "span", Key: "old"},
		},
	}
	next := &dom.StructuredNode{
		Tag: "div",
		Children: []*dom.StructuredNode{
			{Tag: "span", Key: "new1"},
			{Tag: "span", Key: "new2"},
		},
	}

	patches := Diff(prev, next)

	// Find operations
	var delSeq, add1Seq, add2Seq int
	for _, p := range patches {
		switch p.Op {
		case OpDelChild:
			delSeq = p.Seq
		case OpAddChild:
			if p.Index != nil && *p.Index == 0 {
				add1Seq = p.Seq
			} else {
				add2Seq = p.Seq
			}
		}
	}

	if delSeq >= add1Seq || delSeq >= add2Seq {
		t.Errorf("delete (seq=%d) should come before adds (seq=%d, %d)", delSeq, add1Seq, add2Seq)
	}
}

func TestSeqDiffRawAlsoHasSeq(t *testing.T) {
	prev := &dom.StructuredNode{Tag: "div"}
	next := &dom.StructuredNode{Tag: "span"}

	patches := DiffRaw(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Seq != 0 {
		t.Errorf("expected seq 0, got %d", patches[0].Seq)
	}
}

func TestSeqWithFlattenedComponents(t *testing.T) {

	prev := &dom.StructuredNode{
		ComponentID: "App",
		Children: []*dom.StructuredNode{
			{Tag: "div", Children: []*dom.StructuredNode{{Text: "old"}}},
		},
	}
	next := &dom.StructuredNode{
		ComponentID: "App",
		Children: []*dom.StructuredNode{
			{Tag: "div", Children: []*dom.StructuredNode{{Text: "new"}}},
		},
	}

	patches := Diff(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Op != OpSetText {
		t.Errorf("expected setText op, got %s", patches[0].Op)
	}
	if patches[0].Seq != 0 {
		t.Errorf("expected seq 0, got %d", patches[0].Seq)
	}
}

func TestSeqWithNestedChanges(t *testing.T) {

	prev := &dom.StructuredNode{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"outer"}},
		Children: []*dom.StructuredNode{
			{
				Tag:   "span",
				Attrs: map[string][]string{"class": {"inner"}},
				Children: []*dom.StructuredNode{
					{Text: "deep"},
				},
			},
		},
	}
	next := &dom.StructuredNode{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"outer-new"}},
		Children: []*dom.StructuredNode{
			{
				Tag:   "span",
				Attrs: map[string][]string{"class": {"inner-new"}},
				Children: []*dom.StructuredNode{
					{Text: "deep-new"},
				},
			},
		},
	}

	patches := Diff(prev, next)

	if len(patches) != 3 {
		t.Fatalf("expected 3 patches, got %d", len(patches))
	}

	for i := 0; i < len(patches); i++ {
		if patches[i].Seq != i {
			t.Errorf("patch %d: expected seq %d, got %d", i, i, patches[i].Seq)
		}
	}
}
