package tui

import (
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sorfeb/profilegif/internal/render"
	"github.com/sorfeb/profilegif/internal/scene"
)

// inputPurpose is what the current text-input overlay is collecting.
type inputPurpose int

const (
	inputNone inputPurpose = iota
	inputAddText
	inputAddImage
	inputAddStat
	inputAddBackground
	inputEditPrimary
	inputSavePath
	inputExportPath
)

// startInput opens the text-input overlay. Returns the Blink command that animates the
// cursor. Mutates the model, so call it, then return the (now-updated) model with the cmd.
func (m *Model) startInput(p inputPurpose, prompt, initial string) tea.Cmd {
	m.inputFor = p
	m.inputActive = true
	m.input.Prompt = prompt + " "
	m.input.SetValue(initial)
	m.input.CursorEnd()
	return m.input.Focus()
}

// handleInputKey routes keystrokes to the overlay: Enter commits, Esc cancels, else type.
func (m Model) handleInputKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.closeInput("cancelled")
		return m, nil
	case "enter":
		m.commitInput()
		m.closeInput(m.status)
		return m, nil
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m *Model) closeInput(status string) {
	m.inputActive = false
	m.inputFor = inputNone
	m.input.Blur()
	m.status = status
}

// commitInput applies the entered value according to what the overlay was collecting.
func (m *Model) commitInput() {
	v := strings.TrimSpace(m.input.Value())
	switch m.inputFor {
	case inputAddText:
		if v == "" {
			v = "text"
		}
		m.selected = m.scene.Add(scene.NewText(m.centerRect(320, 90), v))
		m.status = "added text"

	case inputAddImage:
		m.selected = m.scene.Add(scene.NewImage(m.centerRect(240, 240), v))
		m.status = "added image"

	case inputAddStat:
		if v == "" {
			v = "octocat"
		}
		sw := scene.NewStatWidget(m.centerRect(200, 180), scene.MetricCommits, v)
		sw.Value = 1000
		sw.Label = "commits"
		m.selected = m.scene.Add(sw)
		m.status = "added stat"

	case inputAddBackground:
		// Backgrounds belong at the bottom of the stack.
		bg := scene.NewBackground(v, m.scene.W, m.scene.H)
		m.scene.Layers = append([]scene.Element{bg}, m.scene.Layers...)
		m.selected = 0
		m.status = "set background"

	case inputEditPrimary:
		m.applyPrimaryEdit(v)

	case inputSavePath:
		m.doSave(v)

	case inputExportPath:
		m.doExport(v)
	}
}

// centerRect returns a w×h rect centered on the canvas.
func (m *Model) centerRect(w, h int) scene.Rect {
	return scene.Rect{X: (m.scene.W - w) / 2, Y: (m.scene.H - h) / 2, W: w, H: h}
}

// primaryEdit reports the label + current value of an element's main editable field.
func primaryEdit(el scene.Element) (label, current string) {
	switch e := el.(type) {
	case *scene.TextElement:
		return "Text:", e.Text
	case *scene.StatWidget:
		return "Login:", e.Login
	case *scene.ImageElement:
		return "Image path:", e.Path
	case *scene.Background:
		return "Background path:", e.Path
	}
	return "Value:", ""
}

func (m *Model) applyPrimaryEdit(v string) {
	switch e := m.sel().(type) {
	case *scene.TextElement:
		e.Text = v
	case *scene.StatWidget:
		e.Login = v
	case *scene.ImageElement:
		e.Path = v
	case *scene.Background:
		e.Path = v
	default:
		return
	}
	m.status = "edited"
}

func (m *Model) doSave(path string) {
	if path == "" {
		path = "scene.json"
	}
	if err := m.scene.Save(path); err != nil {
		m.status = "save failed: " + err.Error()
		return
	}
	m.scenePath = path
	m.status = "saved " + path
}

func (m *Model) doExport(path string) {
	if path == "" {
		path = "profilegif.gif"
	}
	f, err := os.Create(path)
	if err != nil {
		m.status = "export failed: " + err.Error()
		return
	}
	defer f.Close()
	if err := render.EncodeScene(f, m.scene); err != nil {
		m.status = "export failed: " + err.Error()
		return
	}
	m.status = "exported " + path
}

// defaultLogin picks a login for new stat widgets: reuse an existing one if present.
func (m *Model) defaultLogin() string {
	for _, el := range m.scene.Layers {
		if sw, ok := el.(*scene.StatWidget); ok && sw.Login != "" {
			return sw.Login
		}
	}
	return "octocat"
}
