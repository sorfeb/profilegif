package render

import (
	"fmt"
	"image"
	"image/color"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"io"

	"github.com/sorfeb/profilegif/internal/scene"
)

// alphaThreshold splits the 1-bit GIF transparency: pixels at or above are opaque, below are
// fully transparent (GIF has no partial alpha, so anti-aliased edges get thresholded).
const alphaThreshold = 0x8000 // half of 0xffff (16-bit alpha from color.RGBA())

// EncodeGIF writes an infinitely-looping animated GIF. If the frames contain transparency it
// uses a palette with a reserved transparent index and per-frame background disposal;
// otherwise it quantizes to the Plan9 palette with Floyd–Steinberg dithering.
func EncodeGIF(w io.Writer, frames []image.Image, delaysCentis []int) error {
	if len(frames) == 0 {
		return fmt.Errorf("render: no frames to encode")
	}
	if len(delaysCentis) != len(frames) {
		return fmt.Errorf("render: %d frames but %d delays", len(frames), len(delaysCentis))
	}
	if hasTransparency(frames[0]) {
		return encodeTransparent(w, frames, delaysCentis)
	}

	g := &gif.GIF{LoopCount: 0} // 0 == loop forever
	for i, fr := range frames {
		pal := image.NewPaletted(fr.Bounds(), palette.Plan9)
		draw.FloydSteinberg.Draw(pal, fr.Bounds(), fr, image.Point{})
		g.Image = append(g.Image, pal)
		g.Delay = append(g.Delay, delaysCentis[i])
	}
	return gif.EncodeAll(w, g)
}

// encodeTransparent maps sub-threshold-alpha pixels to a reserved transparent palette entry
// (index 0) and quantizes the rest against Plan9. DisposalBackground clears each frame so the
// animation doesn't smear on the transparent canvas.
func encodeTransparent(w io.Writer, frames []image.Image, delaysCentis []int) error {
	pal := make(color.Palette, 0, 256)
	pal = append(pal, color.RGBA{}) // index 0: fully transparent
	pal = append(pal, palette.Plan9[:255]...)

	g := &gif.GIF{LoopCount: 0, BackgroundIndex: 0}
	for i, fr := range frames {
		b := fr.Bounds()
		pimg := image.NewPaletted(b, pal)
		for y := b.Min.Y; y < b.Max.Y; y++ {
			for x := b.Min.X; x < b.Max.X; x++ {
				r, gg, bb, a := fr.At(x, y).RGBA()
				if a < alphaThreshold {
					pimg.SetColorIndex(x, y, 0)
					continue
				}
				// Force full opacity so nearest-color matching ignores the source alpha.
				pimg.Set(x, y, color.RGBA{uint8(r >> 8), uint8(gg >> 8), uint8(bb >> 8), 0xff})
			}
		}
		g.Image = append(g.Image, pimg)
		g.Delay = append(g.Delay, delaysCentis[i])
		g.Disposal = append(g.Disposal, gif.DisposalBackground)
	}
	return gif.EncodeAll(w, g)
}

// hasTransparency reports whether img has any pixel below the opacity threshold.
func hasTransparency(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if _, _, _, a := img.At(x, y).RGBA(); a < alphaThreshold {
				return true
			}
		}
	}
	return false
}

// EncodeScene is the one-call convenience: render every frame of s and encode a GIF to w.
func EncodeScene(w io.Writer, s *scene.Scene) error {
	frames, delays := Frames(s)
	return EncodeGIF(w, frames, delays)
}
