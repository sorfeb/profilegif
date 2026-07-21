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
	"image"
	"strconv"
	"strings"

	"github.com/fogleman/gg"
	"github.com/sorfeb/profilegif/internal/scene"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
)

// bgFill is the canvas base color drawn beneath every layer (GitHub dark).
const bgFill = "#0d1117"

var baseFont *opentype.Font

func init() {
	f, err := opentype.Parse(goregular.TTF)
	if err != nil {
		panic("render: parse embedded font: " + err.Error())
	}
	baseFont = f
}

// faceAt builds a font face at the given pixel size. At 72 DPI, 1 point == 1 pixel, so
// size is effectively the cap height in pixels. Caller must Close the returned face.
func faceAt(size float64) font.Face {
	if size <= 0 {
		size = 24
	}
	face, err := opentype.NewFace(baseFont, &opentype.FaceOptions{
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
	dc.SetHexColor(bgFill)
	dc.Clear()

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
			drawText(dc, e.Text, e.FontSize, e.Color, e.Bounds())
		case *scene.StatWidget:
			drawStatWidget(dc, e, progress)
		}
	}
	return dc.Image()
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

func drawText(dc *gg.Context, text string, size float64, hex string, r scene.Rect) {
	if text == "" {
		return
	}
	face := faceAt(size)
	defer face.Close()
	dc.SetFontFace(face)
	if hex == "" {
		hex = "#ffffff"
	}
	dc.SetHexColor(hex)
	cx := float64(r.X) + float64(r.W)/2
	cy := float64(r.Y) + float64(r.H)/2
	dc.DrawStringAnchored(text, cx, cy, 0.5, 0.5)
}

func drawStatWidget(dc *gg.Context, e *scene.StatWidget, progress float64) {
	r := e.Bounds()
	color := e.Color
	if color == "" {
		color = "#39d353"
	}
	label := e.Label
	if label == "" {
		label = defaultLabel(e.Metric)
	}
	val := int(float64(e.Value) * progress)

	// Big animated number.
	numFace := faceAt(e.FontSize)
	dc.SetFontFace(numFace)
	dc.SetHexColor(color)
	dc.DrawStringAnchored(formatInt(val),
		float64(r.X)+float64(r.W)/2, float64(r.Y)+float64(r.H)*0.35, 0.5, 0.5)
	numFace.Close()

	// Muted label beneath.
	lblSize := e.FontSize * 0.45
	if lblSize < 8 {
		lblSize = 8
	}
	lblFace := faceAt(lblSize)
	dc.SetFontFace(lblFace)
	dc.SetHexColor("#8b949e")
	dc.DrawStringAnchored(label,
		float64(r.X)+float64(r.W)/2, float64(r.Y)+float64(r.H)*0.60, 0.5, 0.5)
	lblFace.Close()

	// Progress bar along the bottom: track + growing fill.
	const barH = 6.0
	barY := float64(r.Y) + float64(r.H) - barH - 2
	dc.SetHexColor("#30363d")
	dc.DrawRectangle(float64(r.X), barY, float64(r.W), barH)
	dc.Fill()
	dc.SetHexColor(color)
	dc.DrawRectangle(float64(r.X), barY, float64(r.W)*progress, barH)
	dc.Fill()
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
