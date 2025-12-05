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
	handlers := computeBodyHandlers(state)

	return &work.Element{
		Tag:      "body",
		Attrs:    attrs,
		Handlers: handlers,
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

func computeBodyHandlers(state *documentState) map[string]work.Handler {
	if state == nil || len(state.entries) == 0 {
		return nil
	}

	allHandlers := make(map[string][]work.Handler)

	for _, entry := range state.entries {
		for event, handlers := range entry.bodyHandlers {
			allHandlers[event] = append(allHandlers[event], handlers...)
		}
	}

	if len(allHandlers) == 0 {
		return nil
	}

	result := make(map[string]work.Handler, len(allHandlers))
	for event, handlers := range allHandlers {
		if len(handlers) == 0 {
			continue
		}
		if len(handlers) == 1 {
			result[event] = handlers[0]
			continue
		}
		merged := mergeHandlers(handlers)
		result[event] = merged
	}

	return result
}

func mergeHandlers(handlers []work.Handler) work.Handler {
	if len(handlers) == 0 {
		return work.Handler{}
	}
	if len(handlers) == 1 {
		return handlers[0]
	}

	merged := handlers[0]
	for i := 1; i < len(handlers); i++ {
		merged.EventOptions = work.MergeEventOptions(merged.EventOptions, handlers[i].EventOptions)
	}

	merged.Fn = func(evt work.Event) work.Updates {
		var result work.Updates
		for _, h := range handlers {
			if h.Fn == nil {
				continue
			}
			if out := h.Fn(evt); out != nil {
				result = out
			}
		}
		return result
	}

	return merged
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
