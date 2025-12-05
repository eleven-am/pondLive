package document

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestDocumentHandleNoopWhenNilState(t *testing.T) {
	handle := &DocumentHandle{}

	result := handle.Html()
	if result != handle {
		t.Error("Expected Html to return same handle")
	}

	result = handle.Body()
	if result != handle {
		t.Error("Expected Body to return same handle")
	}

	result = handle.AddBodyHandler("click", work.Handler{})
	if result != handle {
		t.Error("Expected AddBodyHandler to return same handle")
	}
}

func TestDocumentHandleHtmlSetsClass(t *testing.T) {
	state := &documentState{
		entries: make(map[string]documentEntry),
	}
	state.setEntries = func(e map[string]documentEntry) {
		state.entries = e
	}

	handle := &DocumentHandle{
		componentID: "test",
		depth:       0,
		state:       state,
	}

	classItem := testClassItem{classes: []string{"dark", "theme-blue"}}
	handle.Html(classItem)

	if len(state.entries) != 1 {
		t.Fatalf("Expected 1 entry, got %d", len(state.entries))
	}

	entry := state.entries["test"]
	if entry.doc.HtmlClass != "dark theme-blue" {
		t.Errorf("Expected HtmlClass 'dark theme-blue', got %q", entry.doc.HtmlClass)
	}
}

func TestDocumentHandleHtmlSetsLangAndDir(t *testing.T) {
	state := &documentState{
		entries: make(map[string]documentEntry),
	}
	state.setEntries = func(e map[string]documentEntry) {
		state.entries = e
	}

	handle := &DocumentHandle{
		componentID: "test",
		depth:       0,
		state:       state,
	}

	langItem := testAttrItem{name: "lang", value: "en"}
	dirItem := testAttrItem{name: "dir", value: "ltr"}
	handle.Html(langItem, dirItem)

	entry := state.entries["test"]
	if entry.doc.HtmlLang != "en" {
		t.Errorf("Expected HtmlLang 'en', got %q", entry.doc.HtmlLang)
	}
	if entry.doc.HtmlDir != "ltr" {
		t.Errorf("Expected HtmlDir 'ltr', got %q", entry.doc.HtmlDir)
	}
}

func TestDocumentHandleBodySetsClass(t *testing.T) {
	state := &documentState{
		entries: make(map[string]documentEntry),
	}
	state.setEntries = func(e map[string]documentEntry) {
		state.entries = e
	}

	handle := &DocumentHandle{
		componentID: "test",
		depth:       0,
		state:       state,
	}

	classItem := testClassItem{classes: []string{"bg-white"}}
	handle.Body(classItem)

	entry := state.entries["test"]
	if entry.doc.BodyClass != "bg-white" {
		t.Errorf("Expected BodyClass 'bg-white', got %q", entry.doc.BodyClass)
	}
}

func TestDocumentHandleAddBodyHandler(t *testing.T) {
	state := &documentState{
		entries: make(map[string]documentEntry),
	}
	state.setEntries = func(e map[string]documentEntry) {
		state.entries = e
	}

	handle := &DocumentHandle{
		componentID: "test",
		depth:       0,
		state:       state,
	}

	called := false
	handler := work.Handler{
		Fn: func(evt work.Event) work.Updates {
			called = true
			return nil
		},
	}

	handle.AddBodyHandler("keydown", handler)

	entry := state.entries["test"]
	if len(entry.bodyHandlers["keydown"]) != 1 {
		t.Fatalf("Expected 1 keydown handler, got %d", len(entry.bodyHandlers["keydown"]))
	}

	entry.bodyHandlers["keydown"][0].Fn(work.Event{})
	if !called {
		t.Error("Expected handler to be called")
	}
}

func TestDocumentHandleAddMultipleHandlersSameEvent(t *testing.T) {
	state := &documentState{
		entries: make(map[string]documentEntry),
	}
	state.setEntries = func(e map[string]documentEntry) {
		state.entries = e
	}

	handle := &DocumentHandle{
		componentID: "test",
		depth:       0,
		state:       state,
	}

	count := 0
	handler1 := work.Handler{Fn: func(evt work.Event) work.Updates {
		count++
		return nil
	}}
	handler2 := work.Handler{Fn: func(evt work.Event) work.Updates {
		count++
		return nil
	}}

	handle.AddBodyHandler("click", handler1)
	handle.AddBodyHandler("click", handler2)

	entry := state.entries["test"]
	if len(entry.bodyHandlers["click"]) != 2 {
		t.Errorf("Expected 2 click handlers, got %d", len(entry.bodyHandlers["click"]))
	}
}

func TestDocumentHandleChaining(t *testing.T) {
	state := &documentState{
		entries: make(map[string]documentEntry),
	}
	state.setEntries = func(e map[string]documentEntry) {
		state.entries = e
	}

	handle := &DocumentHandle{
		componentID: "test",
		depth:       0,
		state:       state,
	}

	classItem := testClassItem{classes: []string{"dark"}}
	bodyClassItem := testClassItem{classes: []string{"bg-white"}}
	handler := work.Handler{Fn: func(evt work.Event) work.Updates { return nil }}

	result := handle.Html(classItem).Body(bodyClassItem).AddBodyHandler("click", handler)

	if result != handle {
		t.Error("Expected chaining to return same handle")
	}

	entry := state.entries["test"]
	if entry.doc.HtmlClass != "dark" {
		t.Errorf("Expected HtmlClass 'dark', got %q", entry.doc.HtmlClass)
	}
	if entry.doc.BodyClass != "bg-white" {
		t.Errorf("Expected BodyClass 'bg-white', got %q", entry.doc.BodyClass)
	}
	if len(entry.bodyHandlers["click"]) != 1 {
		t.Errorf("Expected 1 click handler, got %d", len(entry.bodyHandlers["click"]))
	}
}

func TestDocumentHandleSkipsNilHandlerFn(t *testing.T) {
	state := &documentState{
		entries: make(map[string]documentEntry),
	}
	state.setEntries = func(e map[string]documentEntry) {
		state.entries = e
	}

	handle := &DocumentHandle{
		componentID: "test",
		depth:       0,
		state:       state,
	}

	handle.AddBodyHandler("click", work.Handler{Fn: nil})

	if len(state.entries) != 0 {
		t.Error("Expected no entry for nil handler")
	}
}

type testClassItem struct {
	classes []string
}

func (t testClassItem) ApplyTo(el *work.Element) {
	if el.Attrs == nil {
		el.Attrs = make(map[string][]string)
	}
	el.Attrs["class"] = append(el.Attrs["class"], t.classes...)
}

type testAttrItem struct {
	name  string
	value string
}

func (t testAttrItem) ApplyTo(el *work.Element) {
	if el.Attrs == nil {
		el.Attrs = make(map[string][]string)
	}
	el.Attrs[t.name] = []string{t.value}
}
