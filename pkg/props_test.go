package pkg

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestAttr(t *testing.T) {
	el := &work.Element{}
	Attr("data-test", "value").ApplyTo(el)

	if el.Attrs == nil {
		t.Fatal("Expected Attrs to be initialized")
	}
	if len(el.Attrs["data-test"]) != 1 || el.Attrs["data-test"][0] != "value" {
		t.Errorf("Expected data-test='value', got %v", el.Attrs["data-test"])
	}
}

func TestSrc(t *testing.T) {
	el := &work.Element{}
	Src("/images/logo.png").ApplyTo(el)

	if el.Attrs == nil {
		t.Fatal("Expected Attrs to be initialized")
	}
	vals, ok := el.Attrs["src"]
	if !ok {
		t.Fatal("Expected 'src' attribute, not found")
	}
	if len(vals) != 1 || vals[0] != "/images/logo.png" {
		t.Errorf("Expected src='/images/logo.png', got %v", vals)
	}
	if _, hasSrcOld := el.Attrs["src-old"]; hasSrcOld {
		t.Error("Should not have 'src-old' attribute")
	}
}

func TestHref(t *testing.T) {
	el := &work.Element{}
	Href("https://example.com").ApplyTo(el)

	if el.Attrs["href"][0] != "https://example.com" {
		t.Errorf("Expected href='https://example.com', got %v", el.Attrs["href"])
	}
}

func TestID(t *testing.T) {
	el := &work.Element{}
	ID("main-content").ApplyTo(el)

	if el.Attrs["id"][0] != "main-content" {
		t.Errorf("Expected id='main-content', got %v", el.Attrs["id"])
	}
}

func TestClass(t *testing.T) {
	el := &work.Element{}
	Class("btn", "btn-primary").ApplyTo(el)

	classes := el.Attrs["class"]
	if len(classes) != 2 {
		t.Fatalf("Expected 2 classes, got %d", len(classes))
	}
	if classes[0] != "btn" || classes[1] != "btn-primary" {
		t.Errorf("Expected ['btn', 'btn-primary'], got %v", classes)
	}
}

func TestClassFiltersEmpty(t *testing.T) {
	el := &work.Element{}
	Class("btn", "", "  ", "active").ApplyTo(el)

	classes := el.Attrs["class"]
	if len(classes) != 2 {
		t.Fatalf("Expected 2 classes after filtering, got %d: %v", len(classes), classes)
	}
	if classes[0] != "btn" || classes[1] != "active" {
		t.Errorf("Expected ['btn', 'active'], got %v", classes)
	}
}

func TestClassAppends(t *testing.T) {
	el := &work.Element{}
	Class("first").ApplyTo(el)
	Class("second").ApplyTo(el)

	classes := el.Attrs["class"]
	if len(classes) != 2 {
		t.Fatalf("Expected 2 classes after append, got %d", len(classes))
	}
	if classes[0] != "first" || classes[1] != "second" {
		t.Errorf("Expected ['first', 'second'], got %v", classes)
	}
}

func TestStyle(t *testing.T) {
	el := &work.Element{}
	Style("color", "red").ApplyTo(el)
	Style("font-size", "16px").ApplyTo(el)

	if el.Style == nil {
		t.Fatal("Expected Style to be initialized")
	}
	if el.Style["color"] != "red" {
		t.Errorf("Expected color='red', got %q", el.Style["color"])
	}
	if el.Style["font-size"] != "16px" {
		t.Errorf("Expected font-size='16px', got %q", el.Style["font-size"])
	}
}

func TestKey(t *testing.T) {
	el := &work.Element{}
	Key("item-123").ApplyTo(el)

	if el.Key != "item-123" {
		t.Errorf("Expected Key='item-123', got %q", el.Key)
	}
}

func TestDisabled(t *testing.T) {
	el := &work.Element{}
	Disabled().ApplyTo(el)

	vals, ok := el.Attrs["disabled"]
	if !ok {
		t.Fatal("Expected 'disabled' attribute")
	}
	if len(vals) != 1 || vals[0] != "" {
		t.Errorf("Expected disabled=[''], got %v", vals)
	}
}

func TestChecked(t *testing.T) {
	el := &work.Element{}
	Checked().ApplyTo(el)

	if _, ok := el.Attrs["checked"]; !ok {
		t.Error("Expected 'checked' attribute")
	}
}

func TestData(t *testing.T) {
	el := &work.Element{}
	Data("user-id", "42").ApplyTo(el)

	if el.Attrs["data-user-id"][0] != "42" {
		t.Errorf("Expected data-user-id='42', got %v", el.Attrs["data-user-id"])
	}
}

func TestAria(t *testing.T) {
	el := &work.Element{}
	Aria("label", "Close button").ApplyTo(el)

	if el.Attrs["aria-label"][0] != "Close button" {
		t.Errorf("Expected aria-label='Close button', got %v", el.Attrs["aria-label"])
	}
}

func TestUnsafeHTML(t *testing.T) {
	el := &work.Element{}
	UnsafeHTML("<strong>Bold</strong>").ApplyTo(el)

	if el.UnsafeHTML != "<strong>Bold</strong>" {
		t.Errorf("Expected UnsafeHTML='<strong>Bold</strong>', got %q", el.UnsafeHTML)
	}
}
