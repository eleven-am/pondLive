package runtime

import (
	"strings"

	"github.com/eleven-am/pondlive/go/internal/css"
	h "github.com/eleven-am/pondlive/go/pkg/live/html"
)

type StyleLookup interface {
	Get(selector string) string
	Class(className string) string
	ID(idName string) string
	Call(args ...interface{}) string
	StyleTag() h.Node
	Rule(selector string) string
	AllRules() map[string]string
}

type StyleRule struct {
	Selector     string
	Declarations string
}

func Rule(selector, declarations string) StyleRule {
	return StyleRule{Selector: selector, Declarations: declarations}
}

type styleCell struct {
	css         string
	scopedCSS   string
	lookup      *styleLookupImpl
	componentID string
}

type styleLookupImpl struct {
	lookup    *css.StyleLookup
	scopedCSS string
}

func (s *styleLookupImpl) Get(selector string) string {
	return s.lookup.Get(selector)
}

func (s *styleLookupImpl) Class(className string) string {
	return s.lookup.Class(className)
}

func (s *styleLookupImpl) ID(idName string) string {
	return s.lookup.ID(idName)
}

func (s *styleLookupImpl) Call(args ...interface{}) string {
	return s.lookup.Call(args...)
}

func (s *styleLookupImpl) StyleTag() h.Node {
	return h.StyleEl(h.Text(s.scopedCSS))
}

func (s *styleLookupImpl) Rule(selector string) string {
	return s.lookup.Rule(selector)
}

func (s *styleLookupImpl) AllRules() map[string]string {
	return s.lookup.AllRules()
}

func UseStyles(ctx Ctx, cssString string) StyleLookup {
	if ctx.frame == nil {
		panic("runtime: UseStyles called outside render")
	}

	idx := ctx.frame.idx
	ctx.frame.idx++

	if idx >= len(ctx.frame.cells) {
		componentID := ""
		if ctx.comp != nil {
			componentID = ctx.comp.id
		}

		scoped := css.Scope(cssString, componentID)

		cell := &styleCell{
			css:         cssString,
			scopedCSS:   scoped.CSS,
			componentID: componentID,
		}

		cell.lookup = &styleLookupImpl{
			lookup:    css.NewStyleLookupWithRules(scoped.SelectorMap, scoped.RuleMap),
			scopedCSS: scoped.CSS,
		}

		ctx.frame.cells = append(ctx.frame.cells, cell)

		return cell.lookup
	}

	cell, ok := ctx.frame.cells[idx].(*styleCell)
	if !ok {
		panic("runtime: UseStyles cell type mismatch")
	}

	if cell.css != cssString {
		componentID := ""
		if ctx.comp != nil {
			componentID = ctx.comp.id
		}

		scoped := css.Scope(cssString, componentID)

		cell.css = cssString
		cell.scopedCSS = scoped.CSS
		cell.componentID = componentID

		cell.lookup = &styleLookupImpl{
			lookup:    css.NewStyleLookupWithRules(scoped.SelectorMap, scoped.RuleMap),
			scopedCSS: scoped.CSS,
		}
	}

	return cell.lookup
}

func UseStyleRules(ctx Ctx, rules ...StyleRule) StyleLookup {
	if len(rules) == 0 {
		return &styleLookupImpl{
			lookup:    css.NewStyleLookupWithRules(make(map[string]string), make(map[string]string)),
			scopedCSS: "",
		}
	}

	var cssBuilder strings.Builder
	for _, rule := range rules {
		cssBuilder.WriteString(rule.Selector)
		cssBuilder.WriteString(" { ")
		cssBuilder.WriteString(rule.Declarations)
		cssBuilder.WriteString(" }\n")
	}

	return UseStyles(ctx, cssBuilder.String())
}
