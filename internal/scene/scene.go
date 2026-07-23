// Package scene is the profilegif document model: a canvas of stacked elements plus
// animation timing. It is deliberately free of HTTP, TUI, and rendering concerns — it's
// just data + JSON. Both front-ends (the TUI editor and the web server) read and write
// this model, and a scene JSON file is the bridge between them.
//
// For a JS/React dev: think of Scene as the serializable app state (like a Redux store
// snapshot) and Element as a discriminated union of node types, tagged by "kind" in JSON.
package scene

import (
	"encoding/json"
	"fmt"
	"os"
)

// Rect is an axis-aligned rectangle in logical canvas pixels (origin top-left).
// Logical pixels are resolution-independent: the terminal renderer and the GIF rasterizer
// both scale from these, so the same scene looks right at any output size.
type Rect struct {
	X int `json:"x"`
	Y int `json:"y"`
	W int `json:"w"`
	H int `json:"h"`
}

// Contains reports whether the point (px,py) lies inside r.
func (r Rect) Contains(px, py int) bool {
	return px >= r.X && px < r.X+r.W && py >= r.Y && py < r.Y+r.H
}

// Element is one item on the canvas. Concrete elements live in elements.go.
// Implementations are pointer types (SetBounds mutates), so Scene.Layers holds pointers.
type Element interface {
	Bounds() Rect
	SetBounds(Rect)
	Kind() string
}

// base is embedded by every concrete element to supply the common Rect + accessors.
// (Composition instead of inheritance — the Go way.)
type base struct {
	Rect Rect `json:"rect"`
}

func (b base) Bounds() Rect      { return b.Rect }
func (b *base) SetBounds(r Rect) { b.Rect = r }

// Scene is the whole document: canvas size, animation timing, and layers bottom→top.
// Layer order IS z-order: Layers[0] is drawn first (bottom); the last is on top.
//
// Transparent leaves the canvas background unfilled (a transparent GIF that blends into the
// README's theme). Ink is the default foreground color for elements that don't set their own
// — the single color of the monochrome look.
type Scene struct {
	W           int
	H           int
	FPS         int
	DurationMs  int
	Transparent bool
	Ink         string
	Layers      []Element
}

// NewScene returns an empty scene with sensible animation defaults.
func NewScene(w, h int) *Scene {
	return &Scene{W: w, H: h, FPS: 15, DurationMs: 3000}
}

// FrameCount is how many frames the animation spans at the scene's FPS/duration.
func (s *Scene) FrameCount() int {
	n := s.FPS * s.DurationMs / 1000
	if n < 1 {
		return 1
	}
	return n
}

// TopmostAt returns the highest (last-drawn) element containing the point, or (-1, nil).
// Used for click hit-testing in the editor.
func (s *Scene) TopmostAt(x, y int) (int, Element) {
	for i := len(s.Layers) - 1; i >= 0; i-- {
		if s.Layers[i].Bounds().Contains(x, y) {
			return i, s.Layers[i]
		}
	}
	return -1, nil
}

// Add appends an element on top and returns its index.
func (s *Scene) Add(el Element) int {
	s.Layers = append(s.Layers, el)
	return len(s.Layers) - 1
}

// Remove deletes the element at i (no-op if out of range).
func (s *Scene) Remove(i int) {
	if i < 0 || i >= len(s.Layers) {
		return
	}
	s.Layers = append(s.Layers[:i], s.Layers[i+1:]...)
}

// Raise moves the element at i one step toward the top; returns its new index.
func (s *Scene) Raise(i int) int {
	if i < 0 || i >= len(s.Layers)-1 {
		return i
	}
	s.Layers[i], s.Layers[i+1] = s.Layers[i+1], s.Layers[i]
	return i + 1
}

// Lower moves the element at i one step toward the bottom; returns its new index.
func (s *Scene) Lower(i int) int {
	if i <= 0 || i >= len(s.Layers) {
		return i
	}
	s.Layers[i], s.Layers[i-1] = s.Layers[i-1], s.Layers[i]
	return i - 1
}

// --- JSON: interface slices need a type-tagged envelope ---
//
// Go can't unmarshal into an interface, so on disk each layer is an object carrying a
// "kind" discriminator alongside its fields. This is the standard tagged-union pattern.

type sceneJSON struct {
	W           int               `json:"w"`
	H           int               `json:"h"`
	FPS         int               `json:"fps"`
	DurationMs  int               `json:"durationMs"`
	Transparent bool              `json:"transparent,omitempty"`
	Ink         string            `json:"ink,omitempty"`
	Layers      []json.RawMessage `json:"layers"`
}

// MarshalJSON emits each layer as its fields plus a "kind" tag.
func (s Scene) MarshalJSON() ([]byte, error) {
	out := sceneJSON{W: s.W, H: s.H, FPS: s.FPS, DurationMs: s.DurationMs, Transparent: s.Transparent, Ink: s.Ink}
	for _, el := range s.Layers {
		raw, err := json.Marshal(el)
		if err != nil {
			return nil, err
		}
		var m map[string]json.RawMessage
		if err := json.Unmarshal(raw, &m); err != nil {
			return nil, err
		}
		kind, err := json.Marshal(el.Kind())
		if err != nil {
			return nil, err
		}
		m["kind"] = kind
		merged, err := json.Marshal(m)
		if err != nil {
			return nil, err
		}
		out.Layers = append(out.Layers, merged)
	}
	return json.Marshal(out)
}

// UnmarshalJSON reconstructs concrete elements by switching on each layer's "kind".
func (s *Scene) UnmarshalJSON(data []byte) error {
	var sj sceneJSON
	if err := json.Unmarshal(data, &sj); err != nil {
		return err
	}
	s.W, s.H, s.FPS, s.DurationMs = sj.W, sj.H, sj.FPS, sj.DurationMs
	s.Transparent, s.Ink = sj.Transparent, sj.Ink
	s.Layers = nil
	for _, raw := range sj.Layers {
		el, err := unmarshalElement(raw)
		if err != nil {
			return err
		}
		s.Layers = append(s.Layers, el)
	}
	return nil
}

// Load reads a scene JSON file from disk.
func Load(path string) (*Scene, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Scene
	if err := json.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("scene: parse %s: %w", path, err)
	}
	return &s, nil
}

// Save writes the scene to disk as indented JSON.
func (s *Scene) Save(path string) error {
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
