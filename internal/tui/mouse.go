package tui

import tea "github.com/charmbracelet/bubbletea"

// handleMouse turns raw mouse events into select / move / resize interactions.
//
//   - left press  → select the topmost element under the cursor and begin moving it, OR, if
//     the press lands on the selected element's bottom-right handle, begin resizing.
//   - motion (while a button is held) → apply the move/resize.
//   - release → commit and return to idle.
func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case tea.MouseActionPress:
		if msg.Button == tea.MouseButtonLeft {
			m.onPress(msg.X, msg.Y)
		}
	case tea.MouseActionMotion:
		if m.dragging {
			m.onDrag(msg.X, msg.Y)
		}
	case tea.MouseActionRelease:
		m.onRelease()
	}
	return m, nil
}

func (m *Model) onPress(cellX, cellY int) {
	vp := m.viewportGeom()
	lx, ly, inside := vp.cellToLogical(cellX, cellY)
	if !inside {
		return // click landed on the sidebar/help, not the canvas
	}

	// If the current selection's resize handle is grabbed, start resizing.
	if el := m.sel(); el != nil {
		r := el.Bounds()
		hsX := max(int(vp.scaleX*1.5), 1)
		hsY := max(int(vp.scaleY*1.5), 1)
		if lx >= r.X+r.W-hsX && lx <= r.X+r.W && ly >= r.Y+r.H-hsY && ly <= r.Y+r.H {
			m.mode = modeResize
			m.dragging = true
			m.origRect = r
			m.status = "resizing"
			return
		}
	}

	// Otherwise select the topmost element under the cursor and start moving it.
	if idx, el := m.scene.TopmostAt(lx, ly); el != nil {
		r := el.Bounds()
		m.selected = idx
		m.mode = modeMove
		m.dragging = true
		m.origRect = r
		m.grabX = lx - r.X // keep the cursor's offset within the element constant
		m.grabY = ly - r.Y
		m.status = "moving"
		return
	}

	// Clicked empty canvas → deselect.
	m.selected = -1
	m.status = "deselected"
}

func (m *Model) onDrag(cellX, cellY int) {
	el := m.sel()
	if el == nil {
		m.dragging = false
		return
	}
	vp := m.viewportGeom()
	lx, ly := vp.cellToLogicalClamped(cellX, cellY, m.scene.W, m.scene.H)
	r := el.Bounds()
	switch m.mode {
	case modeMove:
		r.X = lx - m.grabX
		r.Y = ly - m.grabY
		el.SetBounds(m.clampToCanvas(r))
	case modeResize:
		r.W = lx - r.X
		r.H = ly - r.Y
		if r.W < minElemSize {
			r.W = minElemSize
		}
		if r.H < minElemSize {
			r.H = minElemSize
		}
		el.SetBounds(r)
	}
	m.invalidate()
}

func (m *Model) onRelease() {
	if m.dragging {
		m.dragging = false
		m.mode = modeIdle
		m.status = "ready"
	}
}
