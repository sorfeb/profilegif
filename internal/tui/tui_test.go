package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/sorfeb/profilegif/internal/scene"
)

func testScene() *scene.Scene {
	s := scene.NewScene(800, 400)
	s.FPS, s.DurationMs = 10, 1000
	s.Add(scene.NewText(scene.Rect{X: 0, Y: 20, W: 800, H: 80}, "@sorfeb"))
	sw := scene.NewStatWidget(scene.Rect{X: 60, Y: 150, W: 200, H: 180}, scene.MetricCommits, "sorfeb")
	sw.Value = 4237
	s.Add(sw)
	return s
}

// sized returns a model that has received a terminal size (so View can lay out).
func sized() Model {
	m := New(testScene(), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return next.(Model)
}

func key(m Model, k tea.Key) Model {
	next, _ := m.Update(tea.KeyMsg(k))
	return next.(Model)
}

func TestViewRendersWithoutPanic(t *testing.T) {
	m := sized()
	out := m.View()
	if out == "" {
		t.Fatal("empty view")
	}
	for _, want := range []string{"profilegif edit", "LAYERS", "INSPECTOR"} {
		if !strings.Contains(out, want) {
			t.Errorf("view missing %q", want)
		}
	}
}

func TestViewTooSmall(t *testing.T) {
	m := New(testScene(), "")
	next, _ := m.Update(tea.WindowSizeMsg{Width: 10, Height: 5})
	if out := next.(Model).View(); !strings.Contains(out, "too small") {
		t.Errorf("expected too-small notice, got %q", out)
	}
}

func TestSelectionCycles(t *testing.T) {
	m := sized() // starts with topmost (index 1) selected
	if m.selected != 1 {
		t.Fatalf("initial selection: got %d want 1", m.selected)
	}
	m = key(m, tea.Key{Type: tea.KeyTab}) // 1 -> 0 (wraps within 2 layers: (1+1)%2=0)
	if m.selected != 0 {
		t.Errorf("after tab: got %d want 0", m.selected)
	}
}

func TestNudgeMovesSelected(t *testing.T) {
	m := sized()
	before := m.sel().Bounds()
	m = key(m, tea.Key{Type: tea.KeyRight})
	after := m.sel().Bounds()
	if after.X != before.X+nudgeStep {
		t.Errorf("nudge right: got X=%d want %d", after.X, before.X+nudgeStep)
	}
}

func TestDeleteRemovesLayer(t *testing.T) {
	m := sized()
	n := len(m.scene.Layers)
	m = key(m, tea.Key{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if len(m.scene.Layers) != n-1 {
		t.Errorf("after delete: got %d layers want %d", len(m.scene.Layers), n-1)
	}
}

func TestZOrderKeys(t *testing.T) {
	m := sized()
	m.selected = 0 // bottom
	m = key(m, tea.Key{Type: tea.KeyRunes, Runes: []rune{']'}})
	if m.selected != 1 {
		t.Errorf("raise: selection should follow to 1, got %d", m.selected)
	}
}

func TestPlayPauseAndTick(t *testing.T) {
	m := sized()
	if !m.playing {
		t.Fatal("should start playing")
	}
	// tick advances the frame while playing.
	next, _ := m.Update(tickMsg{})
	m = next.(Model)
	if m.frame != 1 {
		t.Errorf("after tick: frame=%d want 1", m.frame)
	}
	// space pauses; a tick then should not advance.
	m = key(m, tea.Key{Type: tea.KeySpace})
	if m.playing {
		t.Fatal("space should pause")
	}
	next, _ = m.Update(tickMsg{})
	if next.(Model).frame != 1 {
		t.Errorf("paused tick advanced frame to %d", next.(Model).frame)
	}
}

func TestQuitKey(t *testing.T) {
	m := sized()
	_, cmd := m.Update(tea.KeyMsg(tea.Key{Type: tea.KeyRunes, Runes: []rune{'q'}}))
	if cmd == nil {
		t.Fatal("q should return a command (tea.Quit)")
	}
}
