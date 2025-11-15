package render

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestUploadBindingSlicesAreCloned(t *testing.T) {
	button := h.Button(h.Text("upload"))
	button.UploadBindings = []dom.UploadBinding{{
		UploadID: "slot-1",
		Accept:   []string{"image/png"},
	}}

	root := h.WrapComponent("root", h.Div(button))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.UploadBindings) != 1 {
		t.Fatalf("expected one upload binding, got %d", len(structured.UploadBindings))
	}

	button.UploadBindings[0].Accept[0] = "mutated"
	binding := structured.UploadBindings[0]
	if binding.Accept[0] != "image/png" {
		t.Fatalf("expected upload binding slice to be cloned, got %v", binding.Accept)
	}
}
