package runtime

import (
	"testing"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestBuildMetadataDiffDetectsNoChange(t *testing.T) {
	base := &Meta{
		Title:       "Same",
		Description: "Identical",
		Meta: []h.MetaTag{{
			Name:    "keywords",
			Content: "go,live",
		}},
		Links: []h.LinkTag{{
			Rel:  "canonical",
			Href: "/home",
		}},
		Scripts: []h.ScriptTag{{
			Src:   "/app.js",
			Async: true,
		}},
	}
	clone := CloneMeta(base)
	effect, changed := buildMetadataDiff(base, clone)
	if changed {
		t.Fatalf("expected no changes, got %+v", effect)
	}
	if effect != nil {
		t.Fatalf("expected effect to be nil, got %+v", effect)
	}
}

func TestBuildMetadataDiffDetectsChanges(t *testing.T) {
	prev := &Meta{
		Title:       "Prev",
		Description: "Keep",
		Meta: []h.MetaTag{{
			Name:    "description",
			Content: "Keep",
		}, {
			Name:    "keywords",
			Content: "go,live",
		}},
		Links: []h.LinkTag{{
			Rel:  "canonical",
			Href: "/prev",
		}, {
			Rel:  "alternate",
			Href: "/feed",
		}},
		Scripts: []h.ScriptTag{{
			Src:   "/prev.js",
			Async: true,
		}, {
			Src: "/inline.js",
		}},
	}
	next := &Meta{
		Title:       "Next",
		Description: " ",
		Meta: []h.MetaTag{{
			Name:    "keywords",
			Content: "go,live,fast",
		}, {
			Property: "og:type",
			Content:  "profile",
		}},
		Links: []h.LinkTag{{
			Rel:  "canonical",
			Href: "/next",
		}},
		Scripts: []h.ScriptTag{{
			Src: "/inline.js",
		}, {
			Src:   "/new.js",
			Defer: true,
		}},
	}

	effect, changed := buildMetadataDiff(prev, next)
	if !changed {
		t.Fatal("expected changes to be detected")
	}
	if effect == nil {
		t.Fatal("expected effect to be populated")
	}
	if effect.Title == nil || *effect.Title != "Next" {
		t.Fatalf("expected updated title, got %+v", effect.Title)
	}
	if !effect.ClearDescription {
		t.Fatal("expected description to be cleared")
	}
	if len(effect.MetaRemove) != 1 || effect.MetaRemove[0] != "meta:name:description" {
		t.Fatalf("expected description meta removal, got %+v", effect.MetaRemove)
	}
	if len(effect.MetaAdd) != 2 {
		t.Fatalf("expected two meta additions, got %+v", effect.MetaAdd)
	}
	if effect.MetaAdd[0].Key != "meta:name:keywords" || effect.MetaAdd[0].Content != "go,live,fast" {
		t.Fatalf("expected keywords update, got %+v", effect.MetaAdd[0])
	}
	if effect.MetaAdd[1].Key != "meta:property:og:type" {
		t.Fatalf("expected og:type addition, got %+v", effect.MetaAdd[1])
	}
	if len(effect.LinkRemove) != 2 {
		t.Fatalf("expected two link removals, got %+v", effect.LinkRemove)
	}
	if effect.LinkRemove[0] != "link:rel:canonical|href:/prev" || effect.LinkRemove[1] != "link:rel:alternate|href:/feed" {
		t.Fatalf("unexpected link removals, got %+v", effect.LinkRemove)
	}
	if len(effect.LinkAdd) != 1 || effect.LinkAdd[0].Href != "/next" {
		t.Fatalf("expected canonical addition, got %+v", effect.LinkAdd)
	}
	if effect.LinkAdd[0].Key != "link:rel:canonical|href:/next" {
		t.Fatalf("expected canonical key, got %+v", effect.LinkAdd[0])
	}
	if len(effect.ScriptRemove) != 1 || effect.ScriptRemove[0] != "script:src:/prev.js" {
		t.Fatalf("expected prev.js removal, got %+v", effect.ScriptRemove)
	}
	if len(effect.ScriptAdd) != 1 || effect.ScriptAdd[0].Src != "/new.js" {
		t.Fatalf("expected new.js addition, got %+v", effect.ScriptAdd)
	}
	if effect.ScriptAdd[0].Key != "script:src:/new.js" {
		t.Fatalf("expected new.js key, got %+v", effect.ScriptAdd[0])
	}
}
