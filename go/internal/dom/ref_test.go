package dom

import "testing"

type testDescriptor struct{}

func (testDescriptor) TagName() string { return "test" }

func TestAttachElementRefPanicsWhenReused(t *testing.T) {
	ref := NewElementRef("ref:1", testDescriptor{})
	first := &Element{Tag: "test", Descriptor: testDescriptor{}}
	AttachElementRef[testDescriptor](ref, first)

	second := &Element{Tag: "test", Descriptor: testDescriptor{}}
	defer func() {
		if r := recover(); r == nil {
			t.Fatalf("AttachElementRef should panic when reusing a ref, but it did not")
		}
	}()
	AttachElementRef[testDescriptor](ref, second)
}

func TestAttachElementRefAllowsReattachAfterReset(t *testing.T) {
	ref := NewElementRef("ref:2", testDescriptor{})
	first := &Element{Tag: "test", Descriptor: testDescriptor{}}
	AttachElementRef[testDescriptor](ref, first)

	ref.ResetAttachment()

	second := &Element{Tag: "test", Descriptor: testDescriptor{}}
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("AttachElementRef panicked after ResetAttachment: %v", r)
		}
	}()
	AttachElementRef[testDescriptor](ref, second)
}

func TestAttachElementRefSetsRefID(t *testing.T) {
	ref := NewElementRef("ref:3", testDescriptor{})
	el := &Element{Tag: "test", Descriptor: testDescriptor{}}

	AttachElementRef[testDescriptor](ref, el)

	if got, want := el.RefID, "ref:3"; got != want {
		t.Fatalf("AttachElementRef should set element ref ID to %q, got %q", want, got)
	}
}
