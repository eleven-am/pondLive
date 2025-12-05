package document

import (
	"strings"
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestGetMergedDocumentEmpty(t *testing.T) {
	entries := make(map[string]documentEntry)
	doc := getMergedDocument(entries)

	if doc.HtmlClass != "" {
		t.Errorf("Expected empty HtmlClass, got %q", doc.HtmlClass)
	}
	if doc.HtmlLang != "" {
		t.Errorf("Expected empty HtmlLang, got %q", doc.HtmlLang)
	}
	if doc.BodyClass != "" {
		t.Errorf("Expected empty BodyClass, got %q", doc.BodyClass)
	}
}

func TestGetMergedDocumentSingleEntry(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["comp1"] = documentEntry{
		componentID: "comp1",
		depth:       0,
		doc: &Document{
			HtmlClass: "dark",
			HtmlLang:  "en",
			BodyClass: "bg-gray-900",
		},
	}

	doc := getMergedDocument(entries)

	if doc.HtmlClass != "dark" {
		t.Errorf("Expected HtmlClass 'dark', got %q", doc.HtmlClass)
	}
	if doc.HtmlLang != "en" {
		t.Errorf("Expected HtmlLang 'en', got %q", doc.HtmlLang)
	}
	if doc.BodyClass != "bg-gray-900" {
		t.Errorf("Expected BodyClass 'bg-gray-900', got %q", doc.BodyClass)
	}
}

func TestGetMergedDocumentLangDeepestWins(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["layout"] = documentEntry{
		componentID: "layout",
		depth:       0,
		doc: &Document{
			HtmlLang: "en",
		},
	}
	entries["page"] = documentEntry{
		componentID: "page",
		depth:       1,
		doc: &Document{
			HtmlLang: "fr",
		},
	}

	doc := getMergedDocument(entries)

	if doc.HtmlLang != "fr" {
		t.Errorf("Expected deepest lang 'fr', got %q", doc.HtmlLang)
	}
}

func TestGetMergedDocumentDirDeepestWins(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["layout"] = documentEntry{
		componentID: "layout",
		depth:       0,
		doc: &Document{
			HtmlDir: "ltr",
		},
	}
	entries["page"] = documentEntry{
		componentID: "page",
		depth:       1,
		doc: &Document{
			HtmlDir: "rtl",
		},
	}

	doc := getMergedDocument(entries)

	if doc.HtmlDir != "rtl" {
		t.Errorf("Expected deepest dir 'rtl', got %q", doc.HtmlDir)
	}
}

func TestGetMergedDocumentClassesAdditive(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["theme"] = documentEntry{
		componentID: "theme",
		depth:       0,
		doc: &Document{
			HtmlClass: "dark",
		},
	}
	entries["layout"] = documentEntry{
		componentID: "layout",
		depth:       1,
		doc: &Document{
			HtmlClass: "theme-blue",
		},
	}

	doc := getMergedDocument(entries)

	if !strings.Contains(doc.HtmlClass, "dark") {
		t.Errorf("Expected HtmlClass to contain 'dark', got %q", doc.HtmlClass)
	}
	if !strings.Contains(doc.HtmlClass, "theme-blue") {
		t.Errorf("Expected HtmlClass to contain 'theme-blue', got %q", doc.HtmlClass)
	}
}

func TestGetMergedDocumentBodyClassesAdditive(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["theme"] = documentEntry{
		componentID: "theme",
		depth:       0,
		doc: &Document{
			BodyClass: "bg-gray-900",
		},
	}
	entries["modal"] = documentEntry{
		componentID: "modal",
		depth:       1,
		doc: &Document{
			BodyClass: "overflow-hidden",
		},
	}

	doc := getMergedDocument(entries)

	if !strings.Contains(doc.BodyClass, "bg-gray-900") {
		t.Errorf("Expected BodyClass to contain 'bg-gray-900', got %q", doc.BodyClass)
	}
	if !strings.Contains(doc.BodyClass, "overflow-hidden") {
		t.Errorf("Expected BodyClass to contain 'overflow-hidden', got %q", doc.BodyClass)
	}
}

func TestGetMergedDocumentClassesDeduplicated(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["comp1"] = documentEntry{
		componentID: "comp1",
		depth:       0,
		doc: &Document{
			HtmlClass: "dark theme-blue",
		},
	}
	entries["comp2"] = documentEntry{
		componentID: "comp2",
		depth:       1,
		doc: &Document{
			HtmlClass: "dark theme-green",
		},
	}

	doc := getMergedDocument(entries)

	count := strings.Count(doc.HtmlClass, "dark")
	if count != 1 {
		t.Errorf("Expected 'dark' to appear once, appeared %d times in %q", count, doc.HtmlClass)
	}
}

func TestGetMergedDocumentNilDoc(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["comp1"] = documentEntry{
		componentID: "comp1",
		depth:       0,
		doc:         nil,
	}
	entries["comp2"] = documentEntry{
		componentID: "comp2",
		depth:       1,
		doc: &Document{
			HtmlClass: "dark",
		},
	}

	doc := getMergedDocument(entries)

	if doc.HtmlClass != "dark" {
		t.Errorf("Expected HtmlClass 'dark', got %q", doc.HtmlClass)
	}
}

func TestGetMergedDocumentRemove(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["parent"] = documentEntry{
		componentID: "parent",
		depth:       0,
		doc: &Document{
			HtmlClass: "light",
		},
	}
	entries["child"] = documentEntry{
		componentID: "child",
		depth:       1,
		doc: &Document{
			HtmlClass: "dark",
		},
	}

	doc := getMergedDocument(entries)
	if !strings.Contains(doc.HtmlClass, "dark") {
		t.Errorf("Before remove: expected 'dark' in HtmlClass, got %q", doc.HtmlClass)
	}

	delete(entries, "child")

	doc = getMergedDocument(entries)
	if doc.HtmlClass != "light" {
		t.Errorf("After remove: expected 'light', got %q", doc.HtmlClass)
	}
}

func TestGetMergedDocumentMultipleClasses(t *testing.T) {
	entries := make(map[string]documentEntry)
	entries["comp1"] = documentEntry{
		componentID: "comp1",
		depth:       0,
		doc: &Document{
			HtmlClass: "dark mode-compact",
			BodyClass: "bg-gray-900 text-white",
		},
	}

	doc := getMergedDocument(entries)

	htmlClasses := strings.Fields(doc.HtmlClass)
	if len(htmlClasses) != 2 {
		t.Errorf("Expected 2 HTML classes, got %d", len(htmlClasses))
	}

	bodyClasses := strings.Fields(doc.BodyClass)
	if len(bodyClasses) != 2 {
		t.Errorf("Expected 2 body classes, got %d", len(bodyClasses))
	}
}

func TestComputeBodyHandlersEmpty(t *testing.T) {
	handlers := computeBodyHandlers(nil)
	if handlers != nil {
		t.Error("Expected nil handlers for nil state")
	}

	state := &documentState{entries: make(map[string]documentEntry)}
	handlers = computeBodyHandlers(state)
	if handlers != nil {
		t.Error("Expected nil handlers for empty entries")
	}
}

func TestComputeBodyHandlersSingleEntry(t *testing.T) {
	callCount := 0
	state := &documentState{
		entries: map[string]documentEntry{
			"comp1": {
				componentID: "comp1",
				depth:       0,
				bodyHandlers: map[string][]work.Handler{
					"click": {{Fn: func(evt work.Event) work.Updates {
						callCount++
						return nil
					}}},
				},
			},
		},
	}

	handlers := computeBodyHandlers(state)
	if handlers == nil {
		t.Fatal("Expected non-nil handlers")
	}

	if _, ok := handlers["click"]; !ok {
		t.Error("Expected click handler")
	}

	handlers["click"].Fn(work.Event{})
	if callCount != 1 {
		t.Errorf("Expected handler to be called once, got %d", callCount)
	}
}

func TestComputeBodyHandlersMultipleEntriesSameEvent(t *testing.T) {
	callCount := 0
	state := &documentState{
		entries: map[string]documentEntry{
			"comp1": {
				componentID: "comp1",
				depth:       0,
				bodyHandlers: map[string][]work.Handler{
					"keydown": {{Fn: func(evt work.Event) work.Updates {
						callCount++
						return nil
					}}},
				},
			},
			"comp2": {
				componentID: "comp2",
				depth:       1,
				bodyHandlers: map[string][]work.Handler{
					"keydown": {{Fn: func(evt work.Event) work.Updates {
						callCount++
						return nil
					}}},
				},
			},
		},
	}

	handlers := computeBodyHandlers(state)
	if handlers == nil {
		t.Fatal("Expected non-nil handlers")
	}

	handlers["keydown"].Fn(work.Event{})
	if callCount != 2 {
		t.Errorf("Expected both handlers called, got %d calls", callCount)
	}
}

func TestComputeBodyHandlersDifferentEvents(t *testing.T) {
	clickCalled := false
	keydownCalled := false

	state := &documentState{
		entries: map[string]documentEntry{
			"comp1": {
				componentID: "comp1",
				depth:       0,
				bodyHandlers: map[string][]work.Handler{
					"click": {{Fn: func(evt work.Event) work.Updates {
						clickCalled = true
						return nil
					}}},
				},
			},
			"comp2": {
				componentID: "comp2",
				depth:       1,
				bodyHandlers: map[string][]work.Handler{
					"keydown": {{Fn: func(evt work.Event) work.Updates {
						keydownCalled = true
						return nil
					}}},
				},
			},
		},
	}

	handlers := computeBodyHandlers(state)
	if len(handlers) != 2 {
		t.Errorf("Expected 2 event types, got %d", len(handlers))
	}

	handlers["click"].Fn(work.Event{})
	handlers["keydown"].Fn(work.Event{})

	if !clickCalled {
		t.Error("Expected click handler to be called")
	}
	if !keydownCalled {
		t.Error("Expected keydown handler to be called")
	}
}

func TestMergeHandlersEmpty(t *testing.T) {
	merged := mergeHandlers(nil)
	if merged.Fn != nil {
		t.Error("Expected nil Fn for empty handlers")
	}

	merged = mergeHandlers([]work.Handler{})
	if merged.Fn != nil {
		t.Error("Expected nil Fn for empty slice")
	}
}

func TestMergeHandlersSingle(t *testing.T) {
	called := false
	handlers := []work.Handler{{Fn: func(evt work.Event) work.Updates {
		called = true
		return nil
	}}}

	merged := mergeHandlers(handlers)
	merged.Fn(work.Event{})

	if !called {
		t.Error("Expected single handler to be called")
	}
}

func TestMergeHandlersMultiple(t *testing.T) {
	order := []int{}
	handlers := []work.Handler{
		{Fn: func(evt work.Event) work.Updates {
			order = append(order, 1)
			return nil
		}},
		{Fn: func(evt work.Event) work.Updates {
			order = append(order, 2)
			return nil
		}},
		{Fn: func(evt work.Event) work.Updates {
			order = append(order, 3)
			return nil
		}},
	}

	merged := mergeHandlers(handlers)
	merged.Fn(work.Event{})

	if len(order) != 3 {
		t.Errorf("Expected 3 calls, got %d", len(order))
	}
	for i, v := range order {
		if v != i+1 {
			t.Errorf("Expected call order %d at index %d, got %d", i+1, i, v)
		}
	}
}

func TestMergeHandlersSkipsNilFn(t *testing.T) {
	callCount := 0
	handlers := []work.Handler{
		{Fn: func(evt work.Event) work.Updates {
			callCount++
			return nil
		}},
		{Fn: nil},
		{Fn: func(evt work.Event) work.Updates {
			callCount++
			return nil
		}},
	}

	merged := mergeHandlers(handlers)
	merged.Fn(work.Event{})

	if callCount != 2 {
		t.Errorf("Expected 2 calls (skipping nil), got %d", callCount)
	}
}
