package runtime

import (
	"fmt"
	"reflect"

	"github.com/eleven-am/pondlive/go/internal/work"
)

type StreamItem[T any] struct {
	Key   string
	Value T
}

type StreamHandle[T any] interface {
	Append(StreamItem[T]) bool
	Prepend(StreamItem[T]) bool
	InsertBefore(anchorKey string, item StreamItem[T]) bool
	InsertAfter(anchorKey string, item StreamItem[T]) bool
	Replace(StreamItem[T]) bool
	Upsert(StreamItem[T]) bool
	Delete(key string) bool
	Clear() bool
	Reset(items []StreamItem[T]) bool
	Items() []StreamItem[T]
}

type streamHandle[T any] struct {
	items *[]StreamItem[T]
	set   func([]StreamItem[T])
	index *Ref[map[string]int]
}

func UseStream[T any](
	ctx *Ctx,
	renderRow func(StreamItem[T]) work.Node,
	initial ...StreamItem[T],
) (work.Node, StreamHandle[T]) {
	if ctx == nil || ctx.instance == nil {
		panic("runtime: UseStream called outside component render")
	}
	if renderRow == nil {
		panic("runtime: UseStream requires a row renderer")
	}

	initialCopy := cloneStreamItems(initial)
	ensureUniqueKeys(initialCopy)

	items, set := UseState(ctx, initialCopy)
	indexRef := UseRef(ctx, map[string]int{})

	handle := &streamHandle[T]{
		items: &items,
		set:   set,
		index: indexRef,
	}
	handle.rebuildIndex(items)

	frag := renderStreamFragment(items, renderRow)
	return frag, handle
}

func renderStreamFragment[T any](
	items []StreamItem[T],
	renderRow func(StreamItem[T]) work.Node,
) work.Node {
	if len(items) == 0 {
		return &work.Fragment{}
	}

	children := make([]work.Node, 0, len(items))
	for _, item := range items {
		node := buildStreamRow(item, renderRow)
		if node != nil {
			children = append(children, node)
		}
	}

	return &work.Fragment{
		Children: children,
	}
}

func buildStreamRow[T any](
	item StreamItem[T],
	renderRow func(StreamItem[T]) work.Node,
) work.Node {
	if item.Key == "" {
		panic("runtime: UseStream items require a non-empty key")
	}

	node := renderRow(item)
	if node == nil {
		panic("runtime: UseStream row renderer returned nil")
	}

	if elem, ok := node.(*work.Element); ok {
		if elem.Key != "" && elem.Key != item.Key {
			panic(fmt.Sprintf("runtime: UseStream row renderer set conflicting key %q (expected %q)", elem.Key, item.Key))
		}
		elem.Key = item.Key
	}

	return node
}

func (h *streamHandle[T]) get() []StreamItem[T] {
	if h.items == nil {
		return nil
	}
	return *h.items
}

func (h *streamHandle[T]) Append(item StreamItem[T]) bool {
	return h.insertAt(len(h.get()), item)
}

func (h *streamHandle[T]) Prepend(item StreamItem[T]) bool {
	return h.insertAt(0, item)
}

func (h *streamHandle[T]) InsertBefore(anchorKey string, item StreamItem[T]) bool {
	idx, ok := h.lookup(anchorKey)
	if !ok {
		idx = indexOfKey(h.get(), anchorKey)
		if idx == -1 {
			return false
		}
	}
	return h.insertAt(idx, item)
}

func (h *streamHandle[T]) InsertAfter(anchorKey string, item StreamItem[T]) bool {
	idx, ok := h.lookup(anchorKey)
	if !ok {
		idx = indexOfKey(h.get(), anchorKey)
		if idx == -1 {
			return false
		}
	}
	return h.insertAt(idx+1, item)
}

func (h *streamHandle[T]) Replace(item StreamItem[T]) bool {
	ensureKey(item.Key)
	idx, ok := h.lookup(item.Key)
	if !ok {
		return false
	}

	current := h.get()
	next := cloneStreamItems(current)
	next[idx] = item
	h.setAndReindex(next)
	return true
}

func (h *streamHandle[T]) Upsert(item StreamItem[T]) bool {
	if h.Replace(item) {
		return true
	}
	return h.Append(item)
}

func (h *streamHandle[T]) Delete(key string) bool {
	idx, ok := h.lookup(key)
	if !ok {
		idx = indexOfKey(h.get(), key)
		if idx == -1 {
			return false
		}
	}

	current := h.get()
	next := make([]StreamItem[T], 0, len(current)-1)
	next = append(next, current[:idx]...)
	next = append(next, current[idx+1:]...)
	h.setAndReindex(next)
	return true
}

func (h *streamHandle[T]) Clear() bool {
	if len(h.get()) == 0 {
		return false
	}
	h.setAndReindex(nil)
	return true
}

func (h *streamHandle[T]) Reset(items []StreamItem[T]) bool {
	normalized := cloneStreamItems(items)
	ensureUniqueKeys(normalized)

	if reflect.DeepEqual(h.get(), normalized) {
		return false
	}

	h.setAndReindex(normalized)
	return true
}

func (h *streamHandle[T]) Items() []StreamItem[T] {
	return cloneStreamItems(h.get())
}

func (h *streamHandle[T]) insertAt(idx int, item StreamItem[T]) bool {
	ensureKey(item.Key)

	current := h.get()
	if idx < 0 {
		idx = 0
	}
	if idx > len(current) {
		idx = len(current)
	}

	existingIdx, exists := h.lookup(item.Key)
	base := cloneStreamItems(current)
	if exists {
		base = append(base[:existingIdx], base[existingIdx+1:]...)
		if existingIdx < idx {
			idx--
		}
	}

	next := make([]StreamItem[T], 0, len(base)+1)
	next = append(next, base[:idx]...)
	next = append(next, item)
	next = append(next, base[idx:]...)

	h.setAndReindex(next)
	return true
}

func (h *streamHandle[T]) lookup(key string) (int, bool) {
	if key == "" || h.index == nil || h.index.Current == nil {
		return 0, false
	}
	idx, ok := h.index.Current[key]
	return idx, ok
}

func (h *streamHandle[T]) setAndReindex(items []StreamItem[T]) {
	clone := cloneStreamItems(items)
	h.set(clone)
	*h.items = clone
	h.rebuildIndex(clone)
}

func (h *streamHandle[T]) rebuildIndex(items []StreamItem[T]) {
	if h.index == nil {
		return
	}

	index := make(map[string]int, len(items))
	for i, item := range items {
		ensureKey(item.Key)
		if _, exists := index[item.Key]; exists {
			panic(fmt.Sprintf("runtime: duplicate stream key %q", item.Key))
		}
		index[item.Key] = i
	}
	h.index.Current = index
}

func cloneStreamItems[T any](items []StreamItem[T]) []StreamItem[T] {
	if len(items) == 0 {
		return nil
	}
	out := make([]StreamItem[T], len(items))
	copy(out, items)
	return out
}

func ensureUniqueKeys[T any](items []StreamItem[T]) {
	if len(items) == 0 {
		return
	}
	seen := make(map[string]struct{}, len(items))
	for _, item := range items {
		ensureKey(item.Key)
		if _, exists := seen[item.Key]; exists {
			panic(fmt.Sprintf("runtime: duplicate stream key %q", item.Key))
		}
		seen[item.Key] = struct{}{}
	}
}

func ensureKey(key string) {
	if key == "" {
		panic("runtime: UseStream requires non-empty keys")
	}
}

func indexOfKey[T any](items []StreamItem[T], key string) int {
	for i, item := range items {
		if item.Key == key {
			return i
		}
	}
	return -1
}
