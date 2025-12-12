package styles

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/view"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestProvider(t *testing.T) {
	t.Run("provides styles context", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx, &work.Text{Value: "child content"})
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if sess.View == nil {
			t.Fatal("expected View after Flush")
		}

		html := view.RenderHTML(sess.View)
		if !strings.Contains(html, "child content") {
			t.Error("expected child content in output")
		}
	})

	t.Run("with multiple children", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx,
					&work.Element{Tag: "div", Children: []work.Node{&work.Text{Value: "first"}}},
					&work.Element{Tag: "span", Children: []work.Node{&work.Text{Value: "second"}}},
				)
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		html := view.RenderHTML(sess.View)
		if !strings.Contains(html, "first") || !strings.Contains(html, "second") {
			t.Error("expected both children in output")
		}
	})
}

func TestRender(t *testing.T) {
	t.Run("renders slot content", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return Provider(ctx,
					&work.Element{
						Tag: "html",
						Children: []work.Node{
							&work.Element{
								Tag:      "head",
								Children: []work.Node{Render(ctx)},
							},
							&work.Element{
								Tag:      "body",
								Children: []work.Node{&work.Text{Value: "content"}},
							},
						},
					},
				)
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if sess.View == nil {
			t.Fatal("expected View after Flush")
		}
	})
}

func TestUseStyles(t *testing.T) {
	t.Run("creates styles with CSS", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				styles := UseStyles(ctx, ".test { color: red; }")

				return Provider(ctx,
					&work.Element{
						Tag: "html",
						Children: []work.Node{
							&work.Element{
								Tag:      "head",
								Children: []work.Node{Render(ctx)},
							},
							&work.Element{
								Tag: "body",
								Children: []work.Node{
									&work.Element{
										Tag:      "div",
										Attrs:    map[string][]string{"class": {styles.Class("test")}},
										Children: []work.Node{&work.Text{Value: "styled"}},
									},
								},
							},
						},
					},
				)
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		html := view.RenderHTML(sess.View)
		if !strings.Contains(html, "styled") {
			t.Error("expected styled content in output")
		}
		t.Logf("HTML output: %s", html)
	})

	t.Run("returns styles object", func(t *testing.T) {
		var capturedStyles *Styles

		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				capturedStyles = UseStyles(ctx, ".foo { margin: 0; }")
				return Provider(ctx,
					&work.Element{
						Tag:      "head",
						Children: []work.Node{Render(ctx)},
					},
				)
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		if capturedStyles == nil {
			t.Fatal("expected styles to be returned")
		}

		className := capturedStyles.Class("foo")
		if className == "" {
			t.Error("expected non-empty class name")
		}
	})

	t.Run("multiple UseStyles calls", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				styles1 := UseStyles(ctx, ".a { color: red; }")
				styles2 := UseStyles(ctx, ".b { color: blue; }")

				return Provider(ctx,
					&work.Element{
						Tag: "html",
						Children: []work.Node{
							&work.Element{
								Tag:      "head",
								Children: []work.Node{Render(ctx)},
							},
							&work.Element{
								Tag: "body",
								Children: []work.Node{
									&work.Element{
										Tag:   "div",
										Attrs: map[string][]string{"class": {styles1.Class("a")}},
									},
									&work.Element{
										Tag:   "span",
										Attrs: map[string][]string{"class": {styles2.Class("b")}},
									},
								},
							},
						},
					},
				)
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		html := view.RenderHTML(sess.View)
		t.Logf("Multiple styles HTML: %s", html)
	})
}

func TestProviderAndRenderIntegration(t *testing.T) {
	t.Run("full integration", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				styles := UseStyles(ctx, `
					.container { display: flex; }
					.title { font-size: 24px; }
				`)

				return Provider(ctx,
					&work.Element{
						Tag: "html",
						Children: []work.Node{
							&work.Element{
								Tag:      "head",
								Children: []work.Node{Render(ctx)},
							},
							&work.Element{
								Tag: "body",
								Children: []work.Node{
									&work.Element{
										Tag:   "div",
										Attrs: map[string][]string{"class": {styles.Class("container")}},
										Children: []work.Node{
											&work.Element{
												Tag:      "h1",
												Attrs:    map[string][]string{"class": {styles.Class("title")}},
												Children: []work.Node{&work.Text{Value: "Hello"}},
											},
										},
									},
								},
							},
						},
					},
				)
			},
			HookFrame: []runtime.HookSlot{},
			Children:  []*runtime.Instance{},
		}

		sess := &runtime.Session{
			Root:              root,
			Components:        map[string]*runtime.Instance{"root": root},
			Handlers:          make(map[string]work.Handler),
			MountedComponents: make(map[*runtime.Instance]struct{}),
			Bus:               protocol.NewBus(),
		}

		if err := sess.Flush(); err != nil {
			t.Fatalf("Flush failed: %v", err)
		}

		html := view.RenderHTML(sess.View)

		if !strings.Contains(html, "Hello") {
			t.Error("expected content")
		}
		t.Logf("Integration HTML: %s", html)
	})
}
