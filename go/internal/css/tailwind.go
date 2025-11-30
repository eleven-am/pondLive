package css

import (
	"regexp"
	"strings"
)

var conflictGroupPatterns = map[string]*regexp.Regexp{

	"padding":   regexp.MustCompile(`^p-`),
	"padding-x": regexp.MustCompile(`^px-`),
	"padding-y": regexp.MustCompile(`^py-`),
	"padding-s": regexp.MustCompile(`^ps-`),
	"padding-e": regexp.MustCompile(`^pe-`),
	"padding-t": regexp.MustCompile(`^pt-`),
	"padding-r": regexp.MustCompile(`^pr-`),
	"padding-b": regexp.MustCompile(`^pb-`),
	"padding-l": regexp.MustCompile(`^pl-`),

	"margin":   regexp.MustCompile(`^m-`),
	"margin-x": regexp.MustCompile(`^mx-`),
	"margin-y": regexp.MustCompile(`^my-`),
	"margin-s": regexp.MustCompile(`^ms-`),
	"margin-e": regexp.MustCompile(`^me-`),
	"margin-t": regexp.MustCompile(`^mt-`),
	"margin-r": regexp.MustCompile(`^mr-`),
	"margin-b": regexp.MustCompile(`^mb-`),
	"margin-l": regexp.MustCompile(`^ml-`),

	"gap":     regexp.MustCompile(`^gap-(\d|px|\[)`),
	"gap-x":   regexp.MustCompile(`^gap-x-`),
	"gap-y":   regexp.MustCompile(`^gap-y-`),
	"space-x": regexp.MustCompile(`^space-x-`),
	"space-y": regexp.MustCompile(`^space-y-`),

	"inset":   regexp.MustCompile(`^inset-(\d|-)`),
	"inset-x": regexp.MustCompile(`^inset-x-`),
	"inset-y": regexp.MustCompile(`^inset-y-`),
	"top":     regexp.MustCompile(`^top-`),
	"right":   regexp.MustCompile(`^right-`),
	"bottom":  regexp.MustCompile(`^bottom-`),
	"left":    regexp.MustCompile(`^left-`),
	"start":   regexp.MustCompile(`^start-`),
	"end":     regexp.MustCompile(`^end-`),

	"width":     regexp.MustCompile(`^w-(\d|px|full|screen|min|max|fit|svw|lvw|dvw|\[|1/|2/|3/|4/|5/|6/|11/|12/)`),
	"min-width": regexp.MustCompile(`^min-w-`),
	"max-width": regexp.MustCompile(`^max-w-`),

	"height":     regexp.MustCompile(`^h-(\d|px|full|screen|min|max|fit|svh|lvh|dvh|\[|1/|2/|3/|4/|5/|6/)`),
	"min-height": regexp.MustCompile(`^min-h-`),
	"max-height": regexp.MustCompile(`^max-h-`),

	"size": regexp.MustCompile(`^size-`),

	"font-size": regexp.MustCompile(`^text-(xs|sm|base|lg|xl|2xl|3xl|4xl|5xl|6xl|7xl|8xl|9xl|\[)`),

	"font-weight": regexp.MustCompile(`^font-(thin|extralight|light|normal|medium|semibold|bold|extrabold|black|\d)`),

	"line-height": regexp.MustCompile(`^leading-`),

	"letter-spacing": regexp.MustCompile(`^tracking-`),

	"text-align": regexp.MustCompile(`^text-(left|center|right|justify|start|end)$`),

	"text-decoration": regexp.MustCompile(`^(underline|overline|line-through|no-underline)$`),

	"text-transform": regexp.MustCompile(`^(uppercase|lowercase|capitalize|normal-case)$`),

	"font-style": regexp.MustCompile(`^(italic|not-italic)$`),

	"font-family": regexp.MustCompile(`^font-(sans|serif|mono)$`),

	"text-color": regexp.MustCompile(`^text-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^text-\[`),

	"bg-color": regexp.MustCompile(`^bg-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^bg-\[`),

	"border-color": regexp.MustCompile(`^border-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^border-\[`),

	"ring-color": regexp.MustCompile(`^ring-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^ring-\[`),

	"outline-color": regexp.MustCompile(`^outline-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^outline-\[`),

	"display": regexp.MustCompile(`^(block|inline-block|inline|flex|inline-flex|table|inline-table|table-caption|table-cell|table-column|table-column-group|table-footer-group|table-header-group|table-row-group|table-row|flow-root|grid|inline-grid|contents|list-item|hidden)$`),

	"position": regexp.MustCompile(`^(static|fixed|absolute|relative|sticky)$`),

	"overflow":   regexp.MustCompile(`^overflow-(auto|hidden|clip|visible|scroll)$`),
	"overflow-x": regexp.MustCompile(`^overflow-x-`),
	"overflow-y": regexp.MustCompile(`^overflow-y-`),

	"z-index": regexp.MustCompile(`^z-`),

	"flex-direction": regexp.MustCompile(`^flex-(row|row-reverse|col|col-reverse)$`),

	"flex-wrap": regexp.MustCompile(`^flex-(wrap|wrap-reverse|nowrap)$`),

	"flex": regexp.MustCompile(`^flex-(1|auto|initial|none|\[)`),

	"flex-grow": regexp.MustCompile(`^grow(-0|-\[)?`),

	"flex-shrink": regexp.MustCompile(`^shrink(-0|-\[)?`),

	"order": regexp.MustCompile(`^order-`),

	"grid-template-columns": regexp.MustCompile(`^grid-cols-`),

	"grid-template-rows": regexp.MustCompile(`^grid-rows-`),

	"grid-column": regexp.MustCompile(`^col-(auto|span-|start-|end-)`),

	"grid-row": regexp.MustCompile(`^row-(auto|span-|start-|end-)`),

	"justify-content": regexp.MustCompile(`^justify-(normal|start|end|center|between|around|evenly|stretch)$`),

	"justify-items": regexp.MustCompile(`^justify-items-(start|end|center|stretch)$`),

	"justify-self": regexp.MustCompile(`^justify-self-(auto|start|end|center|stretch)$`),

	"align-content": regexp.MustCompile(`^content-(normal|center|start|end|between|around|evenly|baseline|stretch)$`),

	"align-items": regexp.MustCompile(`^items-(start|end|center|baseline|stretch)$`),

	"align-self": regexp.MustCompile(`^self-(auto|start|end|center|stretch|baseline)$`),

	"place-content": regexp.MustCompile(`^place-content-(center|start|end|between|around|evenly|baseline|stretch)$`),

	"place-items": regexp.MustCompile(`^place-items-(start|end|center|baseline|stretch)$`),

	"place-self": regexp.MustCompile(`^place-self-(auto|start|end|center|stretch)$`),

	"border":   regexp.MustCompile(`^border-(\d|\[)`),
	"border-x": regexp.MustCompile(`^border-x-`),
	"border-y": regexp.MustCompile(`^border-y-`),
	"border-s": regexp.MustCompile(`^border-s-`),
	"border-e": regexp.MustCompile(`^border-e-`),
	"border-t": regexp.MustCompile(`^border-t-(\d|\[)`),
	"border-r": regexp.MustCompile(`^border-r-(\d|\[)`),
	"border-b": regexp.MustCompile(`^border-b-(\d|\[)`),
	"border-l": regexp.MustCompile(`^border-l-(\d|\[)`),

	"rounded":    regexp.MustCompile(`^rounded-(\d|none|sm|md|lg|xl|2xl|3xl|full|\[)`),
	"rounded-s":  regexp.MustCompile(`^rounded-s-`),
	"rounded-e":  regexp.MustCompile(`^rounded-e-`),
	"rounded-t":  regexp.MustCompile(`^rounded-t-`),
	"rounded-r":  regexp.MustCompile(`^rounded-r-`),
	"rounded-b":  regexp.MustCompile(`^rounded-b-`),
	"rounded-l":  regexp.MustCompile(`^rounded-l-`),
	"rounded-ss": regexp.MustCompile(`^rounded-ss-`),
	"rounded-se": regexp.MustCompile(`^rounded-se-`),
	"rounded-ee": regexp.MustCompile(`^rounded-ee-`),
	"rounded-es": regexp.MustCompile(`^rounded-es-`),
	"rounded-tl": regexp.MustCompile(`^rounded-tl-`),
	"rounded-tr": regexp.MustCompile(`^rounded-tr-`),
	"rounded-br": regexp.MustCompile(`^rounded-br-`),
	"rounded-bl": regexp.MustCompile(`^rounded-bl-`),

	"border-style": regexp.MustCompile(`^border-(solid|dashed|dotted|double|hidden|none)$`),

	"opacity": regexp.MustCompile(`^opacity-`),

	"box-shadow": regexp.MustCompile(`^shadow-`),

	"blur": regexp.MustCompile(`^blur-`),

	"backdrop-blur": regexp.MustCompile(`^backdrop-blur-`),

	"ring-width":        regexp.MustCompile(`^ring-(\d|\[)`),
	"ring-offset-width": regexp.MustCompile(`^ring-offset-`),
}

func getConflictGroup(class string) string {

	for group, pattern := range conflictGroupPatterns {
		if pattern.MatchString(class) {
			return group
		}
	}
	return ""
}

func splitVariantAndClass(class string) (variant, baseClass string) {

	parts := strings.Split(class, ":")
	if len(parts) == 1 {
		return "", class
	}

	return strings.Join(parts[:len(parts)-1], ":"), parts[len(parts)-1]
}
