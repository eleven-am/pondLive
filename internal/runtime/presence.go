package runtime

import (
	"sort"
	"strconv"
	"sync"
	"time"
)

type PresenceInput[T any] struct {
	Items    []T
	KeyFunc  func(T) string
	Duration time.Duration
}

type PresenceResult[T any] struct {
	Items []PresenceItem[T]
}

type PresenceItem[T any] struct {
	Value   T
	key     string
	exiting bool
}

func (p PresenceItem[T]) Visible() bool {
	return true
}

func (p PresenceItem[T]) IsExiting() bool {
	return p.exiting
}

func (p PresenceItem[T]) Key() string {
	return p.key
}

func (r PresenceResult[T]) Visible() bool {
	return len(r.Items) > 0
}

func (r PresenceResult[T]) IsExiting() bool {
	return len(r.Items) > 0 && r.Items[0].exiting
}

func Present[T any](value T, dur time.Duration) PresenceInput[T] {
	return PresenceInput[T]{
		Items:    []T{value},
		Duration: dur,
		KeyFunc:  func(_ T) string { return "_single" },
	}
}

func PresentIf[T any](condition bool, value T, dur time.Duration) PresenceInput[T] {
	if condition {
		return PresenceInput[T]{
			Items:    []T{value},
			Duration: dur,
			KeyFunc:  func(_ T) string { return "_single" },
		}
	}
	return PresenceInput[T]{
		Items:    nil,
		Duration: dur,
		KeyFunc:  func(_ T) string { return "_single" },
	}
}

func PresentList[T any](items []T, keyFn func(T) string, dur time.Duration) PresenceInput[T] {
	return PresenceInput[T]{
		Items:    items,
		KeyFunc:  keyFn,
		Duration: dur,
	}
}

type trackedEntry[T any] struct {
	value     T
	exiting   bool
	dropAt    time.Time
	lastIndex int
}

type presenceCell[T any] struct {
	tracked map[string]*trackedEntry[T]
	timer   *time.Timer
	owner   *Instance
	session *Session
	mu      sync.Mutex
}

func UsePresence[T any](ctx *Ctx, in PresenceInput[T]) PresenceResult[T] {
	idx := ctx.hookIndex
	ctx.hookIndex++

	if idx >= len(ctx.instance.HookFrame) {
		cell := &presenceCell[T]{
			tracked: make(map[string]*trackedEntry[T]),
			owner:   ctx.instance,
			session: ctx.session,
		}
		ctx.instance.HookFrame = append(ctx.instance.HookFrame, HookSlot{
			Type:  HookTypePresence,
			Value: cell,
		})

		ctx.instance.RegisterCleanup(func() {
			cell.mu.Lock()
			if cell.timer != nil {
				cell.timer.Stop()
			}
			cell.mu.Unlock()
		})
	}

	cell, ok := ctx.instance.HookFrame[idx].Value.(*presenceCell[T])
	if !ok {
		panic("runtime: UsePresence hook mismatch")
	}

	return cell.reconcile(in)
}

func (c *presenceCell[T]) reconcile(in PresenceInput[T]) PresenceResult[T] {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	incoming := make(map[string]int)

	for i, item := range in.Items {
		key := c.getKey(item, i, in.KeyFunc)
		if _, dup := incoming[key]; dup {
			panic("runtime: UsePresence duplicate key: " + key)
		}
		incoming[key] = i

		if entry, exists := c.tracked[key]; exists {
			entry.value = item
			entry.exiting = false
			entry.dropAt = time.Time{}
		} else {
			c.tracked[key] = &trackedEntry[T]{
				value:     item,
				exiting:   false,
				lastIndex: -1,
			}
		}
	}

	for key, entry := range c.tracked {
		if _, present := incoming[key]; !present {
			if !entry.exiting {
				entry.exiting = true
				if in.Duration > 0 {
					entry.dropAt = now.Add(in.Duration)
				} else {
					delete(c.tracked, key)
				}
			} else if in.Duration == 0 {
				delete(c.tracked, key)
			} else if !entry.dropAt.IsZero() {
				if now.After(entry.dropAt) {
					delete(c.tracked, key)
				} else {
					newDropAt := now.Add(in.Duration)
					if newDropAt.Before(entry.dropAt) {
						entry.dropAt = newDropAt
					}
				}
			}
		}
	}

	c.assignStableIndices(incoming)
	c.scheduleTimer(now)

	return c.buildResult()
}

func (c *presenceCell[T]) assignStableIndices(incoming map[string]int) {
	type keyIndex struct {
		key   string
		index int
	}

	var activeKeys []keyIndex
	var exitingKeys []keyIndex

	for key, entry := range c.tracked {
		if idx, present := incoming[key]; present {
			activeKeys = append(activeKeys, keyIndex{key: key, index: idx})
		} else {
			exitingKeys = append(exitingKeys, keyIndex{key: key, index: entry.lastIndex})
		}
	}

	sort.Slice(activeKeys, func(i, j int) bool {
		return activeKeys[i].index < activeKeys[j].index
	})

	sort.Slice(exitingKeys, func(i, j int) bool {
		return exitingKeys[i].index < exitingKeys[j].index
	})

	merged := make([]string, 0, len(c.tracked))
	ai, ei := 0, 0

	for ai < len(activeKeys) || ei < len(exitingKeys) {
		if ai >= len(activeKeys) {
			merged = append(merged, exitingKeys[ei].key)
			ei++
		} else if ei >= len(exitingKeys) {
			merged = append(merged, activeKeys[ai].key)
			ai++
		} else if exitingKeys[ei].index <= activeKeys[ai].index {
			merged = append(merged, exitingKeys[ei].key)
			ei++
		} else {
			merged = append(merged, activeKeys[ai].key)
			ai++
		}
	}

	for i, key := range merged {
		c.tracked[key].lastIndex = i
	}
}

func (c *presenceCell[T]) getKey(item T, index int, keyFn func(T) string) string {
	if keyFn != nil {
		return keyFn(item)
	}
	return strconv.Itoa(index)
}

func (c *presenceCell[T]) scheduleTimer(now time.Time) {
	if c.timer != nil {
		c.timer.Stop()
		c.timer = nil
	}

	var earliest time.Time
	for _, entry := range c.tracked {
		if entry.exiting && !entry.dropAt.IsZero() {
			if earliest.IsZero() || entry.dropAt.Before(earliest) {
				earliest = entry.dropAt
			}
		}
	}

	if earliest.IsZero() {
		return
	}

	delay := earliest.Sub(now)
	if delay < 0 {
		delay = 0
	}

	c.timer = time.AfterFunc(delay, func() {
		c.mu.Lock()
		session := c.session
		owner := c.owner
		c.mu.Unlock()

		if session != nil && owner != nil {
			session.MarkDirty(owner)
		}
	})
}

func (c *presenceCell[T]) buildResult() PresenceResult[T] {
	type sortableEntry struct {
		key   string
		entry *trackedEntry[T]
	}

	entries := make([]sortableEntry, 0, len(c.tracked))
	for key, entry := range c.tracked {
		entries = append(entries, sortableEntry{key: key, entry: entry})
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].entry.lastIndex < entries[j].entry.lastIndex
	})

	items := make([]PresenceItem[T], len(entries))
	for i, e := range entries {
		items[i] = PresenceItem[T]{
			Value:   e.entry.value,
			key:     e.key,
			exiting: e.entry.exiting,
		}
	}

	return PresenceResult[T]{Items: items}
}
