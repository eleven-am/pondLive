package render

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type DynamicSlotKind uint8

const (
	DynamicText DynamicSlotKind = iota
	DynamicAttrs
	DynamicList
)

type DynamicSlot struct {
	Kind DynamicSlotKind

	Text  string
	Attrs map[string]string
	List  []Row
}

type Row struct {
	Key            string
	HTML           string
	Slots          []int
	Bindings       []HandlerBinding
	SlotPaths      []SlotPath
	ListPaths      []ListPath
	ComponentPaths []ComponentPath
	UploadBindings []UploadBinding
	RefBindings    []RefBinding
	RouterBindings []RouterBinding
	RootCount      int
}

type SlotPath struct {
	Slot           int
	ComponentID    string
	Path           []PathSegment
	TextChildIndex int
}

type ListPath struct {
	Slot        int
	ComponentID string
	Path        []PathSegment
	AtRoot      bool
}

type ComponentPath struct {
	ComponentID string
	ParentID    string
	ParentPath  []PathSegment
	FirstChild  []PathSegment
	LastChild   []PathSegment
}

type Structured struct {
	S          []string
	D          []DynamicSlot
	Components map[string]ComponentSpan
	Bindings   []HandlerBinding

	UploadBindings []UploadBinding
	RefBindings    []RefBinding
	RouterBindings []RouterBinding

	SlotPaths      []SlotPath
	ListPaths      []ListPath
	ComponentPaths []ComponentPath
}

type UploadBinding struct {
	ComponentID string
	Path        []PathSegment
	UploadID    string
	Accept      []string
	Multiple    bool
	MaxSize     int64
}

type RefBinding struct {
	ComponentID string
	Path        []PathSegment
	RefID       string
}

type RouterBinding struct {
	ComponentID string
	Path        []PathSegment
	PathValue   string
	Query       string
	Hash        string
	Replace     string
}

type HandlerBinding struct {
	Slot    int
	Event   string
	Handler string
	Listen  []string
	Props   []string
}

type PromotionTracker interface {
	ResolveTextPromotion(componentID string, path []int, value string, mutable bool) bool
	ResolveAttrPromotion(componentID string, path []int, attrs map[string]string, mutable map[string]bool) bool
}

type NodeVisitor interface {
	VisitElement(el *h.Element) int
	VisitText(text *h.TextNode) int
	VisitFragment(fragment *h.FragmentNode) int
	VisitComponent(component *h.ComponentNode) int
	VisitComment(comment *h.CommentNode) int
}

type StructuredOptions struct {
	Handlers                  handlers.Registry
	Promotions                PromotionTracker
	ConcurrentRows            bool
	RowConcurrencyThreshold   int
	MaxRowWorkers             int
	ConcurrentChildren        bool
	ChildConcurrencyThreshold int
	MaxChildWorkers           int
}

type ComponentSpan struct {
	StaticsStart  int
	StaticsEnd    int
	DynamicsStart int
	DynamicsEnd   int
}

type structuredBuilder struct {
	statics    []string
	current    strings.Builder
	dynamics   []DynamicSlot
	stack      []elementFrame
	components map[string]ComponentSpan
	bindings   *BindingExtractor

	tracker  PromotionTracker
	pathCalc *PathCalculator
	opts     StructuredOptions
}

type elementFrame struct {
	attrSlot      int
	element       *h.Element
	bindings      []slotBinding
	startStatic   int
	void          bool
	staticAttrs   bool
	componentID   string
	componentPath []int
	basePath      []int
}

type slotBinding struct {
	slot       int
	childIndex int
}

type componentFrame struct {
	id        string
	parentID  string
	prevPath  []int
	basePath  []int
	startPath []int
	endPath   []int
}
