package tui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func mouse(m Model, x, y int, a tea.MouseAction, b tea.MouseButton) Model {
	next, _ := m.Update(tea.MouseMsg{X: x, Y: y, Action: a, Button: b})
	return next.(Model)
}

// For a 120×40 terminal with scene 800×400, the canvas is 84×21 cells at origin (1,1),
// scaleX≈9.52, scaleY≈19.05. The commits widget is layer 1 at rect (60,150,200,190),
// so its interior maps around cell (18,14) and its bottom-right handle around cell (27,18).

func TestMouseSelectAndMove(t *testing.T) {
	m := sized()

	m = mouse(m, 18, 14, tea.MouseActionPress, tea.MouseButtonLeft)
	if m.selected != 1 {
		t.Fatalf("press should select commits widget (1), got %d", m.selected)
	}
	if !m.dragging || m.mode != modeMove {
		t.Fatalf("press should start a move drag; dragging=%v mode=%v", m.dragging, m.mode)
	}

	before := m.sel().Bounds()
	m = mouse(m, 34, 14, tea.MouseActionMotion, tea.MouseButtonLeft)
	after := m.sel().Bounds()
	if after.X <= before.X {
		t.Errorf("drag right should increase X: before %d after %d", before.X, after.X)
	}
	if after.W != before.W || after.H != before.H {
		t.Errorf("move should not change size: %v -> %v", before, after)
	}

	m = mouse(m, 34, 14, tea.MouseActionRelease, tea.MouseButtonLeft)
	if m.dragging || m.mode != modeIdle {
		t.Errorf("release should end the drag")
	}
}

func TestMouseResizeFromHandle(t *testing.T) {
	m := sized()
	m.selected = 1 // commits widget

	m = mouse(m, 27, 17, tea.MouseActionPress, tea.MouseButtonLeft)
	if !m.dragging || m.mode != modeResize {
		t.Fatalf("press on handle should start resize; dragging=%v mode=%v", m.dragging, m.mode)
	}

	before := m.sel().Bounds()
	m = mouse(m, 40, 24, tea.MouseActionMotion, tea.MouseButtonLeft)
	after := m.sel().Bounds()
	if after.W <= before.W {
		t.Errorf("resize should grow width: before %d after %d", before.W, after.W)
	}
	if after.H <= before.H {
		t.Errorf("resize should grow height: before %d after %d", before.H, after.H)
	}
	if after.X != before.X || after.Y != before.Y {
		t.Errorf("resize should not move origin: %v -> %v", before, after)
	}
}

func TestMouseResizeClampsMinimum(t *testing.T) {
	m := sized()
	m.selected = 1
	m = mouse(m, 27, 17, tea.MouseActionPress, tea.MouseButtonLeft)
	// Drag the handle far up-left, past the origin → size must clamp, not go negative.
	m = mouse(m, 2, 2, tea.MouseActionMotion, tea.MouseButtonLeft)
	r := m.sel().Bounds()
	if r.W < minElemSize || r.H < minElemSize {
		t.Errorf("size should clamp to >= %d, got %dx%d", minElemSize, r.W, r.H)
	}
}

func TestMouseClickEmptyDeselects(t *testing.T) {
	m := sized()
	// Cell (43,7) → ~(404,123): the gap between the title and the widgets.
	m = mouse(m, 43, 7, tea.MouseActionPress, tea.MouseButtonLeft)
	if m.selected != -1 {
		t.Errorf("clicking empty canvas should deselect, got %d", m.selected)
	}
}

func TestMouseOnSidebarIgnored(t *testing.T) {
	m := sized()
	start := m.selected
	// A press well into the sidebar (x > canvas width) shouldn't change selection.
	m = mouse(m, 110, 10, tea.MouseActionPress, tea.MouseButtonLeft)
	if m.selected != start {
		t.Errorf("sidebar click should be ignored; selection changed %d -> %d", start, m.selected)
	}
}
