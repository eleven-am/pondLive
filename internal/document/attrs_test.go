package document

import (
	"strings"
	"testing"
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
