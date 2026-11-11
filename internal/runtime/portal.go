package runtime

import (
	"github.com/eleven-am/pondlive/internal/reactive"
	"github.com/eleven-am/pondlive/internal/work"
)

type portalEntry struct {
	creator  *reactive.Owner
	children []work.Node
}

type portalHost struct {
	s       *Session
	us      *unitSet[uint32]
	entries map[uint32]*portalEntry
}

type portalTargetDef struct{}

type DuplicatePortalTargetError struct{}

func (DuplicatePortalTargetError) Error() string {
	return "runtime: a session can render only one portal target"
}

func (portalTargetDef) kind() regionKind { return regionPortal }

func (portalTargetDef) mount(e *emitter, sc *scope, id uint32, force bool) {
	s := e.s
	if s.portals != nil && !s.portals.us.dead {
		panic(DuplicatePortalTargetError{})
	}
	us := newUnitSet[uint32](s, sc.o, regionPortal, id, 0)
	host := &portalHost{s: s, us: us, entries: make(map[uint32]*portalEntry)}
	us.run = func(_ *Ctx, r *unitRow[uint32]) work.Node {
		entry := host.entries[r.key]
		if entry == nil {
			panic(determinism(us.owner, "portal entry is missing"))
		}
		return &work.Fragment{Children: entry.children}
	}
	us.parentFor = func(r *unitRow[uint32]) *reactive.Owner {
		if entry := host.entries[r.key]; entry != nil {
			return entry.creator
		}
		return us.owner
	}
	s.portals = host
	sc.o.OnCleanup(func() {
		if s.portals == host {
			s.portals = nil
		}
	})
	us.open(e, force)
	sc.o.Regions = append(sc.o.Regions, us)
	us.closeRegion(e)
}

func (e *emitter) portalTarget(sc *scope, force bool) {
	e.region(sc, &work.RegionNode{Impl: portalTargetDef{}}, force)
}

func (e *emitter) portal(sc *scope, x *work.PortalNode) {
	if e.ser || len(x.Children) == 0 {
		return
	}
	s := e.s
	creator := sc.o
	children := x.Children
	s.portalSeq++
	key := s.portalSeq
	creator.OnCleanup(func() { s.removePortalEntry(key) })
	s.arming = append(s.arming, func() { s.addPortalEntry(creator, key, children) })
}

func (s *Session) removePortalEntry(key uint32) {
	host := s.portals
	if host == nil || host.us.dead {
		return
	}
	row := host.us.byKey[key]
	if row == nil {
		return
	}
	host.us.remove(row)
	for i, x := range host.us.rows {
		if x == row {
			host.us.rows = append(host.us.rows[:i], host.us.rows[i+1:]...)
			break
		}
	}
	delete(host.entries, key)
}

func (s *Session) addPortalEntry(creator *reactive.Owner, key uint32, children []work.Node) {
	host := s.portals
	if host == nil {
		s.Report(nil, Report{Severity: SeverityDiagnostic, Phase: "portal:no-target",
			Message: "a portal was rendered but the page has no portal target",
		})
		return
	}
	if creator.Disposed() {
		return
	}
	host.entries[key] = &portalEntry{creator: creator, children: children}
	row := &unitRow[uint32]{key: key}
	us := host.us
	us.byKey[key] = row
	us.insert(row, 0)
	us.rows = append(us.rows, row)
}
