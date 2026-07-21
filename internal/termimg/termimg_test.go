package termimg

import (
	"image"
	"image/color"
	"strings"
	"testing"
)

func solid(w, h int, c color.Color) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, w, h))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			img.Set(x, y, c)
		}
	}
	return img
}

func TestHalfBlockDimensions(t *testing.T) {
	out := HalfBlock{}.Render(solid(10, 10, color.White), 8, 4)
	lines := strings.Split(out, "\n")
	if len(lines) != 4 {
		t.Fatalf("rows: got %d want 4", len(lines))
	}
	// Each cell emits exactly one half-block glyph; 8 per row.
	for i, ln := range lines {
		if n := strings.Count(ln, string(upperHalfBlock)); n != 8 {
			t.Errorf("row %d: got %d glyphs want 8", i, n)
		}
	}
}

func TestHalfBlockColor(t *testing.T) {
	red := color.RGBA{R: 255, G: 0, B: 0, A: 255}
	out := HalfBlock{}.Render(solid(4, 4, red), 2, 1)
	if !strings.Contains(out, "38;2;255;0;0") {
		t.Errorf("missing red foreground escape in output: %q", out)
	}
	if !strings.Contains(out, "48;2;255;0;0") {
		t.Errorf("missing red background escape in output: %q", out)
	}
	if !strings.HasSuffix(out, reset) {
		t.Errorf("output should end with a reset sequence")
	}
}

func TestHalfBlockElidesRepeatedColor(t *testing.T) {
	// A solid image: the first cell sets fg+bg, subsequent cells in the row shouldn't
	// re-emit identical color escapes.
	out := HalfBlock{}.Render(solid(4, 4, color.White), 5, 1)
	if got := strings.Count(out, fgPrefix); got != 1 {
		t.Errorf("fg escapes: got %d want 1 (elision failed)", got)
	}
	if got := strings.Count(out, bgPrefix); got != 1 {
		t.Errorf("bg escapes: got %d want 1 (elision failed)", got)
	}
}

func TestHalfBlockDegenerate(t *testing.T) {
	if (HalfBlock{}).Render(solid(4, 4, color.White), 0, 5) != "" {
		t.Error("zero cols should render empty")
	}
	if (HalfBlock{}).Render(solid(4, 4, color.White), 5, 0) != "" {
		t.Error("zero rows should render empty")
	}
}

// Compile-time assertion that HalfBlock satisfies Renderer.
var _ Renderer = HalfBlock{}
