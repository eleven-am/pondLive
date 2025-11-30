//go:build ignore

package main

import (
	"fmt"
	"go/format"
	"os"
	"sort"
	"strings"
)

type tagSpec struct {
	Name   string
	Tag    string
	Kind   string
	Mixins []string
}

type actionSpec struct {
	Type        string
	Constructor string
	Field       string
}

var baseActions = []actionSpec{
	{"ElementActions", "NewElementActions", "ElementActions"},
	{"ScrollActions", "NewScrollActions", "ScrollActions"},
}

var mixinActions = map[string][]actionSpec{
	"media":       {{"MediaActions", "NewMediaActions", "MediaActions"}},
	"form":        {{"FormActions", "NewFormActions", "FormActions"}},
	"details":     {{"DetailsActions", "NewDetailsActions", "DetailsActions"}},
	"value":       {{"ValueActions", "NewValueActions", "ValueActions"}},
	"dialog":      {{"DialogActions", "NewDialogActions", "DialogActions"}},
	"canvas":      {{"CanvasActions", "NewCanvasActions", "CanvasActions"}},
	"disableable": {{"DisableableActions", "NewDisableableActions", "DisableableActions"}},
	"selection":   {{"SelectionActions", "NewSelectionActions", "SelectionActions"}},
}

func actionsForTag(spec tagSpec) []actionSpec {
	result := append([]actionSpec{}, baseActions...)
	seen := map[string]bool{"ElementActions": true, "ScrollActions": true}

	for _, mixin := range spec.Mixins {
		if actions, ok := mixinActions[mixin]; ok {
			for _, action := range actions {
				if !seen[action.Type] {
					result = append(result, action)
					seen[action.Type] = true
				}
			}
		}
	}
	return result
}

var tags = []tagSpec{

	{"A", "a", "pkg", nil},
	{"Abbr", "abbr", "pkg", nil},
	{"Address", "address", "pkg", nil},
	{"Area", "area", "pkg", nil},
	{"Article", "article", "pkg", nil},
	{"Aside", "aside", "pkg", nil},
	{"Audio", "audio", "pkg", []string{"media"}},
	{"B", "b", "pkg", nil},
	{"Base", "base", "pkg", nil},
	{"Bdi", "bdi", "pkg", nil},
	{"Bdo", "bdo", "pkg", nil},
	{"Blockquote", "blockquote", "pkg", nil},
	{"Body", "body", "pkg", nil},
	{"Br", "br", "pkg", nil},
	{"Button", "button", "pkg", []string{"disableable"}},
	{"Canvas", "canvas", "pkg", []string{"canvas"}},
	{"Caption", "caption", "pkg", nil},
	{"Cite", "cite", "pkg", nil},
	{"Code", "code", "pkg", nil},
	{"Col", "col", "pkg", nil},
	{"Colgroup", "colgroup", "pkg", nil},
	{"DataEl", "data", "pkg", nil},
	{"Datalist", "datalist", "pkg", nil},
	{"Dd", "dd", "pkg", nil},
	{"DelEl", "del", "pkg", nil},
	{"Details", "details", "pkg", []string{"details"}},
	{"Dfn", "dfn", "pkg", nil},
	{"Dialog", "dialog", "pkg", []string{"dialog"}},
	{"Div", "div", "pkg", nil},
	{"Dl", "dl", "pkg", nil},
	{"Dt", "dt", "pkg", nil},
	{"Em", "em", "pkg", nil},
	{"Embed", "embed", "pkg", nil},
	{"Fieldset", "fieldset", "pkg", []string{"disableable"}},
	{"Figcaption", "figcaption", "pkg", nil},
	{"Figure", "figure", "pkg", nil},
	{"Footer", "footer", "pkg", nil},
	{"Form", "form", "pkg", []string{"form"}},
	{"H1", "h1", "pkg", nil},
	{"H2", "h2", "pkg", nil},
	{"H3", "h3", "pkg", nil},
	{"H4", "h4", "pkg", nil},
	{"H5", "h5", "pkg", nil},
	{"H6", "h6", "pkg", nil},
	{"Head", "head", "pkg", nil},
	{"Header", "header", "pkg", nil},
	{"Hgroup", "hgroup", "pkg", nil},
	{"Hr", "hr", "pkg", nil},
	{"Html", "pkg", "pkg", nil},
	{"I", "i", "pkg", nil},
	{"Iframe", "iframe", "pkg", nil},
	{"Img", "img", "pkg", nil},
	{"Input", "input", "pkg", []string{"value", "disableable", "selection"}},
	{"InsEl", "ins", "pkg", nil},
	{"Kbd", "kbd", "pkg", nil},
	{"Label", "label", "pkg", nil},
	{"Legend", "legend", "pkg", nil},
	{"Li", "li", "pkg", nil},
	{"LinkEl", "link", "pkg", nil},
	{"Main", "main", "pkg", nil},
	{"MapEl", "map", "pkg", nil},
	{"Mark", "mark", "pkg", nil},
	{"Menu", "menu", "pkg", nil},
	{"MenuItem", "menuitem", "pkg", nil},
	{"MetaEl", "meta", "pkg", nil},
	{"Meter", "meter", "pkg", []string{"value"}},
	{"Nav", "nav", "pkg", nil},
	{"Noscript", "noscript", "pkg", nil},
	{"Object", "object", "pkg", nil},
	{"Ol", "ol", "pkg", nil},
	{"Optgroup", "optgroup", "pkg", nil},
	{"Option", "option", "pkg", nil},
	{"Output", "output", "pkg", nil},
	{"P", "p", "pkg", nil},
	{"Param", "param", "pkg", nil},
	{"Picture", "picture", "pkg", nil},
	{"Portal", "portal", "pkg", nil},
	{"Pre", "pre", "pkg", nil},
	{"Progress", "progress", "pkg", []string{"value"}},
	{"Q", "q", "pkg", nil},
	{"Rb", "rb", "pkg", nil},
	{"Rp", "rp", "pkg", nil},
	{"Rt", "rt", "pkg", nil},
	{"Rtc", "rtc", "pkg", nil},
	{"Ruby", "ruby", "pkg", nil},
	{"S", "s", "pkg", nil},
	{"Samp", "samp", "pkg", nil},
	{"ScriptEl", "script", "pkg", nil},
	{"Section", "section", "pkg", nil},
	{"Select", "select", "pkg", []string{"value", "disableable"}},
	{"Slot", "slot", "pkg", nil},
	{"Small", "small", "pkg", nil},
	{"Source", "source", "pkg", nil},
	{"Span", "span", "pkg", nil},
	{"Strong", "strong", "pkg", nil},
	{"StyleEl", "style", "pkg", nil},
	{"Sub", "sub", "pkg", nil},
	{"Summary", "summary", "pkg", nil},
	{"Sup", "sup", "pkg", nil},
	{"Table", "table", "pkg", nil},
	{"Tbody", "tbody", "pkg", nil},
	{"Td", "td", "pkg", nil},
	{"Template", "template", "pkg", nil},
	{"Textarea", "textarea", "pkg", []string{"value", "disableable", "selection"}},
	{"Tfoot", "tfoot", "pkg", nil},
	{"Th", "th", "pkg", nil},
	{"Thead", "thead", "pkg", nil},
	{"Time", "time", "pkg", nil},
	{"TitleEl", "title", "pkg", nil},
	{"Tr", "tr", "pkg", nil},
	{"Track", "track", "pkg", nil},
	{"U", "u", "pkg", nil},
	{"Ul", "ul", "pkg", nil},
	{"Var", "var", "pkg", nil},
	{"Video", "video", "pkg", []string{"media"}},
	{"Wbr", "wbr", "pkg", nil},

	{"Svg", "svg", "svg", nil},
	{"Circle", "circle", "svg", nil},
	{"ClipPath", "clipPath", "svg", nil},
	{"Defs", "defs", "svg", nil},
	{"Ellipse", "ellipse", "svg", nil},
	{"ForeignObject", "foreignObject", "svg", nil},
	{"G", "g", "svg", nil},
	{"Image", "image", "svg", nil},
	{"Line", "line", "svg", nil},
	{"LinearGradient", "linearGradient", "svg", nil},
	{"Marker", "marker", "svg", nil},
	{"Mask", "mask", "svg", nil},
	{"Path", "path", "svg", nil},
	{"Pattern", "pattern", "svg", nil},
	{"Polygon", "polygon", "svg", nil},
	{"Polyline", "polyline", "svg", nil},
	{"RadialGradient", "radialGradient", "svg", nil},
	{"Rect", "rect", "svg", nil},
	{"Stop", "stop", "svg", nil},
	{"SvgText", "text", "svg", nil},
	{"TSpan", "tspan", "svg", nil},
	{"SvgUse", "use", "svg", nil},
}

func main() {
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })

	generateTagBuilders(tags)
	generateRefs(tags)

	fmt.Println("generated tags_generated.go")
	fmt.Println("generated refs_generated.go")
}

func generateTagBuilders(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n\n")
	b.WriteString("package pkg\n\n")

	for _, spec := range specs {
		fmt.Fprintf(&b, "func %s(items ...Item) Node { return El(%q, items...) }\n", spec.Name, spec.Tag)
	}

	writeFormatted("tags_generated.go", b.String())
}

func generateRefs(specs []tagSpec) {
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n\n")
	b.WriteString("package pkg\n\n")

	for _, spec := range specs {
		refName := spec.Name + "Ref"
		actions := actionsForTag(spec)

		fmt.Fprintf(&b, "type %s struct {\n", refName)
		b.WriteString("\t*ElementRef\n")
		for _, action := range actions {
			fmt.Fprintf(&b, "\t*%s\n", action.Type)
		}
		b.WriteString("}\n\n")
	}

	for _, spec := range specs {
		refName := spec.Name + "Ref"
		hookName := "Use" + spec.Name
		actions := actionsForTag(spec)

		fmt.Fprintf(&b, "func %s(ctx *Ctx) *%s {\n", hookName, refName)
		b.WriteString("\tref := UseElement(ctx)\n")
		fmt.Fprintf(&b, "\treturn &%s{\n", refName)
		b.WriteString("\t\tElementRef: ref,\n")
		for _, action := range actions {
			fmt.Fprintf(&b, "\t\t%s: %s(ctx, ref),\n", action.Field, action.Constructor)
		}
		b.WriteString("\t}\n}\n\n")
	}

	writeFormatted("refs_generated.go", b.String())
}

func writeFormatted(target string, src string) {
	formatted, err := format.Source([]byte(src))
	if err != nil {
		fmt.Printf("Error formatting %s: %v\n", target, err)
		fmt.Println("Source:")
		fmt.Println(src)
		panic(err)
	}
	if err := os.WriteFile(target, formatted, 0o644); err != nil {
		panic(err)
	}
}
