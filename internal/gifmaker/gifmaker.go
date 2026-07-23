// Package gifmaker is the GitHub-stats front door: it fetches a user's stats and turns them
// into a ready-to-serve animated GIF. It owns the *default* composition (DefaultScene) but
// delegates all pixel work to internal/render and the document model to internal/scene, so
// the web server and the TUI editor share one rendering pipeline.
//
// Keep this package free of HTTP/CLI assumptions — it just takes inputs and writes bytes.
package gifmaker

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/sorfeb/profilegif/internal/render"
	"github.com/sorfeb/profilegif/internal/scene"
)

// Stats holds the GitHub data we animate into frames.
type Stats struct {
	Login        string
	TotalCommits int
	Followers    int
	Stars        int
	// Future: Languages []LangStat, ContributionDays []int, Streak int, etc.
}

// Theme names for the default composition. GIFs can't adapt to the viewer's GitHub theme,
// so we render a variant per theme and let the README pick with #gh-light/dark-mode-only.
const (
	ThemeDark  = "dark"
	ThemeLight = "light"
)

// DefaultScene builds the standard stats composition (dark theme) from fetched Stats.
func DefaultScene(s Stats) *scene.Scene { return DefaultSceneTheme(s, ThemeDark) }

// DefaultSceneTheme builds the monochrome, terminal/ASCII stats card on a transparent
// background: a monospace title plus one "label  value  [████░░░░]" meter line per stat.
// The single ink color is chosen for the given theme.
func DefaultSceneTheme(s Stats, theme string) *scene.Scene {
	sc := scene.NewScene(560, 220)
	sc.Transparent = true
	sc.Ink = inkForTheme(theme)
	sc.FPS = 15
	sc.DurationMs = 2500

	title := scene.NewText(scene.Rect{X: 28, Y: 26, W: 504, H: 44}, "@"+s.Login)
	title.Mono = true
	title.FontSize = 34
	title.Color = "" // inherit the scene's monochrome ink (NewText defaults to white)
	sc.Add(title)

	add := func(y int, metric, label string, value, max int) {
		w := scene.NewStatWidget(scene.Rect{X: 28, Y: y, W: 504, H: 40}, metric, s.Login)
		w.Label = label
		w.Value = value
		w.Max = max
		w.BarCells = 10
		w.FontSize = 26
		w.Color = "" // inherit the scene's monochrome ink
		sc.Add(w)
	}
	add(92, scene.MetricCommits, "commits", s.TotalCommits, 5000)
	add(132, scene.MetricFollowers, "followers", s.Followers, 500)
	add(172, scene.MetricStars, "stars", s.Stars, 1000)

	return sc
}

// inkForTheme returns the single foreground color legible on that GitHub theme's surface.
func inkForTheme(theme string) string {
	if strings.EqualFold(theme, ThemeLight) {
		return "#1f2328" // near-black ink for light backgrounds
	}
	return "#c9d1d9" // light gray ink for dark backgrounds
}

// Render builds the default scene for these stats and encodes it as an animated GIF to w.
func Render(w io.Writer, s Stats) error {
	return render.EncodeScene(w, DefaultScene(s))
}

// Generate is the one-call pipeline the HTTP handler (and the CLI) use.
func Generate(ctx context.Context, w io.Writer, login, token string) error {
	s, err := Fetch(ctx, login, token)
	if err != nil {
		return fmt.Errorf("fetch %q: %w", login, err)
	}
	return Render(w, s)
}
