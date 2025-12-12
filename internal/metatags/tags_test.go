package metatags

import (
	"testing"

	"github.com/eleven-am/pondlive/internal/work"
)

func TestMetaTagToNode_Basic(t *testing.T) {
	tag := MetaTag{
		Name:    "description",
		Content: "A test page",
	}

	node := tag.ToNode()
	el, ok := node.(*work.Element)
	if !ok {
		t.Fatal("expected *work.Element")
	}

	if el.Tag != "meta" {
		t.Errorf("expected tag 'meta', got %q", el.Tag)
	}
	if len(el.Attrs["name"]) != 1 || el.Attrs["name"][0] != "description" {
		t.Errorf("expected name='description', got %v", el.Attrs["name"])
	}
	if len(el.Attrs["content"]) != 1 || el.Attrs["content"][0] != "A test page" {
		t.Errorf("expected content='A test page', got %v", el.Attrs["content"])
	}
}

func TestMetaTagToNode_AllFields(t *testing.T) {
	tag := MetaTag{
		Name:      "viewport",
		Content:   "width=device-width",
		Property:  "og:title",
		Charset:   "UTF-8",
		HTTPEquiv: "X-UA-Compatible",
		ItemProp:  "name",
		Attrs:     map[string]string{"data-custom": "value"},
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if el.Attrs["name"][0] != "viewport" {
		t.Error("name not set")
	}
	if el.Attrs["content"][0] != "width=device-width" {
		t.Error("content not set")
	}
	if el.Attrs["property"][0] != "og:title" {
		t.Error("property not set")
	}
	if el.Attrs["charset"][0] != "UTF-8" {
		t.Error("charset not set")
	}
	if el.Attrs["http-equiv"][0] != "X-UA-Compatible" {
		t.Error("http-equiv not set")
	}
	if el.Attrs["itemprop"][0] != "name" {
		t.Error("itemprop not set")
	}
	if el.Attrs["data-custom"][0] != "value" {
		t.Error("custom attr not set")
	}
}

func TestMetaTagToNode_Empty(t *testing.T) {
	tag := MetaTag{}
	node := tag.ToNode()
	el := node.(*work.Element)

	if el.Tag != "meta" {
		t.Errorf("expected tag 'meta', got %q", el.Tag)
	}
	if len(el.Attrs) != 0 {
		t.Errorf("expected no attrs for empty tag, got %v", el.Attrs)
	}
}

func TestMetaTags_Multiple(t *testing.T) {
	tags := MetaTags(
		MetaTag{Name: "description", Content: "Test"},
		MetaTag{Name: "keywords", Content: "go, test"},
	)

	if len(tags) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(tags))
	}
}

func TestMetaTags_Empty(t *testing.T) {
	tags := MetaTags()
	if tags != nil {
		t.Errorf("expected nil for empty tags, got %v", tags)
	}
}

func TestLinkTagToNode_Basic(t *testing.T) {
	tag := LinkTag{
		Rel:  "stylesheet",
		Href: "/styles.css",
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if el.Tag != "link" {
		t.Errorf("expected tag 'link', got %q", el.Tag)
	}
	if el.Attrs["rel"][0] != "stylesheet" {
		t.Error("rel not set")
	}
	if el.Attrs["href"][0] != "/styles.css" {
		t.Error("href not set")
	}
}

func TestLinkTagToNode_AllFields(t *testing.T) {
	tag := LinkTag{
		Rel:            "stylesheet",
		Href:           "/styles.css",
		Type:           "text/css",
		As:             "style",
		Media:          "screen",
		HrefLang:       "en",
		Title:          "Main Styles",
		CrossOrigin:    "anonymous",
		Integrity:      "sha384-xxx",
		ReferrerPolicy: "no-referrer",
		Sizes:          "32x32",
		Attrs:          map[string]string{"data-theme": "dark"},
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if el.Attrs["rel"][0] != "stylesheet" {
		t.Error("rel not set")
	}
	if el.Attrs["href"][0] != "/styles.css" {
		t.Error("href not set")
	}
	if el.Attrs["type"][0] != "text/css" {
		t.Error("type not set")
	}
	if el.Attrs["as"][0] != "style" {
		t.Error("as not set")
	}
	if el.Attrs["media"][0] != "screen" {
		t.Error("media not set")
	}
	if el.Attrs["hreflang"][0] != "en" {
		t.Error("hreflang not set")
	}
	if el.Attrs["title"][0] != "Main Styles" {
		t.Error("title not set")
	}
	if el.Attrs["crossorigin"][0] != "anonymous" {
		t.Error("crossorigin not set")
	}
	if el.Attrs["integrity"][0] != "sha384-xxx" {
		t.Error("integrity not set")
	}
	if el.Attrs["referrerpolicy"][0] != "no-referrer" {
		t.Error("referrerpolicy not set")
	}
	if el.Attrs["sizes"][0] != "32x32" {
		t.Error("sizes not set")
	}
	if el.Attrs["data-theme"][0] != "dark" {
		t.Error("custom attr not set")
	}
}

func TestLinkTags_Multiple(t *testing.T) {
	tags := LinkTags(
		LinkTag{Rel: "stylesheet", Href: "/a.css"},
		LinkTag{Rel: "icon", Href: "/favicon.ico"},
	)

	if len(tags) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(tags))
	}
}

func TestLinkTags_Empty(t *testing.T) {
	tags := LinkTags()
	if tags != nil {
		t.Errorf("expected nil for empty tags, got %v", tags)
	}
}

func TestScriptTagToNode_Basic(t *testing.T) {
	tag := ScriptTag{
		Src: "/app.js",
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if el.Tag != "script" {
		t.Errorf("expected tag 'script', got %q", el.Tag)
	}
	if el.Attrs["src"][0] != "/app.js" {
		t.Error("src not set")
	}
}

func TestScriptTagToNode_AllFields(t *testing.T) {
	tag := ScriptTag{
		Src:            "/app.js",
		Type:           "text/javascript",
		Async:          true,
		Defer:          true,
		CrossOrigin:    "anonymous",
		Integrity:      "sha384-xxx",
		ReferrerPolicy: "strict-origin",
		Nonce:          "abc123",
		Attrs:          map[string]string{"data-main": "true"},
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if el.Attrs["src"][0] != "/app.js" {
		t.Error("src not set")
	}
	if el.Attrs["type"][0] != "text/javascript" {
		t.Error("type not set")
	}
	if _, ok := el.Attrs["async"]; !ok {
		t.Error("async not set")
	}
	if _, ok := el.Attrs["defer"]; !ok {
		t.Error("defer not set")
	}
	if el.Attrs["crossorigin"][0] != "anonymous" {
		t.Error("crossorigin not set")
	}
	if el.Attrs["integrity"][0] != "sha384-xxx" {
		t.Error("integrity not set")
	}
	if el.Attrs["referrerpolicy"][0] != "strict-origin" {
		t.Error("referrerpolicy not set")
	}
	if el.Attrs["nonce"][0] != "abc123" {
		t.Error("nonce not set")
	}
	if el.Attrs["data-main"][0] != "true" {
		t.Error("custom attr not set")
	}
}

func TestScriptTagToNode_Module(t *testing.T) {
	tag := ScriptTag{
		Src:    "/app.mjs",
		Type:   "text/javascript",
		Module: true,
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if el.Attrs["type"][0] != "module" {
		t.Errorf("expected type='module' when Module=true, got %q", el.Attrs["type"][0])
	}
}

func TestScriptTagToNode_NoModule(t *testing.T) {
	tag := ScriptTag{
		Src:      "/fallback.js",
		NoModule: true,
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if _, ok := el.Attrs["nomodule"]; !ok {
		t.Error("nomodule not set")
	}
}

func TestScriptTagToNode_InnerContent(t *testing.T) {
	tag := ScriptTag{
		Inner: "console.log('hello')",
	}

	node := tag.ToNode()
	el := node.(*work.Element)

	if el.UnsafeHTML != "console.log('hello')" {
		t.Errorf("expected UnsafeHTML, got %q", el.UnsafeHTML)
	}
}

func TestScriptTags_Multiple(t *testing.T) {
	tags := ScriptTags(
		ScriptTag{Src: "/a.js"},
		ScriptTag{Src: "/b.js"},
	)

	if len(tags) != 2 {
		t.Errorf("expected 2 nodes, got %d", len(tags))
	}
}

func TestScriptTags_Empty(t *testing.T) {
	tags := ScriptTags()
	if tags != nil {
		t.Errorf("expected nil for empty tags, got %v", tags)
	}
}
