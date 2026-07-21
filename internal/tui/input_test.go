package tui

import (
	"image/gif"
	"os"
	"path/filepath"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sorfeb/profilegif/internal/scene"
)

// typeText opens/uses the overlay: set the value and press Enter to commit.
func commitValue(m Model, value string) Model {
	m.input.SetValue(value)
	return key(m, tea.Key{Type: tea.KeyEnter})
}

func TestAddTextElement(t *testing.T) {
	m := sized()
	n := len(m.scene.Layers)

	m = key(m, tea.Key{Type: tea.KeyRunes, Runes: []rune{'t'}})
	if !m.inputActive {
		t.Fatal("'t' should open the text input")
	}
	m = commitValue(m, "Hello world")

	if len(m.scene.Layers) != n+1 {
		t.Fatalf("layer count: got %d want %d", len(m.scene.Layers), n+1)
	}
	txt, ok := m.scene.Layers[len(m.scene.Layers)-1].(*scene.TextElement)
	if !ok {
		t.Fatalf("new layer not text: %T", m.scene.Layers[len(m.scene.Layers)-1])
	}
	if txt.Text != "Hello world" {
		t.Errorf("text: got %q want %q", txt.Text, "Hello world")
	}
	if m.inputActive {
		t.Error("input should close after commit")
	}
}

func TestAddBackgroundGoesToBottom(t *testing.T) {
	m := sized()
	m = key(m, tea.Key{Type: tea.KeyRunes, Runes: []rune{'b'}})
	m = commitValue(m, "bg.png")
	if _, ok := m.scene.Layers[0].(*scene.Background); !ok {
		t.Errorf("background should be layer 0, got %T", m.scene.Layers[0])
	}
}

func TestEscCancelsInput(t *testing.T) {
	m := sized()
	n := len(m.scene.Layers)
	m = key(m, tea.Key{Type: tea.KeyRunes, Runes: []rune{'t'}})
	m = key(m, tea.Key{Type: tea.KeyEsc})
	if m.inputActive {
		t.Error("esc should close the input")
	}
	if len(m.scene.Layers) != n {
		t.Error("esc should not add an element")
	}
}

func TestEditPrimaryField(t *testing.T) {
	m := sized()
	m.selected = 0 // the "@sorfeb" text
	m = key(m, tea.Key{Type: tea.KeyEnter})
	if !m.inputActive {
		t.Fatal("enter should open edit input for the selected element")
	}
	m = commitValue(m, "@newname")
	if got := m.scene.Layers[0].(*scene.TextElement).Text; got != "@newname" {
		t.Errorf("edited text: got %q want %q", got, "@newname")
	}
}

func TestSaveAndReload(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "myscene.json")

	m := sized()
	m = key(m, tea.Key{Type: tea.KeyRunes, Runes: []rune{'s'}})
	m = commitValue(m, path)

	if _, err := os.Stat(path); err != nil {
		t.Fatalf("scene file not written: %v", err)
	}
	reloaded, err := scene.Load(path)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if len(reloaded.Layers) != len(m.scene.Layers) {
		t.Errorf("reloaded layers: got %d want %d", len(reloaded.Layers), len(m.scene.Layers))
	}
	if m.scenePath != path {
		t.Errorf("scenePath not recorded: got %q", m.scenePath)
	}
}

func TestExportGIF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "out.gif")

	m := sized()
	m = key(m, tea.Key{Type: tea.KeyRunes, Runes: []rune{'e'}})
	m = commitValue(m, path)

	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("gif not written: %v", err)
	}
	defer f.Close()
	if _, err := gif.DecodeAll(f); err != nil {
		t.Fatalf("exported gif invalid: %v", err)
	}
}

func TestExportDefaultName(t *testing.T) {
	cases := map[string]string{
		"":                    "profilegif.gif",
		"scene.json":          "scene.gif",
		"dir/my.scene.json":   "my.scene.gif",
		`C:\a\b\profile.json`: "profile.gif",
	}
	for in, want := range cases {
		if got := exportDefault(in); got != want {
			t.Errorf("exportDefault(%q): got %q want %q", in, got, want)
		}
	}
}
