//go:build ignore

package main

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

var tags = []struct {
	Name string
	Tag  string
	Doc  string
}{
	{"A", "a", "A creates an <a> element."},
	{"Abbr", "abbr", "Abbr creates an <abbr> element."},
	{"Address", "address", "Address creates an <address> element."},
	{"Area", "area", "Area creates an <area> element."},
	{"Article", "article", "Article creates an <article> element."},
	{"Aside", "aside", "Aside creates an <aside> element."},
	{"Audio", "audio", "Audio creates an <audio> element."},
	{"B", "b", "B creates a <b> element."},
	{"Base", "base", "Base creates a <base> element."},
	{"Bdi", "bdi", "Bdi creates a <bdi> element."},
	{"Bdo", "bdo", "Bdo creates a <bdo> element."},
	{"Blockquote", "blockquote", "Blockquote creates a <blockquote> element."},
	{"Body", "body", "Body creates a <body> element."},
	{"Br", "br", "Br creates a <br> element."},
	{"Button", "button", "Button creates a <button> element."},
	{"Canvas", "canvas", "Canvas creates a <canvas> element."},
	{"Caption", "caption", "Caption creates a <caption> element."},
	{"Cite", "cite", "Cite creates a <cite> element."},
	{"Code", "code", "Code creates a <code> element."},
	{"Col", "col", "Col creates a <col> element."},
	{"Colgroup", "colgroup", "Colgroup creates a <colgroup> element."},
	{"DataEl", "data", "DataEl creates a <data> element."},
	{"Datalist", "datalist", "Datalist creates a <datalist> element."},
	{"Dd", "dd", "Dd creates a <dd> element."},
	{"DelEl", "del", "DelEl creates a <del> element."},
	{"Details", "details", "Details creates a <details> element."},
	{"Dfn", "dfn", "Dfn creates a <dfn> element."},
	{"Dialog", "dialog", "Dialog creates a <dialog> element."},
	{"Div", "div", "Div creates a <div> element."},
	{"Dl", "dl", "Dl creates a <dl> element."},
	{"Dt", "dt", "Dt creates a <dt> element."},
	{"Em", "em", "Em creates an <em> element."},
	{"Embed", "embed", "Embed creates an <embed> element."},
	{"Fieldset", "fieldset", "Fieldset creates a <fieldset> element."},
	{"Figcaption", "figcaption", "Figcaption creates a <figcaption> element."},
	{"Figure", "figure", "Figure creates a <figure> element."},
	{"Footer", "footer", "Footer creates a <footer> element."},
	{"Form", "form", "Form creates a <form> element."},
	{"H1", "h1", "H1 creates an <h1> element."},
	{"H2", "h2", "H2 creates an <h2> element."},
	{"H3", "h3", "H3 creates an <h3> element."},
	{"H4", "h4", "H4 creates an <h4> element."},
	{"H5", "h5", "H5 creates an <h5> element."},
	{"H6", "h6", "H6 creates an <h6> element."},
	{"Head", "head", "Head creates a <head> element."},
	{"Header", "header", "Header creates a <header> element."},
	{"Hgroup", "hgroup", "Hgroup creates an <hgroup> element."},
	{"Hr", "hr", "Hr creates an <hr> element."},
	{"Html", "html", "Html creates an <html> element."},
	{"I", "i", "I creates an <i> element."},
	{"Iframe", "iframe", "Iframe creates an <iframe> element."},
	{"Img", "img", "Img creates an <img> element."},
	{"Input", "input", "Input creates an <input> element."},
	{"InsEl", "ins", "InsEl creates an <ins> element."},
	{"Kbd", "kbd", "Kbd creates a <kbd> element."},
	{"Label", "label", "Label creates a <label> element."},
	{"Legend", "legend", "Legend creates a <legend> element."},
	{"Li", "li", "Li creates an <li> element."},
	{"Link", "link", "Link creates a <link> element."},
	{"Main", "main", "Main creates a <main> element."},
	{"MapEl", "map", "MapEl creates a <map> element."},
	{"Mark", "mark", "Mark creates a <mark> element."},
	{"Meta", "meta", "Meta creates a <meta> element."},
	{"Meter", "meter", "Meter creates a <meter> element."},
	{"Nav", "nav", "Nav creates a <nav> element."},
	{"Noscript", "noscript", "Noscript creates a <noscript> element."},
	{"Object", "object", "Object creates an <object> element."},
	{"Ol", "ol", "Ol creates an <ol> element."},
	{"Optgroup", "optgroup", "Optgroup creates an <optgroup> element."},
	{"Option", "option", "Option creates an <option> element."},
	{"Output", "output", "Output creates an <output> element."},
	{"P", "p", "P creates a <p> element."},
	{"Picture", "picture", "Picture creates a <picture> element."},
	{"Pre", "pre", "Pre creates a <pre> element."},
	{"Progress", "progress", "Progress creates a <progress> element."},
	{"Q", "q", "Q creates a <q> element."},
	{"Rp", "rp", "Rp creates an <rp> element."},
	{"Rt", "rt", "Rt creates an <rt> element."},
	{"Ruby", "ruby", "Ruby creates a <ruby> element."},
	{"S", "s", "S creates an <s> element."},
	{"Samp", "samp", "Samp creates a <samp> element."},
	{"Script", "script", "Script creates a <script> element."},
	{"Section", "section", "Section creates a <section> element."},
	{"Select", "select", "Select creates a <select> element."},
	{"Slot", "slot", "Slot creates a <slot> element."},
	{"Small", "small", "Small creates a <small> element."},
	{"Source", "source", "Source creates a <source> element."},
	{"Span", "span", "Span creates a <span> element."},
	{"Strong", "strong", "Strong creates a <strong> element."},
	{"StyleEl", "style", "StyleEl creates a <style> element."},
	{"Sub", "sub", "Sub creates a <sub> element."},
	{"Summary", "summary", "Summary creates a <summary> element."},
	{"Sup", "sup", "Sup creates a <sup> element."},
	{"Table", "table", "Table creates a <table> element."},
	{"Tbody", "tbody", "Tbody creates a <tbody> element."},
	{"Td", "td", "Td creates a <td> element."},
	{"Template", "template", "Template creates a <template> element."},
	{"Textarea", "textarea", "Textarea creates a <textarea> element."},
	{"Tfoot", "tfoot", "Tfoot creates a <tfoot> element."},
	{"Th", "th", "Th creates a <th> element."},
	{"Thead", "thead", "Thead creates a <thead> element."},
	{"Time", "time", "Time creates a <time> element."},
	{"TitleEl", "title", "TitleEl creates a <title> element."},
	{"Tr", "tr", "Tr creates a <tr> element."},
	{"Track", "track", "Track creates a <track> element."},
	{"U", "u", "U creates a <u> element."},
	{"Ul", "ul", "Ul creates a <ul> element."},
	{"Var", "var", "Var creates a <var> element."},
	{"Video", "video", "Video creates a <video> element."},
	{"Wbr", "wbr", "Wbr creates a <wbr> element."},

	// SVG elements
	{"Svg", "svg", "Svg creates an <svg> element."},
	{"Circle", "circle", "Circle creates a <circle> element."},
	{"ClipPath", "clipPath", "ClipPath creates a <clipPath> element."},
	{"Defs", "defs", "Defs creates a <defs> element."},
	{"Ellipse", "ellipse", "Ellipse creates an <ellipse> element."},
	{"G", "g", "G creates a <g> element."},
	{"Line", "line", "Line creates a <line> element."},
	{"LinearGradient", "linearGradient", "LinearGradient creates a <linearGradient> element."},
	{"Marker", "marker", "Marker creates a <marker> element."},
	{"Mask", "mask", "Mask creates a <mask> element."},
	{"Path", "path", "Path creates a <path> element."},
	{"Pattern", "pattern", "Pattern creates a <pattern> element."},
	{"Polygon", "polygon", "Polygon creates a <polygon> element."},
	{"Polyline", "polyline", "Polyline creates a <polyline> element."},
	{"RadialGradient", "radialGradient", "RadialGradient creates a <radialGradient> element."},
	{"Rect", "rect", "Rect creates a <rect> element."},
	{"Stop", "stop", "Stop creates a <stop> element."},
	{"SvgText", "text", "SvgText creates a <text> element."},
	{"TSpan", "tspan", "TSpan creates a <tspan> element."},
	{"Use", "use", "Use creates a <use> element."},
}

func main() {
	sort.Slice(tags, func(i, j int) bool { return tags[i].Name < tags[j].Name })
	var b strings.Builder
	b.WriteString("// Code generated by tags_gen.go; DO NOT EDIT.\n")
	b.WriteString("package html\n\n")
	for _, t := range tags {
		b.WriteString("// ")
		b.WriteString(t.Doc)
		b.WriteString("\n")
		b.WriteString("func ")
		b.WriteString(t.Name)
		b.WriteString("(items ...Item) *Element { return el(\"")
		b.WriteString(t.Tag)
		b.WriteString("\", items...) }\n\n")
	}
	if err := os.WriteFile("tags_generated.go", []byte(b.String()), 0o644); err != nil {
		panic(err)
	}
	fmt.Println("generated tags_generated.go")
}
