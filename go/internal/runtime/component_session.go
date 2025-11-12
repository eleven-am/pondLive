package runtime

import (
	"errors"
	"fmt"
	"runtime/debug"
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/eleven-am/pondlive/go/internal/diff"
	"github.com/eleven-am/pondlive/go/internal/dom"
	handlers "github.com/eleven-am/pondlive/go/internal/handlers"
	"github.com/eleven-am/pondlive/go/internal/protocol"
	render "github.com/eleven-am/pondlive/go/internal/render"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// ComponentSession drives component rendering, diffing, and event handling for a live UI connection.
type ComponentSession struct {
	owner        *LiveSession
	root         *component
	rootCallable componentCallable
	rootProps    any
	registry     handlers.Registry
	sendPatch    func([]diff.Op) error

	nextRefID   int
	elementRefs map[string]trackedElementRef
	pendingRefs map[string]protocol.RefMeta
	lastRefs    map[string]protocol.RefMeta

	prev render.Structured

	dirty          map[*component]struct{}
	dirtyRoot      bool
	pendingFlush   bool
	suspend        int
	flushing       bool
	forceTemplate  atomic.Bool
	templateUpdate atomic.Pointer[templateUpdate]
	router         atomic.Pointer[routerSessionState]

	uploads           map[string]*uploadSlot
	uploadByComponent map[*component]map[int]*uploadSlot
	uploadSeq         int
	uploadMu          sync.Mutex

	pendingEffects  []effectTask
	pendingCleanups []cleanupTask
	pendingNav      *protocol.NavDelta
	pendingMetrics  *protocol.FrameMetrics

	pendingPubsub  []pubsubTask
	pubsubProvider PubsubProvider
	pubsubSubs     map[string]pubsubSubscription
	pubsubMu       sync.RWMutex

	reporter       DiagnosticReporter
	renderStack    []*component
	currentPhase   string
	errored        bool
	lastDiagnostic *Diagnostic

	meta            *Meta
	baseMeta        *Meta
	metaByComponent map[*component]*Meta
	metaOrder       []*component
	metaTouched     map[*component]bool
	metaMu          sync.RWMutex

	header   HeaderState
	headerMu sync.RWMutex

	componentsMu          sync.RWMutex
	components            map[string]*component
	componentBoots        map[string]*componentBootRequest
	pendingComponentBoots []componentTemplateUpdate

	promotions map[string]*componentPromotionState

	mu sync.Mutex
}

// ErrFlushInProgress indicates that Flush was invoked while another flush is already running.
var ErrFlushInProgress = errors.New("runtime: flush in progress")

type templateUpdate struct {
	structured render.Structured
	html       string
}

type componentBootRequest struct {
	component *component
}

type spanRange struct {
	start int
	end   int
}

type componentTemplateUpdate struct {
	id             string
	html           string
	staticsRange   spanRange
	statics        []string
	dynamicsRange  spanRange
	dynamics       []protocol.DynamicSlot
	slots          []int
	listSlots      []int
	slotPaths      []protocol.SlotPath
	listPaths      []protocol.ListPath
	componentPaths []protocol.ComponentPath
	handlersAdd    map[string]protocol.HandlerMeta
	handlersDel    []string
	bindings       protocol.TemplateBindings
}

type componentPromotionState struct {
	texts map[string]*textPromotionSlot
	attrs map[string]*attrPromotionSlot
}

type textPromotionSlot struct {
	value   string
	dynamic bool
}

type attrPromotionSlot struct {
	values  map[string]string
	dynamic bool
}

type componentSpanPair struct {
	prev render.ComponentSpan
	next render.ComponentSpan
}

type spanRangePair struct {
	prevStart int
	prevEnd   int
	nextStart int
	nextEnd   int
}

type pubsubTask struct {
	run func()
}

type pubsubSubscription struct {
	token    string
	topic    string
	handler  func([]byte, map[string]string)
	provider PubsubProvider
}

// DiagnosticReporter receives structured diagnostics captured during panic recovery.
type DiagnosticReporter interface {
	ReportDiagnostic(Diagnostic)
}

// NewSession constructs a session rooted at the provided component function.
func NewSession[P any](root Component[P], props P) *ComponentSession {
	sess := &ComponentSession{
		dirty: make(map[*component]struct{}),
	}
	sess.root = newComponent(sess, nil, "root", root, props)
	if sess.root != nil {
		sess.rootCallable = sess.root.callable
		sess.rootProps = props
	}
	return sess
}

// Registry exposes the handler registry, creating one if necessary.
func (s *ComponentSession) Registry() handlers.Registry {
	return s.ensureRegistry()
}

// SetRegistry injects a custom registry implementation.
func (s *ComponentSession) SetRegistry(reg handlers.Registry) { s.registry = reg }

// SetPatchSender installs the transport used to deliver diff operations.
func (s *ComponentSession) SetPatchSender(fn func([]diff.Op) error) { s.sendPatch = fn }

func (s *ComponentSession) setOwner(owner *LiveSession) { s.owner = owner }

func (s *ComponentSession) allocateElementRefID() string {
	if s == nil {
		return ""
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	id := fmt.Sprintf("ref:%d", s.nextRefID)
	s.nextRefID++
	return id
}

type trackedElementRef interface {
	ID() string
	DescriptorTag() string
	BindingSnapshot() map[string]dom.EventBinding
}

func (s *ComponentSession) registerElementRef(ref trackedElementRef) {
	if s == nil || ref == nil {
		return
	}
	s.mu.Lock()
	if s.elementRefs == nil {
		s.elementRefs = make(map[string]trackedElementRef)
	}
	s.elementRefs[ref.ID()] = ref
	s.mu.Unlock()
}

func (s *ComponentSession) snapshotRefs(ids []string) map[string]protocol.RefMeta {
	if s == nil || len(ids) == 0 {
		return nil
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.snapshotRefsLocked(ids)
}

func (s *ComponentSession) snapshotRefsLocked(ids []string) map[string]protocol.RefMeta {
	if len(ids) == 0 || len(s.elementRefs) == 0 {
		return nil
	}
	out := make(map[string]protocol.RefMeta)
	for _, id := range ids {
		ref, ok := s.elementRefs[id]
		if !ok || ref == nil {
			continue
		}
		meta := protocol.RefMeta{Tag: ref.DescriptorTag()}
		bindings := ref.BindingSnapshot()
		if len(bindings) > 0 {
			events := make(map[string]protocol.RefEventMeta, len(bindings))
			for event, binding := range bindings {
				eventMeta := protocol.RefEventMeta{}
				if len(binding.Listen) > 0 {
					eventMeta.Listen = append([]string(nil), binding.Listen...)
				}
				if len(binding.Props) > 0 {
					eventMeta.Props = append([]string(nil), binding.Props...)
				}
				events[event] = eventMeta
			}
			meta.Events = events
		}
		out[id] = meta
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func cloneRefMetaMap(src map[string]protocol.RefMeta) map[string]protocol.RefMeta {
	if len(src) == 0 {
		return nil
	}
	out := make(map[string]protocol.RefMeta, len(src))
	for id, meta := range src {
		clone := protocol.RefMeta{Tag: meta.Tag}
		if len(meta.Events) > 0 {
			events := make(map[string]protocol.RefEventMeta, len(meta.Events))
			for name, event := range meta.Events {
				events[name] = protocol.RefEventMeta{
					Listen: append([]string(nil), event.Listen...),
					Props:  append([]string(nil), event.Props...),
				}
			}
			clone.Events = events
		}
		out[id] = clone
	}
	return out
}

func refMetaMapsEqual(a, b map[string]protocol.RefMeta) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	for id, metaA := range a {
		metaB, ok := b[id]
		if !ok {
			return false
		}
		if metaA.Tag != metaB.Tag {
			return false
		}
		if !refEventMapsEqual(metaA.Events, metaB.Events) {
			return false
		}
	}
	return true
}

func refEventMapsEqual(a, b map[string]protocol.RefEventMeta) bool {
	if len(a) != len(b) {
		return false
	}
	for name, metaA := range a {
		metaB, ok := b[name]
		if !ok {
			return false
		}
		if !stringSliceEqual(metaA.Listen, metaB.Listen) {
			return false
		}
		if !stringSliceEqual(metaA.Props, metaB.Props) {
			return false
		}
	}
	return true
}

func stringSliceEqual(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	acopy := append([]string(nil), a...)
	bcopy := append([]string(nil), b...)
	sort.Strings(acopy)
	sort.Strings(bcopy)
	for i := range acopy {
		if acopy[i] != bcopy[i] {
			return false
		}
	}
	return true
}

func shouldForceDynamicAttr(mutable map[string]bool, attrs map[string]string) bool {
	if len(mutable) == 0 {
		return false
	}
	if mutable["*"] {
		return true
	}
	if len(attrs) == 0 {
		return false
	}
	for key := range attrs {
		if mutable[key] {
			return true
		}
	}
	return false
}

func attrMapsEqual(a, b map[string]string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 {
		return true
	}
	for k, v := range a {
		if b == nil {
			return false
		}
		if bv, ok := b[k]; !ok || bv != v {
			return false
		}
	}
	return true
}

// ResolveTextPromotion implements render.PromotionTracker to decide whether a text node should be dynamic.
func (s *ComponentSession) ResolveTextPromotion(componentID string, path []int, value string, mutable bool) bool {
	if s == nil {
		return mutable
	}
	if mutable {
		if componentID == "" {
			return true
		}
		state := s.ensurePromotionState(componentID)
		if state == nil {
			return true
		}
		key := promotionPathKey(path)
		slot, _ := state.ensureTextSlot(key)
		slot.dynamic = true
		slot.value = value
		return true
	}
	if componentID == "" {
		return false
	}
	state := s.ensurePromotionState(componentID)
	if state == nil {
		return false
	}
	key := promotionPathKey(path)
	slot, created := state.ensureTextSlot(key)
	if created {
		slot.value = value
		return false
	}
	if slot.dynamic {
		slot.value = value
		return true
	}
	if slot.value == value {
		return false
	}
	slot.dynamic = true
	slot.value = value
	s.requestComponentBootInternal(componentID)
	return true
}

// ResolveAttrPromotion implements render.PromotionTracker to decide whether an element's attributes should be dynamic.
func (s *ComponentSession) ResolveAttrPromotion(componentID string, path []int, attrs map[string]string, mutable map[string]bool) bool {
	forceDynamic := shouldForceDynamicAttr(mutable, attrs)
	if s == nil {
		return forceDynamic
	}
	if componentID == "" {
		return forceDynamic
	}
	state := s.ensurePromotionState(componentID)
	if state == nil {
		return forceDynamic
	}
	key := promotionPathKey(path)
	slot, created := state.ensureAttrSlot(key)
	values := cloneStringMap(attrs)
	if forceDynamic {
		slot.dynamic = true
		slot.values = values
		return true
	}
	if created {
		slot.values = values
		return false
	}
	if slot.dynamic {
		slot.values = values
		return true
	}
	if attrMapsEqual(slot.values, attrs) {
		return false
	}
	slot.dynamic = true
	slot.values = values
	s.requestComponentBootInternal(componentID)
	return true
}

func (s *ComponentSession) ensurePromotionState(componentID string) *componentPromotionState {
	if componentID == "" {
		return nil
	}
	if s.promotions == nil {
		s.promotions = make(map[string]*componentPromotionState)
	}
	state, ok := s.promotions[componentID]
	if !ok {
		state = &componentPromotionState{}
		s.promotions[componentID] = state
	}
	return state
}

func (s *ComponentSession) clearPromotionState(componentID string) {
	if s == nil || componentID == "" {
		return
	}
	if s.promotions != nil {
		delete(s.promotions, componentID)
	}
}

func promotionPathKey(path []int) string {
	if len(path) == 0 {
		return ""
	}
	var b strings.Builder
	for i, idx := range path {
		if i > 0 {
			b.WriteByte('/')
		}
		b.WriteString(strconv.Itoa(idx))
	}
	return b.String()
}

func (s *componentPromotionState) ensureTextSlot(key string) (*textPromotionSlot, bool) {
	if s == nil {
		return nil, false
	}
	if s.texts == nil {
		s.texts = make(map[string]*textPromotionSlot)
	}
	slot, ok := s.texts[key]
	if !ok {
		slot = &textPromotionSlot{}
		s.texts[key] = slot
		return slot, true
	}
	return slot, false
}

func (s *componentPromotionState) ensureAttrSlot(key string) (*attrPromotionSlot, bool) {
	if s == nil {
		return nil, false
	}
	if s.attrs == nil {
		s.attrs = make(map[string]*attrPromotionSlot)
	}
	slot, ok := s.attrs[key]
	if !ok {
		slot = &attrPromotionSlot{}
		s.attrs[key] = slot
		return slot, true
	}
	return slot, false
}

func alignStructured(prev, next render.Structured, spans []componentSpanPair) render.Structured {
	if len(spans) == 0 {
		return render.Structured{}
	}
	aligned := render.Structured{}
	staticsPairs := make([]spanRangePair, 0, len(spans))
	dynamicsPairs := make([]spanRangePair, 0, len(spans))
	for _, pair := range spans {
		staticsPairs = append(staticsPairs, spanRangePair{
			prevStart: pair.prev.StaticsStart,
			prevEnd:   pair.prev.StaticsEnd,
			nextStart: pair.next.StaticsStart,
			nextEnd:   pair.next.StaticsEnd,
		})
		dynamicsPairs = append(dynamicsPairs, spanRangePair{
			prevStart: pair.prev.DynamicsStart,
			prevEnd:   pair.prev.DynamicsEnd,
			nextStart: pair.next.DynamicsStart,
			nextEnd:   pair.next.DynamicsEnd,
		})
	}
	sort.Slice(staticsPairs, func(i, j int) bool { return staticsPairs[i].nextStart < staticsPairs[j].nextStart })
	sort.Slice(dynamicsPairs, func(i, j int) bool { return dynamicsPairs[i].nextStart < dynamicsPairs[j].nextStart })
	if len(next.S) > 0 {
		aligned.S = alignStrings(prev.S, next.S, staticsPairs)
	}
	if len(next.D) > 0 {
		aligned.D = alignDynamics(prev.D, next.D, dynamicsPairs)
	}
	return aligned
}

func alignStrings(prev, next []string, pairs []spanRangePair) []string {
	if len(next) == 0 {
		return nil
	}
	aligned := make([]string, len(next))
	delta := 0
	pairIdx := 0
	for nextIdx := 0; nextIdx < len(next); nextIdx++ {
		for pairIdx < len(pairs) && nextIdx >= pairs[pairIdx].nextEnd {
			delta += (pairs[pairIdx].nextEnd - pairs[pairIdx].nextStart) - (pairs[pairIdx].prevEnd - pairs[pairIdx].prevStart)
			pairIdx++
		}
		if pairIdx < len(pairs) && nextIdx >= pairs[pairIdx].nextStart && nextIdx < pairs[pairIdx].nextEnd {
			offset := nextIdx - pairs[pairIdx].nextStart
			prevLen := pairs[pairIdx].prevEnd - pairs[pairIdx].prevStart
			if offset < prevLen {
				idx := pairs[pairIdx].prevStart + offset
				if idx >= 0 && idx < len(prev) {
					aligned[nextIdx] = prev[idx]
					continue
				}
			}
			aligned[nextIdx] = next[nextIdx]
			continue
		}
		prevIdx := nextIdx - delta
		if prevIdx >= 0 && prevIdx < len(prev) {
			aligned[nextIdx] = prev[prevIdx]
		} else {
			aligned[nextIdx] = next[nextIdx]
		}
	}
	return aligned
}

func alignDynamics(prev, next []render.Dyn, pairs []spanRangePair) []render.Dyn {
	if len(next) == 0 {
		return nil
	}
	aligned := make([]render.Dyn, len(next))
	delta := 0
	pairIdx := 0
	for nextIdx := 0; nextIdx < len(next); nextIdx++ {
		for pairIdx < len(pairs) && nextIdx >= pairs[pairIdx].nextEnd {
			delta += (pairs[pairIdx].nextEnd - pairs[pairIdx].nextStart) - (pairs[pairIdx].prevEnd - pairs[pairIdx].prevStart)
			pairIdx++
		}
		if pairIdx < len(pairs) && nextIdx >= pairs[pairIdx].nextStart && nextIdx < pairs[pairIdx].nextEnd {
			offset := nextIdx - pairs[pairIdx].nextStart
			prevLen := pairs[pairIdx].prevEnd - pairs[pairIdx].prevStart
			if offset < prevLen {
				idx := pairs[pairIdx].prevStart + offset
				if idx >= 0 && idx < len(prev) {
					aligned[nextIdx] = prev[idx]
					continue
				}
			}
			aligned[nextIdx] = next[nextIdx]
			continue
		}
		prevIdx := nextIdx - delta
		if prevIdx >= 0 && prevIdx < len(prev) {
			aligned[nextIdx] = prev[prevIdx]
		} else {
			aligned[nextIdx] = next[nextIdx]
		}
	}
	return aligned
}

func (s *ComponentSession) requestTemplateReset() {
	if s == nil {
		return
	}
	s.forceTemplate.Store(true)
}

func (s *ComponentSession) registerComponentInstance(c *component) {
	if s == nil || c == nil || c.id == "" {
		return
	}
	s.componentsMu.Lock()
	if s.components == nil {
		s.components = make(map[string]*component)
	}
	s.components[c.id] = c
	s.componentsMu.Unlock()
}

func (s *ComponentSession) unregisterComponentInstance(c *component) {
	if s == nil || c == nil || c.id == "" {
		return
	}
	s.componentsMu.Lock()
	if s.components != nil {
		delete(s.components, c.id)
	}
	if s.componentBoots != nil {
		delete(s.componentBoots, c.id)
	}
	s.componentsMu.Unlock()
	s.clearPromotionState(c.id)
}

func (s *ComponentSession) componentByID(id string) *component {
	if s == nil || id == "" {
		return nil
	}
	s.componentsMu.RLock()
	comp := s.components[id]
	s.componentsMu.RUnlock()
	if comp != nil {
		return comp
	}
	return s.componentByIDSlow(id)
}

func (s *ComponentSession) componentByIDSlow(id string) *component {
	if s == nil || id == "" {
		return nil
	}
	root := s.root
	if root == nil {
		return nil
	}
	if found := findComponentByID(root, id); found != nil {
		s.componentsMu.Lock()
		if s.components == nil {
			s.components = make(map[string]*component)
		}
		s.components[id] = found
		s.componentsMu.Unlock()
		return found
	}
	return nil
}

func findComponentByID(root *component, id string) *component {
	if root == nil {
		return nil
	}
	if root.id == id {
		return root
	}
	root.mu.Lock()
	children := make([]*component, 0, len(root.children))
	for _, child := range root.children {
		children = append(children, child)
	}
	root.mu.Unlock()
	for _, child := range children {
		if found := findComponentByID(child, id); found != nil {
			return found
		}
	}
	return nil
}

// RequestComponentBoot schedules a template refresh for the component identified by id.
func (s *ComponentSession) requestComponentBootInternal(id string) *component {
	if s == nil || id == "" {
		return nil
	}
	s.componentsMu.RLock()
	comp := s.components[id]
	s.componentsMu.RUnlock()
	if comp == nil {
		return nil
	}

	s.componentsMu.Lock()
	if s.componentBoots == nil {
		s.componentBoots = make(map[string]*componentBootRequest)
	}
	s.componentBoots[id] = &componentBootRequest{component: comp}
	s.componentsMu.Unlock()

	return comp
}

// RequestComponentBoot schedules a template refresh for the component identified by id.
func (s *ComponentSession) RequestComponentBoot(id string) {
	comp := s.requestComponentBootInternal(id)
	if comp == nil {
		return
	}

	if s.currentComponent() != nil {
		return
	}
	s.markDirty(comp)
}

func (s *ComponentSession) consumeTemplateReset() bool {
	if s == nil {
		return false
	}
	return s.forceTemplate.Swap(false)
}

func (s *ComponentSession) setTemplateUpdate(update templateUpdate) {
	if s == nil {
		return
	}
	copy := update
	s.templateUpdate.Store(&copy)
}

func (s *ComponentSession) consumeTemplateUpdate() *templateUpdate {
	if s == nil {
		return nil
	}
	return s.templateUpdate.Swap(nil)
}

func (s *ComponentSession) consumeComponentBoots() []componentTemplateUpdate {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	boots := s.pendingComponentBoots
	if len(boots) > 0 {
		copied := make([]componentTemplateUpdate, len(boots))
		copy(copied, boots)
		s.pendingComponentBoots = nil
		s.mu.Unlock()
		return copied
	}
	s.pendingComponentBoots = nil
	s.mu.Unlock()
	return nil
}

func (s *ComponentSession) consumeComponentBootRequests() map[string]*componentBootRequest {
	if s == nil {
		return nil
	}
	s.componentsMu.Lock()
	defer s.componentsMu.Unlock()
	if len(s.componentBoots) == 0 {
		return nil
	}
	requests := s.componentBoots
	s.componentBoots = nil
	return requests
}

func (s *ComponentSession) ensureRouterState() *routerSessionState {
	if s == nil {
		return nil
	}
	if state := s.router.Load(); state != nil {
		return state
	}
	created := &routerSessionState{}
	if s.router.CompareAndSwap(nil, created) {
		return created
	}
	return s.router.Load()
}

func (s *ComponentSession) loadRouterState() *routerSessionState {
	if s == nil {
		return nil
	}
	return s.router.Load()
}

func (s *ComponentSession) ensureRouterEntry() *sessionEntry {
	if s == nil {
		return nil
	}
	state := s.ensureRouterState()
	if state == nil {
		return nil
	}
	return &state.entry
}

func (s *ComponentSession) loadRouterEntry() *sessionEntry {
	if s == nil {
		return nil
	}
	state := s.loadRouterState()
	if state == nil {
		return nil
	}
	return &state.entry
}

func (s *ComponentSession) storeLinkPlaceholder(frag *h.FragmentNode, node *linkNode) {
	if s == nil || frag == nil || node == nil {
		return
	}
	if state := s.ensureRouterState(); state != nil {
		state.linkPlaceholders.Store(frag, node)
	}
}

func (s *ComponentSession) takeLinkPlaceholder(frag *h.FragmentNode) (*linkNode, bool) {
	if s == nil || frag == nil {
		return nil, false
	}
	if state := s.loadRouterState(); state != nil {
		if value, ok := state.linkPlaceholders.LoadAndDelete(frag); ok {
			if node, okCast := value.(*linkNode); okCast {
				return node, true
			}
		}
	}
	return nil, false
}

func (s *ComponentSession) clearLinkPlaceholder(frag *h.FragmentNode) {
	if s == nil || frag == nil {
		return
	}
	if state := s.loadRouterState(); state != nil {
		state.linkPlaceholders.Delete(frag)
	}
}

func (s *ComponentSession) storeRoutesPlaceholder(frag *h.FragmentNode, node *routesNode) {
	if s == nil || frag == nil || node == nil {
		return
	}
	if state := s.ensureRouterState(); state != nil {
		state.routesPlaceholders.Store(frag, node)
	}
}

func (s *ComponentSession) takeRoutesPlaceholder(frag *h.FragmentNode) (*routesNode, bool) {
	if s == nil || frag == nil {
		return nil, false
	}
	if state := s.loadRouterState(); state != nil {
		if value, ok := state.routesPlaceholders.LoadAndDelete(frag); ok {
			if node, okCast := value.(*routesNode); okCast {
				return node, true
			}
		}
	}
	return nil, false
}

func (s *ComponentSession) clearRoutesPlaceholder(frag *h.FragmentNode) {
	if s == nil || frag == nil {
		return
	}
	if state := s.loadRouterState(); state != nil {
		state.routesPlaceholders.Delete(frag)
	}
}

// SetPubsubProvider wires the session to an external pub/sub provider.
func (s *ComponentSession) SetPubsubProvider(provider PubsubProvider) {
	if s == nil {
		return
	}
	s.pubsubProvider = provider
}

// SetDiagnosticReporter installs a reporter notified when diagnostics are captured.
func (s *ComponentSession) SetDiagnosticReporter(r DiagnosticReporter) { s.reporter = r }

// SetMetadata records document metadata for the most recent render cycle.
func (s *ComponentSession) SetMetadata(meta *Meta) {
	if s == nil {
		return
	}
	if comp := s.currentComponent(); comp != nil {
		s.setComponentMetadata(comp, meta)
		return
	}
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	if meta == nil {
		s.baseMeta = nil
		s.meta = nil
		s.metaByComponent = nil
		s.metaOrder = nil
		s.metaTouched = nil
		return
	}
	s.baseMeta = CloneMeta(meta)
	s.rebuildAggregatedMetaLocked()
}

// Metadata returns a copy of the last metadata provided during rendering.
func (s *ComponentSession) Metadata() *Meta {
	if s == nil {
		return nil
	}
	s.metaMu.RLock()
	defer s.metaMu.RUnlock()
	return CloneMeta(s.meta)
}

func (s *ComponentSession) setComponentMetadata(comp *component, meta *Meta) {
	if s == nil || comp == nil {
		return
	}
	s.metaMu.Lock()
	defer s.metaMu.Unlock()
	if meta == nil {
		if s.metaByComponent != nil {
			s.metaByComponent[comp] = nil
			s.rebuildAggregatedMetaLocked()
		}
		s.markComponentMetadataTouchedLocked(comp, true)
		return
	}
	if s.metaByComponent == nil {
		s.metaByComponent = make(map[*component]*Meta)
	}
	existing := s.metaByComponent[comp]
	merged := MergeMeta(existing, meta)
	s.metaByComponent[comp] = merged
	if !s.componentInMetaOrderLocked(comp) {
		s.metaOrder = append(s.metaOrder, comp)
	}
	s.markComponentMetadataTouchedLocked(comp, true)
	s.rebuildAggregatedMetaLocked()
}

func (s *ComponentSession) beginComponentMetadata(comp *component) {
	if s == nil || comp == nil {
		return
	}
	s.metaMu.Lock()
	if s.metaByComponent == nil {
		s.metaByComponent = make(map[*component]*Meta)
	}
	if _, ok := s.metaByComponent[comp]; ok {
		s.metaByComponent[comp] = nil
	}
	s.rebuildAggregatedMetaLocked()
	s.markComponentMetadataTouchedLocked(comp, false)
	s.metaMu.Unlock()
}

func (s *ComponentSession) finishComponentMetadata(comp *component) {
	if s == nil || comp == nil {
		return
	}
	s.metaMu.Lock()
	if s.metaTouched != nil {
		if touched, ok := s.metaTouched[comp]; ok {
			if !touched && s.metaByComponent != nil {
				if _, exists := s.metaByComponent[comp]; exists {
					s.metaByComponent[comp] = nil
					s.rebuildAggregatedMetaLocked()
				}
			}
			delete(s.metaTouched, comp)
		}
	}
	s.metaMu.Unlock()
}

func (s *ComponentSession) markComponentMetadataTouchedLocked(comp *component, value bool) {
	if s.metaTouched == nil {
		s.metaTouched = make(map[*component]bool)
	}
	s.metaTouched[comp] = value
}

func (s *ComponentSession) componentInMetaOrderLocked(comp *component) bool {
	for _, existing := range s.metaOrder {
		if existing == comp {
			return true
		}
	}
	return false
}

func (s *ComponentSession) removeComponentFromMetaOrderLocked(comp *component) {
	if len(s.metaOrder) == 0 {
		return
	}
	for idx, existing := range s.metaOrder {
		if existing == comp {
			s.metaOrder = append(s.metaOrder[:idx], s.metaOrder[idx+1:]...)
			break
		}
	}
}

func (s *ComponentSession) rebuildAggregatedMetaLocked() {
	var merged *Meta
	if s.baseMeta != nil {
		merged = CloneMeta(s.baseMeta)
	}
	if len(s.metaOrder) > 0 && s.metaByComponent != nil {
		for _, comp := range s.metaOrder {
			if meta := s.metaByComponent[comp]; meta != nil {
				merged = MergeMeta(merged, meta)
			}
		}
	}
	s.meta = merged
}

func (s *ComponentSession) removeMetadataForComponent(comp *component) {
	if s == nil || comp == nil {
		return
	}
	s.metaMu.Lock()
	if s.metaByComponent != nil {
		delete(s.metaByComponent, comp)
	}
	s.removeComponentFromMetaOrderLocked(comp)
	if s.metaTouched != nil {
		delete(s.metaTouched, comp)
	}
	s.rebuildAggregatedMetaLocked()
	s.metaMu.Unlock()
}

func (s *ComponentSession) assignHeaderState(state HeaderState) {
	if s == nil {
		return
	}
	if state == nil {
		state = noopHeaderState{}
	}
	s.headerMu.Lock()
	s.header = state
	s.headerMu.Unlock()
}

func (s *ComponentSession) currentHeaderState() HeaderState {
	if s == nil {
		return noopHeaderState{}
	}
	s.headerMu.RLock()
	state := s.header
	s.headerMu.RUnlock()
	if state == nil {
		return noopHeaderState{}
	}
	return state
}

// HeaderState returns the currently active header state for the session.
func (s *ComponentSession) HeaderState() HeaderState {
	return s.currentHeaderState()
}

// InitialStructured performs an initial render and returns the structured result for SSR boot.
func (s *ComponentSession) InitialStructured() render.Structured {
	if s == nil || s.root == nil {
		return render.Structured{}
	}
	var (
		structured render.Structured
		cleanups   []cleanupTask
		effects    []effectTask
		pubsubs    []pubsubTask
	)
	if err := s.withRecovery("initial", func() error {
		reg := s.ensureRegistry()
		node := s.root.render()
		structured = render.ToStructuredWithOptions(node, render.StructuredOptions{Handlers: reg, Promotions: s})
		s.prev = structured
		s.dirtyRoot = false
		s.pendingFlush = false
		cleanups = append(cleanups, s.pendingCleanups...)
		effects = append(effects, s.pendingEffects...)
		pubsubs = append(pubsubs, s.pendingPubsub...)
		s.pendingEffects = nil
		s.pendingCleanups = nil
		s.pendingNav = nil
		s.pendingMetrics = nil
		s.pendingPubsub = nil
		s.pendingRefs = nil
		s.lastRefs = nil
		s.uploadMu.Lock()
		if s.uploads != nil {
			for _, slot := range s.uploads {
				if slot != nil {
					slot.sess = nil
				}
			}
		}
		s.uploads = nil
		s.uploadByComponent = nil
		s.uploadSeq = 0
		s.uploadMu.Unlock()
		if s.root != nil {
			s.root.markSelfDirty()
		}
		return nil
	}); err != nil {
		return render.Structured{}
	}
	runCleanups(cleanups)
	s.runPubsubTasks(pubsubs)
	runEffects(effects)
	ids := extractRefIDs(structured)
	s.mu.Lock()
	if len(ids) > 0 {
		refs := s.snapshotRefsLocked(ids)
		s.lastRefs = cloneRefMetaMap(refs)
	} else {
		s.lastRefs = nil
	}
	s.mu.Unlock()
	return structured
}

// RenderNode re-renders the root component and returns its HTML node tree.
func (s *ComponentSession) RenderNode() dom.Node {
	if s == nil || s.root == nil {
		return nil
	}
	return s.root.render()
}

// Flush applies pending state updates by rerendering and diffing the component tree.
func (s *ComponentSession) Flush() error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	if s.errored {
		if s.lastDiagnostic != nil {
			return DiagnosticError{diag: *s.lastDiagnostic}
		}
		return errors.New("runtime: session halted after panic")
	}
	s.mu.Lock()
	if s.flushing {
		s.mu.Unlock()
		return ErrFlushInProgress
	}
	s.flushing = true
	s.mu.Unlock()
	reg := s.ensureRegistry()
	var (
		cleanups         []cleanupTask
		effects          []effectTask
		pubsubs          []pubsubTask
		componentUpdates []componentTemplateUpdate
	)
	var (
		nextRefsClone map[string]protocol.RefMeta
		refsApplied   bool
	)
	err := s.withRecovery("flush", func() error {
		s.mu.Lock()
		locked := true
		defer func() {
			if locked {
				s.mu.Unlock()
			}
		}()

		if !s.dirtyRoot && !s.pendingFlush {
			return nil
		}
		if s.root == nil {
			return errors.New("runtime: session has no root component")
		}

		start := time.Now()
		root := s.root
		shouldMarkRoot := s.dirtyRoot && root != nil

		s.mu.Unlock()
		locked = false

		if shouldMarkRoot {
			root.markSelfDirty()
		}

		node := root.render()
		next := render.ToStructuredWithOptions(node, render.StructuredOptions{Handlers: reg, Promotions: s})

		s.mu.Lock()
		locked = true
		requests := s.consumeComponentBootRequests()
		autoRequests, rootChange := s.autoComponentBootRequests(s.prev, next)
		if len(autoRequests) > 0 {
			if requests == nil {
				requests = autoRequests
			} else {
				for id, req := range autoRequests {
					if _, exists := requests[id]; !exists {
						requests[id] = req
					}
				}
			}
		}
		forceTemplate := s.consumeTemplateReset()
		if rootChange {
			forceTemplate = true
		}
		skipDiff := false
		var opDiff []diff.Op
		diffInput := next
		prevForDiff := s.prev
		if len(requests) > 0 {
			var sanitized render.Structured
			var prevAligned render.Structured
			var ok bool
			componentUpdates, sanitized, prevAligned, ok = s.prepareComponentBoots(requests, s.prev, next, reg)
			if !ok {
				forceTemplate = true
				componentUpdates = nil
				requests = nil
			} else if !forceTemplate {
				diffInput = sanitized
				if len(prevAligned.S) > 0 || len(prevAligned.D) > 0 {
					prevForDiff = prevAligned
				}
				if rootChange {
					skipDiff = true
				}
			}
		}
		if forceTemplate {
			html := render.RenderHTML(node, reg)
			s.setTemplateUpdate(templateUpdate{structured: next, html: html})
			opDiff = nil
		} else if skipDiff {
			opDiff = nil
		} else {
			sanitizedPrev := sanitizeStructuredForDiff(prevForDiff)
			sanitizedNext := sanitizeStructuredForDiff(diffInput)
			opDiff = diff.Diff(sanitizedPrev, sanitizedNext)
		}
		meta := s.Metadata()
		var metadataChanged bool
		if owner := s.owner; owner != nil {
			prevMeta := owner.currentMetadata()
			if effect, ok := buildMetadataDiff(prevMeta, meta); ok {
				metadataChanged = true
				owner.enqueueMetadataEffect(effect)
			}
		}

		navUpdate := drainNavUpdate(s)
		var navDelta *protocol.NavDelta
		if !navUpdate.Empty() {
			navDelta = &protocol.NavDelta{
				Push:    navUpdate.Push,
				Replace: navUpdate.Replace,
			}
		}

		metrics := protocol.FrameMetrics{
			RenderMs: float64(time.Since(start)) / float64(time.Millisecond),
			Ops:      len(opDiff),
		}

		ids := extractRefIDs(next)
		var nextRefs map[string]protocol.RefMeta
		if len(ids) > 0 {
			nextRefs = s.snapshotRefsLocked(ids)
		}
		refsChanged := !refMetaMapsEqual(nextRefs, s.lastRefs)
		nextRefsClone = cloneRefMetaMap(nextRefs)

		cookiePending := false
		if owner := s.owner; owner != nil {
			cookiePending = owner.hasPendingCookieMutations()
		}
		shouldSend := forceTemplate || len(opDiff) > 0 || navDelta != nil || metadataChanged || cookiePending || len(componentUpdates) > 0 || refsChanged

		var sendPatchFn func([]diff.Op) error
		if shouldSend {
			if s.sendPatch == nil {
				return errors.New("runtime: SendPatch is nil")
			}
			sendPatchFn = s.sendPatch
			s.pendingNav = navDelta
			s.pendingMetrics = &metrics
			if forceTemplate {
				s.pendingRefs = nil
			} else {
				s.pendingRefs = nextRefs
			}
		} else {
			s.pendingNav = nil
			s.pendingMetrics = nil
			s.pendingRefs = nil
		}

		cleanups = append(cleanups, s.pendingCleanups...)
		effects = append(effects, s.pendingEffects...)
		pubsubs = append(pubsubs, s.pendingPubsub...)
		s.pendingCleanups = nil
		s.pendingEffects = nil
		s.pendingPubsub = nil
		s.pendingComponentBoots = componentUpdates
		s.prev = next
		s.dirty = make(map[*component]struct{})
		s.dirtyRoot = false
		s.pendingFlush = false

		if shouldSend {
			locked = false
			s.mu.Unlock()
			if err := sendPatchFn(opDiff); err != nil {
				s.mu.Lock()
				s.pendingNav = nil
				s.pendingMetrics = nil
				s.pendingRefs = nil
				s.mu.Unlock()
				return err
			}
			refsApplied = true
		} else {
			locked = false
			s.mu.Unlock()
		}

		return nil
	})
	if err != nil {
		s.mu.Lock()
		s.flushing = false
		s.mu.Unlock()
		return err
	}
	runCleanups(cleanups)
	s.runPubsubTasks(pubsubs)
	totalEffects, maxEffect, slowEffects := runEffects(effects)
	if metricsPtr := s.pendingMetrics; metricsPtr != nil {
		metricsPtr.EffectsMs = float64(totalEffects) / float64(time.Millisecond)
		metricsPtr.MaxEffectMs = float64(maxEffect) / float64(time.Millisecond)
		metricsPtr.SlowEffects = slowEffects
	}
	s.mu.Lock()
	if refsApplied {
		s.lastRefs = nextRefsClone
	}
	s.flushing = false
	pending := s.pendingFlush && s.suspend == 0
	owner := s.owner
	s.mu.Unlock()
	if pending && owner != nil {
		owner.flushAsync()
	}
	return nil
}

func runCleanups(tasks []cleanupTask) {
	for _, task := range tasks {
		task.run()
	}
}

func (s *ComponentSession) runPubsubTasks(tasks []pubsubTask) {
	for _, task := range tasks {
		if task.run == nil {
			continue
		}
		if err := s.withRecovery("pubsub", func() error {
			task.run()
			return nil
		}); err != nil {

			return
		}
	}
}

func runEffects(tasks []effectTask) (total time.Duration, max time.Duration, slowCount int) {
	for _, task := range tasks {
		start := time.Now()
		task.run()
		duration := time.Since(start)
		total += duration
		if duration > max {
			max = duration
		}
		if observeEffectDuration(task.comp, duration) {
			slowCount++
		}
	}
	return
}

func (s *ComponentSession) prepareComponentBoots(requests map[string]*componentBootRequest, prev, next render.Structured, reg handlers.Registry) ([]componentTemplateUpdate, render.Structured, render.Structured, bool) {
	if len(requests) == 0 {
		return nil, next, render.Structured{}, true
	}
	sanitized := render.Structured{
		S:              append([]string(nil), next.S...),
		D:              append([]render.Dyn(nil), next.D...),
		Components:     next.Components,
		UploadBindings: append([]render.UploadBinding(nil), next.UploadBindings...),
	}
	globalBindings := encodeBindingTable(next.Bindings)
	updates := make([]componentTemplateUpdate, 0, len(requests))
	spanPairs := make([]componentSpanPair, 0, len(requests))
	for id, req := range requests {
		span, ok := next.Components[id]
		if !ok {
			continue
		}
		if span.StaticsStart < 0 || span.StaticsEnd > len(next.S) || span.DynamicsStart < 0 || span.DynamicsEnd > len(next.D) {
			continue
		}
		update := componentTemplateUpdate{
			id:            id,
			staticsRange:  spanRange{start: span.StaticsStart, end: span.StaticsEnd},
			dynamicsRange: spanRange{start: span.DynamicsStart, end: span.DynamicsEnd},
			statics:       append([]string(nil), next.S[span.StaticsStart:span.StaticsEnd]...),
		}
		dynamicsSlice := append([]render.Dyn(nil), next.D[span.DynamicsStart:span.DynamicsEnd]...)
		update.dynamics = encodeDynamics(dynamicsSlice)
		for slot := span.DynamicsStart; slot < span.DynamicsEnd; slot++ {
			update.slots = append(update.slots, slot)
		}
		for idx, dyn := range dynamicsSlice {
			if dyn.Kind == render.DynList {
				update.listSlots = append(update.listSlots, span.DynamicsStart+idx)
			}
		}
		slotPaths := filterSlotPathsForComponent(next.SlotPaths, id, span)
		if dynPaths := collectSlotPathsFromDynamics(dynamicsSlice); len(dynPaths) > 0 {
			slotPaths = append(slotPaths, dynPaths...)
		}
		if len(slotPaths) > 0 {
			update.slotPaths = encodeSlotPaths(slotPaths)
		}
		listPaths := filterListPathsForComponent(next.ListPaths, id, span)
		if dynListPaths := collectListPathsFromDynamics(dynamicsSlice); len(dynListPaths) > 0 {
			listPaths = append(listPaths, dynListPaths...)
		}
		if len(listPaths) > 0 {
			update.listPaths = encodeListPaths(listPaths)
		}
		componentPaths := filterComponentPathsForSpan(next.ComponentPaths, span, next.Components)
		if dynComponentPaths := collectComponentPathsFromDynamics(dynamicsSlice); len(dynComponentPaths) > 0 {
			componentPaths = append(componentPaths, dynComponentPaths...)
		}
		if len(componentPaths) > 0 {
			update.componentPaths = encodeComponentPaths(componentPaths)
		}
		var uploadBindings []protocol.UploadBinding
		if filtered := filterUploadBindingsForComponent(next.UploadBindings, id); len(filtered) > 0 {
			uploadBindings = filtered
		}
		var refBindings []protocol.RefBinding
		if filteredRefs := filterRefBindingsForComponent(next.RefBindings, id); len(filteredRefs) > 0 {
			refBindings = filteredRefs
		}
		var routerBindings []protocol.RouterBinding
		if filteredRouter := filterRouterBindingsForComponent(next.RouterBindings, id); len(filteredRouter) > 0 {
			routerBindings = filteredRouter
		}
		if len(update.slots) > 0 {
			componentBindings := make(protocol.BindingTable, len(update.slots))
			for _, slot := range update.slots {
				entries := globalBindings[slot]
				if len(entries) == 0 {
					componentBindings[slot] = []protocol.SlotBinding{}
					continue
				}
				cloned := make([]protocol.SlotBinding, len(entries))
				for i, entry := range entries {
					clone := entry
					if len(entry.Listen) > 0 {
						clone.Listen = append([]string(nil), entry.Listen...)
					}
					if len(entry.Props) > 0 {
						clone.Props = append([]string(nil), entry.Props...)
					}
					cloned[i] = clone
				}
				componentBindings[slot] = cloned
			}
			hasSlots := len(componentBindings) > 0
			hasUploads := len(uploadBindings) > 0
			hasRefs := len(refBindings) > 0
			hasRouter := len(routerBindings) > 0
			if hasSlots || hasUploads || hasRefs || hasRouter {
				update.bindings = protocol.TemplateBindings{}
				if hasSlots {
					update.bindings.Slots = componentBindings
				}
				if hasUploads {
					update.bindings.Uploads = uploadBindings
				}
				if hasRefs {
					update.bindings.Refs = refBindings
				}
				if hasRouter {
					update.bindings.Router = routerBindings
				}
			}
		} else {
			hasUploads := len(uploadBindings) > 0
			hasRefs := len(refBindings) > 0
			hasRouter := len(routerBindings) > 0
			if hasUploads || hasRefs || hasRouter {
				update.bindings = protocol.TemplateBindings{}
				if hasUploads {
					update.bindings.Uploads = uploadBindings
				}
				if hasRefs {
					update.bindings.Refs = refBindings
				}
				if hasRouter {
					update.bindings.Router = routerBindings
				}
			}
		}
		if req != nil && req.component != nil {
			node := req.component.render()
			if html := render.RenderHTML(node, reg); html != "" {
				update.html = html
			} else {
				update.html = strings.Join(update.statics, "")
			}
		} else {
			update.html = strings.Join(update.statics, "")
		}
		newStructured := render.Structured{
			S: append([]string(nil), update.statics...),
			D: dynamicsSlice,
		}
		newHandlers := extractHandlerMeta(newStructured)
		var oldHandlers map[string]protocol.HandlerMeta
		if prevSpan, ok := prev.Components[id]; ok && prevSpan.StaticsStart >= 0 && prevSpan.StaticsEnd <= len(prev.S) && prevSpan.DynamicsStart >= 0 && prevSpan.DynamicsEnd <= len(prev.D) {
			oldStructured := render.Structured{
				S: append([]string(nil), prev.S[prevSpan.StaticsStart:prevSpan.StaticsEnd]...),
				D: append([]render.Dyn(nil), prev.D[prevSpan.DynamicsStart:prevSpan.DynamicsEnd]...),
			}
			oldHandlers = extractHandlerMeta(oldStructured)
			spanPairs = append(spanPairs, componentSpanPair{prev: prevSpan, next: span})
		}
		addHandlers, removeHandlers := diffHandlerMeta(oldHandlers, newHandlers)
		if len(addHandlers) > 0 {
			update.handlersAdd = addHandlers
		}
		if len(removeHandlers) > 0 {
			update.handlersDel = removeHandlers
		}
		if span.StaticsStart >= 0 && span.StaticsEnd <= len(prev.S) {
			copy(sanitized.S[span.StaticsStart:span.StaticsEnd], prev.S[span.StaticsStart:span.StaticsEnd])
		}
		if span.DynamicsStart >= 0 && span.DynamicsEnd <= len(prev.D) {
			copy(sanitized.D[span.DynamicsStart:span.DynamicsEnd], prev.D[span.DynamicsStart:span.DynamicsEnd])
		}
		updates = append(updates, update)
	}
	if len(spanPairs) == 0 && (len(prev.S) != len(next.S) || len(prev.D) != len(next.D)) {
		return nil, render.Structured{}, render.Structured{}, false
	}
	alignedPrev := alignStructured(prev, next, spanPairs)
	return updates, sanitized, alignedPrev, true
}

func (s *ComponentSession) autoComponentBootRequests(prev, next render.Structured) (map[string]*componentBootRequest, bool) {
	if len(prev.Components) == 0 || len(next.Components) == 0 {
		return nil, false
	}
	staticsLenPrev := len(prev.S)
	dynamicsLenPrev := len(prev.D)
	staticsLenNext := len(next.S)
	dynamicsLenNext := len(next.D)

	s.componentsMu.RLock()
	defer s.componentsMu.RUnlock()

	prevSanitizedStatics := sanitizeStaticsForComparison(prev.S)
	nextSanitizedStatics := sanitizeStaticsForComparison(next.S)

	var (
		changes    map[string]*componentBootRequest
		rootChange bool
	)
	for id, nextSpan := range next.Components {
		prevSpan, ok := prev.Components[id]
		if !ok {
			continue
		}
		if !componentSpanValid(prevSpan, staticsLenPrev, dynamicsLenPrev) || !componentSpanValid(nextSpan, staticsLenNext, dynamicsLenNext) {
			continue
		}
		prevStatics := prevSanitizedStatics[prevSpan.StaticsStart:prevSpan.StaticsEnd]
		nextStatics := nextSanitizedStatics[nextSpan.StaticsStart:nextSpan.StaticsEnd]
		prevDynamics := prev.D[prevSpan.DynamicsStart:prevSpan.DynamicsEnd]
		nextDynamics := next.D[nextSpan.DynamicsStart:nextSpan.DynamicsEnd]
		equalStatics := componentEqualStrings(prevStatics, nextStatics)
		equalDynamics := componentEqualDyns(prevDynamics, nextDynamics)
		if equalStatics && equalDynamics {
			continue
		}
		comp := s.components[id]
		if comp != nil && comp.parent == nil {
			rootChange = true
			continue
		}
		if changes == nil {
			changes = make(map[string]*componentBootRequest)
		}
		changes[id] = &componentBootRequest{component: comp}
	}
	return changes, rootChange
}

func componentSpanValid(span render.ComponentSpan, staticsLen, dynamicsLen int) bool {
	if span.StaticsStart < 0 || span.StaticsEnd < 0 || span.DynamicsStart < 0 || span.DynamicsEnd < 0 {
		return false
	}
	if span.StaticsStart > span.StaticsEnd || span.DynamicsStart > span.DynamicsEnd {
		return false
	}
	if span.StaticsEnd > staticsLen || span.DynamicsEnd > dynamicsLen {
		return false
	}
	return true
}

func filterSlotPathsForComponent(paths []render.SlotPath, componentID string, span render.ComponentSpan) []render.SlotPath {
	if len(paths) == 0 {
		return nil
	}
	filtered := make([]render.SlotPath, 0, len(paths))
	for _, path := range paths {
		if path.ComponentID != componentID {
			continue
		}
		if path.Slot < span.DynamicsStart || path.Slot >= span.DynamicsEnd {
			continue
		}
		filtered = append(filtered, path)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterListPathsForComponent(paths []render.ListPath, componentID string, span render.ComponentSpan) []render.ListPath {
	if len(paths) == 0 {
		return nil
	}
	filtered := make([]render.ListPath, 0, len(paths))
	for _, path := range paths {
		if path.ComponentID != componentID {
			continue
		}
		if path.Slot < span.DynamicsStart || path.Slot >= span.DynamicsEnd {
			continue
		}
		filtered = append(filtered, path)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func filterComponentPathsForSpan(paths []render.ComponentPath, span render.ComponentSpan, components map[string]render.ComponentSpan) []render.ComponentPath {
	if len(paths) == 0 {
		return nil
	}
	filtered := make([]render.ComponentPath, 0, len(paths))
	for _, path := range paths {
		target, ok := components[path.ComponentID]
		if !ok {
			continue
		}
		if target.StaticsStart < span.StaticsStart || target.StaticsEnd > span.StaticsEnd {
			continue
		}
		if target.DynamicsStart < span.DynamicsStart || target.DynamicsEnd > span.DynamicsEnd {
			continue
		}
		filtered = append(filtered, path)
	}
	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

func collectSlotPathsFromDynamics(dynamics []render.Dyn) []render.SlotPath {
	var out []render.SlotPath
	for _, dyn := range dynamics {
		if dyn.Kind != render.DynList {
			continue
		}
		for _, row := range dyn.List {
			if len(row.SlotPaths) == 0 {
				continue
			}
			out = append(out, row.SlotPaths...)
		}
	}
	return out
}

func collectListPathsFromDynamics(dynamics []render.Dyn) []render.ListPath {
	var out []render.ListPath
	for _, dyn := range dynamics {
		if dyn.Kind != render.DynList {
			continue
		}
		for _, row := range dyn.List {
			if len(row.ListPaths) == 0 {
				continue
			}
			out = append(out, row.ListPaths...)
		}
	}
	return out
}

func collectComponentPathsFromDynamics(dynamics []render.Dyn) []render.ComponentPath {
	var out []render.ComponentPath
	for _, dyn := range dynamics {
		if dyn.Kind != render.DynList {
			continue
		}
		for _, row := range dyn.List {
			if len(row.ComponentPaths) == 0 {
				continue
			}
			out = append(out, row.ComponentPaths...)
		}
	}
	return out
}

func filterUploadBindingsForComponent(bindings []render.UploadBinding, componentID string) []protocol.UploadBinding {
	if componentID == "" || len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.UploadBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.ComponentID != componentID || binding.UploadID == "" {
			continue
		}
		encoded := protocol.UploadBinding{
			ComponentID: binding.ComponentID,
			UploadID:    binding.UploadID,
			Multiple:    binding.Multiple,
			MaxSize:     binding.MaxSize,
		}
		if len(binding.Path) > 0 {
			encoded.Path = append([]int(nil), binding.Path...)
		}
		if len(binding.Accept) > 0 {
			encoded.Accept = append([]string(nil), binding.Accept...)
		}
		out = append(out, encoded)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterRefBindingsForComponent(bindings []render.RefBinding, componentID string) []protocol.RefBinding {
	if componentID == "" || len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.RefBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.ComponentID != componentID {
			continue
		}
		encoded := protocol.RefBinding{
			ComponentID: binding.ComponentID,
			RefID:       binding.RefID,
		}
		if len(binding.Path) > 0 {
			encoded.Path = append([]int(nil), binding.Path...)
		}
		out = append(out, encoded)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func filterRouterBindingsForComponent(bindings []render.RouterBinding, componentID string) []protocol.RouterBinding {
	if componentID == "" || len(bindings) == 0 {
		return nil
	}
	out := make([]protocol.RouterBinding, 0, len(bindings))
	for _, binding := range bindings {
		if binding.ComponentID != componentID {
			continue
		}
		encoded := protocol.RouterBinding{
			ComponentID: binding.ComponentID,
			PathValue:   binding.PathValue,
			Query:       binding.Query,
			Hash:        binding.Hash,
			Replace:     binding.Replace,
		}
		if len(binding.Path) > 0 {
			encoded.Path = append([]int(nil), binding.Path...)
		}
		out = append(out, encoded)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func componentEqualStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func normalizeComponentStatic(s string) string {
	if !strings.Contains(s, dom.ComponentCommentPrefix()) {
		return s
	}
	normalized := normalizeComponentMarker(s, "<!--"+dom.ComponentCommentPrefix()+":start:")
	normalized = normalizeComponentMarker(normalized, "<!--"+dom.ComponentCommentPrefix()+":end:")
	return normalized
}

func sanitizeStaticsForComparison(statics []string) []string {
	if len(statics) == 0 {
		return nil
	}
	sanitized := make([]string, len(statics))
	for i, s := range statics {
		sanitized[i] = normalizeComponentStatic(s)
	}
	return sanitized
}

func sanitizeStructuredForDiff(str render.Structured) render.Structured {
	return render.Structured{
		S: sanitizeStaticsForComparison(str.S),
		D: str.D,
	}
}

func normalizeComponentMarker(s, marker string) string {
	var b strings.Builder
	for {
		idx := strings.Index(s, marker)
		if idx == -1 {
			b.WriteString(s)
			break
		}
		b.WriteString(s[:idx])
		b.WriteString(marker)
		s = s[idx+len(marker):]
		endIdx := strings.Index(s, "-->")
		if endIdx == -1 {
			break
		}
		b.WriteString("-->")
		s = s[endIdx+len("-->"):]
	}
	return b.String()
}

func componentEqualDyns(a, b []render.Dyn) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i].Kind != b[i].Kind {
			return false
		}
		if a[i].Kind == render.DynList {
			if !componentEqualRows(a[i].List, b[i].List) {
				return false
			}
		}
	}
	return true
}

func componentEqualRows(a, b []render.Row) bool {
	if len(a) == 0 || len(b) == 0 {
		return true
	}
	if len(a[0].Slots) != len(b[0].Slots) {
		return false
	}
	return true
}

func diffHandlerMeta(prev, next map[string]protocol.HandlerMeta) (map[string]protocol.HandlerMeta, []string) {
	var add map[string]protocol.HandlerMeta
	var del []string
	if len(next) > 0 {
		for id, meta := range next {
			if prev == nil {
				if add == nil {
					add = make(map[string]protocol.HandlerMeta)
				}
				add[id] = meta
				continue
			}
			if old, ok := prev[id]; !ok || !handlerMetaEqual(old, meta) {
				if add == nil {
					add = make(map[string]protocol.HandlerMeta)
				}
				add[id] = meta
			}
		}
	}
	if len(prev) > 0 {
		for id := range prev {
			if next == nil {
				del = append(del, id)
				continue
			}
			if _, ok := next[id]; !ok {
				del = append(del, id)
			}
		}
	}
	return add, del
}

func handlerMetaEqual(a, b protocol.HandlerMeta) bool {
	if a.Event != b.Event {
		return false
	}
	if !equalStringSlices(a.Listen, b.Listen) {
		return false
	}
	if !equalStringSlices(a.Props, b.Props) {
		return false
	}
	return true
}

func equalStringSlices(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// Reset clears the errored flag and rebuilds the root component so rendering can resume.
func (s *ComponentSession) Reset() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	if !s.errored {
		s.mu.Unlock()
		return false
	}
	var (
		callable = s.rootCallable
		props    any
	)
	if s.root != nil {
		props = s.root.props
	}
	if props == nil {
		props = s.rootProps
	}
	s.rootProps = props
	s.errored = false
	s.lastDiagnostic = nil
	s.pendingFlush = false
	s.dirtyRoot = false
	s.dirty = make(map[*component]struct{})
	s.pendingEffects = nil
	s.pendingCleanups = nil
	s.pendingNav = nil
	s.pendingMetrics = nil
	s.pendingPubsub = nil
	s.pendingRefs = nil
	s.lastRefs = nil
	s.pubsubMu.Lock()
	s.pubsubSubs = nil
	s.pubsubMu.Unlock()
	s.mu.Unlock()

	if s.root != nil {
		s.root.unmount()
	}

	var rebuilt *component
	if callable != nil {
		rebuilt = newComponentWithCallable(s, nil, "root", callable, props)
	}

	s.mu.Lock()
	if rebuilt != nil {
		if s.dirty == nil {
			s.dirty = make(map[*component]struct{})
		}
		s.dirty[rebuilt] = struct{}{}
		s.dirtyRoot = true
		s.pendingFlush = true
	}
	s.root = rebuilt
	s.rootCallable = callable
	if props != nil {
		s.rootProps = props
	}
	s.mu.Unlock()
	return rebuilt != nil
}

func (s *ComponentSession) withRecovery(phase string, fn func() error) (err error) {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	prevPhase := s.currentPhase
	s.currentPhase = phase
	defer func() { s.currentPhase = prevPhase }()
	defer func() {
		if rec := recover(); rec != nil {
			err = s.handlePanic(phase, rec)
		}
	}()
	return fn()
}

func (s *ComponentSession) handlePanic(phase string, value any) error {
	diag := Diagnostic{
		Phase:      phase,
		Message:    fmt.Sprint(value),
		Panic:      fmt.Sprintf("%v", value),
		CapturedAt: time.Now(),
		Code:       normalizeDiagnosticCode(phase),
		Stack:      string(debug.Stack()),
	}
	if comp := s.currentComponent(); comp != nil {
		diag.ComponentID = comp.id
		if comp.callable != nil {
			diag.ComponentName = comp.callable.name()
		}
	}
	if carrier, ok := value.(metadataCarrier); ok {
		diag.Metadata = cloneDiagnosticMetadata(carrier.Metadata())
	}
	if diag.Metadata == nil {
		diag.Metadata = map[string]any{}
	}
	diag.Metadata["panicType"] = fmt.Sprintf("%T", value)
	if hooker, ok := value.(hookCarrier); ok {
		diag.Hook = hooker.HookName()
		diag.HookIndex = hooker.HookIndex()
	}
	if suggester, ok := value.(suggestionCarrier); ok {
		diag.Suggestion = suggester.Suggestion()
	}

	s.mu.Lock()
	s.errored = true
	s.lastDiagnostic = &diag
	s.pendingFlush = false
	s.dirtyRoot = false
	s.dirty = make(map[*component]struct{})
	s.pendingEffects = nil
	s.pendingCleanups = nil
	s.pendingNav = nil
	s.pendingMetrics = nil
	s.pendingPubsub = nil
	s.pendingRefs = nil
	s.lastRefs = nil
	s.pubsubMu.Lock()
	s.pubsubSubs = nil
	s.pubsubMu.Unlock()
	s.mu.Unlock()

	if s.reporter != nil {
		s.reporter.ReportDiagnostic(diag)
	}
	return DiagnosticError{diag: diag}
}

func (s *ComponentSession) pushComponent(c *component) {
	if s == nil || c == nil {
		return
	}
	s.renderStack = append(s.renderStack, c)
}

func (s *ComponentSession) popComponent() {
	if s == nil || len(s.renderStack) == 0 {
		return
	}
	s.renderStack = s.renderStack[:len(s.renderStack)-1]
}

func (s *ComponentSession) currentComponent() *component {
	if s == nil || len(s.renderStack) == 0 {
		return nil
	}
	return s.renderStack[len(s.renderStack)-1]
}

// Dirty reports whether the session has pending renders.
func (s *ComponentSession) Dirty() bool {
	if s == nil {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.pendingFlush || s.dirtyRoot || len(s.dirty) > 0
}

func (s *ComponentSession) suspendFlushScheduling() func() {
	if s == nil {
		return func() {}
	}
	s.mu.Lock()
	s.suspend++
	s.mu.Unlock()
	return func() {
		s.mu.Lock()
		s.suspend--
		if s.suspend < 0 {
			s.suspend = 0
		}
		s.mu.Unlock()
	}
}

func (s *ComponentSession) markDirty(c *component) {
	if s == nil || c == nil {
		return
	}
	if s.errored {
		return
	}
	var (
		owner    *LiveSession
		schedule bool
	)
	s.mu.Lock()
	if s.dirty == nil {
		s.dirty = make(map[*component]struct{})
	}
	s.dirty[c] = struct{}{}
	inEvent := strings.HasPrefix(s.currentPhase, "event:")
	if !s.pendingFlush && s.suspend == 0 && !s.flushing && !inEvent {
		schedule = true
	}
	s.dirtyRoot = true
	s.pendingFlush = true
	owner = s.owner
	s.mu.Unlock()
	c.markDirtyChain()
	if schedule && owner != nil {
		owner.flushAsync()
	}
}

func (s *ComponentSession) clearDirty(c *component) {
	if s == nil || c == nil {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.dirty, c)
	if len(s.dirty) == 0 {
		s.dirtyRoot = false
		s.pendingFlush = false
	}
}

// HandleEvent dispatches an event to a registered handler without flushing.
func (s *ComponentSession) HandleEvent(id handlers.ID, ev handlers.Event) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	if s.errored {
		if s.lastDiagnostic != nil {
			return DiagnosticError{diag: *s.lastDiagnostic}
		}
		return errors.New("runtime: session halted after panic")
	}
	phase := fmt.Sprintf("event:%s", id)
	return s.withRecovery(phase, func() error {
		registry := s.ensureRegistry()
		handler, ok := registry.Get(id)
		if !ok || handler == nil {
			diag := Diagnostic{
				Code:       "handler_not_found",
				Phase:      phase,
				Message:    fmt.Sprintf("runtime: handler %s not found", id),
				Metadata:   map[string]any{"handlerId": string(id)},
				Suggestion: "Ensure the event handler is registered before dispatching events.",
			}
			return DiagnosticError{diag: diag}
		}
		if updates := handler(ev); updates != nil {
			s.markDirty(s.root)
		}
		return nil
	})
}

// DispatchEvent routes an event and flushes if state changed.
func (s *ComponentSession) DispatchEvent(id handlers.ID, ev handlers.Event) error {
	if err := s.HandleEvent(id, ev); err != nil {
		return err
	}
	s.mu.Lock()
	dirty := s.pendingFlush && s.suspend == 0
	s.mu.Unlock()
	if !dirty {
		return nil
	}
	return s.Flush()
}

// effectTask represents an effect setup scheduled after the next flush.
type effectTask struct {
	comp  *component
	index int
	setup func() Cleanup
}

func (t effectTask) run() {
	if t.comp == nil || t.comp.frame == nil {
		if t.setup != nil {
			t.setup()
		}
		return
	}
	if t.setup == nil {
		return
	}
	cleanup := t.setup()
	if t.index >= 0 && t.index < len(t.comp.frame.cells) {
		if cell, ok := t.comp.frame.cells[t.index].(*effectCell); ok {
			cell.cleanup = cleanup
		}
	}
}

// cleanupTask runs stored effect cleanups.
type cleanupTask struct {
	comp  *component
	index int
}

func (t cleanupTask) run() {
	if t.comp == nil || t.comp.frame == nil {
		return
	}
	if t.index < 0 || t.index >= len(t.comp.frame.cells) {
		return
	}
	if cell, ok := t.comp.frame.cells[t.index].(*effectCell); ok {
		if cell.cleanup != nil {
			cell.cleanup()
			cell.cleanup = nil
		}
	}
}

func (s *ComponentSession) enqueueEffect(comp *component, index int, setup func() Cleanup) {
	if s == nil {
		return
	}
	if s.errored {
		return
	}
	s.pendingEffects = append(s.pendingEffects, effectTask{comp: comp, index: index, setup: setup})
}

func (s *ComponentSession) enqueueCleanup(comp *component, index int) {
	if s == nil {
		return
	}
	if s.errored {
		return
	}
	s.pendingCleanups = append(s.pendingCleanups, cleanupTask{comp: comp, index: index})
}

func (s *ComponentSession) enqueuePubsub(fn func()) {
	if s == nil || fn == nil {
		return
	}
	if s.errored {
		return
	}
	s.mu.Lock()
	s.pendingPubsub = append(s.pendingPubsub, pubsubTask{run: fn})
	s.pendingFlush = true
	s.mu.Unlock()
}

func (s *ComponentSession) ensureRegistry() handlers.Registry {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	reg := s.registry
	if reg == nil {
		reg = handlers.NewRegistry()
		s.registry = reg
	}
	s.mu.Unlock()
	return reg
}

func (s *ComponentSession) subscribePubsub(topic string, provider PubsubProvider, handler func([]byte, map[string]string)) (string, error) {
	if s == nil {
		return "", errors.New("runtime: session is nil")
	}
	if handler == nil {
		return "", errors.New("runtime: pubsub handler is nil")
	}
	target := provider
	if target == nil {
		target = s.pubsubProvider
	}
	if target == nil || s.owner == nil {
		return "", ErrPubsubUnavailable
	}
	token, err := target.Subscribe(s.owner, topic, s.handlePubsubDelivery)
	if err != nil {
		return "", err
	}
	s.pubsubMu.Lock()
	if s.pubsubSubs == nil {
		s.pubsubSubs = make(map[string]pubsubSubscription)
	}
	s.pubsubSubs[token] = pubsubSubscription{
		token:    token,
		topic:    topic,
		handler:  handler,
		provider: target,
	}
	s.pubsubMu.Unlock()
	if s.owner != nil && target == s.pubsubProvider {
		s.owner.pubsubSubscribed(topic)
	}
	return token, nil
}

func (s *ComponentSession) unsubscribePubsub(token string) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	if token == "" {
		return nil
	}
	s.pubsubMu.Lock()
	sub, ok := s.pubsubSubs[token]
	if ok {
		delete(s.pubsubSubs, token)
	}
	s.pubsubMu.Unlock()
	if !ok {
		return nil
	}
	provider := sub.provider
	if provider == nil {
		provider = s.pubsubProvider
	}
	if provider == nil || s.owner == nil {
		return ErrPubsubUnavailable
	}
	if err := provider.Unsubscribe(s.owner, sub.token); err != nil {
		s.pubsubMu.Lock()
		if s.pubsubSubs == nil {
			s.pubsubSubs = make(map[string]pubsubSubscription)
		}
		s.pubsubSubs[token] = sub
		s.pubsubMu.Unlock()
		return err
	}
	if s.owner != nil && provider == s.pubsubProvider {
		s.owner.pubsubUnsubscribed(sub.topic)
	}
	return nil
}

func (s *ComponentSession) publishPubsub(topic string, provider PubsubProvider, payload []byte, meta map[string]string) error {
	if s == nil {
		return errors.New("runtime: session is nil")
	}
	target := provider
	if target == nil {
		target = s.pubsubProvider
	}
	if target == nil {
		return ErrPubsubUnavailable
	}
	data := append([]byte(nil), payload...)
	var metaCopy map[string]string
	if meta != nil {
		metaCopy = cloneStringMap(meta)
	}
	return target.Publish(topic, data, metaCopy)
}

func (s *ComponentSession) handlePubsubDelivery(topic string, payload []byte, meta map[string]string) {
	s.deliverPubsub(topic, payload, meta)
}

func (s *ComponentSession) deliverPubsub(topic string, payload []byte, meta map[string]string) {
	if s == nil {
		return
	}
	s.pubsubMu.RLock()
	if len(s.pubsubSubs) == 0 {
		s.pubsubMu.RUnlock()
		return
	}
	handlers := make([]func([]byte, map[string]string), 0, len(s.pubsubSubs))
	for _, sub := range s.pubsubSubs {
		if sub.topic == topic && sub.handler != nil {
			handlers = append(handlers, sub.handler)
		}
	}
	s.pubsubMu.RUnlock()
	if len(handlers) == 0 {
		return
	}
	for _, h := range handlers {
		handler := h
		payloadCopy := append([]byte(nil), payload...)
		var metaCopy map[string]string
		if meta != nil {
			metaCopy = cloneStringMap(meta)
		}
		s.enqueuePubsub(func() {
			handler(payloadCopy, metaCopy)
		})
	}
}
