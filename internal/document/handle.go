package document

import (
	"maps"
	"strings"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/eleven-am/pondlive/internal/work"
)

type DocumentHandle struct {
	ctx         *runtime.Ctx
	componentID string
	depth       int
	state       *documentState
}

func newDocumentHandle(ctx *runtime.Ctx, state *documentState) *DocumentHandle {
	if state == nil {
		return &DocumentHandle{}
	}
	return &DocumentHandle{
		ctx:         ctx,
		componentID: ctx.ComponentID(),
		depth:       ctx.ComponentDepth(),
		state:       state,
	}
}

func (h *DocumentHandle) isNoop() bool {
	return h.state == nil
}

func (h *DocumentHandle) getOrCreateEntry() documentEntry {
	if h.isNoop() {
		return documentEntry{}
	}
	if entry, ok := h.state.entries[h.componentID]; ok {
		return entry
	}
	return documentEntry{
		doc:          &Document{},
		depth:        h.depth,
		componentID:  h.componentID,
		bodyHandlers: make(map[string][]work.Handler),
	}
}

func (h *DocumentHandle) updateEntry(entry documentEntry) {
	if h.isNoop() {
		return
	}
	next := maps.Clone(h.state.entries)
	next[h.componentID] = entry
	h.state.setEntries(next)
}

func (h *DocumentHandle) Html(items ...work.Item) *DocumentHandle {
	if h.isNoop() {
		return h
	}

	el := &work.Element{Attrs: make(map[string][]string)}
	for _, item := range items {
		item.ApplyTo(el)
	}

	entry := h.getOrCreateEntry()
	if entry.doc == nil {
		entry.doc = &Document{}
	}

	if classes, ok := el.Attrs["class"]; ok && len(classes) > 0 {
		entry.doc.HtmlClass = strings.Join(classes, " ")
	}
	if lang, ok := el.Attrs["lang"]; ok && len(lang) > 0 {
		entry.doc.HtmlLang = lang[0]
	}
	if dir, ok := el.Attrs["dir"]; ok && len(dir) > 0 {
		entry.doc.HtmlDir = dir[0]
	}

	h.updateEntry(entry)
	return h
}

func (h *DocumentHandle) Body(items ...work.Item) *DocumentHandle {
	if h.isNoop() {
		return h
	}

	el := &work.Element{Attrs: make(map[string][]string)}
	for _, item := range items {
		item.ApplyTo(el)
	}

	entry := h.getOrCreateEntry()
	if entry.doc == nil {
		entry.doc = &Document{}
	}

	if classes, ok := el.Attrs["class"]; ok && len(classes) > 0 {
		entry.doc.BodyClass = strings.Join(classes, " ")
	}

	h.updateEntry(entry)
	return h
}

func (h *DocumentHandle) AddBodyHandler(event string, handler work.Handler) *DocumentHandle {
	if h.isNoop() || handler.Fn == nil {
		return h
	}

	entry := h.getOrCreateEntry()
	if entry.bodyHandlers == nil {
		entry.bodyHandlers = make(map[string][]work.Handler)
	}

	entry.bodyHandlers[event] = append(entry.bodyHandlers[event], handler)
	h.updateEntry(entry)
	return h
}
