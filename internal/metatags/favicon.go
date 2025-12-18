package metatags

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"sort"
	"strings"

	"github.com/eleven-am/pondlive/internal/work"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func renderIconToSVG(icon *iconInfo) []byte {
	if icon == nil || icon.node == nil {
		return nil
	}

	var b strings.Builder
	renderSVGNode(&b, icon.node)
	return []byte(b.String())
}

func renderIconToPNG(svgData []byte, size int, background, color string) []byte {
	if len(svgData) == 0 {
		return nil
	}

	svgStr := string(svgData)

	strokeColor := "#000000"
	if color != "" {
		strokeColor = color
	} else if background != "" {
		strokeColor = "#ffffff"
	}

	svgStr = strings.ReplaceAll(svgStr, "currentColor", strokeColor)
	svgStr = strings.ReplaceAll(svgStr, "currentcolor", strokeColor)

	if background != "" {
		svgStr = addBackgroundRect(svgStr, background)
	}

	svgData = []byte(svgStr)

	icon, err := oksvg.ReadIconStream(bytes.NewReader(svgData))
	if err != nil {
		fmt.Println("Error reading SVG icon:", err)
		return nil
	}

	icon.SetTarget(0, 0, float64(size), float64(size))
	rgba := image.NewRGBA(image.Rect(0, 0, size, size))
	icon.Draw(rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, rgba, rgba.Bounds())), 1)

	var buf bytes.Buffer
	if err := png.Encode(&buf, rgba); err != nil {
		return nil
	}

	return buf.Bytes()
}

func addBackgroundRect(svgStr, background string) string {
	openTag := "<svg"
	idx := strings.Index(svgStr, openTag)
	if idx == -1 {
		return svgStr
	}

	closeIdx := strings.Index(svgStr[idx:], ">")
	if closeIdx == -1 {
		return svgStr
	}

	insertPos := idx + closeIdx + 1
	bgRect := `<rect x="0" y="0" width="24" height="24" rx="5" ry="5" fill="` + background + `"/>`
	return svgStr[:insertPos] + bgRect + svgStr[insertPos:]
}

func renderSVGNode(b *strings.Builder, node work.Node) {
	if node == nil {
		return
	}

	switch n := node.(type) {
	case *work.Element:
		renderSVGElement(b, n)
	case *work.Text:
		b.WriteString(escapeXML(n.Value))
	case *work.Fragment:
		for _, child := range n.Children {
			renderSVGNode(b, child)
		}
	}
}

func renderSVGElement(b *strings.Builder, el *work.Element) {
	if el == nil {
		return
	}

	b.WriteByte('<')
	b.WriteString(el.Tag)

	if len(el.Attrs) > 0 {
		keys := make([]string, 0, len(el.Attrs))
		for k := range el.Attrs {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			values := el.Attrs[k]
			if len(values) == 0 {
				b.WriteByte(' ')
				b.WriteString(k)
				continue
			}
			b.WriteByte(' ')
			b.WriteString(k)
			b.WriteString(`="`)
			b.WriteString(escapeXML(strings.Join(values, " ")))
			b.WriteByte('"')
		}
	}

	if len(el.Style) > 0 {
		keys := make([]string, 0, len(el.Style))
		for k := range el.Style {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		b.WriteString(` style="`)
		for i, k := range keys {
			if i > 0 {
				b.WriteString("; ")
			}
			b.WriteString(k)
			b.WriteString(": ")
			b.WriteString(escapeXML(el.Style[k]))
		}
		b.WriteByte('"')
	}

	if len(el.Children) == 0 {
		b.WriteString("/>")
		return
	}

	b.WriteByte('>')

	for _, child := range el.Children {
		renderSVGNode(b, child)
	}

	b.WriteString("</")
	b.WriteString(el.Tag)
	b.WriteByte('>')
}

func escapeXML(s string) string {
	s = strings.ReplaceAll(s, "&", "&amp;")
	s = strings.ReplaceAll(s, "<", "&lt;")
	s = strings.ReplaceAll(s, ">", "&gt;")
	s = strings.ReplaceAll(s, "\"", "&quot;")
	s = strings.ReplaceAll(s, "'", "&apos;")
	return s
}
