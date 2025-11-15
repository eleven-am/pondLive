package runtime

import (
	"fmt"
	"reflect"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// StreamItem represents a keyed value managed by UseStream. Keys must be
// stable and unique within the list.
type StreamItem[T any] struct {
	Key   string
	Value T
}

// StreamHandle exposes mutation helpers for a keyed list managed by UseStream.
// Implementations are intended for single-goroutine use from component event
// handlers. Methods return whether they changed the underlying state and
// therefore scheduled a render.
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
	sess  *ComponentSession
	get   func() []StreamItem[T]
	set   func([]StreamItem[T])
	index *Ref[map[string]int]
}

// UseStream manages a keyed collection and renders it as a dynamic list slot.
// The renderRow function must return an *html.Element; UseStream overwrites its
// Key field with the supplied item's key so diffing can emit minimal list ops.
func UseStream[T any](ctx Ctx, renderRow func(StreamItem[T]) h.Node, initial ...StreamItem[T]) (h.Node, StreamHandle[T]) {
	if renderRow == nil {
		panic("runtime: UseStream requires a row renderer")
	}

	initialCopy := cloneStreamItems(initial)
	ensureUniqueKeys(initialCopy)

	get, set := UseState(ctx, initialCopy)
	indexRef := UseRef(ctx, map[string]int{})

	handle := &streamHandle[T]{
		sess:  ctx.sess,
		get:   get,
		set:   set,
		index: indexRef,
	}
	handle.rebuildIndex(initialCopy)

	items := get()
	frag := renderStreamFragment(items, renderRow)
	return frag, handle
}

func renderStreamFragment[T any](items []StreamItem[T], renderRow func(StreamItem[T]) h.Node) h.Node {
	if len(items) == 0 {
		return h.Fragment()
	}
	children := make([]h.Node, 0, len(items))
	for _, item := range items {
		children = append(children, buildStreamRow(item, renderRow))
	}
	return h.Fragment(children...)
}

func buildStreamRow[T any](item StreamItem[T], renderRow func(StreamItem[T]) h.Node) h.Node {
	if item.Key == "" {
		panic("runtime: UseStream items require a non-empty key")
	}
	node := renderRow(item)
	if node == nil {
		panic("runtime: UseStream row renderer returned nil")
	}
	el, ok := node.(*h.Element)
	if !ok {
		panic("runtime: UseStream row renderer must return an *html.Element")
	}
	if el.Key != "" && el.Key != item.Key {
		panic(fmt.Sprintf("runtime: UseStream row renderer set conflicting key %q (expected %q)", el.Key, item.Key))
	}
	el.Key = item.Key
	return el
}

func (hnd *streamHandle[T]) Append(item StreamItem[T]) bool {
	return hnd.insertAt(len(hnd.get()), item)
}

func (hnd *streamHandle[T]) Prepend(item StreamItem[T]) bool {
	return hnd.insertAt(0, item)
}

func (hnd *streamHandle[T]) InsertBefore(anchorKey string, item StreamItem[T]) bool {
	idx, ok := hnd.lookup(anchorKey)
	if !ok {
		idx = indexOfKey(hnd.get(), anchorKey)
		if idx == -1 {
			return false
		}
	}
	return hnd.insertAt(idx, item)
}

func (hnd *streamHandle[T]) InsertAfter(anchorKey string, item StreamItem[T]) bool {
	idx, ok := hnd.lookup(anchorKey)
	if !ok {
		idx = indexOfKey(hnd.get(), anchorKey)
		if idx == -1 {
			return false
		}
	}
	return hnd.insertAt(idx+1, item)
}

func (hnd *streamHandle[T]) Replace(item StreamItem[T]) bool {
	ensureKey(item.Key)
	idx, ok := hnd.lookup(item.Key)
	if !ok {
		return false
	}
	current := hnd.get()
	next := cloneStreamItems(current)
	next[idx] = item
	hnd.setAndReindex(next)
	hnd.onLengthTransition(len(current), len(next))
	return true
}

func (hnd *streamHandle[T]) Upsert(item StreamItem[T]) bool {
	if hnd.Replace(item) {
		return true
	}
	return hnd.Append(item)
}

func (hnd *streamHandle[T]) Delete(key string) bool {
	idx, ok := hnd.lookup(key)
	if !ok {
		idx = indexOfKey(hnd.get(), key)
		if idx == -1 {
			return false
		}
	}
	current := hnd.get()
	next := make([]StreamItem[T], 0, len(current)-1)
	next = append(next, current[:idx]...)
	next = append(next, current[idx+1:]...)
	hnd.setAndReindex(next)
	hnd.onLengthTransition(len(current), len(next))
	return true
}

func (hnd *streamHandle[T]) Clear() bool {
	if len(hnd.get()) == 0 {
		return false
	}
	prevLen := len(hnd.get())
	hnd.setAndReindex(nil)
	hnd.onLengthTransition(prevLen, 0)
	return true
}

func (hnd *streamHandle[T]) Reset(items []StreamItem[T]) bool {
	normalized := cloneStreamItems(items)
	ensureUniqueKeys(normalized)
	prevLen := len(hnd.get())
	if reflect.DeepEqual(hnd.get(), normalized) {
		return false
	}
	hnd.setAndReindex(normalized)
	hnd.onLengthTransition(prevLen, len(normalized))
	return true
}

func (hnd *streamHandle[T]) Items() []StreamItem[T] {
	return cloneStreamItems(hnd.get())
}

func (hnd *streamHandle[T]) insertAt(idx int, item StreamItem[T]) bool {
	ensureKey(item.Key)
	current := hnd.get()
	if idx < 0 {
		idx = 0
	}
	if idx > len(current) {
		idx = len(current)
	}
	existingIdx, exists := hnd.lookup(item.Key)
	base := cloneStreamItems(current)
	if exists {
		base = append(base[:existingIdx], base[existingIdx+1:]...)
		if existingIdx < idx {
			idx--
		}
	} else {
		for i, entry := range base {
			if entry.Key != item.Key {
				continue
			}
			base = append(base[:i], base[i+1:]...)
			if i < idx {
				idx--
			}
			break
		}
	}
	next := make([]StreamItem[T], 0, len(base)+1)
	next = append(next, base[:idx]...)
	next = append(next, item)
	next = append(next, base[idx:]...)
	hnd.setAndReindex(next)
	hnd.onLengthTransition(len(current), len(next))
	return true
}

func (hnd *streamHandle[T]) lookup(key string) (int, bool) {
	if key == "" {
		return 0, false
	}
	if hnd.index == nil || hnd.index.Cur == nil {
		return 0, false
	}
	idx, ok := hnd.index.Cur[key]
	return idx, ok
}

func (hnd *streamHandle[T]) setAndReindex(items []StreamItem[T]) {
	clone := cloneStreamItems(items)
	hnd.set(clone)
	hnd.rebuildIndex(clone)
}

func (hnd *streamHandle[T]) rebuildIndex(items []StreamItem[T]) {
	if hnd.index == nil {
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
	hnd.index.Cur = index
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

func (hnd *streamHandle[T]) onLengthTransition(prevLen, nextLen int) {
	if prevLen == nextLen {
		return
	}
	if hnd != nil && hnd.sess != nil {
		hnd.sess.requestTemplateReset()
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
