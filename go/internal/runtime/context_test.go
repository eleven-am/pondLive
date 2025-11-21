package runtime

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/dom"
	dom2diff "github.com/eleven-am/pondlive/go/internal/dom/diff"
)

func TestContextDefault(t *testing.T) {
	ctx := CreateContext("default")

	var value string
	comp := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		value = ctx.Use(rctx)
		return &dom.StructuredNode{Tag: "div"}
	}

	sess := NewSession(comp, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if value != "default" {
		t.Errorf("expected default value 'default', got %q", value)
	}
}

func TestContextProvide(t *testing.T) {
	ctx := CreateContext("default")

	var childValue string
	child := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		childValue = ctx.Use(rctx)
		return &dom.StructuredNode{Tag: "span"}
	}

	parent := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		return ctx.Provide(rctx, "provided", func(pctx Ctx) *dom.StructuredNode {
			return Render(pctx, child, struct{}{})
		})
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if childValue != "provided" {
		t.Errorf("expected provided value 'provided', got %q", childValue)
	}
}

func TestContextNestedProviders(t *testing.T) {
	ctx := CreateContext(0)

	var value1, value2, value3 int

	leaf := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		value3 = ctx.Use(rctx)
		return &dom.StructuredNode{Tag: "span"}
	}

	middle := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		value2 = ctx.Use(rctx)
		return ctx.Provide(rctx, 20, func(pctx Ctx) *dom.StructuredNode {
			return Render(pctx, leaf, struct{}{})
		})
	}

	root := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		value1 = ctx.Use(rctx)
		return ctx.Provide(rctx, 10, func(pctx Ctx) *dom.StructuredNode {
			return Render(pctx, middle, struct{}{})
		})
	}

	sess := NewSession(root, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if value1 != 0 {
		t.Errorf("expected root to see default 0, got %d", value1)
	}
	if value2 != 10 {
		t.Errorf("expected middle to see 10, got %d", value2)
	}
	if value3 != 20 {
		t.Errorf("expected leaf to see 20, got %d", value3)
	}
}

func TestContextMultipleContexts(t *testing.T) {
	strCtx := CreateContext("default-str")
	intCtx := CreateContext(42)

	var strValue string
	var intValue int

	child := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		strValue = strCtx.Use(rctx)
		intValue = intCtx.Use(rctx)
		return &dom.StructuredNode{Tag: "span"}
	}

	parent := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		return strCtx.Provide(rctx, "hello", func(pctx1 Ctx) *dom.StructuredNode {
			return intCtx.Provide(pctx1, 100, func(pctx2 Ctx) *dom.StructuredNode {
				return Render(pctx2, child, struct{}{})
			})
		})
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if strValue != "hello" {
		t.Errorf("expected strValue 'hello', got %q", strValue)
	}
	if intValue != 100 {
		t.Errorf("expected intValue 100, got %d", intValue)
	}
}

func TestContextProviderScope(t *testing.T) {
	ctx := CreateContext("default")

	var child1Value, child2Value string

	child1 := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		child1Value = ctx.Use(rctx)
		return &dom.StructuredNode{Tag: "span"}
	}

	child2 := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		child2Value = ctx.Use(rctx)
		return &dom.StructuredNode{Tag: "span"}
	}

	root := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		providedChild := ctx.Provide(rctx, "scoped", func(pctx Ctx) *dom.StructuredNode {
			return Render(pctx, child1, struct{}{})
		})

		regularChild := Render(rctx, child2, struct{}{})

		return &dom.StructuredNode{
			Tag:      "div",
			Children: []*dom.StructuredNode{providedChild, regularChild},
		}
	}

	sess := NewSession(root, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if child1Value != "scoped" {
		t.Errorf("expected child1 to see 'scoped', got %q", child1Value)
	}
	if child2Value != "default" {
		t.Errorf("expected child2 to see 'default' (not scoped), got %q", child2Value)
	}
}

func TestContextTypeStability(t *testing.T) {
	ctx1 := CreateContext(0)
	ctx2 := CreateContext(0)

	if ctx1.id == ctx2.id {
		t.Error("expected different context IDs for different contexts")
	}

	id1 := ctx1.id
	id2 := ctx1.id
	if id1 != id2 {
		t.Error("expected context ID to be stable")
	}
}

func TestContextWithStruct(t *testing.T) {
	type User struct {
		Name string
		ID   int
	}

	userCtx := CreateContext(User{Name: "guest", ID: 0})

	var userName string
	var userID int

	child := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		user := userCtx.Use(rctx)
		userName = user.Name
		userID = user.ID
		return &dom.StructuredNode{Tag: "div"}
	}

	parent := func(rctx Ctx, props struct{}) *dom.StructuredNode {
		return userCtx.Provide(rctx, User{Name: "Alice", ID: 123}, func(pctx Ctx) *dom.StructuredNode {
			return Render(pctx, child, struct{}{})
		})
	}

	sess := NewSession(parent, struct{}{})
	sess.SetPatchSender(func(patches []dom2diff.Patch) error { return nil })

	if err := sess.Flush(); err != nil {
		t.Fatalf("flush failed: %v", err)
	}

	if userName != "Alice" {
		t.Errorf("expected userName 'Alice', got %q", userName)
	}
	if userID != 123 {
		t.Errorf("expected userID 123, got %d", userID)
	}
}
