package runtime

import (
	"testing"

	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

func TestCloneMetaDeepCopy(t *testing.T) {
	meta := &Meta{
		Title:       "Title",
		Description: "Desc",
		Meta: []h.MetaTag{{
			Name:    "keywords",
			Content: "go,live",
			Attrs: map[string]string{
				"data-test": "value",
			},
		}},
		Links: []h.LinkTag{{
			Rel:  "canonical",
			Href: "/home",
		}},
		Scripts: []h.ScriptTag{{
			Src:   "/inline.js",
			Async: true,
		}},
	}
	clone := CloneMeta(meta)
	if clone == nil {
		t.Fatal("expected clone to be non-nil")
	}
	if clone == meta {
		t.Fatal("expected clone to be distinct pointer")
	}
	clone.Meta[0].Attrs["data-test"] = "changed"
	if meta.Meta[0].Attrs["data-test"] != "value" {
		t.Fatalf("expected original attrs to remain unchanged, got %q", meta.Meta[0].Attrs["data-test"])
	}
	clone.Links[0].Href = "/other"
	if meta.Links[0].Href != "/home" {
		t.Fatalf("expected original link href to remain, got %q", meta.Links[0].Href)
	}
	clone.Scripts[0].Src = "/other.js"
	if meta.Scripts[0].Src != "/inline.js" {
		t.Fatalf("expected original script src to remain, got %q", meta.Scripts[0].Src)
	}
}

func TestMergeMetaOverridesAndAppends(t *testing.T) {
	base := &Meta{
		Title:       "Base",
		Description: "Base description",
		Meta:        []h.MetaTag{{Name: "description", Content: "legacy"}},
	}
	override := &Meta{
		Title:   "Override",
		Meta:    []h.MetaTag{{Property: "og:title", Content: "Override"}},
		Scripts: []h.ScriptTag{{Src: "/bundle.js"}},
	}
	merged := MergeMeta(base, override)
	if merged.Title != "Override" {
		t.Fatalf("expected title to be Override, got %q", merged.Title)
	}
	if merged.Description != "Base description" {
		t.Fatalf("expected description to carry over, got %q", merged.Description)
	}
	if len(merged.Meta) != 2 {
		t.Fatalf("expected two meta tags, got %d", len(merged.Meta))
	}
	if merged.Meta[1].Property != "og:title" {
		t.Fatalf("expected appended property tag, got %+v", merged.Meta[1])
	}
	if len(merged.Scripts) != 1 || merged.Scripts[0].Src != "/bundle.js" {
		t.Fatalf("expected scripts to be appended, got %+v", merged.Scripts)
	}
}

func TestRenderResultUpdatesSessionMetadata(t *testing.T) {
	component := func(ctx Ctx, _ struct{}) h.Node {
		if sess := ctx.Session(); sess != nil {
			sess.SetMetadata(&Meta{Title: "Base"})
		}
		return WithMetadata(h.Div(), &Meta{
			Description: "Combined",
			Meta:        []h.MetaTag{{Name: "robots", Content: "index"}},
		})
	}
	sess := NewLiveSession("meta-session", 1, component, struct{}{}, nil)
	_ = render.RenderHTML(sess.RenderRoot())
	meta := sess.Metadata()
	if meta == nil {
		t.Fatal("expected metadata to be recorded")
	}
	if meta.Title != "Base" {
		t.Fatalf("expected title to remain Base, got %q", meta.Title)
	}
	if meta.Description != "Combined" {
		t.Fatalf("expected description Combined, got %q", meta.Description)
	}
	if len(meta.Meta) != 1 || meta.Meta[0].Name != "robots" {
		t.Fatalf("expected robots meta tag, got %+v", meta.Meta)
	}
}
