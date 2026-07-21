package render

import (
	"fmt"
	"image"
	"image/color/palette"
	"image/draw"
	"image/gif"
	"io"

	"github.com/sorfeb/profilegif/internal/scene"
)

// EncodeGIF quantizes each frame to a 256-color paletted image (Plan9 palette with
// Floyd–Steinberg dithering) and writes an infinitely-looping animated GIF.
func EncodeGIF(w io.Writer, frames []image.Image, delaysCentis []int) error {
	if len(frames) == 0 {
		return fmt.Errorf("render: no frames to encode")
	}
	if len(delaysCentis) != len(frames) {
		return fmt.Errorf("render: %d frames but %d delays", len(frames), len(delaysCentis))
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

// EncodeScene is the one-call convenience: render every frame of s and encode a GIF to w.
func EncodeScene(w io.Writer, s *scene.Scene) error {
	frames, delays := Frames(s)
	return EncodeGIF(w, frames, delays)
}
