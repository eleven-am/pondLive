package dom

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

// TestValidation tests the StructuredNode validation logic
func TestValidation(t *testing.T) {
	tests := []struct {
		name    string
		node    *StructuredNode
		wantErr bool
		errMsg  string
	}{
		{
			name:    "nil node",
			node:    nil,
			wantErr: true,
			errMsg:  "cannot validate nil node",
		},
		{
			name:    "no discriminator",
			node:    &StructuredNode{},
			wantErr: true,
			errMsg:  "exactly one type discriminator",
		},
		{
			name: "multiple discriminators",
			node: &StructuredNode{
				Tag:  "div",
				Text: "hello",
			},
			wantErr: true,
			errMsg:  "exactly one type discriminator",
		},
		{
			name: "valid text node",
			node: &StructuredNode{
				Text: "hello world",
			},
			wantErr: false,
		},
		{
			name: "valid element node",
			node: &StructuredNode{
				Tag: "div",
			},
			wantErr: false,
		},
		{
			name: "valid component node",
			node: &StructuredNode{
				ComponentID: "comp-1",
			},
			wantErr: false,
		},
		{
			name: "element with both children and unsafe html",
			node: &StructuredNode{
				Tag:        "div",
				Children:   []*StructuredNode{{Text: "child"}},
				UnsafeHTML: "<span>raw</span>",
			},
			wantErr: true,
			errMsg:  "cannot have both UnsafeHTML and Children",
		},
		{
			name: "styles on non-style element",
			node: &StructuredNode{
				Tag: "div",
				Styles: map[string]map[string]string{
					"card": {"color": "red"},
				},
			},
			wantErr: true,
			errMsg:  "styles map only valid on <style> elements",
		},
		{
			name: "styles on style element",
			node: &StructuredNode{
				Tag: "style",
				Styles: map[string]map[string]string{
					"card": {"color": "red"},
				},
			},
			wantErr: false,
		},
		{
			name: "text node with handlers",
			node: &StructuredNode{
				Text:     "hello",
				Handlers: []HandlerMeta{{Event: "click"}},
			},
			wantErr: true,
			errMsg:  "only elements can have handlers",
		},
		{
			name: "text node with attrs",
			node: &StructuredNode{
				Text:  "hello",
				Attrs: map[string][]string{"class": {"btn"}},
			},
			wantErr: true,
			errMsg:  "only elements can have attributes or styles",
		},
		{
			name: "component node with ref",
			node: &StructuredNode{
				ComponentID: "comp-1",
				RefID:       "ref1",
			},
			wantErr: true,
			errMsg:  "only elements can have ref IDs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.node.Validate()
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error containing %q, got nil", tt.errMsg)
					return
				}
				if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("expected error containing %q, got %q", tt.errMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("expected no error, got %v", err)
				}
			}
		})
	}
}

// TestHTMLRendering tests the ToHTML method
func TestHTMLRendering(t *testing.T) {
	tests := []struct {
		name string
		node *StructuredNode
		want string
	}{
		{
			name: "text node",
			node: &StructuredNode{Text: "hello world"},
			want: "hello world",
		},
		{
			name: "text node with escaping",
			node: &StructuredNode{Text: "<script>alert('xss')</script>"},
			want: "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;",
		},
		{
			name: "simple element",
			node: &StructuredNode{Tag: "div"},
			want: "<div></div>",
		},
		{
			name: "element with text child",
			node: &StructuredNode{
				Tag: "p",
				Children: []*StructuredNode{
					{Text: "Hello"},
				},
			},
			want: "<p>Hello</p>",
		},
		{
			name: "element with attributes",
			node: &StructuredNode{
				Tag: "button",
				Attrs: map[string][]string{
					"class": {"btn", "btn-primary"},
					"type":  {"submit"},
				},
			},
			want: `<button class="btn btn-primary" type="submit"></button>`,
		},
		{
			name: "element with inline styles",
			node: &StructuredNode{
				Tag: "div",
				Style: map[string]string{
					"color":      "red",
					"background": "blue",
				},
			},
			want: `<div style="background: blue; color: red"></div>`,
		},
		{
			name: "style element with stylesheet",
			node: &StructuredNode{
				Tag: "style",
				Styles: map[string]map[string]string{
					"card": {
						"background": "#fff",
						"color":      "#111",
					},
					"card:hover": {
						"background": "#eee",
					},
				},
			},
			want: `<style>card { background: #fff; color: #111; } card:hover { background: #eee; }</style>`,
		},
		{
			name: "element with unsafe html",
			node: &StructuredNode{
				Tag:        "div",
				UnsafeHTML: "<span>raw html</span>",
			},
			want: `<div><span>raw html</span></div>`,
		},
		{
			name: "component boundary",
			node: &StructuredNode{
				ComponentID: "comp-1",
				Children: []*StructuredNode{
					{Tag: "div", Children: []*StructuredNode{{Text: "inside component"}}},
				},
			},
			want: "<div>inside component</div>",
		},
		{
			name: "nested structure",
			node: &StructuredNode{
				Tag: "div",
				Attrs: map[string][]string{
					"class": {"container"},
				},
				Children: []*StructuredNode{
					{
						Tag: "h1",
						Children: []*StructuredNode{
							{Text: "Title"},
						},
					},
					{
						Tag: "p",
						Children: []*StructuredNode{
							{Text: "Paragraph text"},
						},
					},
				},
			},
			want: `<div class="container"><h1>Title</h1><p>Paragraph text</p></div>`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.node.ToHTML()
			if got != tt.want {
				t.Errorf("ToHTML() = %q, want %q", got, tt.want)
			}
		})
	}
}

// TestJSONSerialization tests ToJSON and FromJSON
func TestJSONSerialization(t *testing.T) {
	tests := []struct {
		name    string
		node    *StructuredNode
		wantErr bool
	}{
		{
			name: "simple element",
			node: &StructuredNode{
				Tag: "div",
				Attrs: map[string][]string{
					"class": {"container"},
				},
			},
			wantErr: false,
		},
		{
			name: "text node",
			node: &StructuredNode{
				Text: "hello",
			},
			wantErr: false,
		},
		{
			name: "component with children",
			node: &StructuredNode{
				ComponentID: "comp-1",
				Children: []*StructuredNode{
					{Tag: "div"},
					{Text: "text"},
				},
			},
			wantErr: false,
		},
		{
			name: "invalid node",
			node: &StructuredNode{
				Tag:  "div",
				Text: "invalid",
			},
			wantErr: true,
		},
		{
			name: "element with handlers",
			node: &StructuredNode{
				Tag: "button",
				Handlers: []HandlerMeta{
					{
						Event: "click",
						Props: []string{"target.value"},
					},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			data, err := tt.node.ToJSON()
			if tt.wantErr {
				if err == nil {
					t.Errorf("ToJSON() expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("ToJSON() unexpected error: %v", err)
				return
			}

			restored, err := FromJSON(data)
			if err != nil {
				t.Errorf("FromJSON() unexpected error: %v", err)
				return
			}

			restoredData, err := json.Marshal(restored)
			if err != nil {
				t.Errorf("Marshal restored unexpected error: %v", err)
				return
			}

			var compact1, compact2 bytes.Buffer
			if err := json.Compact(&compact1, data); err != nil {
				t.Errorf("Compact original error: %v", err)
				return
			}
			if err := json.Compact(&compact2, restoredData); err != nil {
				t.Errorf("Compact restored error: %v", err)
				return
			}

			if compact1.String() != compact2.String() {
				t.Errorf("Roundtrip mismatch:\noriginal: %s\nrestored: %s", compact1.String(), compact2.String())
			}
		})
	}
}

// TestBuilderHelpers tests the builder pattern helpers
func TestBuilderHelpers(t *testing.T) {
	node := ElementNode("div").
		WithAttr("class", "container", "flex").
		WithStyle("color", "red").
		WithKey("item-1").
		WithChildren(
			TextNode("Hello"),
			ElementNode("span").WithChildren(TextNode("World")),
		)

	html := node.ToHTML()
	expected := `<div class="container flex" style="color: red">Hello<span>World</span></div>`

	if html != expected {
		t.Errorf("Builder pattern failed:\ngot:  %s\nwant: %s", html, expected)
	}

	if node.Key != "item-1" {
		t.Errorf("Key not set correctly: got %q, want %q", node.Key, "item-1")
	}
}

// TestApplyTo tests the Item interface implementation
func TestApplyTo(t *testing.T) {
	parent := ElementNode("div")
	child1 := TextNode("Hello")
	child2 := ElementNode("span")

	child1.ApplyTo(parent)
	child2.ApplyTo(parent)

	if len(parent.Children) != 2 {
		t.Errorf("ApplyTo failed: expected 2 children, got %d", len(parent.Children))
	}

	html := parent.ToHTML()
	expected := `<div>Hello<span></span></div>`

	if html != expected {
		t.Errorf("ApplyTo HTML output:\ngot:  %s\nwant: %s", html, expected)
	}
}
