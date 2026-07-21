package tui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sorfeb/profilegif/internal/scene"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		return m, nil

	case tickMsg:
		if m.playing {
			m.frame = (m.frame + 1) % m.scene.FrameCount()
		}
		return m, tickCmd(m.scene.FPS)

	case tea.KeyMsg:
		if m.inputActive {
			return m.handleInputKey(msg)
		}
		return m.handleKey(msg)

	case tea.MouseMsg:
		if m.inputActive {
			return m, nil // ignore mouse while typing
		}
		return m.handleMouse(msg)
	}

	// Forward any other message (e.g. the cursor's blink) to the text input when it's open.
	if m.inputActive {
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) handleKey(k tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch k.String() {
	case "q", "ctrl+c":
		m.quit = true
		return m, tea.Quit

	case "tab":
		m.cycleSelection(1)
	case "shift+tab":
		m.cycleSelection(-1)

	case " ":
		m.playing = !m.playing
		m.status = map[bool]string{true: "playing", false: "paused"}[m.playing]
	case "r":
		m.frame = 0
		m.status = "rewound"

	case "up":
		m.nudge(0, -nudgeStep)
	case "down":
		m.nudge(0, nudgeStep)
	case "left":
		m.nudge(-nudgeStep, 0)
	case "right":
		m.nudge(nudgeStep, 0)

	case "]":
		if m.selected >= 0 {
			m.selected = m.scene.Raise(m.selected)
			m.status = "raised"
		}
	case "[":
		if m.selected >= 0 {
			m.selected = m.scene.Lower(m.selected)
			m.status = "lowered"
		}

	case "d", "delete", "backspace":
		m.deleteSelected()

	// --- element authoring (open the text-input overlay) ---
	// NOTE: startInput mutates m, so capture its cmd first, then return the mutated m —
	// `return m, m.startInput(...)` would copy the pre-mutation m into the first result.
	case "t":
		cmd := m.startInput(inputAddText, "Text:", "New text")
		return m, cmd
	case "i":
		cmd := m.startInput(inputAddImage, "Image path:", "")
		return m, cmd
	case "g":
		cmd := m.startInput(inputAddStat, "GitHub login:", m.defaultLogin())
		return m, cmd
	case "b":
		cmd := m.startInput(inputAddBackground, "Background path:", "")
		return m, cmd
	case "enter":
		if el := m.sel(); el != nil {
			label, cur := primaryEdit(el)
			cmd := m.startInput(inputEditPrimary, label, cur)
			return m, cmd
		}

	// --- persistence ---
	case "s":
		if m.scenePath != "" {
			m.doSave(m.scenePath)
			return m, nil
		}
		cmd := m.startInput(inputSavePath, "Save scene as:", "scene.json")
		return m, cmd
	case "e":
		cmd := m.startInput(inputExportPath, "Export GIF as:", exportDefault(m.scenePath))
		return m, cmd
	}
	return m, nil
}

// exportDefault suggests a .gif filename, derived from the scene path when there is one.
func exportDefault(scenePath string) string {
	if scenePath == "" {
		return "profilegif.gif"
	}
	base := scenePath
	if i := strings.LastIndexAny(base, `/\`); i >= 0 {
		base = base[i+1:]
	}
	if i := strings.LastIndex(base, "."); i > 0 {
		base = base[:i]
	}
	return base + ".gif"
}

// --- mutations (pointer receivers; m is addressable as a value param) ---

func (m *Model) cycleSelection(dir int) {
	n := len(m.scene.Layers)
	if n == 0 {
		m.selected = -1
		return
	}
	if m.selected < 0 {
		m.selected = 0
		return
	}
	m.selected = (m.selected + dir + n) % n
}

func (m *Model) nudge(dx, dy int) {
	el := m.sel()
	if el == nil {
		return
	}
	r := el.Bounds()
	r.X += dx
	r.Y += dy
	el.SetBounds(m.clampToCanvas(r))
	m.status = "moved"
}

func (m *Model) deleteSelected() {
	if m.selected < 0 {
		return
	}
	m.scene.Remove(m.selected)
	if m.selected >= len(m.scene.Layers) {
		m.selected = len(m.scene.Layers) - 1
	}
	m.status = "deleted"
}

// clampToCanvas keeps an element's origin within the canvas so it can't be lost off-screen.
func (m *Model) clampToCanvas(r scene.Rect) scene.Rect {
	if r.X < 0 {
		r.X = 0
	}
	if r.Y < 0 {
		r.Y = 0
	}
	if r.X > m.scene.W-1 {
		r.X = m.scene.W - 1
	}
	if r.Y > m.scene.H-1 {
		r.Y = m.scene.H - 1
	}
	return r
}
