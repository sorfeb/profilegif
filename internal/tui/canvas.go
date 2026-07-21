package tui

import (
	"image"
	"image/color"
	"math"

	"github.com/sorfeb/profilegif/internal/render"
	"github.com/sorfeb/profilegif/internal/scene"
)

// selection colors for the outline + resize handle painted over the canvas.
var (
	selColor    = color.RGBA{R: 0x3f, G: 0xb0, B: 0xff, A: 0xff}
	handleColor = color.RGBA{R: 0xff, G: 0xd3, B: 0x3d, A: 0xff}
)

// viewport describes where the canvas content sits on screen and how cells map to logical px.
type viewport struct {
	originX, originY int     // top-left cell of the canvas *content* (inside the border)
	cols, rows       int     // canvas size in cells
	scaleX, scaleY   float64 // logical px per cell
}

// viewportGeom recomputes the canvas geometry from the current terminal size. Both View and
// the mouse handler call this so screen↔logical mapping stays consistent without caching.
func (m Model) viewportGeom() viewport {
	helpH := 1
	contentH := m.height - helpH
	innerCols := (m.width - sidebarWidth) - 2 // minus canvas border
	innerRows := contentH - 2
	cols, rows := fitCanvas(innerCols, innerRows, m.scene.W, m.scene.H)
	return viewport{
		originX: 1, originY: 1, // canvas box sits at screen (0,0); border is 1 cell
		cols: cols, rows: rows,
		scaleX: float64(m.scene.W) / float64(cols),
		scaleY: float64(m.scene.H) / float64(rows),
	}
}

// cellToLogical maps a screen cell to a logical canvas point (center of the cell), plus a
// flag for whether the cell is inside the canvas area at all.
func (vp viewport) cellToLogical(cellX, cellY int) (x, y int, inside bool) {
	cx := cellX - vp.originX
	cy := cellY - vp.originY
	if cx < 0 || cy < 0 || cx >= vp.cols || cy >= vp.rows {
		return 0, 0, false
	}
	x = int((float64(cx) + 0.5) * vp.scaleX)
	y = int((float64(cy) + 0.5) * vp.scaleY)
	return x, y, true
}

// cellToLogicalClamped maps a screen cell to a logical point, clamped into [0,W]×[0,H] so a
// drag that leaves the canvas still yields a sensible edge position (used during move/resize).
func (vp viewport) cellToLogicalClamped(cellX, cellY, W, H int) (int, int) {
	x := int((float64(cellX-vp.originX) + 0.5) * vp.scaleX)
	y := int((float64(cellY-vp.originY) + 0.5) * vp.scaleY)
	if x < 0 {
		x = 0
	}
	if y < 0 {
		y = 0
	}
	if x > W {
		x = W
	}
	if y > H {
		y = H
	}
	return x, y
}

// fitCanvas returns the largest cols×rows (in cells) that fits the available area while
// preserving the scene's aspect ratio. Half-blocks pack 2 pixels per cell vertically, so the
// pixel grid is cols×(rows*2) and we match cols/(rows*2) == W/H.
func fitCanvas(availCols, availRows, W, H int) (int, int) {
	if availCols < 1 || availRows < 1 || W <= 0 || H <= 0 {
		return 1, 1
	}
	cols := availCols
	rows := int(math.Round(float64(cols) * float64(H) / (2 * float64(W))))
	if rows > availRows {
		rows = availRows
		cols = int(math.Round(float64(rows) * 2 * float64(W) / float64(H)))
	}
	if cols < 1 {
		cols = 1
	}
	if rows < 1 {
		rows = 1
	}
	if cols > availCols {
		cols = availCols
	}
	return cols, rows
}

// canvasView rasterizes the current frame, paints the selection overlay, and converts to
// half-block cells.
func (m Model) canvasView(vp viewport) string {
	img := ensureRGBA(render.Rasterize(m.scene, m.frame))
	if el := m.sel(); el != nil {
		th := int(math.Max(vp.scaleX, vp.scaleY)) + 1 // keep the outline visible after downscale
		drawOutline(img, el.Bounds(), selColor, th)
		drawHandle(img, el.Bounds(), handleColor, th*3)
	}
	return m.renderer.Render(img, vp.cols, vp.rows)
}

// ensureRGBA returns img as *image.RGBA, copying only if necessary (gg already returns one).
func ensureRGBA(img image.Image) *image.RGBA {
	if r, ok := img.(*image.RGBA); ok {
		return r
	}
	b := img.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			dst.Set(x, y, img.At(x, y))
		}
	}
	return dst
}

// drawOutline paints a th-thick rectangle border at r (logical px) onto img.
func drawOutline(img *image.RGBA, r scene.Rect, c color.Color, th int) {
	if th < 1 {
		th = 1
	}
	fillRect(img, r.X, r.Y, r.W, th, c)        // top
	fillRect(img, r.X, r.Y+r.H-th, r.W, th, c) // bottom
	fillRect(img, r.X, r.Y, th, r.H, c)        // left
	fillRect(img, r.X+r.W-th, r.Y, th, r.H, c) // right
}

// drawHandle paints a filled square at the bottom-right corner (the resize grab zone).
func drawHandle(img *image.RGBA, r scene.Rect, c color.Color, size int) {
	if size < 2 {
		size = 2
	}
	fillRect(img, r.X+r.W-size, r.Y+r.H-size, size, size, c)
}

func fillRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	b := img.Bounds()
	for yy := y; yy < y+h; yy++ {
		if yy < b.Min.Y || yy >= b.Max.Y {
			continue
		}
		for xx := x; xx < x+w; xx++ {
			if xx < b.Min.X || xx >= b.Max.X {
				continue
			}
			img.Set(xx, yy, c)
		}
	}
}
