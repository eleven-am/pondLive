package render

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// TestChildComponentBoundaries tests that when a parent component renders a child component,
// the child component's FirstChild and LastChild paths are correctly set.
// This reproduces the bug where child components have empty boundary paths.
func TestChildComponentBoundaries(t *testing.T) {
	// Simulate parent component with structure:
	// <div>
	//   <h1>Title</h1>              <!-- index 0 -->
	//   <div>First div</div>        <!-- index 1 -->
	//   <button>Click</button>      <!-- index 2 -->
	//   <input />                   <!-- index 3 -->
	//   <ComponentNode>             <!-- index 4: child component -->
	//     <div>Child content</div>
	//   </ComponentNode>
	// </div>

	parent := &h.Element{
		Tag: "div",
		Children: []h.Node{
			&h.Element{Tag: "h1", Children: []h.Node{&h.TextNode{Value: "Title"}}},
			&h.Element{Tag: "div", Children: []h.Node{&h.TextNode{Value: "First div"}}},
			&h.Element{Tag: "button", Children: []h.Node{&h.TextNode{Value: "Click"}}},
			&h.Element{Tag: "input"},
			&h.ComponentNode{
				ID: "child-component-id",
				Child: &h.Element{
					Tag: "div",
					Children: []h.Node{
						&h.TextNode{Value: "Child content"},
					},
				},
			},
		},
	}

	root := &h.ComponentNode{
		ID:    "parent-component-id",
		Child: parent,
	}

	result, err := ToStructuredWithOptions(root, StructuredOptions{})
	if err != nil {
		t.Fatalf("ToStructuredWithOptions failed: %v", err)
	}

	// Find the child component path
	var childPath *ComponentPath
	for i := range result.ComponentPaths {
		if result.ComponentPaths[i].ComponentID == "child-component-id" {
			childPath = &result.ComponentPaths[i]
			break
		}
	}

	if childPath == nil {
		t.Fatal("child component path not found in ComponentPaths")
	}

	t.Logf("Child component path: %+v", childPath)
	t.Logf("Child FirstChild: %+v", childPath.FirstChild)
	t.Logf("Child LastChild: %+v", childPath.LastChild)

	// The child component is at index 4 in the parent div
	// So FirstChild should point to that location
	if len(childPath.FirstChild) == 0 {
		t.Fatalf("child component FirstChild is empty, expected path to index 4")
	}

	// FirstChild should have a DOM path segment pointing to index 4
	// The path should be relative to the parent component's range
	if childPath.FirstChild[0].Kind != PathDomChild {
		t.Fatalf("expected FirstChild[0] to be DOM child segment, got %v", childPath.FirstChild[0].Kind)
	}

	if childPath.FirstChild[0].Index != 4 {
		t.Fatalf("expected FirstChild[0].Index to be 4 (position in parent), got %d", childPath.FirstChild[0].Index)
	}

	// LastChild should also point to index 4 since the child only occupies one slot
	if len(childPath.LastChild) == 0 {
		t.Fatalf("child component LastChild is empty, expected path to index 4")
	}

	if childPath.LastChild[0].Kind != PathDomChild {
		t.Fatalf("expected LastChild[0] to be DOM child segment, got %v", childPath.LastChild[0].Kind)
	}

	if childPath.LastChild[0].Index != 4 {
		t.Fatalf("expected LastChild[0].Index to be 4, got %d", childPath.LastChild[0].Index)
	}

	// Verify parent component also has correct boundaries
	var parentPath *ComponentPath
	for i := range result.ComponentPaths {
		if result.ComponentPaths[i].ComponentID == "parent-component-id" {
			parentPath = &result.ComponentPaths[i]
			break
		}
	}

	if parentPath == nil {
		t.Fatal("parent component path not found")
	}

	// Parent should have FirstChild pointing to its first element (the div)
	if len(parentPath.FirstChild) == 0 {
		t.Fatal("parent component FirstChild is empty")
	}

	// Parent's LastChild should point to its last element (also the div, since it wraps everything)
	if len(parentPath.LastChild) == 0 {
		t.Fatal("parent component LastChild is empty")
	}
}

// TestNestedChildComponentBoundaries tests a more complex scenario with nested child components
func TestNestedChildComponentBoundaries(t *testing.T) {
	// Structure:
	// <div>
	//   <h1>Header</h1>                    <!-- index 0 -->
	//   <ComponentNode id="middle">        <!-- index 1 -->
	//     <div>
	//       <p>Middle para</p>              <!-- index 0 within middle -->
	//       <ComponentNode id="inner">      <!-- index 1 within middle -->
	//         <span>Inner</span>
	//       </ComponentNode>
	//     </div>
	//   </ComponentNode>
	//   <footer>Footer</footer>            <!-- index 2 -->
	// </div>

	innerComponent := &h.ComponentNode{
		ID: "inner-component",
		Child: &h.Element{
			Tag: "span",
			Children: []h.Node{
				&h.TextNode{Value: "Inner"},
			},
		},
	}

	middleComponent := &h.ComponentNode{
		ID: "middle-component",
		Child: &h.Element{
			Tag: "div",
			Children: []h.Node{
				&h.Element{
					Tag: "p",
					Children: []h.Node{
						&h.TextNode{Value: "Middle para"},
					},
				},
				innerComponent,
			},
		},
	}

	root := &h.ComponentNode{
		ID: "root-component",
		Child: &h.Element{
			Tag: "div",
			Children: []h.Node{
				&h.Element{
					Tag: "h1",
					Children: []h.Node{
						&h.TextNode{Value: "Header"},
					},
				},
				middleComponent,
				&h.Element{
					Tag: "footer",
					Children: []h.Node{
						&h.TextNode{Value: "Footer"},
					},
				},
			},
		},
	}

	result, err := ToStructuredWithOptions(root, StructuredOptions{})
	if err != nil {
		t.Fatalf("ToStructuredWithOptions failed: %v", err)
	}

	// Find all component paths
	paths := make(map[string]*ComponentPath)
	for i := range result.ComponentPaths {
		paths[result.ComponentPaths[i].ComponentID] = &result.ComponentPaths[i]
	}

	// Verify middle component boundaries
	middlePath := paths["middle-component"]
	if middlePath == nil {
		t.Fatal("middle component path not found")
	}

	if len(middlePath.FirstChild) == 0 {
		t.Fatalf("middle component FirstChild is empty")
	}

	// Middle component is at index 1 in root's div
	if middlePath.FirstChild[0].Index != 1 {
		t.Fatalf("expected middle FirstChild at index 1, got %d", middlePath.FirstChild[0].Index)
	}

	if len(middlePath.LastChild) == 0 {
		t.Fatalf("middle component LastChild is empty")
	}

	if middlePath.LastChild[0].Index != 1 {
		t.Fatalf("expected middle LastChild at index 1, got %d", middlePath.LastChild[0].Index)
	}

	// Verify inner component boundaries
	innerPath := paths["inner-component"]
	if innerPath == nil {
		t.Fatal("inner component path not found")
	}

	if len(innerPath.FirstChild) == 0 {
		t.Fatalf("inner component FirstChild is empty")
	}

	// Inner component is at index 1 within middle component's div
	if innerPath.FirstChild[0].Index != 1 {
		t.Fatalf("expected inner FirstChild at index 1, got %d", innerPath.FirstChild[0].Index)
	}

	if len(innerPath.LastChild) == 0 {
		t.Fatalf("inner component LastChild is empty")
	}

	if innerPath.LastChild[0].Index != 1 {
		t.Fatalf("expected inner LastChild at index 1, got %d", innerPath.LastChild[0].Index)
	}
}
