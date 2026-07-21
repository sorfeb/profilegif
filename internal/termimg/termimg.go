// Package termimg draws an image.Image into a terminal-printable string. It is the ONLY
// layer in profilegif that knows about terminal pixels — the scene model and the rasterizer
// are resolution-independent, so swapping in a higher-fidelity backend (Sixel, Kitty) later
// means adding another Renderer here, with zero changes to the editor or the document model.
//
// The default HalfBlock renderer uses the "▀" (upper half block) trick: each character cell
// stacks two pixels — the cell's foreground color is the top pixel, its background color is
// the bottom pixel — doubling vertical resolution. 24-bit truecolor ANSI, so it works in
// essentially any modern terminal (Windows Terminal included).
package termimg

import (
	"image"
	"strconv"
	"strings"

	xdraw "golang.org/x/image/draw"
)

// Renderer converts an image into a string of terminal cells sized cols×rows.
type Renderer interface {
	Render(img image.Image, cols, rows int) string
}

// upperHalfBlock renders the top half of a cell; its fg is the top pixel, bg the bottom.
const upperHalfBlock = '▀'

// ANSI truecolor set-foreground / set-background prefixes and the reset sequence.
const (
	fgPrefix = "\x1b[38;2;"
	bgPrefix = "\x1b[48;2;"
	reset    = "\x1b[0m"
)

// HalfBlock is the portable truecolor renderer. The zero value is ready to use.
type HalfBlock struct{}

// Render scales img to cols×(rows*2) pixels and emits one "▀" per cell. Redundant color
// escapes are elided (a cell only re-emits fg/bg when they change from the previous cell),
// which keeps frames small enough to redraw smoothly.
func (HalfBlock) Render(src image.Image, cols, rows int) string {
	if cols < 1 || rows < 1 {
		return ""
	}
	pw, ph := cols, rows*2
	dst := image.NewRGBA(image.Rect(0, 0, pw, ph))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)

	var b strings.Builder
	b.Grow(pw*ph*6 + rows*len(reset))

	for r := 0; r < rows; r++ {
		// Each row ends with a reset, so terminal color state is cleared between rows;
		// track "last color" per row starting from unset (-1).
		lastFg, lastBg := -1, -1
		for c := 0; c < cols; c++ {
			tr, tg, tb := rgb8(dst, c, r*2)
			br, bg, bl := rgb8(dst, c, r*2+1)
			fg := tr<<16 | tg<<8 | tb
			bgc := br<<16 | bg<<8 | bl
			if fg != lastFg {
				writeColor(&b, fgPrefix, tr, tg, tb)
				lastFg = fg
			}
			if bgc != lastBg {
				writeColor(&b, bgPrefix, br, bg, bl)
				lastBg = bgc
			}
			b.WriteRune(upperHalfBlock)
		}
		b.WriteString(reset)
		if r < rows-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

func writeColor(b *strings.Builder, prefix string, r, g, bl int) {
	b.WriteString(prefix)
	b.WriteString(strconv.Itoa(r))
	b.WriteByte(';')
	b.WriteString(strconv.Itoa(g))
	b.WriteByte(';')
	b.WriteString(strconv.Itoa(bl))
	b.WriteByte('m')
}

func rgb8(img *image.RGBA, x, y int) (r, g, b int) {
	i := img.PixOffset(x, y)
	s := img.Pix[i : i+3 : i+3]
	return int(s[0]), int(s[1]), int(s[2])
}
