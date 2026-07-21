package render

import (
	"image"

	"github.com/sorfeb/profilegif/internal/scene"
)

// Frames rasterizes every frame of the scene's animation and returns the frames alongside
// per-frame delays in GIF hundredths-of-a-second (matching the scene's FPS).
func Frames(s *scene.Scene) (frames []image.Image, delaysCentis []int) {
	n := s.FrameCount()
	fps := s.FPS
	if fps < 1 {
		fps = 1
	}
	delay := 100 / fps // hundredths of a second per frame
	if delay < 1 {
		delay = 1
	}

	frames = make([]image.Image, n)
	delaysCentis = make([]int, n)
	for i := 0; i < n; i++ {
		frames[i] = Rasterize(s, i)
		delaysCentis[i] = delay
	}
	return frames, delaysCentis
}
