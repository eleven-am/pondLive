package server

import (
	"testing"

	"github.com/eleven-am/pondlive/go/internal/headers"
	"github.com/eleven-am/pondlive/go/internal/html"
	"github.com/eleven-am/pondlive/go/internal/metatags"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	"github.com/eleven-am/pondlive/go/internal/router"
	"github.com/eleven-am/pondlive/go/internal/runtime"
	"github.com/eleven-am/pondlive/go/internal/session"
	"github.com/eleven-am/pondlive/go/internal/styles"
	"github.com/eleven-am/pondlive/go/internal/view/diff"
	"github.com/eleven-am/pondlive/go/internal/work"
)

func TestDirectSessionHandlerExtraction(t *testing.T) {
	component := func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
		return html.Div(
			html.Button(
				html.On("click", func(evt work.Event) work.Updates { return nil }),
				html.Text("Click Me"),
			),
		)
	}

	root := &runtime.Instance{
		ID:        "root",
		Fn:        component,
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

	patches := diff.ExtractMetadata(sess.View)

	hasSetHandlers := false
	for _, p := range patches {
		if p.Op == "setHandlers" {
			hasSetHandlers = true
			t.Logf("Found setHandlers: path=%v value=%v", p.Path, p.Value)
		}
	}

	if !hasSetHandlers {
		t.Errorf("expected setHandlers in patches, got: %+v", patches)
	}
}

func TestLiveSessionHandlerExtraction(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return html.Div(
			html.Button(
				html.On("click", func(evt work.Event) work.Updates { return nil }),
				html.Text("Click Me"),
			),
		)
	}

	sess := session.NewLiveSession("test-sid", 1, component, nil)

	if err := sess.Flush(); err != nil {
		t.Fatalf("Flush failed: %v", err)
	}

	rtSession := sess.Session()
	if rtSession == nil || rtSession.View == nil {
		t.Fatal("expected View after Flush")
	}

	patches := diff.ExtractMetadata(rtSession.View)

	hasSetHandlers := false
	for _, p := range patches {
		if p.Op == "setHandlers" {
			hasSetHandlers = true
			t.Logf("Found setHandlers: path=%v value=%v", p.Path, p.Value)
		}
	}

	if !hasSetHandlers {
		t.Errorf("expected setHandlers in patches, got: %+v", patches)
	}
}

func TestMinimalProviderFlush(t *testing.T) {

	t.Run("simple element", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				return html.Div(html.Text("Hello"))
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
		t.Log("simple element: OK")
	})

	t.Run("metatags.Provider only", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				return metatags.Provider(ctx, html.Text("Hello"))
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
		t.Log("metatags.Provider only: OK")
	})

	t.Run("nested providers", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				return metatags.Provider(ctx,
					styles.Provider(ctx,
						html.Text("Hello"),
					),
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
		t.Log("nested providers: OK")
	})

	t.Run("provider + render", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				return metatags.Provider(ctx,
					html.Div(
						metatags.Render(ctx),
					),
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
		t.Log("provider + render: OK")
	})

	t.Run("router.ProvideRouter", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				return router.ProvideRouter(ctx,
					html.Text("Hello"),
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
		t.Log("router.ProvideRouter: OK")
	})

	t.Run("all three providers", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				return metatags.Provider(ctx,
					router.ProvideRouter(ctx,
						styles.Provider(ctx,
							html.Text("Hello"),
						),
					),
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
		t.Log("all three providers: OK")
	})

	t.Run("boot-like structure", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, children []work.Node) work.Node {
				headers.UseProvideRequestState(ctx, nil)

				return metatags.Provider(ctx,
					router.ProvideRouter(ctx,
						styles.Provider(ctx,
							html.Html(
								html.Head(
									metatags.Render(ctx),
									styles.Render(ctx),
								),
								html.Body(
									html.Div(html.Text("User App")),
									html.ScriptEl(
										html.Src("/static/pondlive.js"),
									),
								),
							),
						),
					),
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
		t.Log("boot-like structure: OK")
	})
}

func TestBootMetadataExtraction(t *testing.T) {
	t.Run("handlers in boot structure", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				headers.UseProvideRequestState(ctx, nil)

				return metatags.Provider(ctx,
					router.ProvideRouter(ctx,
						styles.Provider(ctx,
							html.Html(
								html.Head(
									metatags.Render(ctx),
									styles.Render(ctx),
								),
								html.Body(
									html.Button(
										html.On("click", func(evt work.Event) work.Updates { return nil }),
										html.Text("Click"),
									),
								),
							),
						),
					),
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

		patches := diff.ExtractMetadata(sess.View)

		hasSetHandlers := false
		for _, p := range patches {
			if p.Op == "setHandlers" {
				hasSetHandlers = true
				t.Logf("Found setHandlers: path=%v value=%v", p.Path, p.Value)
			}
		}

		if !hasSetHandlers {
			t.Errorf("expected setHandlers in boot structure patches, got: %+v", patches)
		}
	})

	t.Run("refs in boot structure", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				headers.UseProvideRequestState(ctx, nil)
				ref := runtime.UseElement(ctx)

				return metatags.Provider(ctx,
					router.ProvideRouter(ctx,
						styles.Provider(ctx,
							html.Html(
								html.Head(
									metatags.Render(ctx),
									styles.Render(ctx),
								),
								html.Body(
									html.Div(
										html.Attach(ref),
										html.Text("With Ref"),
									),
								),
							),
						),
					),
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

		patches := diff.ExtractMetadata(sess.View)

		hasSetRef := false
		for _, p := range patches {
			if p.Op == "setRef" {
				hasSetRef = true
				t.Logf("Found setRef: path=%v value=%v", p.Path, p.Value)
			}
		}

		if !hasSetRef {
			t.Errorf("expected setRef in boot structure patches, got: %+v", patches)
		}
	})

	t.Run("scripts in boot structure", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				headers.UseProvideRequestState(ctx, nil)
				script := runtime.UseScript(ctx, "console.log('hello')")

				scriptDiv := &work.Element{Tag: "div"}
				script.AttachTo(scriptDiv)
				scriptDiv.Children = []work.Node{&work.Text{Value: "With Script"}}

				return metatags.Provider(ctx,
					router.ProvideRouter(ctx,
						styles.Provider(ctx,
							html.Html(
								html.Head(
									metatags.Render(ctx),
									styles.Render(ctx),
								),
								html.Body(scriptDiv),
							),
						),
					),
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

		patches := diff.ExtractMetadata(sess.View)

		hasSetScript := false
		for _, p := range patches {
			if p.Op == "setScript" {
				hasSetScript = true
				t.Logf("Found setScript: path=%v value=%v", p.Path, p.Value)
			}
		}

		if !hasSetScript {
			t.Errorf("expected setScript in boot structure patches, got: %+v", patches)
		}
	})

	t.Run("all metadata types combined", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Node) work.Node {
				headers.UseProvideRequestState(ctx, nil)
				ref := runtime.UseElement(ctx)
				script := runtime.UseScript(ctx, "console.log('combined')")

				btn := &work.Element{Tag: "button"}
				btn.Handlers = map[string]work.Handler{
					"click": {Fn: func(evt work.Event) work.Updates { return nil }},
				}
				btn.RefID = ref.RefID()
				script.AttachTo(btn)
				btn.Children = []work.Node{&work.Text{Value: "All Metadata"}}

				return metatags.Provider(ctx,
					router.ProvideRouter(ctx,
						styles.Provider(ctx,
							html.Html(
								html.Head(
									metatags.Render(ctx),
									styles.Render(ctx),
								),
								html.Body(btn),
							),
						),
					),
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

		patches := diff.ExtractMetadata(sess.View)

		hasHandlers := false
		hasRef := false
		hasScript := false

		for _, p := range patches {
			switch p.Op {
			case "setHandlers":
				hasHandlers = true
				t.Logf("Found setHandlers: path=%v", p.Path)
			case "setRef":
				hasRef = true
				t.Logf("Found setRef: path=%v value=%v", p.Path, p.Value)
			case "setScript":
				hasScript = true
				t.Logf("Found setScript: path=%v", p.Path)
			}
		}

		if !hasHandlers {
			t.Error("expected setHandlers in combined test")
		}
		if !hasRef {
			t.Error("expected setRef in combined test")
		}
		if !hasScript {
			t.Error("expected setScript in combined test")
		}

		t.Logf("All metadata types extracted successfully: handlers=%v, ref=%v, script=%v",
			hasHandlers, hasRef, hasScript)
	})
}
