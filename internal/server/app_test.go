package server

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/headers"
	"github.com/eleven-am/pondlive/internal/metatags"
	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/router"
	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/session"
	"github.com/eleven-am/pondlive/internal/styles"
	"github.com/eleven-am/pondlive/internal/upload"
	"github.com/eleven-am/pondlive/internal/view/diff"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestDirectSessionHandlerExtraction(t *testing.T) {
	component := func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
		return &work.Element{
			Tag: "div",
			Children: []work.Node{
				&work.Element{
					Tag: "button",
					Handlers: map[string]work.Handler{
						"click": {Fn: func(evt work.Event) work.Updates { return nil }},
					},
					Children: []work.Node{&work.Text{Value: "Click Me"}},
				},
			},
		}
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
		return &work.Element{
			Tag: "div",
			Children: []work.Node{
				&work.Element{
					Tag: "button",
					Handlers: map[string]work.Handler{
						"click": {Fn: func(evt work.Event) work.Updates { return nil }},
					},
					Children: []work.Node{&work.Text{Value: "Click Me"}},
				},
			},
		}
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return &work.Element{
					Tag:      "div",
					Children: []work.Node{&work.Text{Value: "Hello"}},
				}
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return metatags.Provider(ctx, &work.Text{Value: "Hello"})
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return metatags.Provider(ctx,
					styles.Provider(ctx,
						&work.Text{Value: "Hello"},
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return metatags.Provider(ctx,
					&work.Element{
						Tag:      "div",
						Children: []work.Node{metatags.Render(ctx)},
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
		t.Log("provider + render: OK")
	})

	t.Run("router.Provide", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return router.Provide(ctx,
					&work.Text{Value: "Hello"},
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
		t.Log("router.Provide: OK")
	})

	t.Run("all three providers", func(t *testing.T) {
		root := &runtime.Instance{
			ID: "root",
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				return metatags.Provider(ctx,
					router.Provide(ctx,
						styles.Provider(ctx,
							&work.Text{Value: "Hello"},
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
			Fn: func(ctx *runtime.Ctx, _ any, children []work.Item) work.Node {
				return headers.Provider(ctx, nil,
					metatags.Provider(ctx,
						router.Provide(ctx,
							styles.Provider(ctx,
								&work.Element{
									Tag: "html",
									Children: []work.Node{
										&work.Element{
											Tag: "head",
											Children: []work.Node{
												metatags.Render(ctx),
												styles.Render(ctx),
												headers.Render(ctx),
											},
										},
										&work.Element{
											Tag: "body",
											Children: []work.Node{
												&work.Element{
													Tag:      "div",
													Children: []work.Node{&work.Text{Value: "User App"}},
												},
												&work.Element{
													Tag: "script",
													Attrs: map[string][]string{
														"src": {"/static/pondlive.js"},
													},
												},
											},
										},
									},
								},
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				_ = headers.UseRequestState(ctx)

				return metatags.Provider(ctx,
					router.Provide(ctx,
						styles.Provider(ctx,
							&work.Element{
								Tag: "html",
								Children: []work.Node{
									&work.Element{
										Tag: "head",
										Children: []work.Node{
											metatags.Render(ctx),
											styles.Render(ctx),
										},
									},
									&work.Element{
										Tag: "body",
										Children: []work.Node{
											&work.Element{
												Tag: "button",
												Handlers: map[string]work.Handler{
													"click": {Fn: func(evt work.Event) work.Updates { return nil }},
												},
												Children: []work.Node{&work.Text{Value: "Click"}},
											},
										},
									},
								},
							},
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				_ = headers.UseRequestState(ctx)
				ref := runtime.UseElement(ctx)

				return metatags.Provider(ctx,
					router.Provide(ctx,
						styles.Provider(ctx,
							&work.Element{
								Tag: "html",
								Children: []work.Node{
									&work.Element{
										Tag: "head",
										Children: []work.Node{
											metatags.Render(ctx),
											styles.Render(ctx),
										},
									},
									&work.Element{
										Tag: "body",
										Children: []work.Node{
											&work.Element{
												Tag:      "div",
												RefID:    ref.RefID(),
												Children: []work.Node{&work.Text{Value: "With Ref"}},
											},
										},
									},
								},
							},
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				_ = headers.UseRequestState(ctx)
				script := runtime.UseScript(ctx, "console.log('hello')")

				scriptDiv := &work.Element{Tag: "div"}
				script.AttachTo(scriptDiv)
				scriptDiv.Children = []work.Node{&work.Text{Value: "With Script"}}

				return metatags.Provider(ctx,
					router.Provide(ctx,
						styles.Provider(ctx,
							&work.Element{
								Tag: "html",
								Children: []work.Node{
									&work.Element{
										Tag: "head",
										Children: []work.Node{
											metatags.Render(ctx),
											styles.Render(ctx),
										},
									},
									&work.Element{
										Tag:      "body",
										Children: []work.Node{scriptDiv},
									},
								},
							},
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
			Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
				_ = headers.UseRequestState(ctx)
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
					router.Provide(ctx,
						styles.Provider(ctx,
							&work.Element{
								Tag: "html",
								Children: []work.Node{
									&work.Element{
										Tag: "head",
										Children: []work.Node{
											metatags.Render(ctx),
											styles.Render(ctx),
										},
									},
									&work.Element{
										Tag:      "body",
										Children: []work.Node{btn},
									},
								},
							},
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

func TestLiveSessionUploadRegistry(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	t.Run("session has upload registry", func(t *testing.T) {
		sess := session.NewLiveSession("test-sid", 1, component, nil)
		defer sess.Close()

		reg := sess.UploadRegistry()
		if reg == nil {
			t.Fatal("expected upload registry to be non-nil")
		}
	})

	t.Run("registry is functional", func(t *testing.T) {
		sess := session.NewLiveSession("test-sid", 1, component, nil)
		defer sess.Close()

		reg := sess.UploadRegistry()
		if reg == nil {
			t.Fatal("expected upload registry")
		}

		cb := upload.UploadCallback{
			Token:   "test-token",
			MaxSize: 1024,
		}
		reg.Register(cb)

		found, ok := reg.Lookup("test-token")
		if !ok {
			t.Fatal("expected to find registered callback")
		}
		if found.Token != "test-token" {
			t.Errorf("expected token test-token, got %s", found.Token)
		}
	})

	t.Run("nil session returns nil registry", func(t *testing.T) {
		var sess *session.LiveSession
		reg := sess.UploadRegistry()
		if reg != nil {
			t.Error("expected nil registry for nil session")
		}
	})
}

func TestAppLookupUploadCallback(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	t.Run("lookup finds callback across sessions", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		sess1 := session.NewLiveSession("sess-1", 1, component, nil)
		sess2 := session.NewLiveSession("sess-2", 1, component, nil)
		defer sess1.Close()
		defer sess2.Close()

		app.registry.Put(sess1)
		app.registry.Put(sess2)

		cb := upload.UploadCallback{
			Token:   "target-token",
			MaxSize: 2048,
		}
		sess2.UploadRegistry().Register(cb)

		found, ok := app.lookupUploadCallback("target-token")
		if !ok {
			t.Fatal("expected to find callback")
		}
		if found.Token != "target-token" {
			t.Errorf("expected token target-token, got %s", found.Token)
		}
		if found.MaxSize != 2048 {
			t.Errorf("expected MaxSize 2048, got %d", found.MaxSize)
		}
	})

	t.Run("lookup returns false for nonexistent token", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		sess := session.NewLiveSession("sess-1", 1, component, nil)
		defer sess.Close()
		app.registry.Put(sess)

		_, ok := app.lookupUploadCallback("nonexistent")
		if ok {
			t.Error("expected false for nonexistent token")
		}
	})

	t.Run("lookup works with no sessions", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		_, ok := app.lookupUploadCallback("any-token")
		if ok {
			t.Error("expected false with no sessions")
		}
	})
}

func TestEscapeJSON(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`{"key": "value"}`, `{"key": "value"}`},
		{`<script></script>`, `<script><\/script>`},
		{`</body>`, `<\/body>`},
		{`test</test`, `test<\/test`},
		{`no slash tags`, `no slash tags`},
		{`multiple </a></b></c>`, `multiple <\/a><\/b><\/c>`},
	}

	for _, tc := range tests {
		result := escapeJSON(tc.input)
		if result != tc.expected {
			t.Errorf("escapeJSON(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestLastIndexFold(t *testing.T) {
	tests := []struct {
		haystack string
		needle   string
		expected int
	}{
		{"hello</body>world", "</body>", 5},
		{"hello</BODY>world", "</body>", 5},
		{"hello</Body>world", "</body>", 5},
		{"<body></body></BODY>", "</body>", 13},
		{"no match here", "</body>", -1},
		{"short", "</body>", -1},
		{"", "</body>", -1},
		{"test", "", -1},
		{"</body>", "</body>", 0},
	}

	for _, tc := range tests {
		result := lastIndexFold(tc.haystack, tc.needle)
		if result != tc.expected {
			t.Errorf("lastIndexFold(%q, %q) = %d, want %d", tc.haystack, tc.needle, result, tc.expected)
		}
	}
}

func TestDecorateDocument(t *testing.T) {
	t.Run("injects before closing body", func(t *testing.T) {
		doc := "<html><body><div>content</div></body></html>"
		boot := []byte(`{"state":"test"}`)
		result := decorateDocument(doc, boot)

		if !strings.Contains(result, `<script id="live-boot"`) {
			t.Error("expected boot script in result")
		}
		if !strings.Contains(result, `{"state":"test"}`) {
			t.Error("expected boot JSON in result")
		}
		bodyIdx := strings.Index(result, "</body>")
		scriptIdx := strings.Index(result, `<script id="live-boot"`)
		if scriptIdx > bodyIdx {
			t.Error("expected script before </body>")
		}
	})

	t.Run("case insensitive body tag", func(t *testing.T) {
		doc := "<html><BODY>content</BODY></html>"
		boot := []byte(`{"test":true}`)
		result := decorateDocument(doc, boot)

		if !strings.Contains(result, `<script id="live-boot"`) {
			t.Error("expected boot script in result")
		}
	})

	t.Run("appends if no body tag", func(t *testing.T) {
		doc := "<html><div>content</div></html>"
		boot := []byte(`{"test":true}`)
		result := decorateDocument(doc, boot)

		if !strings.HasSuffix(result, "</script>") {
			t.Error("expected script appended at end")
		}
	})

	t.Run("escapes JSON closing tags", func(t *testing.T) {
		doc := "<html><body></body></html>"
		boot := []byte(`{"html":"</script>"}`)
		result := decorateDocument(doc, boot)

		if strings.Contains(result, `</script>"}`) {
			t.Error("expected </script> to be escaped in JSON")
		}
	})
}

func TestDefaultSessionID(t *testing.T) {
	t.Run("generates valid session ID", func(t *testing.T) {
		id, err := defaultSessionID(nil)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if id == "" {
			t.Error("expected non-empty session ID")
		}
		if len(id) < 20 {
			t.Errorf("expected longer session ID, got %d chars", len(id))
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		ids := make(map[session.SessionID]bool)
		for i := 0; i < 100; i++ {
			id, err := defaultSessionID(nil)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ids[id] {
				t.Errorf("duplicate session ID: %s", id)
			}
			ids[id] = true
		}
	})
}

func TestCloneSessionConfig(t *testing.T) {
	t.Run("clones non-nil config", func(t *testing.T) {
		original := &session.Config{
			ClientAsset: "/test.js",
		}
		clone := cloneSessionConfig(original)

		if clone.ClientAsset != original.ClientAsset {
			t.Error("expected same ClientAsset")
		}
	})

	t.Run("returns empty config for nil", func(t *testing.T) {
		clone := cloneSessionConfig(nil)
		if clone.ClientAsset != "" {
			t.Errorf("expected empty config for nil input, got ClientAsset=%q", clone.ClientAsset)
		}
	})
}

func TestAppMux(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	mux := app.Mux()
	if mux == nil {
		t.Error("expected non-nil mux")
	}
}

func TestAppHandler(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handler := app.Handler()
	if handler == nil {
		t.Error("expected non-nil handler")
	}
}

func TestAppHandlerFunc(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	handlerFunc := app.HandlerFunc()
	if handlerFunc == nil {
		t.Error("expected non-nil handler func")
	}
}

func TestAppServer(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	server := app.Server(":0")
	if server == nil {
		t.Error("expected non-nil server")
	}
	if server.Addr != ":0" {
		t.Errorf("expected addr :0, got %s", server.Addr)
	}
}

func TestAppRegistry(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	registry := app.Registry()
	if registry == nil {
		t.Error("expected non-nil registry")
	}
}

func TestAppRemoveUploadCallback(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	t.Run("remove callback from correct session", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		sess1 := session.NewLiveSession("sess-1", 1, component, nil)
		sess2 := session.NewLiveSession("sess-2", 1, component, nil)
		defer sess1.Close()
		defer sess2.Close()

		app.registry.Put(sess1)
		app.registry.Put(sess2)

		cb1 := upload.UploadCallback{Token: "token-1"}
		cb2 := upload.UploadCallback{Token: "token-2"}
		sess1.UploadRegistry().Register(cb1)
		sess2.UploadRegistry().Register(cb2)

		app.removeUploadCallback("token-2")

		_, ok := sess1.UploadRegistry().Lookup("token-1")
		if !ok {
			t.Error("expected token-1 to still exist in sess1")
		}

		_, ok = sess2.UploadRegistry().Lookup("token-2")
		if ok {
			t.Error("expected token-2 to be removed from sess2")
		}
	})

	t.Run("remove nonexistent token does not panic", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		sess := session.NewLiveSession("sess-1", 1, component, nil)
		defer sess.Close()
		app.registry.Put(sess)

		app.removeUploadCallback("nonexistent")
	})

	t.Run("remove with no sessions does not panic", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.removeUploadCallback("any-token")
	})
}

func TestCloneHeader(t *testing.T) {
	t.Run("clones headers", func(t *testing.T) {
		original := http.Header{
			"Content-Type":  {"application/json"},
			"Authorization": {"Bearer token"},
		}

		clone := cloneHeader(original)

		if clone.Get("Content-Type") != "application/json" {
			t.Error("expected Content-Type to be cloned")
		}
		if clone.Get("Authorization") != "Bearer token" {
			t.Error("expected Authorization to be cloned")
		}
	})

	t.Run("clone is independent", func(t *testing.T) {
		original := http.Header{
			"X-Test": {"value1"},
		}

		clone := cloneHeader(original)
		clone.Set("X-Test", "modified")

		if original.Get("X-Test") != "value1" {
			t.Error("original should not be affected by clone modification")
		}
	})

	t.Run("nil returns empty header", func(t *testing.T) {
		clone := cloneHeader(nil)
		if clone == nil {
			t.Error("expected non-nil header for nil input")
		}
		if len(clone) != 0 {
			t.Error("expected empty header")
		}
	})

	t.Run("multi-value headers", func(t *testing.T) {
		original := http.Header{
			"Accept": {"text/html", "application/json", "text/plain"},
		}

		clone := cloneHeader(original)

		values := clone["Accept"]
		if len(values) != 3 {
			t.Errorf("expected 3 Accept values, got %d", len(values))
		}

		original["Accept"][0] = "modified"
		if clone["Accept"][0] == "modified" {
			t.Error("clone values should not be affected by original modification")
		}
	})
}

func TestRegisterEndpointEdgeCases(t *testing.T) {
	t.Run("nil server returns error", func(t *testing.T) {
		_, err := Register(nil, "/live", NewSessionRegistry())
		if err == nil {
			t.Error("expected error for nil server")
		}
	})

	t.Run("nil registry returns error", func(t *testing.T) {
		component := func(ctx *runtime.Ctx) work.Node {
			return &work.Element{Tag: "div"}
		}

		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		_, err = Register(app.pondManager, "/test", nil)
		if err == nil {
			t.Error("expected error for nil registry")
		}
	})
}

func TestPondChannelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"my-channel", "pondlive:my-channel"},
		{"", "pondlive:"},
		{"test:channel", "pondlive:test:channel"},
	}

	for _, tc := range tests {
		result := PondChannelName(tc.input)
		if result != tc.expected {
			t.Errorf("PondChannelName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestExtractAppChannelName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"pondlive:my-channel", "my-channel"},
		{"pondlive:", ""},
		{"pondlive:test:channel", "test:channel"},
		{"other:channel", ""},
		{"", ""},
		{"my-channel", ""},
	}

	for _, tc := range tests {
		result := extractAppChannelName(tc.input)
		if result != tc.expected {
			t.Errorf("extractAppChannelName(%q) = %q, want %q", tc.input, result, tc.expected)
		}
	}
}

func TestNewAppConfigurations(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	t.Run("nil component returns error", func(t *testing.T) {
		_, err := New(Config{Component: nil})
		if err == nil {
			t.Error("expected error for nil component")
		}
	})

	t.Run("custom client asset", func(t *testing.T) {
		app, err := New(Config{
			Component:   component,
			ClientAsset: "/custom/path.js",
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.clientAsset != "/custom/path.js" {
			t.Errorf("expected custom client asset, got %s", app.clientAsset)
		}
	})

	t.Run("empty client asset uses default", func(t *testing.T) {
		app, err := New(Config{
			Component:   component,
			ClientAsset: "   ",
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.clientAsset != "/static/pondlive.js" {
			t.Errorf("expected default client asset, got %s", app.clientAsset)
		}
	})

	t.Run("dev mode changes client asset", func(t *testing.T) {
		app, err := New(Config{
			Component:     component,
			SessionConfig: &session.Config{DevMode: true},
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.clientAsset != "/static/pondlive-dev.js" {
			t.Errorf("expected dev client asset, got %s", app.clientAsset)
		}
	})

	t.Run("custom ID generator", func(t *testing.T) {
		customGenerator := func(*http.Request) (session.SessionID, error) {
			return "custom-id", nil
		}

		app, err := New(Config{
			Component:   component,
			IDGenerator: customGenerator,
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		id, _ := app.idGenerator(nil)
		if id != "custom-id" {
			t.Errorf("expected custom-id, got %s", id)
		}
	})

	t.Run("with upload config", func(t *testing.T) {
		app, err := New(Config{
			Component: component,
			UploadConfig: &upload.Config{
				StoragePath: t.TempDir(),
			},
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.uploadHandler == nil {
			t.Error("expected upload handler to be set")
		}
	})

	t.Run("version defaults to 1", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}
		if app.version != 1 {
			t.Errorf("expected version 1, got %d", app.version)
		}
	})
}

func TestServeSSR(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{
			Tag: "html",
			Children: []work.Node{
				&work.Element{Tag: "head"},
				&work.Element{
					Tag: "body",
					Children: []work.Node{
						&work.Element{
							Tag:      "div",
							Children: []work.Node{&work.Text{Value: "Hello, World!"}},
						},
					},
				},
			},
		}
	}

	t.Run("basic SSR render", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		resp := rec.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		body := rec.Body.String()
		if !strings.Contains(body, "Hello, World!") {
			t.Error("expected body to contain 'Hello, World!'")
		}
		if !strings.Contains(body, `<script id="live-boot"`) {
			t.Error("expected body to contain boot script")
		}
		if !strings.Contains(body, "<!DOCTYPE html>") {
			t.Error("expected DOCTYPE in response")
		}

		contentType := resp.Header.Get("Content-Type")
		if !strings.Contains(contentType, "text/html") {
			t.Errorf("expected text/html content type, got %s", contentType)
		}

		cacheControl := resp.Header.Get("Cache-Control")
		if cacheControl != "no-store" {
			t.Errorf("expected no-store cache control, got %s", cacheControl)
		}
	})

	t.Run("SSR with path", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/test/path?query=value", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		resp := rec.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}
	})

	t.Run("session ID error", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.idGenerator = func(*http.Request) (session.SessionID, error) {
			return "", nil
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("dev mode config", func(t *testing.T) {
		app, err := New(Config{
			Component:     component,
			SessionConfig: &session.Config{DevMode: true},
		})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		resp := rec.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected 200, got %d", resp.StatusCode)
		}

		body := rec.Body.String()
		if !strings.Contains(body, `"debug":true`) {
			t.Error("expected debug flag in boot JSON for dev mode")
		}
	})

	t.Run("with query params and fragment", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/page?foo=bar&baz=qux#section", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("id generator returns error", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.idGenerator = func(*http.Request) (session.SessionID, error) {
			return "", errors.New("generator error")
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusInternalServerError {
			t.Errorf("expected 500, got %d", rec.Code)
		}
	})

	t.Run("with redirect in request state", func(t *testing.T) {
		redirectComponent := func(ctx *runtime.Ctx) work.Node {
			requestState := headers.UseRequestState(ctx)
			if requestState != nil {
				requestState.SetRedirect("/redirected", http.StatusFound)
			}
			return &work.Element{Tag: "div"}
		}

		app, err := New(Config{Component: redirectComponent})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusFound {
			t.Errorf("expected 302, got %d", rec.Code)
		}

		location := rec.Header().Get("Location")
		if location != "/redirected" {
			t.Errorf("expected redirect to /redirected, got %s", location)
		}
	})

	t.Run("version defaults when zero", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.version = 0

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})

	t.Run("nil returning component", func(t *testing.T) {
		nilComponent := func(ctx *runtime.Ctx) work.Node {
			return nil
		}

		app, err := New(Config{Component: nilComponent})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code == http.StatusOK {
			t.Log("nil component rendered successfully (boot wrapper provides structure)")
		}
	})

	t.Run("negative version defaults to 1", func(t *testing.T) {
		app, err := New(Config{Component: component})
		if err != nil {
			t.Fatalf("failed to create app: %v", err)
		}

		app.version = -5

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		app.serveSSR(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestBroadcastReturnsError(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	err = app.Broadcast("test-channel", "test-event", map[string]string{"key": "value"})
	if err == nil {
		t.Error("expected error when broadcasting to nonexistent channel")
	}
}

func TestNewPubSubLobby(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if app.endpoint == nil {
		t.Fatal("expected endpoint to be created")
	}

	if app.endpoint.pubsubLobby == nil {
		t.Fatal("expected pubsub lobby to be created")
	}
}

func TestEndpointConfigure(t *testing.T) {
	component := func(ctx *runtime.Ctx) work.Node {
		return &work.Element{Tag: "div"}
	}

	app, err := New(Config{Component: component})
	if err != nil {
		t.Fatalf("failed to create app: %v", err)
	}

	if app.endpoint.endpoint == nil {
		t.Fatal("expected pondsocket endpoint to be configured")
	}

	if app.endpoint.registry == nil {
		t.Fatal("expected registry to be set")
	}
}
