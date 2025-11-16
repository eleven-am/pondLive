package http

import (
	"html"
	"sort"
	"strings"

	runtime "github.com/eleven-am/pondlive/go/internal/runtime"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

const (
	liveuiHeadAttr    = "data-live-head"
	liveuiHeadKeyAttr = "data-live-key"
)

// buildDocument constructs the outer HTML document used for SSR responses. The
// supplied body fragment is embedded within the shared skeleton so individual
// components only need to return their inner application markup.
func buildDocument(body string, meta *runtime.Meta, assetURL string) string {
	if strings.TrimSpace(assetURL) == "" {
		assetURL = defaultClientAssetURL
	}

	var builder strings.Builder
	builder.Grow(len(body) + 256)
	builder.WriteString("<!DOCTYPE html>")
	builder.WriteString("<html lang=\"en\">")
	builder.WriteString("<head>")
	builder.WriteString("<meta charset=\"utf-8\">")
	builder.WriteString("<meta name=\"viewport\" content=\"width=device-width, initial-scale=1\">")
	writeMetadata(&builder, meta)
	builder.WriteString("</head>")
	builder.WriteString("<body>")
	builder.WriteString(body)
	if assetURL != "" {
		builder.WriteString("<script src=\"")
		builder.WriteString(html.EscapeString(assetURL))
		if assetURL == defaultClientAssetURL || assetURL == devClientAssetURL {
			builder.WriteString("\" defer></script>")
		} else {
			builder.WriteString("\"></script>")
		}
	}
	builder.WriteString("</body></html>")
	return builder.String()
}

func writeMetadata(builder *strings.Builder, meta *runtime.Meta) {
	if builder == nil || meta == nil {
		return
	}
	clone := runtime.CloneMeta(meta)
	if clone == nil {
		return
	}
	if title := strings.TrimSpace(clone.Title); title != "" {
		builder.WriteString("<title>")
		builder.WriteString(html.EscapeString(title))
		builder.WriteString("</title>")
	}
	if desc := strings.TrimSpace(clone.Description); desc != "" {
		builder.WriteString("<meta")
		writeAttr(builder, "name", "description")
		writeAttr(builder, "content", desc)
		writeAttr(builder, liveuiHeadAttr, "description")
		writeAttr(builder, liveuiHeadKeyAttr, "description")
		builder.WriteString(">")
	}
	metaKeys := runtime.MetaTagKeys(clone.Meta)
	for i, tag := range clone.Meta {
		writeMetaTag(builder, tag, metaKeys[i])
	}
	linkKeys := runtime.LinkTagKeys(clone.Links)
	for i, link := range clone.Links {
		writeLinkTag(builder, link, linkKeys[i])
	}
	scriptKeys := runtime.ScriptTagKeys(clone.Scripts)
	for i, script := range clone.Scripts {
		writeScriptTag(builder, script, scriptKeys[i])
	}
}

func writeMetaTag(builder *strings.Builder, tag h.MetaTag, key string) {
	if builder == nil {
		return
	}
	builder.WriteString("<meta")
	writeAttr(builder, "name", tag.Name)
	writeAttr(builder, "content", tag.Content)
	writeAttr(builder, "property", tag.Property)
	writeAttr(builder, "charset", tag.Charset)
	writeAttr(builder, "http-equiv", tag.HTTPEquiv)
	writeAttr(builder, "itemprop", tag.ItemProp)
	writeAttr(builder, liveuiHeadAttr, "meta")
	writeAttr(builder, liveuiHeadKeyAttr, key)
	writeAttrMap(builder, tag.Attrs)
	builder.WriteString(">")
}

func writeLinkTag(builder *strings.Builder, tag h.LinkTag, key string) {
	if builder == nil {
		return
	}
	builder.WriteString("<link")
	writeAttr(builder, "rel", tag.Rel)
	writeAttr(builder, "href", tag.Href)
	writeAttr(builder, "type", tag.Type)
	writeAttr(builder, "as", tag.As)
	writeAttr(builder, "media", tag.Media)
	writeAttr(builder, "hreflang", tag.HrefLang)
	writeAttr(builder, "title", tag.Title)
	writeAttr(builder, "crossorigin", tag.CrossOrigin)
	writeAttr(builder, "integrity", tag.Integrity)
	writeAttr(builder, "referrerpolicy", tag.ReferrerPolicy)
	writeAttr(builder, "sizes", tag.Sizes)
	writeAttr(builder, liveuiHeadAttr, "link")
	writeAttr(builder, liveuiHeadKeyAttr, key)
	writeAttrMap(builder, tag.Attrs)
	builder.WriteString(">")
}

func writeScriptTag(builder *strings.Builder, tag h.ScriptTag, key string) {
	if builder == nil {
		return
	}
	builder.WriteString("<script")
	writeAttr(builder, "src", tag.Src)
	if tag.Module {
		writeAttr(builder, "type", "module")
	} else {
		writeAttr(builder, "type", tag.Type)
	}
	writeBoolAttr(builder, "async", tag.Async)
	writeBoolAttr(builder, "defer", tag.Defer)
	writeBoolAttr(builder, "nomodule", tag.NoModule)
	writeAttr(builder, "crossorigin", tag.CrossOrigin)
	writeAttr(builder, "integrity", tag.Integrity)
	writeAttr(builder, "referrerpolicy", tag.ReferrerPolicy)
	writeAttr(builder, "nonce", tag.Nonce)
	writeAttr(builder, liveuiHeadAttr, "script")
	writeAttr(builder, liveuiHeadKeyAttr, key)
	writeAttrMap(builder, tag.Attrs)
	builder.WriteString(">")
	if tag.Inner != "" {
		builder.WriteString(tag.Inner)
	}
	builder.WriteString("</script>")
}

func writeAttr(builder *strings.Builder, key, value string) {
	if builder == nil {
		return
	}
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return
	}
	builder.WriteByte(' ')
	builder.WriteString(key)
	builder.WriteString("=\"")
	builder.WriteString(html.EscapeString(trimmed))
	builder.WriteString("\"")
}

func writeBoolAttr(builder *strings.Builder, key string, enabled bool) {
	if !enabled || builder == nil {
		return
	}
	builder.WriteByte(' ')
	builder.WriteString(key)
}

func writeAttrMap(builder *strings.Builder, attrs map[string]string) {
	if builder == nil || len(attrs) == 0 {
		return
	}
	keys := make([]string, 0, len(attrs))
	for k := range attrs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		writeAttr(builder, k, attrs[k])
	}
}
