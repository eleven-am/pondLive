package metatags

import (
	"testing"
)

func TestControllerGetEmpty(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	meta := controller.Get()

	if meta.Title != "PondLive Application" {
		t.Errorf("Expected default title, got %q", meta.Title)
	}
	if meta.Description != "A PondLive application" {
		t.Errorf("Expected default description, got %q", meta.Description)
	}
}

func TestControllerGetSingleEntry(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("comp1", 0, &Meta{
		Title:       "My Page",
		Description: "Page description",
	})

	meta := controller.Get()

	if meta.Title != "My Page" {
		t.Errorf("Expected title 'My Page', got %q", meta.Title)
	}
	if meta.Description != "Page description" {
		t.Errorf("Expected description 'Page description', got %q", meta.Description)
	}
}

func TestControllerMergeChildWins(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("layout", 0, &Meta{
		Title:       "My App",
		Description: "App description",
	})

	controller.Set("page", 1, &Meta{
		Title: "Home Page",
	})

	meta := controller.Get()

	if meta.Title != "Home Page" {
		t.Errorf("Expected child title 'Home Page', got %q", meta.Title)
	}

	if meta.Description != "App description" {
		t.Errorf("Expected parent description 'App description', got %q", meta.Description)
	}
}

func TestControllerMergeGrandchildWins(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("root", 0, &Meta{
		Title:       "Root Title",
		Description: "Root description",
	})

	controller.Set("parent", 1, &Meta{
		Title:       "Parent Title",
		Description: "Parent description",
	})

	controller.Set("child", 2, &Meta{
		Title: "Child Title",
	})

	meta := controller.Get()

	if meta.Title != "Child Title" {
		t.Errorf("Expected grandchild title 'Child Title', got %q", meta.Title)
	}

	if meta.Description != "Parent description" {
		t.Errorf("Expected parent description 'Parent description', got %q", meta.Description)
	}
}

func TestControllerMergeMetaTags(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("parent", 0, &Meta{
		Meta: []MetaTag{
			{Property: "og:title", Content: "Parent OG Title"},
			{Name: "viewport", Content: "width=device-width"},
		},
	})

	controller.Set("child", 1, &Meta{
		Meta: []MetaTag{
			{Property: "og:title", Content: "Child OG Title"},
			{Property: "og:description", Content: "Child OG Description"},
		},
	})

	meta := controller.Get()

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

func TestControllerMergeLinks(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("parent", 0, &Meta{
		Links: []LinkTag{
			{Rel: "stylesheet", Href: "/styles.css"},
			{Rel: "icon", Href: "/favicon.ico"},
		},
	})

	controller.Set("child", 1, &Meta{
		Links: []LinkTag{
			{Rel: "stylesheet", Href: "/page.css"},
		},
	})

	meta := controller.Get()

	if len(meta.Links) != 3 {
		t.Errorf("Expected 3 links, got %d", len(meta.Links))
	}
}

func TestControllerRemove(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("parent", 0, &Meta{
		Title: "Parent Title",
	})

	controller.Set("child", 1, &Meta{
		Title: "Child Title",
	})

	meta := controller.Get()
	if meta.Title != "Child Title" {
		t.Errorf("Before remove: expected 'Child Title', got %q", meta.Title)
	}

	controller.Remove("child")

	meta = controller.Get()
	if meta.Title != "Parent Title" {
		t.Errorf("After remove: expected 'Parent Title', got %q", meta.Title)
	}
}

func TestControllerNilSafety(t *testing.T) {
	var controller *Controller

	meta := controller.Get()
	if meta == nil {
		t.Error("Get on nil controller should return default meta, not nil")
	}

	controller.Set("id", 0, &Meta{})
	controller.Remove("id")
}

func TestControllerMergeScripts(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("parent", 0, &Meta{
		Scripts: []ScriptTag{
			{Src: "/analytics.js", Async: true},
		},
	})

	controller.Set("child", 1, &Meta{
		Scripts: []ScriptTag{
			{Src: "/analytics.js", Defer: true},
		},
	})

	meta := controller.Get()

	if len(meta.Scripts) != 1 {
		t.Errorf("Expected 1 script, got %d", len(meta.Scripts))
	}

	if len(meta.Scripts) > 0 && !meta.Scripts[0].Defer {
		t.Error("Expected child's script with Defer=true to win")
	}
}

func TestControllerMergeInlineScripts(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("comp1", 0, &Meta{
		Scripts: []ScriptTag{
			{Type: "text/javascript"},
			{Type: "module"},
		},
	})

	controller.Set("comp2", 0, &Meta{
		Scripts: []ScriptTag{
			{Async: true},
		},
	})

	controller.Set("comp3", 1, &Meta{
		Scripts: []ScriptTag{
			{Defer: true},
		},
	})

	meta := controller.Get()

	if len(meta.Scripts) != 4 {
		t.Errorf("Expected 4 inline scripts from different components, got %d", len(meta.Scripts))
	}
}

func TestControllerMergeInlineScriptsDeepWins(t *testing.T) {
	entries := make(map[string]metaEntry)
	controller := &Controller{
		get:    func() map[string]metaEntry { return entries },
		set:    func(id string, e metaEntry) { entries[id] = e },
		remove: func(id string) { delete(entries, id) },
	}

	controller.Set("parent", 0, &Meta{
		Scripts: []ScriptTag{
			{Src: "/shared.js", Async: true},
		},
	})

	controller.Set("child", 1, &Meta{
		Scripts: []ScriptTag{
			{Src: "/shared.js", Defer: true},
		},
	})

	meta := controller.Get()

	if len(meta.Scripts) != 1 {
		t.Fatalf("Expected 1 script, got %d", len(meta.Scripts))
	}

	if !meta.Scripts[0].Defer {
		t.Error("Expected deeper component's script to win")
	}
}
