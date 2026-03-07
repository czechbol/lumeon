package core

import (
	"bytes"
	"fmt"
	"image"
	"image/draw"
	_ "image/png" // register PNG decoder
	"unicode/utf8"

	bitmapfont "github.com/hajimehoshi/bitmapfont/v3"
	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
	"periph.io/x/devices/v3/ssd1306/image1bit"
)

const (
	canvasW = 128
	canvasH = 64

	// Layout constants for stat pages.
	headerHeight = 16                     // icon row height
	contentH     = canvasH - headerHeight // scrollable content area height (48px, 3 rows)
	iconSize     = 16
	lineHeight   = 16 // text line height (matches bitmapfont 6×16)
	barHeight    = 16 // matches lineHeight so bar and text share the same row slot
	barBorder    = 1
)

// newCanvas creates a blank 128×64 monochrome canvas (all black).
func newCanvas() *image1bit.VerticalLSB {
	return image1bit.NewVerticalLSB(image.Rect(0, 0, canvasW, canvasH))
}

// newContentCanvas creates a blank 128×48 monochrome canvas for a scrollable subpage content area.
func newContentCanvas() *image1bit.VerticalLSB {
	return image1bit.NewVerticalLSB(image.Rect(0, 0, canvasW, contentH))
}

// drawIcon blits a decoded icon image onto the canvas at (x, y).
func drawIcon(canvas draw.Image, icon image.Image, x, y int) {
	// Convert to monochrome: anything brighter than mid-gray becomes white.
	bounds := icon.Bounds()
	for iy := bounds.Min.Y; iy < bounds.Max.Y; iy++ {
		for ix := bounds.Min.X; ix < bounds.Max.X; ix++ {
			r, g, b, _ := icon.At(ix, iy).RGBA()
			lum := (r + g + b) / 3
			if lum > 0x8000 {
				canvas.Set(x+ix-bounds.Min.X, y+iy-bounds.Min.Y, image1bit.On)
			}
		}
	}
}

// drawText renders a string at (x, y) using bitmapfont.Face (6×16).
// y is the top of the text area (ascent is added internally).
// bitmapfont covers printable ASCII plus many Unicode characters including
// degree sign (U+00B0), up arrow (U+2191), and down arrow (U+2193).
func drawText(canvas draw.Image, text string, x, y int) {
	face := bitmapfont.Face
	d := &font.Drawer{
		Dst:  canvas,
		Src:  image.NewUniform(image1bit.On),
		Face: face,
		Dot:  fixed.Point26_6{X: fixed.I(x), Y: fixed.I(y + face.Metrics().Ascent.Ceil())},
	}
	d.DrawString(text)
}

// textWidth returns the pixel width of a string in bitmapfont.Face (6px per glyph).
func textWidth(text string) int {
	return utf8.RuneCountInString(text) * 6
}

// drawProgressBar draws a horizontal bar with 1px border.
// percent should be 0–100.
func drawProgressBar(canvas draw.Image, x, y, width int, percent float64) {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}

	// Outer border
	for bx := x; bx < x+width; bx++ {
		canvas.Set(bx, y, image1bit.On)
		canvas.Set(bx, y+barHeight-1, image1bit.On)
	}
	for by := y; by < y+barHeight; by++ {
		canvas.Set(x, by, image1bit.On)
		canvas.Set(x+width-1, by, image1bit.On)
	}

	// Inner fill
	innerW := width - 2*barBorder - 2 // 1px border + 1px padding each side
	innerH := barHeight - 2*barBorder - 2
	fillW := int(float64(innerW) * percent / 100.0)
	startX := x + barBorder + 1
	startY := y + barBorder + 1

	for by := startY; by < startY+innerH; by++ {
		for bx := startX; bx < startX+fillW; bx++ {
			canvas.Set(bx, by, image1bit.On)
		}
	}
}

// decodeIcon decodes a PNG icon from embedded bytes.
func decodeIcon(data []byte) image.Image {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		// Return a blank 16×16 image on decode failure rather than crashing.
		return image.NewGray(image.Rect(0, 0, iconSize, iconSize))
	}
	return img
}

// drawHeader renders an icon + title at the top of a page.
// Returns the y position below the header for content.
func drawHeader(canvas draw.Image, iconData []byte, title string) int {
	icon := decodeIcon(iconData)
	drawIcon(canvas, icon, 0, 0)
	drawText(canvas, title, iconSize+2, 0)
	return headerHeight
}

// rightAlignX returns the x position to right-align text on the canvas.
func rightAlignX(text string) int {
	return canvasW - textWidth(text)
}

// truncateToFit shortens text so its pixel width does not exceed maxPx.
func truncateToFit(text string, maxPx int) string {
	runes := []rune(text)
	for len(runes)*6 > maxPx {
		runes = runes[:len(runes)-1]
	}
	return string(runes)
}

const (
	bytesPerMB      = 1 << 20
	mbDisplayThresh = 0.1 // show MB/s above this, KB/s below
)

// formatSpeed formats a byte/s value as a compact string with unit suffix.
// Uses K for kilobytes/s and M for megabytes/s.
func formatSpeed(bytesPerSec float64) string {
	mb := bytesPerSec / bytesPerMB
	if mb >= mbDisplayThresh {
		return fmt.Sprintf("%.1fM", mb)
	}
	kb := bytesPerSec / 1024
	return fmt.Sprintf("%.0fK", kb)
}

const (
	bytesPerGB = 1 << 30
	bytesPerTB = 1 << 40
)

// formatBytes formats a byte count as a compact human-readable string.
// Uses M for megabytes, G for gigabytes, T for terabytes.
func formatBytes(bytes uint64) string {
	switch {
	case bytes >= bytesPerTB:
		return fmt.Sprintf("%.1fT", float64(bytes)/bytesPerTB)
	case bytes >= bytesPerGB:
		return fmt.Sprintf("%.1fG", float64(bytes)/bytesPerGB)
	default:
		return fmt.Sprintf("%.0fM", float64(bytes)/(1<<20))
	}
}
