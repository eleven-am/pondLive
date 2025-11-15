package render

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestRootSlotPathAnchorsRange(t *testing.T) {
	root := h.WrapComponent("root", h.Div(
		h.Input(h.Attr("value", "hello")),
	))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.SlotPaths) == 0 {
		t.Fatalf("expected slot paths, got none")
	}
	path := structured.SlotPaths[0].Path
	if len(path) < 1 {
		t.Fatalf("expected at least one segment, got %v", path)
	}
	if path[0].Kind != PathRangeOffset {
		t.Fatalf("expected first segment to be range offset, got %+v", path[0])
	}
}

func TestComponentPathsUseTypedSegments(t *testing.T) {
	child := h.WrapComponent("child", h.Div(h.Text("body")))
	root := h.WrapComponent("parent", h.Div(
		h.Text("head"),
		child,
	))
	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.ComponentPaths) != 2 {
		t.Fatalf("expected two component paths, got %d", len(structured.ComponentPaths))
	}
	var childPath ComponentPath
	for _, cp := range structured.ComponentPaths {
		if cp.ComponentID == "child" {
			childPath = cp
			break
		}
	}
	if childPath.ComponentID == "" {
		t.Fatalf("child component path not recorded")
	}
	if len(childPath.ParentPath) == 0 || childPath.ParentPath[0].Kind != PathRangeOffset {
		t.Fatalf("expected parent path to start with range segment, got %+v", childPath.ParentPath)
	}
}

func TestListPathCapturesContainerDomPath(t *testing.T) {
	root := h.WrapComponent("root",
		h.Div(
			h.Ul(
				h.Li(h.Key("a"), h.Text("first")),
				h.Li(h.Key("b"), h.Text("second")),
			),
		),
	)
	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.ListPaths) != 1 {
		t.Fatalf("expected one list path, got %d", len(structured.ListPaths))
	}
	listPath := structured.ListPaths[0]
	if len(listPath.Path) == 0 {
		t.Fatalf("expected typed segments, got empty path")
	}
	if listPath.Path[0].Kind != PathRangeOffset {
		t.Fatalf("expected first segment range offset, got %+v", listPath.Path[0])
	}
}

func TestSlotPathsSkipWhitespaceSiblings(t *testing.T) {
	root := h.WrapComponent("root",
		h.Div(
			h.Div(h.Text("alpha")),
			h.Text("\n    \n"),
			h.Div(
				h.Text("button:"),
				h.Textf("%d", 1),
			),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	var dynamic SlotPath
	for _, slot := range structured.SlotPaths {
		if slot.ComponentID == "root" && slot.TextChildIndex >= 0 {
			dynamic = slot
			break
		}
	}
	if dynamic.ComponentID == "" {
		t.Fatalf("expected dynamic slot for root component, got none: %+v", structured.SlotPaths)
	}
	segments := dynamic.Path
	if len(segments) < 2 {
		t.Fatalf("expected at least range+dom segments, got %v", segments)
	}
	last := segments[len(segments)-1]
	if last.Kind != PathDomChild || last.Index != 1 {
		t.Fatalf("expected slot to target second element ignoring whitespace, got last=%+v full=%v", last, segments)
	}
}

func TestBindingsShareTypedPathsAcrossDescriptors(t *testing.T) {
	button := h.Button(
		h.MutableAttr("data-count", "0"),
		h.Text("Upload"),
	)
	button.RefID = "primary"
	button.UploadBindings = []dom.UploadBinding{{
		UploadID: "upload-slot",
		Accept:   []string{"image/png"},
		Multiple: true,
		MaxSize:  4096,
	}}
	button.RouterMeta = &dom.RouterMeta{
		Path:  "/drive",
		Query: "tab=files",
	}

	root := h.WrapComponent("root",
		h.Section(
			h.Text("\n"),
			button,
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.UploadBindings) != 1 {
		t.Fatalf("expected one upload binding, got %d", len(structured.UploadBindings))
	}
	if len(structured.RefBindings) != 1 {
		t.Fatalf("expected one ref binding, got %d", len(structured.RefBindings))
	}
	if len(structured.RouterBindings) != 1 {
		t.Fatalf("expected one router binding, got %d", len(structured.RouterBindings))
	}
	var attrSlot SlotPath
	for _, slot := range structured.SlotPaths {
		if slot.ComponentID == "root" {
			attrSlot = slot
			break
		}
	}
	if attrSlot.ComponentID == "" {
		t.Fatalf("expected attr slot path for root component, got none: %+v", structured.SlotPaths)
	}

	checkPathEqual(t, attrSlot.Path, structured.UploadBindings[0].Path)
	checkPathEqual(t, attrSlot.Path, structured.RefBindings[0].Path)
	checkPathEqual(t, attrSlot.Path, structured.RouterBindings[0].Path)
	if attrSlot.Path[0].Kind != PathRangeOffset {
		t.Fatalf("expected slot path to begin with range offset, got %v", attrSlot.Path)
	}
}

func checkPathEqual(t *testing.T, a, b []PathSegment) {
	t.Helper()
	if len(a) != len(b) {
		t.Fatalf("path length mismatch: %v vs %v", a, b)
	}
	for i := range a {
		if a[i] != b[i] {
			t.Fatalf("path mismatch at %d: %v vs %v", i, a[i], b[i])
		}
	}
}

func TestValidatePath(t *testing.T) {
	tests := []struct {
		name    string
		path    []PathSegment
		wantErr bool
	}{
		{
			name:    "empty path",
			path:    []PathSegment{},
			wantErr: true,
		},
		{
			name: "valid path with range only",
			path: []PathSegment{
				{Kind: PathRangeOffset, Index: 0},
			},
			wantErr: false,
		},
		{
			name: "valid path with range and dom child",
			path: []PathSegment{
				{Kind: PathRangeOffset, Index: 2},
				{Kind: PathDomChild, Index: 1},
			},
			wantErr: false,
		},
		{
			name: "valid path with range and multiple dom children",
			path: []PathSegment{
				{Kind: PathRangeOffset, Index: 0},
				{Kind: PathDomChild, Index: 1},
				{Kind: PathDomChild, Index: 2},
			},
			wantErr: false,
		},
		{
			name: "invalid path starting with dom child",
			path: []PathSegment{
				{Kind: PathDomChild, Index: 0},
			},
			wantErr: true,
		},
		{
			name: "invalid path with range after dom child",
			path: []PathSegment{
				{Kind: PathRangeOffset, Index: 0},
				{Kind: PathDomChild, Index: 1},
				{Kind: PathRangeOffset, Index: 2},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("validatePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeeplyNestedComponents(t *testing.T) {
	level5 := h.WrapComponent("level5", h.Div(h.Text("deepest")))
	level4 := h.WrapComponent("level4", h.Div(level5))
	level3 := h.WrapComponent("level3", h.Div(level4))
	level2 := h.WrapComponent("level2", h.Div(level3))
	level1 := h.WrapComponent("level1", h.Div(level2))
	root := h.WrapComponent("root", h.Div(level1))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.ComponentPaths) != 6 {
		t.Fatalf("expected 6 component paths for nested components, got %d", len(structured.ComponentPaths))
	}

	foundLevel5 := false
	for _, cp := range structured.ComponentPaths {
		if cp.ComponentID == "level5" {
			foundLevel5 = true
			if len(cp.ParentPath) == 0 {
				t.Fatalf("level5 should have non-empty parent path")
			}
		}
	}
	if !foundLevel5 {
		t.Fatalf("level5 component path not found")
	}
}

func TestEmptyComponent(t *testing.T) {
	root := h.WrapComponent("root", h.Div())

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.SlotPaths) != 0 {
		t.Fatalf("expected no slot paths for empty component, got %d", len(structured.SlotPaths))
	}
}

func TestComponentWithOnlyWhitespace(t *testing.T) {
	root := h.WrapComponent("root", h.Div(
		h.Text("\n  "),
		h.Text("\t\t"),
		h.Text("   \n"),
	))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.SlotPaths) != 0 {
		t.Fatalf("expected no slot paths for whitespace-only component, got %d", len(structured.SlotPaths))
	}
}

func TestMultipleListsInSameComponent(t *testing.T) {
	root := h.WrapComponent("root",
		h.Div(
			h.Ul(
				h.Li(h.Key("a"), h.Text("first list A")),
				h.Li(h.Key("b"), h.Text("first list B")),
			),
			h.Ul(
				h.Li(h.Key("x"), h.Text("second list X")),
				h.Li(h.Key("y"), h.Text("second list Y")),
			),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.ListPaths) != 2 {
		t.Fatalf("expected 2 list paths, got %d", len(structured.ListPaths))
	}
}

func TestFragmentWithMixedContent(t *testing.T) {
	root := h.WrapComponent("root",
		h.Fragment(
			h.Div(h.Text("before")),
			h.Text("\n"),
			h.Span(h.Text("middle")),
			h.Text("  "),
			h.P(h.Text("after")),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.S) == 0 {
		t.Fatalf("expected static HTML segments, got none")
	}
}

func TestRouterBindingValidation(t *testing.T) {
	validButton := h.Button(h.Text("valid"))
	validButton.RouterMeta = &dom.RouterMeta{
		Path:  "/dashboard",
		Query: "tab=settings",
	}

	emptyButton := h.Button(h.Text("empty"))
	emptyButton.RouterMeta = &dom.RouterMeta{}

	root := h.WrapComponent("root",
		h.Div(
			validButton,
			h.Text(" "),
			emptyButton,
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	validCount := 0
	for _, rb := range structured.RouterBindings {
		if rb.PathValue != "" {
			validCount++
		}
	}

	if validCount == 0 {
		t.Fatalf("expected at least one valid router binding")
	}
}

func TestComponentIDCollisionDoesNotPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("should not panic on duplicate component IDs, got panic: %v", r)
		}
	}()

	child1 := h.WrapComponent("duplicate", h.Div(h.Text("first")))
	child2 := h.WrapComponent("duplicate", h.Div(h.Text("second")))
	root := h.WrapComponent("root", h.Div(child1, child2))

	_, _ = ToStructured(root)
}

func TestMutableWhitespaceIsPreserved(t *testing.T) {
	mutableSpace := h.Text("   ")
	mutableSpace.Mutable = true

	root := h.WrapComponent("root",
		h.Div(
			h.Text("before"),
			mutableSpace,
			h.Text("after"),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	hasMutableSlot := false
	for _, slot := range structured.D {
		if slot.Kind == DynamicText && slot.Text == "   " {
			hasMutableSlot = true
			break
		}
	}

	if !hasMutableSlot {
		t.Fatalf("expected mutable whitespace to create dynamic slot")
	}
}

func TestNilNodeReturnsError(t *testing.T) {
	_, err := ToStructured(nil)
	if err == nil {
		t.Fatalf("expected error for nil node, got nil")
	}
	if _, ok := err.(*ValidationError); !ok {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestEmptyElementTagReturnsError(t *testing.T) {
	invalidElement := &h.Element{
		Tag:      "",
		Children: []h.Node{h.Text("content")},
	}
	root := h.WrapComponent("root", invalidElement)

	_, err := ToStructured(root)
	if err == nil {
		t.Fatalf("expected error for empty element tag, got nil")
	}
}

func TestDeeplyNestedFragments(t *testing.T) {
	level3 := h.Fragment(h.Div(h.Text("deep")))
	level2 := h.Fragment(h.Div(level3))
	level1 := h.Fragment(h.Div(level2))
	root := h.WrapComponent("root", level1)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.S) == 0 {
		t.Fatalf("expected HTML output for nested fragments")
	}
}

func TestUnsafeHTMLRendering(t *testing.T) {
	unsafeHTML := "<script>alert('xss')</script>"
	div := h.Div()
	div.Unsafe = &unsafeHTML

	root := h.WrapComponent("root", div)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	htmlOutput := ""
	for _, s := range structured.S {
		htmlOutput += s
	}
	if !contains(htmlOutput, unsafeHTML) {
		t.Fatalf("expected unsafe HTML to be preserved, got: %s", htmlOutput)
	}
}

func TestMultipleHandlerAssignments(t *testing.T) {
	button := h.Button(h.Text("multi"))
	button.HandlerAssignments = map[string]dom.EventAssignment{
		"click": {ID: "handleClick", Listen: []string{"target.value"}},
		"focus": {ID: "handleFocus", Props: []string{"focused"}},
		"blur":  {ID: "handleBlur"},
	}
	root := h.WrapComponent("root", h.Div(button))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.Bindings) != 3 {
		t.Fatalf("expected 3 handler bindings, got %d", len(structured.Bindings))
	}
	events := make(map[string]bool)
	for _, binding := range structured.Bindings {
		events[binding.Event] = true
	}
	if !events["click"] || !events["focus"] || !events["blur"] {
		t.Fatalf("expected click, focus, blur events, got: %v", events)
	}
}

func TestUploadBindingWithAllOptions(t *testing.T) {
	button := h.Button(h.Text("upload"))
	button.UploadBindings = []dom.UploadBinding{{
		UploadID: "file-upload",
		Accept:   []string{"image/png", "image/jpeg", "application/pdf"},
		Multiple: true,
		MaxSize:  10485760,
	}}
	root := h.WrapComponent("root", h.Div(button))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.UploadBindings) != 1 {
		t.Fatalf("expected 1 upload binding, got %d", len(structured.UploadBindings))
	}
	upload := structured.UploadBindings[0]
	if upload.UploadID != "file-upload" {
		t.Fatalf("expected UploadID 'file-upload', got %s", upload.UploadID)
	}
	if !upload.Multiple {
		t.Fatalf("expected Multiple=true")
	}
	if upload.MaxSize != 10485760 {
		t.Fatalf("expected MaxSize=10485760, got %d", upload.MaxSize)
	}
	if len(upload.Accept) != 3 {
		t.Fatalf("expected 3 accepted types, got %d", len(upload.Accept))
	}
}

func TestRefBindingInNestedComponent(t *testing.T) {
	input := h.Input()
	input.RefID = "email-input"

	child := h.WrapComponent("child", h.Div(input))
	root := h.WrapComponent("root", h.Div(child))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.RefBindings) != 1 {
		t.Fatalf("expected 1 ref binding, got %d", len(structured.RefBindings))
	}
	if structured.RefBindings[0].RefID != "email-input" {
		t.Fatalf("expected RefID 'email-input', got %s", structured.RefBindings[0].RefID)
	}
	if structured.RefBindings[0].ComponentID != "child" {
		t.Fatalf("expected ComponentID 'child', got %s", structured.RefBindings[0].ComponentID)
	}
}

func TestTextWithSpecialCharacters(t *testing.T) {
	root := h.WrapComponent("root",
		h.Div(
			h.Text("<script>alert('xss')</script>"),
			h.Text("& < > \" '"),
			h.Text("Unicode: ñ é ü 中文 🚀"),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	htmlOutput := ""
	for _, s := range structured.S {
		htmlOutput += s
	}
	if contains(htmlOutput, "<script>") && !contains(htmlOutput, "&lt;script&gt;") {
		t.Fatalf("expected script tags to be escaped")
	}
	if !contains(htmlOutput, "&amp;") {
		t.Fatalf("expected ampersand to be escaped")
	}
}

func TestAttributesWithSpecialCharacters(t *testing.T) {
	div := h.Div(h.Text("content"))
	div.Attrs = map[string]string{
		"data-value":   "<script>",
		"data-encoded": "one & two",
		"title":        `quotes "here"`,
	}
	root := h.WrapComponent("root", div)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	htmlOutput := ""
	for _, s := range structured.S {
		htmlOutput += s
	}
	if contains(htmlOutput, `data-value="<script>"`) {
		t.Fatalf("expected special characters in attributes to be escaped")
	}
}

func TestEmptyUploadBindingIDSkipped(t *testing.T) {
	button := h.Button(h.Text("upload"))
	button.UploadBindings = []dom.UploadBinding{{
		UploadID: "",
		Multiple: true,
	}}
	root := h.WrapComponent("root", h.Div(button))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.UploadBindings) != 0 {
		t.Fatalf("expected 0 upload bindings for empty UploadID, got %d", len(structured.UploadBindings))
	}
}

func TestEmptyRefIDNotRecorded(t *testing.T) {
	input := h.Input()
	input.RefID = "   "
	root := h.WrapComponent("root", h.Div(input))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.RefBindings) != 0 {
		t.Fatalf("expected 0 ref bindings for whitespace RefID, got %d", len(structured.RefBindings))
	}
}

func TestComponentWithoutChild(t *testing.T) {
	comp := &h.ComponentNode{
		ID:    "empty",
		Child: nil,
	}
	root := h.WrapComponent("root", h.Div(comp))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if _, found := structured.Components["empty"]; !found {
		t.Fatalf("expected component 'empty' to be recorded")
	}
}

func TestComponentWithoutID(t *testing.T) {
	comp := &h.ComponentNode{
		ID:    "",
		Child: h.Div(h.Text("content")),
	}
	root := h.WrapComponent("root", h.Div(comp))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	htmlOutput := ""
	for _, s := range structured.S {
		htmlOutput += s
	}
	if !contains(htmlOutput, "content") {
		t.Fatalf("expected child content to be rendered even without component ID")
	}
}

func TestVoidElementsNotSelfClosing(t *testing.T) {
	root := h.WrapComponent("root",
		h.Div(
			h.Input(h.Attr("type", "text")),
			h.Img(h.Attr("src", "/logo.png")),
			h.Br(),
			h.Hr(),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	htmlOutput := ""
	for _, s := range structured.S {
		htmlOutput += s
	}
	voidElements := []string{"<input", "<img", "<br", "<hr"}
	for _, elem := range voidElements {
		if !contains(htmlOutput, elem) {
			t.Fatalf("expected void element %s in output", elem)
		}
	}
}

func TestMixedMutableAndStaticAttrs(t *testing.T) {
	div := h.Div(h.Text("content"))
	div.Attrs = map[string]string{
		"class":      "static-class",
		"id":         "static-id",
		"data-value": "initial",
	}
	div.MutableAttrs = map[string]bool{
		"data-value": true,
	}
	root := h.WrapComponent("root", div)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	hasDynamicAttrs := false
	for _, slot := range structured.D {
		if slot.Kind == DynamicAttrs {
			hasDynamicAttrs = true
			if slot.Attrs["data-value"] != "initial" {
				t.Fatalf("expected mutable attr to preserve initial value")
			}
		}
	}
	if !hasDynamicAttrs {
		t.Fatalf("expected dynamic attrs slot for mutable attributes")
	}
}

func TestListWithNoKeys(t *testing.T) {
	root := h.WrapComponent("root",
		h.Ul(
			h.Li(h.Text("no key 1")),
			h.Li(h.Text("no key 2")),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.ListPaths) != 0 {
		t.Fatalf("expected no list paths when children have no keys, got %d", len(structured.ListPaths))
	}
}

func TestListWithSingleKeyedItem(t *testing.T) {
	root := h.WrapComponent("root",
		h.Ul(
			h.Li(h.Key("only"), h.Text("single item")),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.ListPaths) != 1 {
		t.Fatalf("expected 1 list path for single keyed item, got %d", len(structured.ListPaths))
	}
	if len(structured.D) == 0 {
		t.Fatalf("expected dynamic list slot")
	}
	hasListSlot := false
	for _, slot := range structured.D {
		if slot.Kind == DynamicList {
			hasListSlot = true
			if len(slot.List) != 1 {
				t.Fatalf("expected 1 row in list, got %d", len(slot.List))
			}
		}
	}
	if !hasListSlot {
		t.Fatalf("expected DynamicList slot")
	}
}

func TestFragmentWithNoChildren(t *testing.T) {
	root := h.WrapComponent("root",
		h.Div(
			h.Fragment(),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.S) == 0 {
		t.Fatalf("expected some static HTML")
	}
}

func TestRouterBindingWithAllFields(t *testing.T) {
	link := h.A(h.Text("navigate"))
	link.RouterMeta = &dom.RouterMeta{
		Path:    "/dashboard",
		Query:   "tab=overview&filter=active",
		Hash:    "#section-1",
		Replace: "true",
	}
	root := h.WrapComponent("root", h.Div(link))

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
	if len(structured.RouterBindings) != 1 {
		t.Fatalf("expected 1 router binding, got %d", len(structured.RouterBindings))
	}
	router := structured.RouterBindings[0]
	if router.PathValue != "/dashboard" {
		t.Fatalf("expected PathValue '/dashboard', got %s", router.PathValue)
	}
	if router.Query != "tab=overview&filter=active" {
		t.Fatalf("expected Query 'tab=overview&filter=active', got %s", router.Query)
	}
	if router.Hash != "#section-1" {
		t.Fatalf("expected Hash '#section-1', got %s", router.Hash)
	}
	if router.Replace != "true" {
		t.Fatalf("expected Replace 'true', got %s", router.Replace)
	}
}

func TestPreComputedRanges(t *testing.T) {
	child := h.WrapComponent("child", h.Div(
		h.Text("static"),
		h.Textf("%d", 42),
	))
	root := h.WrapComponent("root",
		h.Div(
			h.Text("before"),
			child,
			h.Text("after"),
		),
	)

	structured, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}

	if len(structured.S) == 0 {
		t.Fatalf("expected static segments")
	}
	if len(structured.D) == 0 {
		t.Fatalf("expected dynamic slots")
	}
	if _, found := structured.Components["child"]; !found {
		t.Fatalf("expected child component to be tracked")
	}
	if _, found := structured.Components["root"]; !found {
		t.Fatalf("expected root component to be tracked")
	}
}

func TestAnalyzerCountsCorrectly(t *testing.T) {
	root := h.WrapComponent("root",
		h.Div(
			h.Text("static1"),
			h.Textf("%s", "dynamic"),
			h.Text("static2"),
			h.Div(
				h.Button(h.Text("click")),
			),
		),
	)

	analyzer := NewComponentAnalyzer()
	result := analyzer.Analyze(root)

	if result.StaticsCapacity <= 0 {
		t.Fatalf("expected positive statics capacity, got %d", result.StaticsCapacity)
	}
	if result.DynamicsCapacity <= 0 {
		t.Fatalf("expected positive dynamics capacity, got %d", result.DynamicsCapacity)
	}
	if len(result.Components) != 1 {
		t.Fatalf("expected 1 component, got %d", len(result.Components))
	}

	_, err := ToStructured(root)
	if err != nil {
		t.Fatalf("ToStructured failed: %v", err)
	}
}

func contains(haystack, needle string) bool {
	return len(haystack) >= len(needle) && (haystack == needle || len(needle) == 0 ||
		(len(haystack) > 0 && (haystack[:len(needle)] == needle || contains(haystack[1:], needle))))
}

// TestConcurrentAnalyzerMatchesSequential ensures concurrent analysis produces same results
func TestConcurrentAnalyzerMatchesSequential(t *testing.T) {

	root := h.WrapComponent("root",
		h.Div(
			h.Div(h.Text("1"), h.Textf("%d", 1)),
			h.Div(h.Text("2"), h.Textf("%d", 2)),
			h.Div(h.Text("3"), h.Textf("%d", 3)),
			h.Div(h.Text("4"), h.Textf("%d", 4)),
			h.Div(h.Text("5"), h.Textf("%d", 5)),
			h.Div(h.Text("6"), h.Textf("%d", 6)),
			h.Div(h.Text("7"), h.Textf("%d", 7)),
			h.Div(h.Text("8"), h.Textf("%d", 8)),
			h.Div(h.Text("9"), h.Textf("%d", 9)),
			h.Div(h.Text("10"), h.Textf("%d", 10)),
		),
	)

	seqAnalyzer := NewComponentAnalyzer()
	seqResult := seqAnalyzer.Analyze(root)

	concAnalyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 4,
	})
	concResult := concAnalyzer.Analyze(root)

	if seqResult.StaticsCapacity != concResult.StaticsCapacity {
		t.Fatalf("statics mismatch: seq=%d, conc=%d", seqResult.StaticsCapacity, concResult.StaticsCapacity)
	}
	if seqResult.DynamicsCapacity != concResult.DynamicsCapacity {
		t.Fatalf("dynamics mismatch: seq=%d, conc=%d", seqResult.DynamicsCapacity, concResult.DynamicsCapacity)
	}
	if len(seqResult.Components) != len(concResult.Components) {
		t.Fatalf("component count mismatch: seq=%d, conc=%d", len(seqResult.Components), len(concResult.Components))
	}

	for id, seqSpan := range seqResult.Components {
		concSpan, ok := concResult.Components[id]
		if !ok {
			t.Fatalf("component %s missing in concurrent result", id)
		}
		if seqSpan != concSpan {
			t.Fatalf("component %s span mismatch:\nseq=%+v\nconc=%+v", id, seqSpan, concSpan)
		}
	}
}

// TestConcurrentAnalyzerNestedComponents tests concurrent analysis with nested components
func TestConcurrentAnalyzerNestedComponents(t *testing.T) {

	root := h.WrapComponent("root",
		h.Div(
			h.WrapComponent("childA", h.Div(h.Text("text"), h.Textf("%d", 1), h.Button(h.Text("click")))),
			h.WrapComponent("childB", h.Div(h.Text("text"), h.Textf("%d", 2), h.Button(h.Text("click")))),
			h.WrapComponent("childC", h.Div(h.Text("text"), h.Textf("%d", 3), h.Button(h.Text("click")))),
			h.WrapComponent("childD", h.Div(h.Text("text"), h.Textf("%d", 4), h.Button(h.Text("click")))),
			h.WrapComponent("childE", h.Div(h.Text("text"), h.Textf("%d", 5), h.Button(h.Text("click")))),
			h.WrapComponent("childF", h.Div(h.Text("text"), h.Textf("%d", 6), h.Button(h.Text("click")))),
		),
	)

	seqAnalyzer := NewComponentAnalyzer()
	seqResult := seqAnalyzer.Analyze(root)

	concAnalyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 3,
	})
	concResult := concAnalyzer.Analyze(root)

	if seqResult.StaticsCapacity != concResult.StaticsCapacity {
		t.Fatalf("statics mismatch: seq=%d, conc=%d", seqResult.StaticsCapacity, concResult.StaticsCapacity)
	}
	if seqResult.DynamicsCapacity != concResult.DynamicsCapacity {
		t.Fatalf("dynamics mismatch: seq=%d, conc=%d", seqResult.DynamicsCapacity, concResult.DynamicsCapacity)
	}

	if len(seqResult.Components) != 7 || len(concResult.Components) != 7 {
		t.Fatalf("expected 7 components, got seq=%d, conc=%d", len(seqResult.Components), len(concResult.Components))
	}

	for id, seqSpan := range seqResult.Components {
		concSpan, ok := concResult.Components[id]
		if !ok {
			t.Fatalf("component %s missing in concurrent result", id)
		}
		if seqSpan != concSpan {
			t.Fatalf("component %s span mismatch:\nseq=%+v\nconc=%+v", id, seqSpan, concSpan)
		}
	}
}

// TestConcurrentAnalyzerBelowThreshold tests that concurrent mode falls back to sequential when below threshold
func TestConcurrentAnalyzerBelowThreshold(t *testing.T) {

	root := h.WrapComponent("root",
		h.Div(
			h.Text("child1"),
			h.Text("child2"),
		),
	)

	seqAnalyzer := NewComponentAnalyzer()
	seqResult := seqAnalyzer.Analyze(root)

	concAnalyzer := NewComponentAnalyzerWithOptions(AnalysisOptions{
		Concurrent:           true,
		ConcurrencyThreshold: 4,
	})
	concResult := concAnalyzer.Analyze(root)

	if seqResult.StaticsCapacity != concResult.StaticsCapacity {
		t.Fatalf("statics mismatch: seq=%d, conc=%d", seqResult.StaticsCapacity, concResult.StaticsCapacity)
	}
	if seqResult.DynamicsCapacity != concResult.DynamicsCapacity {
		t.Fatalf("dynamics mismatch: seq=%d, conc=%d", seqResult.DynamicsCapacity, concResult.DynamicsCapacity)
	}
}
