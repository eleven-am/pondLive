package css

import (
	"regexp"
	"strings"
)

// Tailwind CSS conflict groups define which utility classes conflict with each other.
// When multiple classes from the same conflict group are provided, only the last one is kept.

// conflictGroupPatterns maps conflict group names to their regex patterns.
var conflictGroupPatterns = map[string]*regexp.Regexp{
	// Padding
	"padding":   regexp.MustCompile(`^p-`),
	"padding-x": regexp.MustCompile(`^px-`),
	"padding-y": regexp.MustCompile(`^py-`),
	"padding-s": regexp.MustCompile(`^ps-`),
	"padding-e": regexp.MustCompile(`^pe-`),
	"padding-t": regexp.MustCompile(`^pt-`),
	"padding-r": regexp.MustCompile(`^pr-`),
	"padding-b": regexp.MustCompile(`^pb-`),
	"padding-l": regexp.MustCompile(`^pl-`),

	// Margin
	"margin":   regexp.MustCompile(`^m-`),
	"margin-x": regexp.MustCompile(`^mx-`),
	"margin-y": regexp.MustCompile(`^my-`),
	"margin-s": regexp.MustCompile(`^ms-`),
	"margin-e": regexp.MustCompile(`^me-`),
	"margin-t": regexp.MustCompile(`^mt-`),
	"margin-r": regexp.MustCompile(`^mr-`),
	"margin-b": regexp.MustCompile(`^mb-`),
	"margin-l": regexp.MustCompile(`^ml-`),

	// Spacing
	"gap":     regexp.MustCompile(`^gap-(\d|px|\[)`),
	"gap-x":   regexp.MustCompile(`^gap-x-`),
	"gap-y":   regexp.MustCompile(`^gap-y-`),
	"space-x": regexp.MustCompile(`^space-x-`),
	"space-y": regexp.MustCompile(`^space-y-`),

	// Inset
	"inset":   regexp.MustCompile(`^inset-(\d|-)`),
	"inset-x": regexp.MustCompile(`^inset-x-`),
	"inset-y": regexp.MustCompile(`^inset-y-`),
	"top":     regexp.MustCompile(`^top-`),
	"right":   regexp.MustCompile(`^right-`),
	"bottom":  regexp.MustCompile(`^bottom-`),
	"left":    regexp.MustCompile(`^left-`),
	"start":   regexp.MustCompile(`^start-`),
	"end":     regexp.MustCompile(`^end-`),

	// Sizing (including arbitrary values like w-[200px])
	"width":     regexp.MustCompile(`^w-(\d|px|full|screen|min|max|fit|svw|lvw|dvw|\[|1/|2/|3/|4/|5/|6/|11/|12/)`),
	"min-width": regexp.MustCompile(`^min-w-`),
	"max-width": regexp.MustCompile(`^max-w-`),

	"height":     regexp.MustCompile(`^h-(\d|px|full|screen|min|max|fit|svh|lvh|dvh|\[|1/|2/|3/|4/|5/|6/)`),
	"min-height": regexp.MustCompile(`^min-h-`),
	"max-height": regexp.MustCompile(`^max-h-`),

	"size": regexp.MustCompile(`^size-`),

	// Typography - Font size
	"font-size": regexp.MustCompile(`^text-(xs|sm|base|lg|xl|2xl|3xl|4xl|5xl|6xl|7xl|8xl|9xl|\[)`),

	// Typography - Font weight
	"font-weight": regexp.MustCompile(`^font-(thin|extralight|light|normal|medium|semibold|bold|extrabold|black|\d)`),

	// Typography - Line height
	"line-height": regexp.MustCompile(`^leading-`),

	// Typography - Letter spacing
	"letter-spacing": regexp.MustCompile(`^tracking-`),

	// Typography - Text alignment
	"text-align": regexp.MustCompile(`^text-(left|center|right|justify|start|end)$`),

	// Typography - Text decoration
	"text-decoration": regexp.MustCompile(`^(underline|overline|line-through|no-underline)$`),

	// Typography - Text transform
	"text-transform": regexp.MustCompile(`^(uppercase|lowercase|capitalize|normal-case)$`),

	// Typography - Font style
	"font-style": regexp.MustCompile(`^(italic|not-italic)$`),

	// Typography - Font family
	"font-family": regexp.MustCompile(`^font-(sans|serif|mono)$`),

	// Colors - Text (including arbitrary values like text-[#ff0000])
	"text-color": regexp.MustCompile(`^text-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^text-\[`),

	// Colors - Background (including arbitrary values like bg-[#ff0000])
	"bg-color": regexp.MustCompile(`^bg-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^bg-\[`),

	// Colors - Border (including arbitrary values)
	"border-color": regexp.MustCompile(`^border-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^border-\[`),

	// Colors - Ring (including arbitrary values)
	"ring-color": regexp.MustCompile(`^ring-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^ring-\[`),

	// Colors - Outline (including arbitrary values)
	"outline-color": regexp.MustCompile(`^outline-(inherit|current|transparent|black|white|slate|gray|zinc|neutral|stone|red|orange|amber|yellow|lime|green|emerald|teal|cyan|sky|blue|indigo|violet|purple|fuchsia|pink|rose)-|^outline-\[`),

	// Layout - Display
	"display": regexp.MustCompile(`^(block|inline-block|inline|flex|inline-flex|table|inline-table|table-caption|table-cell|table-column|table-column-group|table-footer-group|table-header-group|table-row-group|table-row|flow-root|grid|inline-grid|contents|list-item|hidden)$`),

	// Layout - Position
	"position": regexp.MustCompile(`^(static|fixed|absolute|relative|sticky)$`),

	// Layout - Overflow
	"overflow":   regexp.MustCompile(`^overflow-(auto|hidden|clip|visible|scroll)$`),
	"overflow-x": regexp.MustCompile(`^overflow-x-`),
	"overflow-y": regexp.MustCompile(`^overflow-y-`),

	// Layout - Z-index
	"z-index": regexp.MustCompile(`^z-`),

	// Flexbox & Grid - Flex direction
	"flex-direction": regexp.MustCompile(`^flex-(row|row-reverse|col|col-reverse)$`),

	// Flexbox & Grid - Flex wrap
	"flex-wrap": regexp.MustCompile(`^flex-(wrap|wrap-reverse|nowrap)$`),

	// Flexbox & Grid - Flex
	"flex": regexp.MustCompile(`^flex-(1|auto|initial|none|\[)`),

	// Flexbox & Grid - Flex grow
	"flex-grow": regexp.MustCompile(`^grow(-0|-\[)?`),

	// Flexbox & Grid - Flex shrink
	"flex-shrink": regexp.MustCompile(`^shrink(-0|-\[)?`),

	// Flexbox & Grid - Order
	"order": regexp.MustCompile(`^order-`),

	// Flexbox & Grid - Grid template columns
	"grid-template-columns": regexp.MustCompile(`^grid-cols-`),

	// Flexbox & Grid - Grid template rows
	"grid-template-rows": regexp.MustCompile(`^grid-rows-`),

	// Flexbox & Grid - Grid column start/end
	"grid-column": regexp.MustCompile(`^col-(auto|span-|start-|end-)`),

	// Flexbox & Grid - Grid row start/end
	"grid-row": regexp.MustCompile(`^row-(auto|span-|start-|end-)`),

	// Flexbox & Grid - Justify content
	"justify-content": regexp.MustCompile(`^justify-(normal|start|end|center|between|around|evenly|stretch)$`),

	// Flexbox & Grid - Justify items
	"justify-items": regexp.MustCompile(`^justify-items-(start|end|center|stretch)$`),

	// Flexbox & Grid - Justify self
	"justify-self": regexp.MustCompile(`^justify-self-(auto|start|end|center|stretch)$`),

	// Flexbox & Grid - Align content
	"align-content": regexp.MustCompile(`^content-(normal|center|start|end|between|around|evenly|baseline|stretch)$`),

	// Flexbox & Grid - Align items
	"align-items": regexp.MustCompile(`^items-(start|end|center|baseline|stretch)$`),

	// Flexbox & Grid - Align self
	"align-self": regexp.MustCompile(`^self-(auto|start|end|center|stretch|baseline)$`),

	// Flexbox & Grid - Place content
	"place-content": regexp.MustCompile(`^place-content-(center|start|end|between|around|evenly|baseline|stretch)$`),

	// Flexbox & Grid - Place items
	"place-items": regexp.MustCompile(`^place-items-(start|end|center|baseline|stretch)$`),

	// Flexbox & Grid - Place self
	"place-self": regexp.MustCompile(`^place-self-(auto|start|end|center|stretch)$`),

	// Borders - Width
	"border":   regexp.MustCompile(`^border-(\d|\[)`),
	"border-x": regexp.MustCompile(`^border-x-`),
	"border-y": regexp.MustCompile(`^border-y-`),
	"border-s": regexp.MustCompile(`^border-s-`),
	"border-e": regexp.MustCompile(`^border-e-`),
	"border-t": regexp.MustCompile(`^border-t-(\d|\[)`),
	"border-r": regexp.MustCompile(`^border-r-(\d|\[)`),
	"border-b": regexp.MustCompile(`^border-b-(\d|\[)`),
	"border-l": regexp.MustCompile(`^border-l-(\d|\[)`),

	// Borders - Radius
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

	// Borders - Style
	"border-style": regexp.MustCompile(`^border-(solid|dashed|dotted|double|hidden|none)$`),

	// Effects - Opacity
	"opacity": regexp.MustCompile(`^opacity-`),

	// Effects - Box shadow
	"box-shadow": regexp.MustCompile(`^shadow-`),

	// Effects - Blur
	"blur": regexp.MustCompile(`^blur-`),

	// Effects - Backdrop blur
	"backdrop-blur": regexp.MustCompile(`^backdrop-blur-`),

	// Ring
	"ring-width":        regexp.MustCompile(`^ring-(\d|\[)`),
	"ring-offset-width": regexp.MustCompile(`^ring-offset-`),
}

// getConflictGroup returns the conflict group for a given class name.
// Returns empty string if no conflict group is found.
func getConflictGroup(class string) string {

	for group, pattern := range conflictGroupPatterns {
		if pattern.MatchString(class) {
			return group
		}
	}
	return ""
}

// splitVariantAndClass splits a class like "hover:bg-blue-500" into
// variant ("hover") and base class ("bg-blue-500").
func splitVariantAndClass(class string) (variant, baseClass string) {

	parts := strings.Split(class, ":")
	if len(parts) == 1 {
		return "", class
	}

	return strings.Join(parts[:len(parts)-1], ":"), parts[len(parts)-1]
}
