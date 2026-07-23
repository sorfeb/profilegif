package render

import (
	"bytes"
	"image"
	"image/gif"
	"testing"

	"github.com/sorfeb/profilegif/internal/scene"
)

func buildScene() *scene.Scene {
	s := scene.NewScene(320, 160)
	s.FPS, s.DurationMs = 10, 1000 // 10 frames
	s.Add(scene.NewText(scene.Rect{X: 0, Y: 0, W: 320, H: 60}, "hi"))
	sw := scene.NewStatWidget(scene.Rect{X: 20, Y: 70, W: 280, H: 80}, scene.MetricCommits, "sorfeb")
	sw.Value = 1234
	s.Add(sw)
	return s
}

func TestRasterizeSize(t *testing.T) {
	s := buildScene()
	img := Rasterize(s, 0)
	if img.Bounds().Dx() != s.W || img.Bounds().Dy() != s.H {
		t.Errorf("frame size: got %v want %dx%d", img.Bounds(), s.W, s.H)
	}
}

func TestFramesCountAndDelay(t *testing.T) {
	s := buildScene()
	frames, delays := Frames(s)
	if len(frames) != 10 {
		t.Fatalf("frame count: got %d want 10", len(frames))
	}
	if len(delays) != 10 {
		t.Fatalf("delay count: got %d want 10", len(delays))
	}
	if delays[0] != 10 { // 100/10 fps = 10 centiseconds
		t.Errorf("delay: got %d want 10", delays[0])
	}
}

func TestEncodeSceneDecodes(t *testing.T) {
	s := buildScene()
	var buf bytes.Buffer
	if err := EncodeScene(&buf, s); err != nil {
		t.Fatalf("encode: %v", err)
	}
	g, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(g.Image) != 10 {
		t.Errorf("decoded frame count: got %d want 10", len(g.Image))
	}
	if g.LoopCount != 0 {
		t.Errorf("loop count: got %d want 0 (infinite)", g.LoopCount)
	}
}

// The animation must actually change: the last frame should differ from the first,
// since the StatWidget counter/bar grows with progress.
func TestAnimationProgresses(t *testing.T) {
	s := buildScene()
	first := Rasterize(s, 0)
	last := Rasterize(s, s.FrameCount()-1)
	if imagesEqual(first, last) {
		t.Error("first and last frames are identical; expected animation to progress")
	}
}

func imagesEqual(a, b image.Image) bool {
	if a.Bounds() != b.Bounds() {
		return false
	}
	bnd := a.Bounds()
	for y := bnd.Min.Y; y < bnd.Max.Y; y++ {
		for x := bnd.Min.X; x < bnd.Max.X; x++ {
			r1, g1, b1, a1 := a.At(x, y).RGBA()
			r2, g2, b2, a2 := b.At(x, y).RGBA()
			if r1 != r2 || g1 != g2 || b1 != b2 || a1 != a2 {
				return false
			}
		}
	}
	return true
}

func TestRasterizeTransparency(t *testing.T) {
	// Transparent scene → corner pixel is fully transparent.
	s := scene.NewScene(50, 50)
	s.Transparent = true
	if _, _, _, a := Rasterize(s, 0).At(0, 0).RGBA(); a != 0 {
		t.Errorf("transparent scene corner: got alpha %d want 0", a)
	}
	// Default scene → corner pixel is opaque.
	s2 := scene.NewScene(50, 50)
	if _, _, _, a := Rasterize(s2, 0).At(0, 0).RGBA(); a == 0 {
		t.Error("opaque scene corner should not be transparent")
	}
}

func TestEncodeTransparentGIF(t *testing.T) {
	s := scene.NewScene(80, 40)
	s.Transparent = true
	s.Ink = "#c9d1d9"
	s.FPS, s.DurationMs = 5, 1000
	s.Add(scene.NewText(scene.Rect{X: 2, Y: 2, W: 76, H: 24}, "hi"))

	var buf bytes.Buffer
	if err := EncodeScene(&buf, s); err != nil {
		t.Fatalf("encode: %v", err)
	}
	g, err := gif.DecodeAll(&buf)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Some frame's palette must include a fully-transparent entry.
	found := false
	for _, img := range g.Image {
		for _, c := range img.Palette {
			if _, _, _, a := c.RGBA(); a == 0 {
				found = true
			}
		}
	}
	if !found {
		t.Error("transparent scene should produce a transparent palette entry")
	}
}

func TestFormatInt(t *testing.T) {
	cases := map[int]string{0: "0", 42: "42", 1234: "1,234", 1000000: "1,000,000", -1234: "-1,234"}
	for in, want := range cases {
		if got := formatInt(in); got != want {
			t.Errorf("formatInt(%d): got %q want %q", in, got, want)
		}
	}
}
