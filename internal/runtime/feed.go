package runtime

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/eleven-am/pondlive/internal/protocol"
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type Feed[T any, K comparable] struct {
	key  func(T) K
	load func() []T
	reg  *feedRegion[T, K]
}

func NewFeed[T any, K comparable](key func(T) K, load func() []T) *Feed[T, K] {
	return &Feed[T, K]{key: key, load: load}
}

type feedRegion[T any, K comparable] struct {
	*unitSet[K]
	feed   *Feed[T, K]
	row    func(reactive.Readable[T]) work.Node
	cache  map[K]T
	pushed map[K]struct{}
}

type feedHandle interface {
	resync()
	ensureCache() bool
	dropCache()
	alive() bool
}

type feedDef[T any, K comparable] struct {
	feed *Feed[T, K]
	row  func(reactive.Readable[T]) work.Node
	exit time.Duration
}

func (f *Feed[T, K]) newDef(row func(reactive.Readable[T]) work.Node, exit time.Duration) regionDef {
	return &feedDef[T, K]{feed: f, row: row, exit: exit}
}

func (d *feedDef[T, K]) kind() regionKind { return regionFeed }

type FeedMountedTwiceError struct{}

func (FeedMountedTwiceError) Error() string { return "runtime: a Feed is mounted by two live regions" }

func (FeedMountedTwiceError) earlyRegionError() {}

func (d *feedDef[T, K]) mount(e *emitter, sc *scope, id uint32, force bool) {
	s := e.s
	f := d.feed
	if f.reg != nil && f.reg.alive() {
		panic(FeedMountedTwiceError{})
	}
	r := &feedRegion[T, K]{unitSet: newUnitSet[K](s, sc.o, regionFeed, id, d.exit), feed: f, row: d.row, cache: map[K]T{}}
	f.reg = r
	us := r.unitSet
	us.dropper = true
	if s.devMode {
		r.pushed = make(map[K]struct{})
		us.forget = func(k K) { delete(r.pushed, k) }
	}
	us.run = func(ctx *Ctx, row *unitRow[K]) work.Node {
		item, ok := r.cache[row.key]
		if !ok {
			panic(determinism(us.owner, fmt.Sprintf("feed item for key %v is not available", row.key)))
		}
		return inlineComponent(ctx, r.row(Const(item)))
	}
	us.open(e, force)
	us.initial = e.w
	sc.o.Regions = append(sc.o.Regions, r)
	s.feeds = append(s.feeds, r)
	us.spawn(func() {
		r.fill(f.load(), false)
	})
	us.initial = nil
	us.closeRegion(e)
}

func (r *feedRegion[T, K]) alive() bool { return !r.dead }

func (r *feedRegion[T, K]) fill(items []T, keepCache bool) {
	us := r.unitSet
	keys := make([]K, 0, len(items))
	cache := make(map[K]T, len(items))
	for _, it := range items {
		k := r.feed.key(it)
		if _, dup := cache[k]; dup {
			r.duplicate(k)
			continue
		}
		cache[k] = it
		keys = append(keys, k)
	}
	r.cache = cache
	us.reconcile(keys)
	if !keepCache && !r.s.holdFeeds {
		r.cache = nil
	}
}

func (r *feedRegion[T, K]) duplicate(k K) {
	s := r.s
	if s.devMode {
		panic(&DuplicateKeyError{Key: fmt.Sprint(k)})
	}
	s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "feed:duplicate-key",
		Message: "duplicate feed key " + fmt.Sprint(k) + " ignored",
	})
}

func (r *feedRegion[T, K]) resync() {
	us := r.unitSet
	exit := us.exit
	us.exit = 0
	defer func() { us.exit = exit }()
	items := r.feed.load()
	r.checkPushed(items)
	r.s.graph.Untracked(func() { r.fill(items, true) })
}

type FeedLostRowsError struct {
	Keys []string
}

func (e *FeedLostRowsError) Error() string {
	return fmt.Sprintf("a Feed's load omitted %d row(s) that were pushed since the last load (keys %s); load is the source of truth for a reset and must return every current row, including the ones added with Append, Prepend, InsertBefore or Replace", len(e.Keys), strings.Join(e.Keys, ", "))
}

func (r *feedRegion[T, K]) checkPushed(items []T) {
	if len(r.pushed) == 0 {
		return
	}
	loaded := make(map[K]struct{}, len(items))
	for _, it := range items {
		loaded[r.feed.key(it)] = struct{}{}
	}
	var lost []string
	for k := range r.pushed {
		row := r.byKey[k]
		if _, ok := loaded[k]; ok || row == nil || row.exiting {
			continue
		}
		lost = append(lost, fmt.Sprint(k))
	}
	clear(r.pushed)
	if len(lost) == 0 {
		return
	}
	slices.Sort(lost)
	panic(determinism(r.owner, (&FeedLostRowsError{Keys: lost}).Error()))
}

func (r *feedRegion[T, K]) ensureCache() bool {
	if r.cache != nil {
		return false
	}
	r.cache = make(map[K]T)
	for _, it := range r.feed.load() {
		k := r.feed.key(it)
		if _, dup := r.cache[k]; !dup {
			r.cache[k] = it
		}
	}
	return true
}

func (r *feedRegion[T, K]) dropCache() { r.cache = nil }

func (s *Session) liveFeeds() []feedHandle {
	kept := s.feeds[:0]
	for _, f := range s.feeds {
		if f.alive() {
			kept = append(kept, f)
		}
	}
	clear(s.feeds[len(kept):])
	s.feeds = kept
	return kept
}

func (s *Session) resyncFeeds() error {
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = s.nondeterministic(r)
			}
		}()
		for _, f := range s.liveFeeds() {
			f.resync()
		}
	}()
	return err
}

func (s *Session) dropFeedCaches() {
	for _, f := range s.liveFeeds() {
		f.dropCache()
	}
}

func (s *Session) ensureFeedCaches() (undo func()) {
	var made []feedHandle
	for _, f := range s.liveFeeds() {
		if f.ensureCache() {
			made = append(made, f)
		}
	}
	return func() {
		for _, f := range made {
			f.dropCache()
		}
	}
}

func (f *Feed[T, K]) op(name string, fn func(r *feedRegion[T, K])) {
	r := f.reg
	if r == nil || !r.alive() {
		return
	}
	s := r.s
	if s.graph.Mode() != reactive.ModeMount {
		if s.allowCreate("Feed." + name) {
			return
		}
		return
	}
	run := func() {
		if !r.alive() {
			return
		}
		fn(r)
	}
	if s.graph.OnActor() {
		run()
		return
	}
	_ = s.Post(run)
}

func (r *feedRegion[T, K]) mountItem(it T, ref uint32) *unitRow[K] {
	us := r.unitSet
	k := r.feed.key(it)
	if _, exists := us.byKey[k]; exists {
		r.duplicate(k)
		return nil
	}
	row := &unitRow[K]{key: k}
	us.byKey[k] = row
	if r.pushed != nil {
		r.pushed[k] = struct{}{}
	}
	if r.cache == nil {
		r.cache = make(map[K]T, 1)
		defer func() { r.cache = nil }()
	}
	r.cache[k] = it
	us.insert(row, ref)
	return row
}

func (f *Feed[T, K]) Append(items ...T) {
	f.op("Append", func(r *feedRegion[T, K]) {
		for _, it := range items {
			if row := r.mountItem(it, 0); row != nil {
				r.rows = append(r.rows, row)
			}
		}
	})
}

func (f *Feed[T, K]) Prepend(items ...T) {
	f.op("Prepend", func(r *feedRegion[T, K]) {
		ref := uint32(0)
		if len(r.rows) > 0 {
			ref = r.rows[0].id
		}
		fresh := make([]*unitRow[K], 0, len(items))
		for _, it := range items {
			if row := r.mountItem(it, ref); row != nil {
				fresh = append(fresh, row)
			}
		}
		r.rows = append(fresh, r.rows...)
	})
}

func (f *Feed[T, K]) InsertBefore(k K, items ...T) {
	f.op("InsertBefore", func(r *feedRegion[T, K]) {
		at := r.indexOf(k)
		if at < 0 {
			r.missing("InsertBefore", k)
			return
		}
		ref := r.rows[at].id
		fresh := make([]*unitRow[K], 0, len(items))
		for _, it := range items {
			if row := r.mountItem(it, ref); row != nil {
				fresh = append(fresh, row)
			}
		}
		rows := make([]*unitRow[K], 0, len(r.rows)+len(fresh))
		rows = append(rows, r.rows[:at]...)
		rows = append(rows, fresh...)
		r.rows = append(rows, r.rows[at:]...)
	})
}

func (f *Feed[T, K]) Replace(item T) {
	f.op("Replace", func(r *feedRegion[T, K]) {
		us := r.unitSet
		k := f.key(item)
		at := r.indexOf(k)
		if at < 0 {
			r.missing("Replace", k)
			return
		}
		old := r.rows[at]
		delete(us.byKey, k)
		row := r.mountItem(item, old.id)
		if row == nil {
			us.byKey[k] = old
			return
		}
		us.s.edit(protocol.RemoveEdit(old.id))
		us.dispose(old)
		r.rows[at] = row
	})
}

func (f *Feed[T, K]) Delete(k K) {
	f.op("Delete", func(r *feedRegion[T, K]) {
		us := r.unitSet
		at := r.indexOf(k)
		if at < 0 {
			return
		}
		row := r.rows[at]
		if us.exit > 0 {
			if !row.exiting {
				us.beginExit(row)
				us.schedule()
			}
			return
		}
		us.remove(row)
		r.rows = append(r.rows[:at], r.rows[at+1:]...)
	})
}

func (f *Feed[T, K]) Reload() {
	f.op("Reload", func(r *feedRegion[T, K]) {
		us := r.unitSet
		clear(r.pushed)
		if len(us.rows) > 0 {
			us.clearAll()
		}
		items := f.load()
		r.cache = make(map[K]T, len(items))
		defer func() { r.cache = nil }()
		for _, it := range items {
			k := f.key(it)
			if _, dup := us.byKey[k]; dup {
				r.duplicate(k)
				continue
			}
			row := &unitRow[K]{key: k}
			us.byKey[k] = row
			r.cache[k] = it
			us.insert(row, 0)
			us.rows = append(us.rows, row)
		}
	})
}

func (r *feedRegion[T, K]) indexOf(k K) int {
	row := r.byKey[k]
	if row == nil {
		return -1
	}
	for i, x := range r.rows {
		if x == row {
			return i
		}
	}
	return -1
}

func (r *feedRegion[T, K]) missing(op string, k K) {
	if r.s.devMode {
		panic(fmt.Errorf("runtime: Feed.%s: no row with key %v", op, k))
	}
	r.s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "feed:missing-key",
		Message: fmt.Sprintf("Feed.%s ignored: no row with key %v", op, k),
	})
}
