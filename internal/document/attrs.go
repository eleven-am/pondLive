package document

import (
	"strings"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

var HtmlElement = runtime.Component(func(ctx *runtime.Ctx, children []work.Item) work.Node {
	state := documentCtx.UseContextValue(ctx)
	attrs := computeHtmlAttrs(state)

	return &work.Element{
		Tag:      "html",
		Attrs:    attrs,
		Children: work.ItemsToNodes(children),
	}
})

var BodyElement = runtime.Component(func(ctx *runtime.Ctx, children []work.Item) work.Node {
	state := documentCtx.UseContextValue(ctx)
	attrs := computeBodyAttrs(state)

	return &work.Element{
		Tag:      "body",
		Attrs:    attrs,
		Children: work.ItemsToNodes(children),
	}
})

func computeHtmlAttrs(state *documentState) map[string][]string {
	if state == nil || len(state.entries) == 0 {
		return nil
	}

	merged := getMergedDocument(state.entries)
	attrs := make(map[string][]string)

	if merged.HtmlClass != "" {
		attrs["class"] = []string{merged.HtmlClass}
	}

	if merged.HtmlLang != "" {
		attrs["lang"] = []string{merged.HtmlLang}
	}

	if merged.HtmlDir != "" {
		attrs["dir"] = []string{merged.HtmlDir}
	}

	if len(attrs) == 0 {
		return nil
	}

	return attrs
}

func computeBodyAttrs(state *documentState) map[string][]string {
	if state == nil || len(state.entries) == 0 {
		return nil
	}

	merged := getMergedDocument(state.entries)
	if merged.BodyClass == "" {
		return nil
	}

	return map[string][]string{
		"class": {merged.BodyClass},
	}
}

func getMergedDocument(entries map[string]documentEntry) *Document {
	if len(entries) == 0 {
		return &Document{}
	}

	var deepestLang *documentEntry
	var deepestDir *documentEntry
	htmlClasses := make([]string, 0)
	bodyClasses := make([]string, 0)
	seenHtmlClasses := make(map[string]struct{})
	seenBodyClasses := make(map[string]struct{})

	for _, entry := range entries {
		if entry.doc == nil {
			continue
		}

		if entry.doc.HtmlLang != "" {
			if deepestLang == nil || entry.depth > deepestLang.depth {
				deepestLang = &entry
			}
		}

		if entry.doc.HtmlDir != "" {
			if deepestDir == nil || entry.depth > deepestDir.depth {
				deepestDir = &entry
			}
		}

		if entry.doc.HtmlClass != "" {
			for _, class := range strings.Fields(entry.doc.HtmlClass) {
				if _, seen := seenHtmlClasses[class]; !seen {
					seenHtmlClasses[class] = struct{}{}
					htmlClasses = append(htmlClasses, class)
				}
			}
		}

		if entry.doc.BodyClass != "" {
			for _, class := range strings.Fields(entry.doc.BodyClass) {
				if _, seen := seenBodyClasses[class]; !seen {
					seenBodyClasses[class] = struct{}{}
					bodyClasses = append(bodyClasses, class)
				}
			}
		}
	}

	result := &Document{}

	if deepestLang != nil {
		result.HtmlLang = deepestLang.doc.HtmlLang
	}

	if deepestDir != nil {
		result.HtmlDir = deepestDir.doc.HtmlDir
	}

	if len(htmlClasses) > 0 {
		result.HtmlClass = strings.Join(htmlClasses, " ")
	}

	if len(bodyClasses) > 0 {
		result.BodyClass = strings.Join(bodyClasses, " ")
	}

	return result
}
