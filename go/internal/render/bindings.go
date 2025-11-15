package render

import "strings"

// BindingExtractor handles extraction of all binding types from elements.
type BindingExtractor struct {
	handlerBindings []HandlerBinding
	slotPaths       []SlotPath
	uploadBindings  []UploadBinding
	refBindings     []RefBinding
	routerBindings  []RouterBinding
}

// NewBindingExtractor creates a new binding extractor.
func NewBindingExtractor() *BindingExtractor {
	return &BindingExtractor{
		handlerBindings: make([]HandlerBinding, 0),
		slotPaths:       make([]SlotPath, 0),
		uploadBindings:  make([]UploadBinding, 0),
		refBindings:     make([]RefBinding, 0),
		routerBindings:  make([]RouterBinding, 0),
	}
}

// ExtractAll extracts all bindings from an element frame.
func (be *BindingExtractor) ExtractAll(frame elementFrame) {
	be.ExtractHandlerBindings(frame)
	be.ExtractSlotPaths(frame)
	be.ExtractUploadBindings(frame)
	be.ExtractRefBinding(frame)
	be.ExtractRouterBinding(frame)
}

// HandlerBindings returns all extracted handler bindings.
func (be *BindingExtractor) HandlerBindings() []HandlerBinding {
	return append([]HandlerBinding(nil), be.handlerBindings...)
}

// SlotPaths returns all extracted slot paths.
func (be *BindingExtractor) SlotPaths() []SlotPath {
	return append([]SlotPath(nil), be.slotPaths...)
}

// UploadBindings returns all extracted upload bindings.
func (be *BindingExtractor) UploadBindings() []UploadBinding {
	return append([]UploadBinding(nil), be.uploadBindings...)
}

// RefBindings returns all extracted ref bindings.
func (be *BindingExtractor) RefBindings() []RefBinding {
	return append([]RefBinding(nil), be.refBindings...)
}

// RouterBindings returns all extracted router bindings.
func (be *BindingExtractor) RouterBindings() []RouterBinding {
	return append([]RouterBinding(nil), be.routerBindings...)
}

// Internal helper methods for snapshot support

func (be *BindingExtractor) handlerBindingsLen() int {
	return len(be.handlerBindings)
}

func (be *BindingExtractor) slotPathsLen() int {
	return len(be.slotPaths)
}

func (be *BindingExtractor) uploadBindingsLen() int {
	return len(be.uploadBindings)
}

func (be *BindingExtractor) refBindingsLen() int {
	return len(be.refBindings)
}

func (be *BindingExtractor) routerBindingsLen() int {
	return len(be.routerBindings)
}

func (be *BindingExtractor) extractHandlerBindingsSlice(startIdx int) []HandlerBinding {
	return extractSlice(be.handlerBindings, startIdx)
}

func (be *BindingExtractor) extractSlotPathsSlice(startIdx int) []SlotPath {
	return extractSlice(be.slotPaths, startIdx)
}

func (be *BindingExtractor) extractUploadBindingsSlice(startIdx int) []UploadBinding {
	return extractSlice(be.uploadBindings, startIdx)
}

func (be *BindingExtractor) extractRefBindingsSlice(startIdx int) []RefBinding {
	return extractSlice(be.refBindings, startIdx)
}

func (be *BindingExtractor) extractRouterBindingsSlice(startIdx int) []RouterBinding {
	return extractSlice(be.routerBindings, startIdx)
}

func (be *BindingExtractor) ExtractHandlerBindings(frame elementFrame) {
	if len(frame.element.HandlerAssignments) == 0 || frame.attrSlot < 0 {
		return
	}

	for event, assignment := range frame.element.HandlerAssignments {
		be.handlerBindings = append(be.handlerBindings, HandlerBinding{
			Slot:    frame.attrSlot,
			Event:   event,
			Handler: assignment.ID,
			Listen:  append([]string(nil), assignment.Listen...),
			Props:   append([]string(nil), assignment.Props...),
		})
	}
}

func (be *BindingExtractor) ExtractSlotPaths(frame elementFrame) {
	if frame.componentID == "" {
		return
	}

	seen := make(map[[2]int]struct{}, len(frame.bindings))
	path := combineTypedPath(frame.basePath, frame.componentPath)

	for _, binding := range frame.bindings {
		key := [2]int{binding.slot, binding.childIndex}
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}

		anchor := SlotPath{
			Slot:           binding.slot,
			ComponentID:    frame.componentID,
			Path:           clonePath(path),
			TextChildIndex: -1,
		}
		if binding.childIndex >= 0 {
			anchor.TextChildIndex = binding.childIndex
		}
		be.slotPaths = append(be.slotPaths, anchor)
	}
}

func (be *BindingExtractor) ExtractUploadBindings(frame elementFrame) {
	if frame.componentID == "" || len(frame.element.UploadBindings) == 0 {
		return
	}

	path := combineTypedPath(frame.basePath, frame.componentPath)

	for _, upload := range frame.element.UploadBindings {
		if upload.UploadID == "" {
			continue
		}

		binding := UploadBinding{
			ComponentID: frame.componentID,
			Path:        clonePath(path),
			UploadID:    upload.UploadID,
			Multiple:    upload.Multiple,
			MaxSize:     upload.MaxSize,
		}
		if len(upload.Accept) > 0 {
			binding.Accept = append([]string(nil), upload.Accept...)
		}
		be.uploadBindings = append(be.uploadBindings, binding)
	}
}

func (be *BindingExtractor) ExtractRefBinding(frame elementFrame) {
	refID := strings.TrimSpace(frame.element.RefID)
	if refID == "" {
		return
	}

	be.refBindings = append(be.refBindings, RefBinding{
		ComponentID: frame.componentID,
		Path:        combineTypedPath(frame.basePath, frame.componentPath),
		RefID:       refID,
	})
}

func (be *BindingExtractor) ExtractRouterBinding(frame elementFrame) {
	router := buildRouterBinding(frame)
	if router != nil {
		be.routerBindings = append(be.routerBindings, *router)
	}
}

func buildRouterBinding(frame elementFrame) *RouterBinding {
	el := frame.element
	if el == nil || el.RouterMeta == nil {
		return nil
	}
	meta := el.RouterMeta
	if meta.Path == "" && meta.Query == "" && meta.Hash == "" && meta.Replace == "" {
		return nil
	}
	return &RouterBinding{
		ComponentID: frame.componentID,
		Path:        combineTypedPath(frame.basePath, frame.componentPath),
		PathValue:   meta.Path,
		Query:       meta.Query,
		Hash:        meta.Hash,
		Replace:     meta.Replace,
	}
}

func (be *BindingExtractor) mergeFrom(other *BindingExtractor, dynamicsOffset int) {
	for _, binding := range other.handlerBindings {
		binding.Slot += dynamicsOffset
		be.handlerBindings = append(be.handlerBindings, binding)
	}
	for _, slotPath := range other.slotPaths {
		slotPath.Slot += dynamicsOffset
		be.slotPaths = append(be.slotPaths, slotPath)
	}
	be.uploadBindings = append(be.uploadBindings, other.uploadBindings...)
	be.refBindings = append(be.refBindings, other.refBindings...)
	be.routerBindings = append(be.routerBindings, other.routerBindings...)
}
