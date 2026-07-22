// Package render turns a scene.Scene into real pixels: single frames (Rasterize), the full
// animation (Frames), and an encoded GIF (EncodeGIF). It knows nothing about terminals or
// HTTP — it just produces image.Image values. Both the web server (via gifmaker) and the
// TUI's export use this package, so what you compose is exactly what ships.
//
// Text is drawn with the Go font (goregular), parsed from golang.org/x/image — so no font
// file needs bundling. Fonts faces are created per draw call (not shared), because a
// font.Face is not safe for concurrent use and the web server renders in parallel.
package render

import (
	"fmt"
	"image"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/sorfeb/profilegif/internal/scene"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// bgFill is the canvas base color drawn beneath every layer when the scene is NOT transparent.
const bgFill = "#0d1117"

// defaultInk is the monochrome foreground used when a scene sets no Ink (GitHub neutral gray).
const defaultInk = "#8b949e"

var (
	sansFont *opentype.Font
	monoFont *opentype.Font
)

func init() {
	var err error
	if sansFont, err = opentype.Parse(goregular.TTF); err != nil {
		panic("render: parse sans font: " + err.Error())
	}
	if monoFont, err = opentype.Parse(gomono.TTF); err != nil {
		panic("render: parse mono font: " + err.Error())
	}
}

// faceAt builds a proportional font face at the given pixel size (72 DPI → 1pt == 1px).
// Caller must Close the returned face.
func faceAt(size float64) font.Face { return newFace(sansFont, size) }

// monoFaceAt builds a monospace face — the terminal/ASCII look. Caller must Close it.
func monoFaceAt(size float64) font.Face { return newFace(monoFont, size) }

func newFace(f *opentype.Font, size float64) font.Face {
	if size <= 0 {
		size = 24
	}
	face, err := opentype.NewFace(f, &opentype.FaceOptions{
		Size:    size,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return basicfont.Face7x13
	}
	return face
}

// Rasterize renders a single animation frame of the scene to an image.
// frame is 0-based; progress across the animation drives StatWidget animation.
func Rasterize(s *scene.Scene, frame int) image.Image {
	dc := gg.NewContext(s.W, s.H)
	if !s.Transparent {
		dc.SetHexColor(bgFill)
		dc.Clear()
	}
	// gg.NewContext starts fully transparent, so a transparent scene needs no fill.

	ink := s.Ink
	if ink == "" {
		ink = defaultInk
	}

	total := s.FrameCount()
	progress := 1.0
	if total > 1 {
		p := float64(frame) / float64(total-1)
		if p < 0 {
			p = 0
		} else if p > 1 {
			p = 1
		}
		progress = p
	}

	for _, el := range s.Layers {
		switch e := el.(type) {
		case *scene.Background:
			drawImage(dc, e.Path, e.Bounds(), e.Fit)
		case *scene.ImageElement:
			drawImage(dc, e.Path, e.Bounds(), e.Fit)
		case *scene.TextElement:
			drawText(dc, e, colorOr(e.Color, ink))
		case *scene.StatWidget:
			drawStatWidget(dc, e, colorOr(e.Color, ink), progress)
		}
	}
	return dc.Image()
}

func colorOr(c, fallback string) string {
	if c == "" {
		return fallback
	}
	return c
}

func drawImage(dc *gg.Context, path string, r scene.Rect, fit string) {
	if r.W <= 0 || r.H <= 0 {
		return
	}
	if path == "" {
		drawPlaceholder(dc, r)
		return
	}
	img, err := gg.LoadImage(path) // decodes png/jpg/gif(first frame)
	if err != nil {
		drawPlaceholder(dc, r)
		return
	}
	dc.DrawImage(scaleToRect(img, r.W, r.H, fit), r.X, r.Y)
}

// drawPlaceholder marks a missing/empty image slot so it's still visible while editing.
func drawPlaceholder(dc *gg.Context, r scene.Rect) {
	dc.Push()
	dc.SetRGBA(0.5, 0.5, 0.5, 0.4)
	dc.DrawRectangle(float64(r.X), float64(r.Y), float64(r.W), float64(r.H))
	dc.Fill()
	dc.Pop()
}

func drawText(dc *gg.Context, e *scene.TextElement, hex string) {
	if e.Text == "" {
		return
	}
	face := faceAt(e.FontSize)
	if e.Mono {
		face = monoFaceAt(e.FontSize)
	}
	defer face.Close()
	dc.SetFontFace(face)
	dc.SetHexColor(hex)
	r := e.Bounds()
	// Left-aligned, vertically centered — reads like a terminal line.
	dc.DrawStringAnchored(e.Text, float64(r.X), float64(r.Y)+float64(r.H)/2, 0, 0.5)
}

// barGlyphs are the filled/empty cells of the ASCII meter.
const (
	barFull  = '█'
	barEmpty = '░'
)

// drawStatWidget renders one monospace terminal line: "label   value  [████░░░░]".
// The counter ticks 0→Value and the bar fills 0→Value/Max as the animation progresses.
func drawStatWidget(dc *gg.Context, e *scene.StatWidget, hex string, progress float64) {
	r := e.Bounds()
	label := e.Label
	if label == "" {
		label = defaultLabel(e.Metric)
	}
	cells := e.BarCells
	if cells <= 0 {
		cells = 10
	}
	max := e.Max
	if max <= 0 {
		max = e.Value // no meter target → bar simply fills as the counter completes
	}

	val := int(float64(e.Value) * progress)
	frac := 0.0
	if max > 0 {
		frac = float64(val) / float64(max)
	}
	if frac > 1 {
		frac = 1
	}
	filled := int(frac*float64(cells) + 0.5)

	bar := make([]rune, 0, cells+2)
	bar = append(bar, '[')
	for i := 0; i < cells; i++ {
		if i < filled {
			bar = append(bar, barFull)
		} else {
			bar = append(bar, barEmpty)
		}
	}
	bar = append(bar, ']')

	// label left-padded value, right-aligned within a fixed column, then the bar.
	line := fmt.Sprintf("%-10s %6s  %s", label, formatInt(val), string(bar))

	face := monoFaceAt(e.FontSize)
	defer face.Close()
	dc.SetFontFace(face)
	dc.SetHexColor(hex)
	dc.DrawStringAnchored(line, float64(r.X), float64(r.Y)+float64(r.H)/2, 0, 0.5)
}

func defaultLabel(metric string) string {
	switch metric {
	case scene.MetricCommits:
		return "commits"
	case scene.MetricFollowers:
		return "followers"
	case scene.MetricStars:
		return "stars"
	case scene.MetricContributions:
		return "contributions"
	default:
		return metric
	}
}

// formatInt renders n with thousands separators, e.g. 12345 -> "12,345".
func formatInt(n int) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.Itoa(n)
	nd := len(s)
	var b strings.Builder
	for i := 0; i < nd; i++ {
		if i > 0 && (nd-i)%3 == 0 { // comma every 3 digits from the right
			b.WriteByte(',')
		}
		b.WriteByte(s[i])
	}
	if neg {
		return "-" + b.String()
	}
	return b.String()
}

// scaleToRect returns a w×h RGBA with src drawn per the fit mode (contain letterboxes with
// transparency; cover crops; stretch distorts). High-quality CatmullRom resampling.
func scaleToRect(src image.Image, w, h int, fit string) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	sb := src.Bounds()
	sw, sh := sb.Dx(), sb.Dy()
	if sw == 0 || sh == 0 {
		return dst
	}

	switch fit {
	case scene.FitStretch:
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, sb, xdraw.Over, nil)

	case scene.FitContain:
		scale := min(float64(w)/float64(sw), float64(h)/float64(sh))
		tw, th := int(float64(sw)*scale), int(float64(sh)*scale)
		ox, oy := (w-tw)/2, (h-th)/2
		xdraw.CatmullRom.Scale(dst, image.Rect(ox, oy, ox+tw, oy+th), src, sb, xdraw.Over, nil)

	default: // cover: crop the source to the target aspect, then fill.
		scale := max(float64(w)/float64(sw), float64(h)/float64(sh))
		cw, ch := int(float64(w)/scale), int(float64(h)/scale)
		if cw > sw {
			cw = sw
		}
		if ch > sh {
			ch = sh
		}
		ox, oy := (sw-cw)/2, (sh-ch)/2
		srcRect := image.Rect(sb.Min.X+ox, sb.Min.Y+oy, sb.Min.X+ox+cw, sb.Min.Y+oy+ch)
		xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, srcRect, xdraw.Over, nil)
	}
	return dst
}
