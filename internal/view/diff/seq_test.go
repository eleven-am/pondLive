package diff

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/view"
)

func TestSeqStartsAtZero(t *testing.T) {
	prev := &view.Element{Tag: "div", Children: []view.Node{&view.Text{Text: "old"}}}
	next := &view.Element{Tag: "div", Children: []view.Node{&view.Text{Text: "new"}}}

	patches := Diff(prev, next)

	if len(patches) == 0 {
		t.Fatal("expected patches")
	}
	if patches[0].Seq != 0 {
		t.Errorf("expected first patch seq to be 0, got %d", patches[0].Seq)
	}
}

func TestSeqIncrementsSequentially(t *testing.T) {
	prev := &view.Element{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"old"}},
		Style: map[string]string{"color": "red"},
		Children: []view.Node{
			&view.Text{Text: "text1"},
		},
	}
	next := &view.Element{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"new"}},
		Style: map[string]string{"color": "blue"},
		Children: []view.Node{
			&view.Text{Text: "text2"},
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
	prev := &view.Element{
		Tag: "ul",
		Children: []view.Node{
			&view.Element{Tag: "li", Children: []view.Node{&view.Text{Text: "a"}}},
			&view.Element{Tag: "li", Children: []view.Node{&view.Text{Text: "b"}}},
			&view.Element{Tag: "li", Children: []view.Node{&view.Text{Text: "c"}}},
		},
	}
	next := &view.Element{
		Tag: "ul",
		Children: []view.Node{
			&view.Element{Tag: "li", Children: []view.Node{&view.Text{Text: "x"}}},
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
	prev := &view.Element{
		Tag: "ul",
		Children: []view.Node{
			&view.Element{Tag: "li", Key: "a", Children: []view.Node{&view.Text{Text: "A"}}},
			&view.Element{Tag: "li", Key: "b", Children: []view.Node{&view.Text{Text: "B"}}},
			&view.Element{Tag: "li", Key: "c", Children: []view.Node{&view.Text{Text: "C"}}},
		},
	}
	next := &view.Element{
		Tag: "ul",
		Children: []view.Node{
			&view.Element{Tag: "li", Key: "c", Children: []view.Node{&view.Text{Text: "C"}}},
			&view.Element{Tag: "li", Key: "a", Children: []view.Node{&view.Text{Text: "A"}}},
			&view.Element{Tag: "li", Key: "b", Children: []view.Node{&view.Text{Text: "B"}}},
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
	prev := &view.Element{
		Tag: "div",
		Children: []view.Node{
			&view.Element{Tag: "span", Key: "a"},
			&view.Element{Tag: "span", Key: "b"},
			&view.Element{Tag: "span", Key: "c"},
		},
	}
	next := &view.Element{
		Tag: "div",
		Children: []view.Node{
			&view.Element{Tag: "span", Key: "a"},
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
	prev := &view.Element{
		Tag: "div",
		Children: []view.Node{
			&view.Element{Tag: "span", Key: "old"},
		},
	}
	next := &view.Element{
		Tag: "div",
		Children: []view.Node{
			&view.Element{Tag: "span", Key: "new1"},
			&view.Element{Tag: "span", Key: "new2"},
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
	prev := &view.Element{Tag: "div"}
	next := &view.Element{Tag: "span"}

	patches := DiffRaw(prev, next)

	if len(patches) != 1 {
		t.Fatalf("expected 1 patch, got %d", len(patches))
	}
	if patches[0].Seq != 0 {
		t.Errorf("expected seq 0, got %d", patches[0].Seq)
	}
}

func TestSeqWithFlattenedFragments(t *testing.T) {

	prev := &view.Fragment{
		Fragment: true,
		Children: []view.Node{
			&view.Element{Tag: "div", Children: []view.Node{&view.Text{Text: "old"}}},
		},
	}
	next := &view.Fragment{
		Fragment: true,
		Children: []view.Node{
			&view.Element{Tag: "div", Children: []view.Node{&view.Text{Text: "new"}}},
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
	prev := &view.Element{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"outer"}},
		Children: []view.Node{
			&view.Element{
				Tag:   "span",
				Attrs: map[string][]string{"class": {"inner"}},
				Children: []view.Node{
					&view.Text{Text: "deep"},
				},
			},
		},
	}
	next := &view.Element{
		Tag:   "div",
		Attrs: map[string][]string{"class": {"outer-new"}},
		Children: []view.Node{
			&view.Element{
				Tag:   "span",
				Attrs: map[string][]string{"class": {"inner-new"}},
				Children: []view.Node{
					&view.Text{Text: "deep-new"},
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
