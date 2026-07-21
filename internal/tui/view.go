package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/sorfeb/profilegif/internal/scene"
)

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("240"))

	sidebarStyle = borderStyle.Padding(0, 1)

	titleStyle   = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("39"))
	sectionStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("244"))
	selStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39")).Bold(true)
	dimStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("244"))
	keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("39"))
)

func (m Model) View() string {
	if m.width < 20 || m.height < 8 {
		return "profilegif edit — terminal too small (need at least 20×8)"
	}

	vp := m.viewportGeom()

	canvasBox := borderStyle.Render(m.canvasView(vp))
	// Match the sidebar height to the canvas content so the two boxes align.
	sidebar := sidebarStyle.
		Width(sidebarWidth - 2).
		Height(vp.rows).
		Render(m.sidebarContent())

	body := lipgloss.JoinHorizontal(lipgloss.Top, canvasBox, sidebar)

	bottom := m.helpView()
	if m.inputActive {
		bottom = keyStyle.Render(m.input.View())
	}
	return lipgloss.JoinVertical(lipgloss.Left, body, bottom)
}

func (m Model) sidebarContent() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("profilegif edit"))
	b.WriteString("\n")
	b.WriteString(dimStyle.Render(fmt.Sprintf("canvas %d×%d · %dfps", m.scene.W, m.scene.H, m.scene.FPS)))
	b.WriteString("\n\n")

	b.WriteString(sectionStyle.Render("LAYERS (top→bottom)"))
	b.WriteString("\n")
	if len(m.scene.Layers) == 0 {
		b.WriteString(dimStyle.Render("  (empty)\n"))
	}
	for i := len(m.scene.Layers) - 1; i >= 0; i-- {
		el := m.scene.Layers[i]
		line := fmt.Sprintf("%s %s", layerIcon(el), layerLabel(el))
		if i == m.selected {
			b.WriteString(selStyle.Render("▸ " + line))
		} else {
			b.WriteString(dimStyle.Render("  " + line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(sectionStyle.Render("INSPECTOR"))
	b.WriteString("\n")
	b.WriteString(m.inspectorContent())

	return b.String()
}

func (m Model) inspectorContent() string {
	el := m.sel()
	if el == nil {
		return dimStyle.Render("  no selection")
	}
	r := el.Bounds()
	var b strings.Builder
	fmt.Fprintf(&b, "  kind   %s\n", el.Kind())
	fmt.Fprintf(&b, "  pos    %d, %d\n", r.X, r.Y)
	fmt.Fprintf(&b, "  size   %d × %d\n", r.W, r.H)

	switch e := el.(type) {
	case *scene.TextElement:
		fmt.Fprintf(&b, "  text   %s\n", truncate(e.Text, 18))
		fmt.Fprintf(&b, "  color  %s\n", e.Color)
		fmt.Fprintf(&b, "  font   %.0f\n", e.FontSize)
	case *scene.StatWidget:
		fmt.Fprintf(&b, "  metric %s\n", e.Metric)
		fmt.Fprintf(&b, "  login  %s\n", truncate(e.Login, 18))
		fmt.Fprintf(&b, "  value  %d\n", e.Value)
		fmt.Fprintf(&b, "  color  %s\n", e.Color)
	case *scene.Background:
		fmt.Fprintf(&b, "  path   %s\n", truncate(e.Path, 18))
		fmt.Fprintf(&b, "  fit    %s\n", e.Fit)
	case *scene.ImageElement:
		fmt.Fprintf(&b, "  path   %s\n", truncate(e.Path, 18))
		fmt.Fprintf(&b, "  fit    %s\n", e.Fit)
	}
	return dimStyle.Render(b.String())
}

func (m Model) helpView() string {
	keys := []struct{ k, d string }{
		{"tab", "select"},
		{"↑↓←→/drag", "move"},
		{"t/i/g/b", "add"},
		{"↵", "edit"},
		{"[ ]", "z"},
		{"space", "play"},
		{"d", "del"},
		{"s/e", "save/export"},
		{"q", "quit"},
	}
	parts := make([]string, 0, len(keys)+1)
	for _, kv := range keys {
		parts = append(parts, keyStyle.Render(kv.k)+" "+dimStyle.Render(kv.d))
	}
	help := strings.Join(parts, dimStyle.Render(" · "))
	if m.status != "" {
		help += dimStyle.Render("   [" + m.status + "]")
	}
	return help
}

func layerIcon(el scene.Element) string {
	switch el.(type) {
	case *scene.Background:
		return "▦"
	case *scene.TextElement:
		return "T"
	case *scene.ImageElement:
		return "▤"
	case *scene.StatWidget:
		return "▮"
	default:
		return "?"
	}
}

func layerLabel(el scene.Element) string {
	switch e := el.(type) {
	case *scene.Background:
		return "background"
	case *scene.TextElement:
		return "text: " + truncate(e.Text, 14)
	case *scene.ImageElement:
		return "image"
	case *scene.StatWidget:
		if e.Label != "" {
			return "stat: " + e.Label
		}
		return "stat: " + e.Metric
	default:
		return el.Kind()
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}
