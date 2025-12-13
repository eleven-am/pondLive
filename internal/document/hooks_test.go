package document

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

func TestUseDocumentEffectRunsOnDocumentChange(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	effectRunCount := 0
	var capturedClass string

	var setDocFn func(*Document)

	root := &runtime.Instance{
		ID:        "root",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			doc, setDoc := runtime.UseState(ctx, &Document{HtmlClass: ""})
			setDocFn = setDoc

			runtime.UseEffect(ctx, func() func() {
				effectRunCount++
				capturedClass = doc.HtmlClass
				return nil
			}, doc)

			return nil
		},
	}
	sess.Root = root

	err := sess.Flush()
	if err != nil {
		t.Fatalf("first flush failed: %v", err)
	}

	if effectRunCount != 1 {
		t.Fatalf("expected effect to run once on initial render, ran %d times", effectRunCount)
	}

	if capturedClass != "" {
		t.Fatalf("expected initial HtmlClass to be empty, got %q", capturedClass)
	}

	effectRunCount = 0
	setDocFn(&Document{HtmlClass: "dark"})

	err = sess.Flush()
	if err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	if effectRunCount != 1 {
		t.Fatalf("expected effect to run once after doc change, ran %d times", effectRunCount)
	}

	if capturedClass != "dark" {
		t.Fatalf("expected HtmlClass to be 'dark', got %q", capturedClass)
	}
}

func TestUseDocumentEffectDoesNotRunWhenDocumentUnchanged(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	effectRunCount := 0
	renderCount := 0
	var forceRerender func()

	root := &runtime.Instance{
		ID:        "root",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			renderCount++
			counter, setCounter := runtime.UseState(ctx, 0)
			forceRerender = func() { setCounter(counter + 1) }

			doc := &Document{HtmlClass: "static"}

			runtime.UseEffect(ctx, func() func() {
				effectRunCount++
				return nil
			}, doc)

			return nil
		},
	}
	sess.Root = root

	err := sess.Flush()
	if err != nil {
		t.Fatalf("first flush failed: %v", err)
	}

	if effectRunCount != 1 {
		t.Fatalf("expected effect to run once on initial render, ran %d times", effectRunCount)
	}

	initialRenderCount := renderCount

	for i := 0; i < 3; i++ {
		forceRerender()
		err = sess.Flush()
		if err != nil {
			t.Fatalf("flush %d failed: %v", i+2, err)
		}
	}

	if renderCount <= initialRenderCount {
		t.Fatalf("expected component to re-render, but render count didn't increase")
	}

	if effectRunCount != 1 {
		t.Fatalf("expected effect to run only once (same doc value each time), ran %d times", effectRunCount)
	}
}

func TestUseDocumentDepsCompareByValue(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	effectRunCount := 0
	var forceRerender func()

	root := &runtime.Instance{
		ID:        "root",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			counter, setCounter := runtime.UseState(ctx, 0)
			forceRerender = func() { setCounter(counter + 1) }

			doc := &Document{HtmlClass: "same-value"}

			runtime.UseEffect(ctx, func() func() {
				effectRunCount++
				return nil
			}, doc)

			return nil
		},
	}
	sess.Root = root

	err := sess.Flush()
	if err != nil {
		t.Fatalf("first flush failed: %v", err)
	}

	if effectRunCount != 1 {
		t.Fatalf("expected effect to run once initially, ran %d times", effectRunCount)
	}

	for i := 0; i < 3; i++ {
		forceRerender()
		err = sess.Flush()
		if err != nil {
			t.Fatalf("flush %d failed: %v", i+2, err)
		}
	}

	if effectRunCount != 1 {
		t.Fatalf("expected effect to run only once (same doc value each time), ran %d times", effectRunCount)
	}
}

func TestUseDocumentFullFlowWithProvider(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	htmlRenderCount := 0
	var lastClass string
	var setDarkFn func(bool)
	effectRuns := 0

	root := &runtime.Instance{
		ID:        "app",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			dark, setDark := runtime.UseState(ctx, false)
			setDarkFn = setDark

			initialState := &documentState{
				entries:    make(map[string]documentEntry),
				setEntries: func(map[string]documentEntry) {},
			}
			state, setState := documentCtx.UseProvider(ctx, initialState)
			state.setEntries = func(newEntries map[string]documentEntry) {
				next := &documentState{
					entries:    newEntries,
					setEntries: state.setEntries,
				}
				setState(next)
			}

			htmlClass := ""
			if dark {
				htmlClass = "dark"
			}
			componentID := ctx.ComponentID()
			depth := ctx.ComponentDepth()

			runtime.UseEffect(ctx, func() func() {
				effectRuns++
				next := make(map[string]documentEntry)
				for k, v := range state.entries {
					next[k] = v
				}
				next[componentID] = documentEntry{
					doc:         &Document{HtmlClass: htmlClass},
					depth:       depth,
					componentID: componentID,
				}
				state.setEntries(next)
				return nil
			}, htmlClass)

			return &work.ComponentNode{
				Key: "html",
				Fn: func(hctx *runtime.Ctx, _ any, children []work.Item) work.Node {
					htmlRenderCount++
					hstate := documentCtx.UseContextValue(hctx)
					attrs := computeHtmlAttrs(hstate)
					if attrs != nil && len(attrs["class"]) > 0 {
						lastClass = attrs["class"][0]
					} else {
						lastClass = ""
					}
					return &work.Element{
						Tag:   "html",
						Attrs: attrs,
					}
				},
			}
		},
	}
	sess.Root = root

	err := sess.Flush()
	if err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	t.Logf("After first flush: htmlRenderCount=%d, lastClass=%q, effectRuns=%d", htmlRenderCount, lastClass, effectRuns)

	if effectRuns != 1 {
		t.Fatalf("expected 1 effect run after initial flush, got %d", effectRuns)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush after initial failed: %v", err)
		}
	}

	t.Logf("After pending flushes: htmlRenderCount=%d, lastClass=%q, effectRuns=%d", htmlRenderCount, lastClass, effectRuns)

	initialHtmlRenderCount := htmlRenderCount

	setDarkFn(true)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("second flush failed: %v", err)
	}

	t.Logf("After setDark flush: htmlRenderCount=%d, lastClass=%q, effectRuns=%d, pending=%v",
		htmlRenderCount, lastClass, effectRuns, sess.IsFlushPending())

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
		t.Logf("After pending flush: htmlRenderCount=%d, lastClass=%q, effectRuns=%d, pending=%v",
			htmlRenderCount, lastClass, effectRuns, sess.IsFlushPending())
	}

	if htmlRenderCount <= initialHtmlRenderCount {
		t.Fatalf("expected html element to re-render after dark toggle, initial=%d, final=%d",
			initialHtmlRenderCount, htmlRenderCount)
	}

	if lastClass != "dark" {
		t.Fatalf("expected class to be 'dark', got %q", lastClass)
	}
}

func TestUseDocumentDeeplyNested4Levels(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	htmlRenderCount := 0
	var lastClass string
	var setDarkFn func(bool)
	level4RenderCount := 0

	root := &runtime.Instance{
		ID:        "root",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			dark, setDark := runtime.UseState(ctx, false)
			setDarkFn = setDark

			initialState := &documentState{
				entries:    make(map[string]documentEntry),
				setEntries: func(map[string]documentEntry) {},
			}
			state, setState := documentCtx.UseProvider(ctx, initialState)
			state.setEntries = func(newEntries map[string]documentEntry) {
				next := &documentState{
					entries:    newEntries,
					setEntries: state.setEntries,
				}
				setState(next)
			}

			return &work.Fragment{
				Children: []work.Node{
					&work.ComponentNode{
						Key: "html-consumer",
						Fn: func(hctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
							htmlRenderCount++
							hstate := documentCtx.UseContextValue(hctx)
							attrs := computeHtmlAttrs(hstate)
							if attrs != nil && len(attrs["class"]) > 0 {
								lastClass = attrs["class"][0]
							} else {
								lastClass = ""
							}
							return &work.Element{
								Tag:   "html",
								Attrs: attrs,
							}
						},
					},
					&work.ComponentNode{
						Key:   "level1",
						Props: dark,
						Fn: func(l1ctx *runtime.Ctx, l1dark any, _ []work.Item) work.Node {
							return &work.ComponentNode{
								Key:   "level2",
								Props: l1dark,
								Fn: func(l2ctx *runtime.Ctx, l2dark any, _ []work.Item) work.Node {
									return &work.ComponentNode{
										Key:   "level3",
										Props: l2dark,
										Fn: func(l3ctx *runtime.Ctx, l3dark any, _ []work.Item) work.Node {
											return &work.ComponentNode{
												Key:   "level4-theme",
												Props: l3dark,
												Fn: func(l4ctx *runtime.Ctx, darkProp any, _ []work.Item) work.Node {
													level4RenderCount++
													isDark := darkProp.(bool)
													l4state := documentCtx.UseContextValue(l4ctx)
													if l4state == nil {
														return nil
													}

													htmlClass := ""
													if isDark {
														htmlClass = "dark"
													}

													componentID := l4ctx.ComponentID()
													depth := l4ctx.ComponentDepth()

													runtime.UseEffect(l4ctx, func() func() {
														next := make(map[string]documentEntry)
														for k, v := range l4state.entries {
															next[k] = v
														}
														next[componentID] = documentEntry{
															doc:         &Document{HtmlClass: htmlClass},
															depth:       depth,
															componentID: componentID,
														}
														l4state.setEntries(next)
														return nil
													}, htmlClass)

													return nil
												},
											}
										},
									}
								},
							}
						},
					},
				},
			}
		},
	}
	sess.Root = root

	err := sess.Flush()
	if err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	initialHtmlRenderCount := htmlRenderCount

	setDarkFn(true)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("flush after setDark failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	if htmlRenderCount <= initialHtmlRenderCount {
		t.Fatalf("expected html to re-render, initial=%d, final=%d", initialHtmlRenderCount, htmlRenderCount)
	}

	if lastClass != "dark" {
		t.Fatalf("expected class 'dark', got %q", lastClass)
	}
}

var darkModeCtx = runtime.CreateContext(false)

func TestUseDocumentWithBootStructure(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	htmlRenderCount := 0
	var lastClass string
	var setDarkFn func(bool)
	themeToggleRenderCount := 0

	root := &runtime.Instance{
		ID:        "boot",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			return Provider(ctx,
				HtmlElement(ctx,
					BodyElement(ctx,
						&work.ComponentNode{
							Key: "theme-toggle",
							Fn: func(tctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
								themeToggleRenderCount++
								dark, setDark := runtime.UseState(tctx, false)
								setDarkFn = setDark

								t.Logf("ThemeToggle render #%d, dark=%v", themeToggleRenderCount, dark)

								htmlClass := ""
								if dark {
									htmlClass = "dark"
								}
								UseDocument(tctx, &Document{HtmlClass: htmlClass})

								return &work.Element{Tag: "button"}
							},
						},
					),
				),
			)
		},
	}
	sess.Root = root

	HtmlElement = runtime.Component(func(ctx *runtime.Ctx, children []work.Item) work.Node {
		htmlRenderCount++
		state := documentCtx.UseContextValue(ctx)
		attrs := computeHtmlAttrs(state)
		if attrs != nil && len(attrs["class"]) > 0 {
			lastClass = attrs["class"][0]
		} else {
			lastClass = ""
		}
		t.Logf("HtmlElement render #%d, class=%q, entries=%d", htmlRenderCount, lastClass, len(state.entries))
		return &work.Element{
			Tag:      "html",
			Attrs:    attrs,
			Children: work.ItemsToNodes(children),
		}
	})

	err := sess.Flush()
	if err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	t.Logf("=== After initial render ===")
	t.Logf("htmlRenderCount=%d, lastClass=%q", htmlRenderCount, lastClass)

	initialHtmlRenderCount := htmlRenderCount

	t.Logf("=== Calling setDarkFn(true) ===")
	setDarkFn(true)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("flush after setDark failed: %v", err)
	}

	for sess.IsFlushPending() {
		t.Logf("Pending flush...")
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	t.Logf("=== After setDark(true) ===")
	t.Logf("htmlRenderCount=%d, lastClass=%q", htmlRenderCount, lastClass)

	if htmlRenderCount <= initialHtmlRenderCount {
		t.Fatalf("expected html to re-render, initial=%d, final=%d", initialHtmlRenderCount, htmlRenderCount)
	}

	if lastClass != "dark" {
		t.Fatalf("expected class 'dark', got %q", lastClass)
	}
}

func TestUseDocumentHookWithDarkModeContext(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	htmlRenderCount := 0
	var lastClass string
	var setDarkFn func(bool)
	level4RenderCount := 0

	root := &runtime.Instance{
		ID:        "root",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			dark, setDark := runtime.UseState(ctx, false)
			setDarkFn = setDark

			_, setDarkMode := darkModeCtx.UseProvider(ctx, dark)
			runtime.UseEffect(ctx, func() func() {
				setDarkMode(dark)
				return nil
			}, dark)

			initialState := &documentState{
				entries:    make(map[string]documentEntry),
				setEntries: func(map[string]documentEntry) {},
			}
			state, setState := documentCtx.UseProvider(ctx, initialState)
			state.setEntries = func(newEntries map[string]documentEntry) {
				next := &documentState{
					entries:    newEntries,
					setEntries: state.setEntries,
				}
				setState(next)
			}

			return &work.Fragment{
				Children: []work.Node{
					&work.ComponentNode{
						Key: "html-consumer",
						Fn: func(hctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
							htmlRenderCount++
							hstate := documentCtx.UseContextValue(hctx)
							attrs := computeHtmlAttrs(hstate)
							if attrs != nil && len(attrs["class"]) > 0 {
								lastClass = attrs["class"][0]
							} else {
								lastClass = ""
							}
							return &work.Element{
								Tag:   "html",
								Attrs: attrs,
							}
						},
					},
					&work.ComponentNode{
						Key: "level1",
						Fn: func(l1ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
							return &work.ComponentNode{
								Key: "level2",
								Fn: func(l2ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
									return &work.ComponentNode{
										Key: "level3",
										Fn: func(l3ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
											return &work.ComponentNode{
												Key: "level4-theme",
												Fn: func(l4ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
													level4RenderCount++

													isDark := darkModeCtx.UseContextValue(l4ctx)
													htmlClass := ""
													if isDark {
														htmlClass = "dark"
													}

													UseDocument(l4ctx, &Document{HtmlClass: htmlClass})
													return nil
												},
											}
										},
									}
								},
							}
						},
					},
				},
			}
		},
	}
	sess.Root = root

	err := sess.Flush()
	if err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	initialHtmlRenderCount := htmlRenderCount
	initialLevel4RenderCount := level4RenderCount

	setDarkFn(true)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("flush after setDark failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	t.Logf("level4RenderCount: initial=%d, final=%d", initialLevel4RenderCount, level4RenderCount)
	t.Logf("htmlRenderCount: initial=%d, final=%d", initialHtmlRenderCount, htmlRenderCount)
	t.Logf("lastClass: %q", lastClass)

	if level4RenderCount <= initialLevel4RenderCount {
		t.Fatalf("expected level4 to re-render via context, initial=%d, final=%d",
			initialLevel4RenderCount, level4RenderCount)
	}

	if htmlRenderCount <= initialHtmlRenderCount {
		t.Fatalf("expected html to re-render, initial=%d, final=%d", initialHtmlRenderCount, htmlRenderCount)
	}

	if lastClass != "dark" {
		t.Fatalf("expected class 'dark', got %q", lastClass)
	}
}

func TestUseDocumentDeeplyNestedWithContext(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	htmlRenderCount := 0
	var lastClass string
	var setDarkFn func(bool)
	level4RenderCount := 0

	root := &runtime.Instance{
		ID:        "root",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			dark, setDark := runtime.UseState(ctx, false)
			setDarkFn = setDark

			_, setDarkMode := darkModeCtx.UseProvider(ctx, dark)
			runtime.UseEffect(ctx, func() func() {
				setDarkMode(dark)
				return nil
			}, dark)

			initialState := &documentState{
				entries:    make(map[string]documentEntry),
				setEntries: func(map[string]documentEntry) {},
			}
			state, setState := documentCtx.UseProvider(ctx, initialState)
			state.setEntries = func(newEntries map[string]documentEntry) {
				next := &documentState{
					entries:    newEntries,
					setEntries: state.setEntries,
				}
				setState(next)
			}

			return &work.Fragment{
				Children: []work.Node{
					&work.ComponentNode{
						Key: "html-consumer",
						Fn: func(hctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
							htmlRenderCount++
							hstate := documentCtx.UseContextValue(hctx)
							attrs := computeHtmlAttrs(hstate)
							if attrs != nil && len(attrs["class"]) > 0 {
								lastClass = attrs["class"][0]
							} else {
								lastClass = ""
							}
							return &work.Element{
								Tag:   "html",
								Attrs: attrs,
							}
						},
					},
					&work.ComponentNode{
						Key: "level1",
						Fn: func(l1ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
							return &work.ComponentNode{
								Key: "level2",
								Fn: func(l2ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
									return &work.ComponentNode{
										Key: "level3",
										Fn: func(l3ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
											return &work.ComponentNode{
												Key: "level4-theme",
												Fn: func(l4ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
													level4RenderCount++

													isDark := darkModeCtx.UseContextValue(l4ctx)
													l4state := documentCtx.UseContextValue(l4ctx)
													if l4state == nil {
														return nil
													}

													htmlClass := ""
													if isDark {
														htmlClass = "dark"
													}

													componentID := l4ctx.ComponentID()
													depth := l4ctx.ComponentDepth()

													runtime.UseEffect(l4ctx, func() func() {
														next := make(map[string]documentEntry)
														for k, v := range l4state.entries {
															next[k] = v
														}
														next[componentID] = documentEntry{
															doc:         &Document{HtmlClass: htmlClass},
															depth:       depth,
															componentID: componentID,
														}
														l4state.setEntries(next)
														return nil
													}, htmlClass)

													return nil
												},
											}
										},
									}
								},
							}
						},
					},
				},
			}
		},
	}
	sess.Root = root

	err := sess.Flush()
	if err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	initialHtmlRenderCount := htmlRenderCount
	initialLevel4RenderCount := level4RenderCount

	setDarkFn(true)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("flush after setDark failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	if level4RenderCount <= initialLevel4RenderCount {
		t.Fatalf("expected level4 to re-render via context, initial=%d, final=%d",
			initialLevel4RenderCount, level4RenderCount)
	}

	if htmlRenderCount <= initialHtmlRenderCount {
		t.Fatalf("expected html to re-render, initial=%d, final=%d", initialHtmlRenderCount, htmlRenderCount)
	}

	if lastClass != "dark" {
		t.Fatalf("expected class 'dark', got %q", lastClass)
	}
}

func TestUseDocumentRealHook(t *testing.T) {
	sess := &runtime.Session{
		Components: make(map[string]*runtime.Instance),
	}

	htmlRenderCount := 0
	var lastClass string
	var setDarkFn func(bool)

	root := &runtime.Instance{
		ID:        "app",
		HookFrame: []runtime.HookSlot{},
		Fn: func(ctx *runtime.Ctx, _ any, _ []work.Item) work.Node {
			dark, setDark := runtime.UseState(ctx, false)
			setDarkFn = setDark

			initialState := &documentState{
				entries:    make(map[string]documentEntry),
				setEntries: func(map[string]documentEntry) {},
			}
			state, setState := documentCtx.UseProvider(ctx, initialState)
			state.setEntries = func(newEntries map[string]documentEntry) {
				next := &documentState{
					entries:    newEntries,
					setEntries: state.setEntries,
				}
				setState(next)
			}

			htmlClass := ""
			if dark {
				htmlClass = "dark"
			}
			UseDocument(ctx, &Document{HtmlClass: htmlClass})

			return &work.ComponentNode{
				Key: "html",
				Fn: func(hctx *runtime.Ctx, _ any, children []work.Item) work.Node {
					htmlRenderCount++
					hstate := documentCtx.UseContextValue(hctx)
					attrs := computeHtmlAttrs(hstate)
					if attrs != nil && len(attrs["class"]) > 0 {
						lastClass = attrs["class"][0]
					} else {
						lastClass = ""
					}
					return &work.Element{
						Tag:   "html",
						Attrs: attrs,
					}
				},
			}
		},
	}
	sess.Root = root
	sess.Root.Render(sess)

	err := sess.Flush()
	if err != nil {
		t.Fatalf("initial flush failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	t.Logf("After initial: htmlRenderCount=%d, lastClass=%q", htmlRenderCount, lastClass)

	initialHtmlRenderCount := htmlRenderCount

	setDarkFn(true)

	err = sess.Flush()
	if err != nil {
		t.Fatalf("flush after setDark failed: %v", err)
	}

	for sess.IsFlushPending() {
		err = sess.Flush()
		if err != nil {
			t.Fatalf("pending flush failed: %v", err)
		}
	}

	t.Logf("After setDark: htmlRenderCount=%d, lastClass=%q", htmlRenderCount, lastClass)

	if htmlRenderCount <= initialHtmlRenderCount {
		t.Fatalf("expected html to re-render, initial=%d, final=%d", initialHtmlRenderCount, htmlRenderCount)
	}

	if lastClass != "dark" {
		t.Fatalf("expected class 'dark', got %q", lastClass)
	}
}
