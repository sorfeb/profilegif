// Package tui is the interactive terminal editor. It follows Bubble Tea's Elm architecture:
// a Model holds all state, Update maps messages (keys, mouse, ticks) to a new Model, and
// View renders the Model to a string. The canvas is drawn by rasterizing the scene to real
// pixels (internal/render) and converting to half-block cells (internal/termimg); selection
// outlines and resize handles are painted onto that image before conversion.
//
// For a React dev: Model ≈ your component state, Update ≈ a reducer (msg in, state out),
// View ≈ render(). There's no virtual DOM — View returns the whole screen as a string each
// frame, and Bubble Tea diffs it against the terminal for you.
package tui

import (
	"image"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/sorfeb/profilegif/internal/scene"
	"github.com/sorfeb/profilegif/internal/termimg"
)

const (
	sidebarWidth = 34 // right panel width in cells
	nudgeStep    = 10 // logical px moved per arrow press
	minElemSize  = 16 // smallest element W/H when resizing
)

// interactionMode is what a mouse drag currently does.
type interactionMode int

const (
	modeIdle interactionMode = iota
	modeMove
	modeResize
)

// tickMsg advances the animation preview.
type tickMsg time.Time

// Model is the whole editor state.
type Model struct {
	scene     *scene.Scene
	scenePath string
	renderer  termimg.Renderer

	selected int  // index into scene.Layers, or -1
	frame    int  // current preview frame
	playing  bool // animation playing?

	width, height int // terminal size in cells

	// baseCache holds rasterized frames (without the selection overlay), keyed by frame
	// index. It's size-independent — a resize only re-scales cached pixels instead of
	// re-drawing them — which keeps rapid resizes (Ctrl +/-) from backing up the event loop.
	// Invalidated via invalidate() whenever the scene changes. (Maps are reference types, so
	// writes from View and clears from Update share one backing store.)
	baseCache map[int]*image.RGBA

	// drag state (Phase 5)
	mode     interactionMode
	dragging bool
	origRect scene.Rect // element bounds at drag start
	grabX    int        // logical-px grab offset within the element
	grabY    int

	// text-input overlay (adding elements, editing fields, save/export paths)
	input       textinput.Model
	inputActive bool
	inputFor    inputPurpose

	status string // one-line status/help feedback
	quit   bool
}

// New builds a Model around an existing scene.
func New(s *scene.Scene, path string) Model {
	sel := -1
	if len(s.Layers) > 0 {
		sel = len(s.Layers) - 1 // start with the topmost element selected
	}
	ti := textinput.New()
	ti.CharLimit = 256
	ti.Width = 40
	return Model{
		scene:     s,
		scenePath: path,
		renderer:  termimg.HalfBlock{},
		selected:  sel,
		playing:   true,
		input:     ti,
		baseCache: map[int]*image.RGBA{},
		status:    "ready",
	}
}

// Run launches the editor and blocks until the user quits.
func Run(s *scene.Scene, path string) error {
	p := tea.NewProgram(
		New(s, path),
		tea.WithAltScreen(),       // full-screen, restore terminal on exit
		tea.WithMouseCellMotion(), // press/release + motion *while a button is held* → drag
	)
	_, err := p.Run()
	return err
}

func (m Model) Init() tea.Cmd {
	return tickCmd(m.scene.FPS)
}

func tickCmd(fps int) tea.Cmd {
	if fps < 1 {
		fps = 1
	}
	d := time.Second / time.Duration(fps)
	return tea.Tick(d, func(t time.Time) tea.Msg { return tickMsg(t) })
}

// sel returns the currently selected element, or nil.
func (m *Model) sel() scene.Element {
	if m.selected < 0 || m.selected >= len(m.scene.Layers) {
		return nil
	}
	return m.scene.Layers[m.selected]
}

// invalidate drops cached frames after a scene edit so the next render reflects the change.
// (Selection/size changes don't need this — the overlay is applied per-render and the cache
// is size-independent.)
func (m *Model) invalidate() {
	clear(m.baseCache)
}
