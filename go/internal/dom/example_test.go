package dom

import "testing"

func TestButtonExample(t *testing.T) {
	button := &StructuredNode{Tag: "button"}

	Class("btn", "btn-primary").ApplyTo(button)
	ID("submit-btn").ApplyTo(button)
	Type("submit").ApplyTo(button)
	Style("margin", "10px").ApplyTo(button)
	Handler("click", "comp-1:h0", "target.value").ApplyTo(button)

	TextNode("Click Me").ApplyTo(button)

	if err := button.Validate(); err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if len(button.Attrs["class"]) != 2 || button.Attrs["class"][0] != "btn" || button.Attrs["class"][1] != "btn-primary" {
		t.Errorf("Class attr incorrect: %v", button.Attrs["class"])
	}

	if len(button.Attrs["id"]) != 1 || button.Attrs["id"][0] != "submit-btn" {
		t.Errorf("ID attr incorrect: %v", button.Attrs["id"])
	}

	if len(button.Attrs["type"]) != 1 || button.Attrs["type"][0] != "submit" {
		t.Errorf("Type attr incorrect: %v", button.Attrs["type"])
	}

	if button.Style["margin"] != "10px" {
		t.Errorf("Style incorrect: %v", button.Style)
	}

	if len(button.Handlers) != 1 {
		t.Errorf("Expected 1 handler, got %d", len(button.Handlers))
	} else {
		h := button.Handlers[0]
		if h.Event != "click" {
			t.Errorf("Handler incorrect: %+v", h)
		}
	}

	if len(button.Children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(button.Children))
	}

	html := button.ToHTML()
	expected := `<button class="btn btn-primary" id="submit-btn" type="submit" style="margin: 10px">Click Me</button>`

	if html != expected {
		t.Errorf("HTML mismatch:\ngot:  %s\nwant: %s", html, expected)
	}
}
