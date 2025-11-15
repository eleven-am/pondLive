// Package render transforms a component tree into a structured template
// that separates static HTML from dynamic slots.
//
// Path Navigation Model:
//   - PathRangeOffset: Navigate within a component's child range by skipping
//     non-whitespace children. Example: PathSegment{PathRangeOffset, 2} means
//     skip 2 DOM-visible children within the component's range.
//   - PathDomChild: Navigate to a specific DOM child by index in childNodes.
//     Example: PathSegment{PathDomChild, 1} means the second child in the
//     parent's childNodes array.
//
// Whitespace Handling:
//   - The domWidth() function returns 0 for whitespace-only text nodes,
//     ensuring server-side path calculation aligns with client-side DOM pruning.
//   - Mutable text nodes always count (even if whitespace-only) since they
//     represent dynamic content slots.
//
// Path Composition:
//   - Valid paths must start with PathRangeOffset (component boundary)
//   - PathDomChild segments can follow PathRangeOffset or other PathDomChild
//   - PathRangeOffset cannot follow PathDomChild (invalid navigation)
package render

import (
	"html"
	"runtime"
	"sort"
	"strings"
	"sync"

	"github.com/eleven-am/pondlive/go/internal/dom"
	"github.com/eleven-am/pondlive/go/internal/handlers"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

var _ NodeVisitor = (*structuredBuilder)(nil)

// builderPool reuses structuredBuilder instances to reduce allocations
var builderPool = sync.Pool{
	New: func() interface{} {
		return &structuredBuilder{
			statics:    make([]string, 0, 32),
			dynamics:   make([]DynamicSlot, 0, 8),
			components: make(map[string]ComponentSpan, 4),
		}
	},
}

func ToStructured(n h.Node) (Structured, error) {
	return ToStructuredWithHandlers(n, StructuredOptions{})
}

// ToStructuredWithOptions is an alias for ToStructuredWithHandlers for compatibility
func ToStructuredWithOptions(n h.Node, opts StructuredOptions) (Structured, error) {
	return ToStructuredWithHandlers(n, opts)
}

func ToStructuredWithHandlers(n h.Node, opts StructuredOptions) (Structured, error) {
	if n == nil {
		return Structured{}, &ValidationError{Message: "root node cannot be nil"}
	}

	validator := NewTreeValidator()
	if err := validator.Validate(n); err != nil {
		return Structured{}, err
	}

	FinalizeWithHandlers(n, opts.Handlers)

	if opts.RowConcurrencyThreshold <= 0 {
		opts.RowConcurrencyThreshold = 10
	}
	if opts.MaxRowWorkers <= 0 {
		opts.MaxRowWorkers = runtime.GOMAXPROCS(0)
	}
	if opts.ChildConcurrencyThreshold <= 0 {
		opts.ChildConcurrencyThreshold = 8
	}
	if opts.MaxChildWorkers <= 0 {
		opts.MaxChildWorkers = runtime.GOMAXPROCS(0)
	}

	analyzer := NewComponentAnalyzer()
	analysis := analyzer.Analyze(n)

	builder := &structuredBuilder{
		tracker:    opts.Promotions,
		bindings:   NewBindingExtractor(),
		pathCalc:   NewPathCalculator(),
		statics:    make([]string, 0, analysis.StaticsCapacity),
		dynamics:   make([]DynamicSlot, 0, analysis.DynamicsCapacity),
		components: analysis.Components,
		opts:       opts,
	}
	builder.visit(n)
	builder.flush()
	return Structured{
		S:              builder.statics,
		D:              builder.dynamics,
		Components:     builder.components,
		Bindings:       builder.bindings.HandlerBindings(),
		UploadBindings: builder.bindings.UploadBindings(),
		RefBindings:    builder.bindings.RefBindings(),
		RouterBindings: builder.bindings.RouterBindings(),
		SlotPaths:      builder.bindings.SlotPaths(),
		ListPaths:      builder.pathCalc.ListPaths(),
		ComponentPaths: builder.pathCalc.ComponentPaths(),
	}, nil
}

func (b *structuredBuilder) writeStatic(s string) {
	if s == "" {
		return
	}
	b.current.WriteString(s)
}

func (b *structuredBuilder) flush() {
	b.statics = append(b.statics, b.current.String())
	b.current.Reset()
}

func (b *structuredBuilder) addDyn(d DynamicSlot) int {
	b.flush()
	b.dynamics = append(b.dynamics, d)
	return len(b.dynamics) - 1
}

func (b *structuredBuilder) visit(n h.Node) int {
	switch v := n.(type) {
	case *h.TextNode:
		return b.VisitText(v)
	case *h.Element:
		return b.VisitElement(v)
	case *h.FragmentNode:
		return b.VisitFragment(v)
	case *h.CommentNode:
		return b.VisitComment(v)
	case *h.ComponentNode:
		return b.VisitComponent(v)
	default:
		return 0
	}
}

func (b *structuredBuilder) VisitText(t *h.TextNode) int {
	if t == nil {
		return 0
	}
	dynamic := t.Mutable
	if !dynamic && b.tracker != nil {
		componentID := b.pathCalc.CurrentComponentID()
		path := b.pathCalc.CurrentComponentPath()
		dynamic = b.tracker.ResolveTextPromotion(componentID, path, t.Value, t.Mutable)
	}
	if dynamic {
		idx := b.addDyn(DynamicSlot{Kind: DynamicText, Text: t.Value})
		b.appendSlotToCurrent(idx, t)
		return 1
	}
	b.writeStatic(html.EscapeString(t.Value))
	return 1
}

func (b *structuredBuilder) VisitComment(c *h.CommentNode) int {
	if c == nil {
		return 0
	}
	b.writeStatic("<!--")
	b.writeStatic(escapeComment(c.Value))
	b.writeStatic("-->")
	return 1
}

func (b *structuredBuilder) VisitComponent(v *h.ComponentNode) int {
	if v == nil || v.ID == "" {
		if v != nil && v.Child != nil {
			return b.visit(v.Child)
		}
		return 0
	}
	b.flush()
	staticsStart := len(b.statics)
	dynamicsStart := len(b.dynamics)
	b.pathCalc.PushComponent(v.ID)
	width := 0
	if v.Child != nil {
		width = b.visit(v.Child)
	}
	b.pathCalc.PopComponent()
	b.flush()
	span := ComponentSpan{
		StaticsStart:  staticsStart,
		StaticsEnd:    len(b.statics),
		DynamicsStart: dynamicsStart,
		DynamicsEnd:   len(b.dynamics),
	}
	if b.components == nil {
		b.components = make(map[string]ComponentSpan)
	}
	b.components[v.ID] = span
	return width
}

func (b *structuredBuilder) VisitElement(v *h.Element) int {
	if v == nil {
		return 0
	}

	void, attrSlot, startStatic := b.renderOpeningTag(v)
	b.pushFrame(v, attrSlot, startStatic, void, attrSlot < 0)
	defer b.popFrame()

	if void {
		return 1
	}

	b.processElementChildren(v, attrSlot)
	b.renderClosingTag(v)
	return 1
}

func (b *structuredBuilder) renderOpeningTag(v *h.Element) (void bool, attrSlot int, startStatic int) {
	void = dom.IsVoidElement(v.Tag)
	dynamicAttrs := b.shouldUseDynamicAttrs(v)
	attrSlot = -1
	startStatic = -1

	if dynamicAttrs {
		b.writeStatic("<")
		b.writeStatic(v.Tag)
		attrs := copyAttrs(v.Attrs)
		if attrs == nil {
			attrs = map[string]string{}
		}
		attrSlot = b.addDyn(DynamicSlot{Kind: DynamicAttrs, Attrs: attrs})
		startStatic = len(b.statics) - 1
		if void {
			b.writeStatic("/>")
		} else {
			b.writeStatic(">")
		}
	} else {
		start := renderStartTag(v, void)
		b.writeStatic(start)
		b.flush()
		startStatic = len(b.statics) - 1
	}

	return void, attrSlot, startStatic
}

func (b *structuredBuilder) processElementChildren(v *h.Element, attrSlot int) {
	if v.Unsafe != nil {
		b.writeStatic(*v.Unsafe)
	} else if !b.tryKeyedChildren(v.Children) {
		b.visitChildren(v.Children)
	}
}

func (b *structuredBuilder) renderClosingTag(v *h.Element) {
	b.writeStatic("</")
	b.writeStatic(v.Tag)
	b.writeStatic(">")
}

func (b *structuredBuilder) VisitFragment(f *h.FragmentNode) int {
	if f == nil {
		return 0
	}
	if b.tryKeyedChildren(f.Children) {
		return 0
	}
	return b.visitChildren(f.Children)
}

func (b *structuredBuilder) shouldUseDynamicAttrs(el *h.Element) bool {
	if el == nil {
		return false
	}
	if len(el.HandlerAssignments) > 0 {
		return true
	}
	for name, value := range el.Attrs {
		if value == "" {
			continue
		}
		if isDynamicAttr(name) {
			return true
		}
	}
	mutable := el.MutableAttrs
	if tracker := b.tracker; tracker != nil {
		componentID := b.pathCalc.CurrentComponentID()
		path := b.pathCalc.CurrentComponentPath()
		if tracker.ResolveAttrPromotion(componentID, path, el.Attrs, mutable) {
			return true
		}
	}
	return shouldForceDynamicAttrs(mutable, el.Attrs)
}

func isDynamicAttr(name string) bool {
	if strings.HasPrefix(name, "data-on") {
		return true
	}
	switch name {
	case "value", "checked", "selected":
		return true
	default:
		return false
	}
}

func shouldForceDynamicAttrs(mutable map[string]bool, attrs map[string]string) bool {
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

func (b *structuredBuilder) pushFrame(el *h.Element, attrSlot, startStatic int, void, staticAttrs bool) {
	frame := elementFrame{
		attrSlot:      attrSlot,
		element:       el,
		startStatic:   startStatic,
		void:          void,
		staticAttrs:   staticAttrs,
		componentID:   b.pathCalc.CurrentComponentID(),
		componentPath: b.pathCalc.CurrentComponentPath(),
		basePath:      b.pathCalc.CurrentComponentBasePath(),
	}
	if attrSlot >= 0 {
		frame.bindings = append(frame.bindings, slotBinding{slot: attrSlot, childIndex: -1})
	}
	b.stack = append(b.stack, frame)
}

func (b *structuredBuilder) appendSlotToCurrent(slot int, node h.Node) {
	if len(b.stack) == 0 {
		return
	}
	frame := &b.stack[len(b.stack)-1]
	binding := slotBinding{slot: slot, childIndex: -1}
	if txt, ok := node.(*h.TextNode); ok {
		binding.childIndex = childIndexOf(frame.element, txt)
	}
	frame.bindings = append(frame.bindings, binding)
}

func (b *structuredBuilder) pushChildIndex(idx int) {
	b.pathCalc.AppendToPath(idx)
	b.pathCalc.RecordComponentTraversal()
}

func (b *structuredBuilder) popChildIndex() {
	b.pathCalc.TrimPath(1)
}

func (b *structuredBuilder) popFrame() {
	if len(b.stack) == 0 {
		return
	}
	last := len(b.stack) - 1
	frame := b.stack[last]
	b.stack = b.stack[:last]
	b.assignSlotIndices(frame)
}

func (b *structuredBuilder) assignSlotIndices(frame elementFrame) {
	if frame.staticAttrs && frame.startStatic >= 0 && frame.startStatic < len(b.statics) {
		b.statics[frame.startStatic] = renderStartTag(frame.element, frame.void)
	}

	b.bindings.ExtractAll(frame)
}

func escapeComment(value string) string {
	return strings.ReplaceAll(value, "--", "- -")
}

func childIndexOf(parent *h.Element, target h.Node) int {
	if parent == nil || target == nil {
		return -1
	}
	for idx, child := range parent.Children {
		if child == target {
			return idx
		}
	}
	return -1
}

func (b *structuredBuilder) visitChildren(children []h.Node) int {
	return b.visitChildrenWithOffset(children, 0)
}

func (b *structuredBuilder) visitChildrenWithOffset(children []h.Node, start int) int {
	return b.visitChildrenSequential(children, start)
}

func (b *structuredBuilder) visitChildrenSequential(children []h.Node, start int) int {
	offset := start
	for _, child := range children {
		offset += b.visitChildNode(child, offset)
	}
	return offset - start
}

func (b *structuredBuilder) visitChildNode(child h.Node, domIndex int) int {
	switch v := child.(type) {
	case nil:
		return 0
	case *h.FragmentNode:
		consumed := 0
		offset := domIndex
		for _, inner := range v.Children {
			width := b.visitChildNode(inner, offset)
			offset += width
			consumed += width
		}
		return consumed
	default:
		width := domWidth(child)
		if width == 0 {
			b.visit(child)
			return 0
		}
		b.pushChildIndex(domIndex)
		w := b.visit(child)
		b.popChildIndex()
		if w <= 0 {
			return width
		}
		return width
	}
}

func domWidth(n h.Node) int {
	switch v := n.(type) {
	case *h.Element:
		return 1
	case *h.CommentNode:
		return 1
	case *h.TextNode:
		if strings.TrimSpace(v.Value) == "" && !v.Mutable {
			return 0
		}
		return 1
	case *h.FragmentNode:
		total := 0
		for _, child := range v.Children {
			total += domWidth(child)
		}
		return total
	case *h.ComponentNode:
		if v.Child == nil {
			return 0
		}
		return domWidth(v.Child)
	default:
		return 0
	}
}

func copyAttrs(attrs map[string]string) map[string]string {
	if len(attrs) == 0 {
		return nil
	}
	out := make(map[string]string, len(attrs))
	for k, v := range attrs {
		out[k] = v
	}
	return out
}

func renderStartTag(el *h.Element, void bool) string {
	if el == nil {
		return ""
	}
	var b strings.Builder
	b.WriteByte('<')
	b.WriteString(el.Tag)
	if len(el.Attrs) > 0 {
		keys := make([]string, 0, len(el.Attrs))
		for k := range el.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if strings.HasPrefix(k, "data-on") {
				continue
			}
			v := el.Attrs[k]
			if v == "" {
				continue
			}
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString("=\"")
			b.WriteString(html.EscapeString(v))
			b.WriteString("\"")
		}
	}
	if void {
		b.WriteByte('>')
		return b.String()
	}
	b.WriteByte('>')
	return b.String()
}

func renderFinalizedNode(n h.Node) string {
	if n == nil {
		return ""
	}
	var b strings.Builder
	renderNode(&b, n)
	return b.String()
}

func collectKeyedNodes(children []h.Node) []h.Node {
	rows := make([]h.Node, 0, len(children))
	for _, child := range children {
		if child == nil {
			continue
		}

		if frag, ok := child.(*h.FragmentNode); ok {
			if len(frag.Children) == 0 {
				continue
			}

			fragRows := collectKeyedNodes(frag.Children)
			if fragRows == nil {
				return nil
			}
			rows = append(rows, fragRows...)
			continue
		}
		key := nodeKey(child)
		if key == "" {
			return nil
		}
		rows = append(rows, child)
	}
	if len(rows) == 0 {
		return nil
	}
	return rows
}

func nodeKey(node h.Node) string {
	switch v := node.(type) {
	case *h.Element:
		return v.Key
	case *h.ComponentNode:
		return v.Key
	default:
		return ""
	}
}

func countTopLevelNodes(node h.Node) int {
	switch v := node.(type) {
	case *h.FragmentNode:
		total := 0
		for _, child := range v.Children {
			total += countTopLevelNodes(child)
		}
		return total
	case *h.ComponentNode:
		if v.Child == nil {
			return 0
		}
		return countTopLevelNodes(v.Child)
	case nil:
		return 0
	default:
		return 1
	}
}

func (b *structuredBuilder) tryKeyedChildren(children []h.Node) bool {
	rows := collectKeyedNodes(children)
	if len(rows) == 0 {
		return false
	}

	listSlot := b.addDyn(DynamicSlot{Kind: DynamicList})
	rowEntries := b.processKeyedRows(rows)

	if listSlot >= 0 && listSlot < len(b.dynamics) {
		b.dynamics[listSlot].List = rowEntries
	}

	var frame *elementFrame
	if len(b.stack) > 0 {
		f := b.stack[len(b.stack)-1]
		frame = &f
	}
	b.pathCalc.RecordListPath(listSlot, frame)
	return true
}

func (b *structuredBuilder) processKeyedRows(rows []h.Node) []Row {
	if !b.opts.ConcurrentRows || len(rows) < b.opts.RowConcurrencyThreshold {
		return b.processKeyedRowsSequential(rows)
	}

	if areRowsHomogeneous(rows) {
		return b.processKeyedRowsSequential(rows)
	}
	return b.processKeyedRowsConcurrent(rows)
}

// areRowsHomogeneous detects when rows are simple enough that concurrent processing overhead isn't worth it
func areRowsHomogeneous(rows []h.Node) bool {
	if len(rows) == 0 {
		return true
	}

	firstType := ""
	maxComplexity := 0

	for i, row := range rows {
		if row == nil {
			continue
		}

		complexity := estimateNodeComplexity(row)
		if complexity > maxComplexity {
			maxComplexity = complexity
		}

		rowType := getNodeTypeName(row)
		if i == 0 {
			firstType = rowType
		} else if rowType != firstType {

			return false
		}
	}

	return maxComplexity <= 10
}

// estimateNodeComplexity provides a rough complexity score for a node
func estimateNodeComplexity(node h.Node) int {
	if node == nil {
		return 0
	}

	switch v := node.(type) {
	case *h.Element:
		complexity := 1
		if len(v.Children) > 0 {
			complexity += len(v.Children)

			for i := 0; i < len(v.Children) && i < 3; i++ {
				if child, ok := v.Children[i].(*h.Element); ok && len(child.Children) > 0 {
					complexity += len(child.Children)
				}
			}
		}
		return complexity
	case *h.ComponentNode:

		return 5
	case *h.FragmentNode:
		complexity := 0
		for _, child := range v.Children {
			complexity += estimateNodeComplexity(child)
		}
		return complexity
	case *h.TextNode:
		return 1
	case *h.CommentNode:
		return 1
	default:
		return 1
	}
}

// getNodeTypeName returns a string representing the node type
func getNodeTypeName(node h.Node) string {
	switch v := node.(type) {
	case *h.Element:
		return "element:" + v.Tag
	case *h.ComponentNode:
		return "component:" + v.ID
	case *h.FragmentNode:
		return "fragment"
	case *h.TextNode:
		return "text"
	case *h.CommentNode:
		return "comment"
	default:
		return "unknown"
	}
}

func (b *structuredBuilder) processKeyedRowsSequential(rows []h.Node) []Row {
	rowEntries := make([]Row, 0, len(rows))
	domOffset := 0

	for _, row := range rows {
		if row == nil {
			continue
		}

		rowKey := nodeKey(row)
		snapshot := b.captureBuilderState()

		savedPathCalc := b.pathCalc
		b.pathCalc = b.pathCalc.CloneState()

		width := b.visitChildNode(row, domOffset)
		domOffset += width
		b.flush()

		savedPathCalc.mergeFrom(b.pathCalc, snapshot.dynamicsLen)
		b.pathCalc = savedPathCalc

		rowData := b.extractRowData(snapshot, row, rowKey)
		rowEntries = append(rowEntries, rowData)
	}

	return rowEntries
}

type childRenderResult struct {
	statics    []string
	dynamics   []DynamicSlot
	components map[string]ComponentSpan
	bindings   *BindingExtractor
	pathCalc   *PathCalculator
	builder    *structuredBuilder
	width      int
	err        error
}

type rowRenderResult struct {
	render  childRenderResult
	builder *structuredBuilder
	key     string
	node    h.Node
	err     error
}

func (b *structuredBuilder) processKeyedRowsConcurrent(rows []h.Node) []Row {
	const batchSize = 4
	sem := make(chan struct{}, b.opts.MaxRowWorkers)
	results := make([]rowRenderResult, len(rows))
	var wg sync.WaitGroup

	for batchStart := 0; batchStart < len(rows); batchStart += batchSize {
		batchEnd := batchStart + batchSize
		if batchEnd > len(rows) {
			batchEnd = len(rows)
		}

		wg.Add(1)
		sem <- struct{}{}

		go func(start, end int) {
			defer func() {
				if r := recover(); r != nil {
					for i := start; i < end; i++ {
						if results[i].err == nil {
							results[i] = rowRenderResult{err: &PanicError{Value: r}}
						}
					}
				}
				<-sem
				wg.Done()
			}()

			domOffset := 0
			for i := 0; i < start; i++ {
				domOffset += domWidth(rows[i])
			}

			for i := start; i < end; i++ {
				row := rows[i]
				if row == nil {
					domOffset += domWidth(row)
					continue
				}

				rowBuilder := b.cloneForRow()
				width := rowBuilder.visitChildNode(row, domOffset)
				rowBuilder.flush()

				results[i] = rowRenderResult{
					render: childRenderResult{
						statics:    rowBuilder.statics,
						dynamics:   rowBuilder.dynamics,
						components: rowBuilder.components,
						bindings:   rowBuilder.bindings,
						pathCalc:   rowBuilder.pathCalc,
						width:      width,
					},
					builder: rowBuilder,
					key:     nodeKey(row),
					node:    row,
				}

				domOffset += domWidth(row)
			}
		}(batchStart, batchEnd)
	}

	wg.Wait()

	totalStatics := 0
	totalDynamics := 0
	validResults := 0
	for _, result := range results {
		if result.err == nil && result.key != "" {
			totalStatics += len(result.render.statics)
			totalDynamics += len(result.render.dynamics)
			validResults++
		}
	}

	if cap(b.statics)-len(b.statics) < totalStatics {
		newStatics := make([]string, len(b.statics), len(b.statics)+totalStatics)
		copy(newStatics, b.statics)
		b.statics = newStatics
	}
	if cap(b.dynamics)-len(b.dynamics) < totalDynamics {
		newDynamics := make([]DynamicSlot, len(b.dynamics), len(b.dynamics)+totalDynamics)
		copy(newDynamics, b.dynamics)
		b.dynamics = newDynamics
	}

	rowEntries := make([]Row, 0, validResults)
	for _, result := range results {
		if result.err != nil {
			if result.builder != nil {
				putBuilder(result.builder)
			}
			continue
		}
		if result.key == "" {
			if result.builder != nil {
				putBuilder(result.builder)
			}
			continue
		}

		snapshot := b.captureBuilderState()
		b.mergeChildRenderResult(result.render)

		row := b.extractRowData(snapshot, result.node, result.key)
		rowEntries = append(rowEntries, row)

		putBuilder(result.builder)
	}

	return rowEntries
}

func (b *structuredBuilder) cloneForRow() *structuredBuilder {
	child := builderPool.Get().(*structuredBuilder)
	child.resetForReuse(b.tracker, b.pathCalc.CloneState(), b.opts)
	return child
}

func (b *structuredBuilder) resetForReuse(tracker PromotionTracker, pathCalc *PathCalculator, opts StructuredOptions) {
	b.tracker = tracker
	b.bindings = NewBindingExtractor()
	b.pathCalc = pathCalc
	b.opts = opts
	b.current.Reset()
	b.statics = b.statics[:0]
	b.dynamics = b.dynamics[:0]
	for k := range b.components {
		delete(b.components, k)
	}
	b.stack = b.stack[:0]
}

func putBuilder(b *structuredBuilder) {

	if cap(b.statics) > 256 {
		b.statics = nil
	}
	if cap(b.dynamics) > 64 {
		b.dynamics = nil
	}
	builderPool.Put(b)
}

func (b *structuredBuilder) visitChildrenConcurrent(children []h.Node, start int) int {
	b.flush()

	sem := make(chan struct{}, b.opts.MaxChildWorkers)
	results := make([]childRenderResult, len(children))
	var wg sync.WaitGroup

	offset := start
	for i, child := range children {
		if child == nil {
			continue
		}
		wg.Add(1)
		sem <- struct{}{}
		domIndex := offset
		go func(idx int, node h.Node, di int) {
			defer func() {
				if r := recover(); r != nil {
					results[idx] = childRenderResult{err: &PanicError{Value: r}}
				}
				<-sem
				wg.Done()
			}()

			childBuilder := b.cloneForChild()
			width := childBuilder.visitChildNode(node, di)
			childBuilder.flush()

			results[idx] = childRenderResult{
				statics:    childBuilder.statics,
				dynamics:   childBuilder.dynamics,
				components: childBuilder.components,
				bindings:   childBuilder.bindings,
				pathCalc:   childBuilder.pathCalc,
				builder:    childBuilder,
				width:      width,
			}
		}(i, child, domIndex)
		offset += domWidth(child)
	}

	wg.Wait()

	totalStatics := 0
	totalDynamics := 0
	for _, result := range results {
		if result.err == nil {
			totalStatics += len(result.statics)
			totalDynamics += len(result.dynamics)
		}
	}

	if cap(b.statics)-len(b.statics) < totalStatics {
		newStatics := make([]string, len(b.statics), len(b.statics)+totalStatics)
		copy(newStatics, b.statics)
		b.statics = newStatics
	}
	if cap(b.dynamics)-len(b.dynamics) < totalDynamics {
		newDynamics := make([]DynamicSlot, len(b.dynamics), len(b.dynamics)+totalDynamics)
		copy(newDynamics, b.dynamics)
		b.dynamics = newDynamics
	}

	totalWidth := offset - start
	for _, result := range results {
		if result.err != nil {
			if result.builder != nil {
				putBuilder(result.builder)
			}
			continue
		}

		b.mergeChildRenderResult(result)

		if result.builder != nil {
			putBuilder(result.builder)
		}
	}

	return totalWidth
}

func (b *structuredBuilder) cloneForChild() *structuredBuilder {
	child := builderPool.Get().(*structuredBuilder)
	child.resetForReuse(b.tracker, b.pathCalc.CloneState(), b.opts)
	return child
}

func (b *structuredBuilder) mergeChildRenderResult(result childRenderResult) {
	staticsOffset := len(b.statics)
	dynamicsOffset := len(b.dynamics)

	b.statics = append(b.statics, result.statics...)
	b.dynamics = append(b.dynamics, result.dynamics...)

	for id, span := range result.components {
		b.components[id] = ComponentSpan{
			StaticsStart:  span.StaticsStart + staticsOffset,
			StaticsEnd:    span.StaticsEnd + staticsOffset,
			DynamicsStart: span.DynamicsStart + dynamicsOffset,
			DynamicsEnd:   span.DynamicsEnd + dynamicsOffset,
		}
	}

	b.bindings.mergeFrom(result.bindings, dynamicsOffset)
	b.pathCalc.mergeFrom(result.pathCalc, dynamicsOffset)
}

type builderSnapshot struct {
	dynamicsLen      int
	bindingsLen      int
	slotPathsLen     int
	listPathsLen     int
	componentPathLen int
	uploadLen        int
	refLen           int
	routerLen        int
}

func (b *structuredBuilder) captureBuilderState() builderSnapshot {
	return builderSnapshot{
		dynamicsLen:      len(b.dynamics),
		bindingsLen:      b.bindings.handlerBindingsLen(),
		slotPathsLen:     b.bindings.slotPathsLen(),
		listPathsLen:     b.pathCalc.ListPathsLen(),
		componentPathLen: b.pathCalc.ComponentPathsLen(),
		uploadLen:        b.bindings.uploadBindingsLen(),
		refLen:           b.bindings.refBindingsLen(),
		routerLen:        b.bindings.routerBindingsLen(),
	}
}

func (b *structuredBuilder) extractRowData(snapshot builderSnapshot, row h.Node, rowKey string) Row {
	endDynamics := len(b.dynamics)

	slots := make([]int, 0, endDynamics-snapshot.dynamicsLen)
	for i := snapshot.dynamicsLen; i < endDynamics; i++ {
		slots = append(slots, i)
	}

	return Row{
		Key:            rowKey,
		HTML:           renderFinalizedNode(row),
		Slots:          slots,
		Bindings:       b.bindings.extractHandlerBindingsSlice(snapshot.bindingsLen),
		SlotPaths:      b.bindings.extractSlotPathsSlice(snapshot.slotPathsLen),
		ListPaths:      b.pathCalc.ExtractListPaths(snapshot.listPathsLen),
		ComponentPaths: b.pathCalc.ExtractComponentPaths(snapshot.componentPathLen),
		UploadBindings: b.bindings.extractUploadBindingsSlice(snapshot.uploadLen),
		RefBindings:    b.bindings.extractRefBindingsSlice(snapshot.refLen),
		RouterBindings: b.bindings.extractRouterBindingsSlice(snapshot.routerLen),
		RootCount:      countTopLevelNodes(row),
	}
}

func extractSlice[T any](slice []T, startIdx int) []T {
	endIdx := len(slice)
	if endIdx <= startIdx {
		return nil
	}
	return append([]T(nil), slice[startIdx:endIdx]...)
}

// Finalization helpers – minimal versions scoped to this package.

func FinalizeWithHandlers(n h.Node, reg handlers.Registry) h.Node {
	finalizeNode(n, reg)
	return n
}

func finalizeNode(n h.Node, reg handlers.Registry) {
	switch v := n.(type) {
	case *h.Element:
		finalizeElement(v, reg)
		for _, child := range v.Children {
			if child != nil {
				finalizeNode(child, reg)
			}
		}
	case *h.FragmentNode:
		for _, child := range v.Children {
			if child != nil {
				finalizeNode(child, reg)
			}
		}
	case *h.ComponentNode:
		if v.Child != nil {
			finalizeNode(v.Child, reg)
		}
	}
}

func finalizeElement(e *h.Element, reg handlers.Registry) {
	if e == nil {
		return
	}
	attachHandlers(e, reg)
}

func attachHandlers(e *h.Element, reg handlers.Registry) {
	if e == nil || len(e.Events) == 0 || reg == nil {
		return
	}
	if e.HandlerAssignments == nil {
		e.HandlerAssignments = map[string]dom.EventAssignment{}
	}
	keys := make([]string, 0, len(e.Events))
	for k := range e.Events {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		binding := e.Events[name]
		id := reg.Ensure(binding.Handler, binding.Key)
		if id == "" {
			continue
		}
		e.HandlerAssignments[name] = dom.EventAssignment{
			ID:     string(id),
			Listen: append([]string(nil), binding.Listen...),
			Props:  append([]string(nil), binding.Props...),
		}
	}
}
