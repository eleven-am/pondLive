package metatags

import (
	"testing"
)

func TestGetMergedMetaEmpty(t *testing.T) {
	entries := make(map[string]metaEntry)
	meta := getMergedMeta(entries)

	if meta.Title != "PondLive Application" {
		t.Errorf("Expected default title, got %q", meta.Title)
	}
	if meta.Description != "A PondLive application" {
		t.Errorf("Expected default description, got %q", meta.Description)
	}
}

func TestGetMergedMetaSingleEntry(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["comp1"] = metaEntry{
		componentID: "comp1",
		depth:       0,
		meta: &Meta{
			Title:       "My Page",
			Description: "Page description",
		},
	}

	meta := getMergedMeta(entries)

	if meta.Title != "My Page" {
		t.Errorf("Expected title 'My Page', got %q", meta.Title)
	}
	if meta.Description != "Page description" {
		t.Errorf("Expected description 'Page description', got %q", meta.Description)
	}
}

func TestGetMergedMetaChildWins(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["layout"] = metaEntry{
		componentID: "layout",
		depth:       0,
		meta: &Meta{
			Title:       "My App",
			Description: "App description",
		},
	}
	entries["page"] = metaEntry{
		componentID: "page",
		depth:       1,
		meta: &Meta{
			Title: "Home Page",
		},
	}

	meta := getMergedMeta(entries)

	if meta.Title != "Home Page" {
		t.Errorf("Expected child title 'Home Page', got %q", meta.Title)
	}

	if meta.Description != "App description" {
		t.Errorf("Expected parent description 'App description', got %q", meta.Description)
	}
}

func TestGetMergedMetaGrandchildWins(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["root"] = metaEntry{
		componentID: "root",
		depth:       0,
		meta: &Meta{
			Title:       "Root Title",
			Description: "Root description",
		},
	}
	entries["parent"] = metaEntry{
		componentID: "parent",
		depth:       1,
		meta: &Meta{
			Title:       "Parent Title",
			Description: "Parent description",
		},
	}
	entries["child"] = metaEntry{
		componentID: "child",
		depth:       2,
		meta: &Meta{
			Title: "Child Title",
		},
	}

	meta := getMergedMeta(entries)

	if meta.Title != "Child Title" {
		t.Errorf("Expected grandchild title 'Child Title', got %q", meta.Title)
	}

	if meta.Description != "Parent description" {
		t.Errorf("Expected parent description 'Parent description', got %q", meta.Description)
	}
}

func TestGetMergedMetaTags(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["parent"] = metaEntry{
		componentID: "parent",
		depth:       0,
		meta: &Meta{
			Meta: []MetaTag{
				{Property: "og:title", Content: "Parent OG Title"},
				{Name: "viewport", Content: "width=device-width"},
			},
		},
	}
	entries["child"] = metaEntry{
		componentID: "child",
		depth:       1,
		meta: &Meta{
			Meta: []MetaTag{
				{Property: "og:title", Content: "Child OG Title"},
				{Property: "og:description", Content: "Child OG Description"},
			},
		},
	}

	meta := getMergedMeta(entries)

	if len(meta.Meta) != 3 {
		t.Errorf("Expected 3 meta tags, got %d", len(meta.Meta))
	}

	foundOGTitle := false
	for _, m := range meta.Meta {
		if m.Property == "og:title" {
			foundOGTitle = true
			if m.Content != "Child OG Title" {
				t.Errorf("Expected og:title 'Child OG Title', got %q", m.Content)
			}
		}
	}
	if !foundOGTitle {
		t.Error("og:title not found in merged meta")
	}
}

func TestGetMergedMetaLinks(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["parent"] = metaEntry{
		componentID: "parent",
		depth:       0,
		meta: &Meta{
			Links: []LinkTag{
				{Rel: "stylesheet", Href: "/styles.css"},
				{Rel: "icon", Href: "/favicon.ico"},
			},
		},
	}
	entries["child"] = metaEntry{
		componentID: "child",
		depth:       1,
		meta: &Meta{
			Links: []LinkTag{
				{Rel: "stylesheet", Href: "/page.css"},
			},
		},
	}

	meta := getMergedMeta(entries)

	if len(meta.Links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(meta.Links))
	}
}

func TestGetMergedMetaRemove(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["parent"] = metaEntry{
		componentID: "parent",
		depth:       0,
		meta: &Meta{
			Title: "Parent Title",
		},
	}
	entries["child"] = metaEntry{
		componentID: "child",
		depth:       1,
		meta: &Meta{
			Title: "Child Title",
		},
	}

	meta := getMergedMeta(entries)
	if meta.Title != "Child Title" {
		t.Errorf("Before remove: expected 'Child Title', got %q", meta.Title)
	}

	delete(entries, "child")

	meta = getMergedMeta(entries)
	if meta.Title != "Parent Title" {
		t.Errorf("After remove: expected 'Parent Title', got %q", meta.Title)
	}
}

func TestGetMergedMetaScripts(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["parent"] = metaEntry{
		componentID: "parent",
		depth:       0,
		meta: &Meta{
			Scripts: []ScriptTag{
				{Src: "/analytics.js", Async: true},
			},
		},
	}
	entries["child"] = metaEntry{
		componentID: "child",
		depth:       1,
		meta: &Meta{
			Scripts: []ScriptTag{
				{Src: "/analytics.js", Defer: true},
			},
		},
	}

	meta := getMergedMeta(entries)

	if len(meta.Scripts) != 1 {
		t.Errorf("Expected 1 script, got %d", len(meta.Scripts))
	}

	if len(meta.Scripts) > 0 && !meta.Scripts[0].Defer {
		t.Error("Expected child's script with Defer=true to win")
	}
}

func TestGetMergedMetaInlineScripts(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["comp1"] = metaEntry{
		componentID: "comp1",
		depth:       0,
		meta: &Meta{
			Scripts: []ScriptTag{
				{Type: "text/javascript"},
				{Type: "module"},
			},
		},
	}
	entries["comp2"] = metaEntry{
		componentID: "comp2",
		depth:       0,
		meta: &Meta{
			Scripts: []ScriptTag{
				{Async: true},
			},
		},
	}
	entries["comp3"] = metaEntry{
		componentID: "comp3",
		depth:       1,
		meta: &Meta{
			Scripts: []ScriptTag{
				{Defer: true},
			},
		},
	}

	meta := getMergedMeta(entries)

	if len(meta.Scripts) != 4 {
		t.Errorf("Expected 4 inline scripts from different components, got %d", len(meta.Scripts))
	}
}

func TestGetMergedMetaInlineScriptsDeepWins(t *testing.T) {
	entries := make(map[string]metaEntry)
	entries["parent"] = metaEntry{
		componentID: "parent",
		depth:       0,
		meta: &Meta{
			Scripts: []ScriptTag{
				{Src: "/shared.js", Async: true},
			},
		},
	}
	entries["child"] = metaEntry{
		componentID: "child",
		depth:       1,
		meta: &Meta{
			Scripts: []ScriptTag{
				{Src: "/shared.js", Defer: true},
			},
		},
	}

	meta := getMergedMeta(entries)

	if len(meta.Scripts) != 1 {
		t.Fatalf("Expected 1 script, got %d", len(meta.Scripts))
	}

	if !meta.Scripts[0].Defer {
		t.Error("Expected deeper component's script to win")
	}
}
