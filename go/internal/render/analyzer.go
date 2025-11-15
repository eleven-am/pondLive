package render

import (
	"fmt"
	"runtime"
	"sync"

	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

// PanicError wraps a panic value from concurrent analysis
type PanicError struct {
	Value interface{}
}

func (e *PanicError) Error() string {
	return fmt.Sprintf("panic during concurrent analysis: %v", e.Value)
}

type AnalysisResult struct {
	StaticsCapacity  int
	DynamicsCapacity int
	Components       map[string]ComponentSpan
}

type AnalysisOptions struct {
	// Concurrent enables parallel analysis of sibling nodes
	Concurrent bool
	// ConcurrencyThreshold is minimum number of siblings to parallelize (default: 4)
	ConcurrencyThreshold int
	// MaxWorkers limits concurrent goroutines (default: GOMAXPROCS)
	// Use 0 for unlimited workers (not recommended for large trees)
	MaxWorkers int
}

type ComponentAnalyzer struct {
	staticsCount  int
	dynamicsCount int
	components    map[string]ComponentSpan
	inComponent   string
	componentSpan *ComponentSpan
	opts          AnalysisOptions
}

func NewComponentAnalyzer() *ComponentAnalyzer {
	return &ComponentAnalyzer{
		components: make(map[string]ComponentSpan),
		opts: AnalysisOptions{
			Concurrent:           false,
			ConcurrencyThreshold: 4,
		},
	}
}

func NewComponentAnalyzerWithOptions(opts AnalysisOptions) *ComponentAnalyzer {
	if opts.ConcurrencyThreshold <= 0 {
		opts.ConcurrencyThreshold = 4
	}
	if opts.MaxWorkers <= 0 {
		opts.MaxWorkers = runtime.GOMAXPROCS(0)
	}
	return &ComponentAnalyzer{
		components: make(map[string]ComponentSpan),
		opts:       opts,
	}
}

func (a *ComponentAnalyzer) Analyze(root h.Node) AnalysisResult {
	a.visit(root)
	return AnalysisResult{
		StaticsCapacity:  a.staticsCount,
		DynamicsCapacity: a.dynamicsCount,
		Components:       a.components,
	}
}

func (a *ComponentAnalyzer) visit(n h.Node) {
	if n == nil {
		return
	}

	switch v := n.(type) {
	case *h.TextNode:
		a.visitText(v)
	case *h.Element:
		a.visitElement(v)
	case *h.FragmentNode:
		a.visitFragment(v)
	case *h.CommentNode:
		a.visitComment(v)
	case *h.ComponentNode:
		a.visitComponent(v)
	}
}

func (a *ComponentAnalyzer) visitText(t *h.TextNode) {
	if t == nil {
		return
	}
	if t.Mutable {
		a.dynamicsCount++
	}
	a.staticsCount++
}

func (a *ComponentAnalyzer) visitComment(c *h.CommentNode) {
	if c == nil {
		return
	}
	a.staticsCount++
}

func (a *ComponentAnalyzer) visitComponent(v *h.ComponentNode) {
	if v == nil {
		return
	}

	if v.ID == "" {
		if v.Child != nil {
			a.visit(v.Child)
		}
		return
	}

	prevComponent := a.inComponent
	prevSpan := a.componentSpan

	a.inComponent = v.ID
	staticsStart := a.staticsCount
	dynamicsStart := a.dynamicsCount

	span := ComponentSpan{
		StaticsStart:  staticsStart,
		DynamicsStart: dynamicsStart,
	}
	a.componentSpan = &span

	if v.Child != nil {
		a.visit(v.Child)
	}

	span.StaticsEnd = a.staticsCount
	span.DynamicsEnd = a.dynamicsCount
	a.components[v.ID] = span

	a.inComponent = prevComponent
	a.componentSpan = prevSpan
}

func (a *ComponentAnalyzer) visitElement(v *h.Element) {
	if v == nil {
		return
	}

	a.staticsCount++

	if a.shouldUseDynamicAttrs(v) {
		a.dynamicsCount++
	}

	if v.Unsafe != nil {
		return
	}

	if a.hasKeyedChildren(v.Children) {
		a.dynamicsCount++
		keyedChildren := make([]h.Node, 0)
		for _, child := range v.Children {
			if el, ok := child.(*h.Element); ok && el.Key != "" {
				keyedChildren = append(keyedChildren, child)
			}
		}
		a.analyzeChildren(keyedChildren)
		return
	}

	a.analyzeChildren(v.Children)
}

func (a *ComponentAnalyzer) visitFragment(f *h.FragmentNode) {
	if f == nil {
		return
	}

	if a.hasKeyedChildren(f.Children) {
		a.dynamicsCount++
		keyedChildren := make([]h.Node, 0)
		for _, child := range f.Children {
			if el, ok := child.(*h.Element); ok && el.Key != "" {
				keyedChildren = append(keyedChildren, child)
			}
		}
		a.analyzeChildren(keyedChildren)
		return
	}

	a.analyzeChildren(f.Children)
}

func (a *ComponentAnalyzer) shouldUseDynamicAttrs(el *h.Element) bool {
	if el == nil {
		return false
	}
	if len(el.HandlerAssignments) > 0 {
		return true
	}
	for _, mutable := range el.MutableAttrs {
		if mutable {
			return true
		}
	}
	if len(el.UploadBindings) > 0 {
		return true
	}
	if el.RefID != "" {
		return true
	}
	if el.Attrs != nil {
		if _, hasPath := el.Attrs["data-router-path"]; hasPath {
			return true
		}
		if _, hasQuery := el.Attrs["data-router-query"]; hasQuery {
			return true
		}
		if _, hasHash := el.Attrs["data-router-hash"]; hasHash {
			return true
		}
		if _, hasReplace := el.Attrs["data-router-replace"]; hasReplace {
			return true
		}
	}
	return false
}

func (a *ComponentAnalyzer) hasKeyedChildren(children []h.Node) bool {
	for _, child := range children {
		if el, ok := child.(*h.Element); ok && el.Key != "" {
			return true
		}
	}
	return false
}

// childAnalysisResult holds the counts from analyzing a single child
type childAnalysisResult struct {
	staticsCount  int
	dynamicsCount int
	components    map[string]ComponentSpan
	err           error
}

// analyzeChildren processes children either sequentially or concurrently
func (a *ComponentAnalyzer) analyzeChildren(children []h.Node) {
	if !a.opts.Concurrent || len(children) < a.opts.ConcurrencyThreshold {
		for _, child := range children {
			a.visit(child)
		}
		return
	}

	sem := make(chan struct{}, a.opts.MaxWorkers)
	results := make([]childAnalysisResult, len(children))
	var wg sync.WaitGroup
	wg.Add(len(children))

	for i, child := range children {
		sem <- struct{}{}
		go func(idx int, node h.Node) {
			defer func() {
				if r := recover(); r != nil {
					results[idx] = childAnalysisResult{
						err: &PanicError{Value: r},
					}
				}
				<-sem
				wg.Done()
			}()
			childAnalyzer := NewComponentAnalyzerWithOptions(a.opts)
			childAnalyzer.visit(node)
			results[idx] = childAnalysisResult{
				staticsCount:  childAnalyzer.staticsCount,
				dynamicsCount: childAnalyzer.dynamicsCount,
				components:    childAnalyzer.components,
			}
		}(i, child)
	}

	wg.Wait()

	for _, result := range results {
		staticsOffset := a.staticsCount
		dynamicsOffset := a.dynamicsCount

		a.staticsCount += result.staticsCount
		a.dynamicsCount += result.dynamicsCount

		for id, span := range result.components {
			adjustedSpan := ComponentSpan{
				StaticsStart:  span.StaticsStart + staticsOffset,
				StaticsEnd:    span.StaticsEnd + staticsOffset,
				DynamicsStart: span.DynamicsStart + dynamicsOffset,
				DynamicsEnd:   span.DynamicsEnd + dynamicsOffset,
			}
			a.components[id] = adjustedSpan
		}
	}
}
