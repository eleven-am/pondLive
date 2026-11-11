package head

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
	"strconv"
	"strings"

	"github.com/eleven-am/pondlive/internal/runtime"
	"github.com/srwiley/oksvg"
	"github.com/srwiley/rasterx"
)

func renderIconToSVG(icon *iconInfo) []byte {
	if icon == nil || icon.node == nil {
		return nil
	}

	return []byte(runtime.RenderHTML(icon.node))
}

func renderIconToPNG(svgData []byte, size int, background, iconColor string) []byte {
	if len(svgData) == 0 {
		return nil
	}

	svgStr := string(svgData)
	svgStr = strings.ReplaceAll(svgStr, "currentColor", "#000000")
	svgStr = strings.ReplaceAll(svgStr, "currentcolor", "#000000")

	icon, err := oksvg.ReadIconStream(bytes.NewReader([]byte(svgStr)))
	if err != nil {
		return nil
	}

	icon.SetTarget(0, 0, float64(size), float64(size))
	strokeCanvas := image.NewRGBA(image.Rect(0, 0, size, size))
	icon.Draw(rasterx.NewDasher(size, size, rasterx.NewScannerGV(size, size, strokeCanvas, strokeCanvas.Bounds())), 1)

	if background == "" {
		strokeColor := parseHexColor(iconColor)
		if strokeColor == nil {
			strokeColor = &color.RGBA{R: 0, G: 0, B: 0, A: 255}
		}
		recolorStrokes(strokeCanvas, strokeColor)

		var buf bytes.Buffer
		if err := png.Encode(&buf, strokeCanvas); err != nil {
			return nil
		}
		return buf.Bytes()
	}

	bgColor := parseHexColor(background)
	if bgColor == nil {
		bgColor = &color.RGBA{R: 0, G: 0, B: 0, A: 255}
	}

	bgCanvas := image.NewRGBA(image.Rect(0, 0, size, size))
	radius := float64(size) * 0.2
	drawRoundedRect(bgCanvas, bgColor, size, radius)

	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			_, _, _, a := strokeCanvas.At(x, y).RGBA()
			if a > 0 {
				bgCanvas.Set(x, y, color.Transparent)
			}
		}
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, bgCanvas); err != nil {
		return nil
	}

	return buf.Bytes()
}

func parseHexColor(hex string) *color.RGBA {
	if hex == "" {
		return nil
	}

	hex = strings.TrimPrefix(hex, "#")

	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}

	if len(hex) != 6 {
		return nil
	}

	r, err := strconv.ParseUint(hex[0:2], 16, 8)
	if err != nil {
		return nil
	}
	g, err := strconv.ParseUint(hex[2:4], 16, 8)
	if err != nil {
		return nil
	}
	b, err := strconv.ParseUint(hex[4:6], 16, 8)
	if err != nil {
		return nil
	}

	return &color.RGBA{R: uint8(r), G: uint8(g), B: uint8(b), A: 255}
}

func recolorStrokes(img *image.RGBA, c *color.RGBA) {
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				img.Set(x, y, color.RGBA{R: c.R, G: c.G, B: c.B, A: uint8(a >> 8)})
			}
		}
	}
}

func drawRoundedRect(img *image.RGBA, c *color.RGBA, size int, radius float64) {
	for y := 0; y < size; y++ {
		for x := 0; x < size; x++ {
			if isInsideRoundedRect(float64(x), float64(y), float64(size), float64(size), radius) {
				img.Set(x, y, c)
			}
		}
	}
}

func isInsideRoundedRect(x, y, w, h, r float64) bool {
	if x < r && y < r {
		return distFromCorner(x, y, r, r, r)
	}
	if x > w-r && y < r {
		return distFromCorner(x, y, w-r, r, r)
	}
	if x < r && y > h-r {
		return distFromCorner(x, y, r, h-r, r)
	}
	if x > w-r && y > h-r {
		return distFromCorner(x, y, w-r, h-r, r)
	}
	return true
}

func distFromCorner(x, y, cx, cy, r float64) bool {
	dx := x - cx
	dy := y - cy
	return math.Sqrt(dx*dx+dy*dy) <= r
}
