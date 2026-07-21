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

// DefaultScene builds the standard stats composition from fetched Stats. This is the layout
// the web server serves out of the box; the TUI editor can load, rearrange, and re-save it.
func DefaultScene(s Stats) *scene.Scene {
	sc := scene.NewScene(800, 400)

	title := scene.NewText(scene.Rect{X: 0, Y: 26, W: 800, H: 80}, "@"+s.Login)
	title.FontSize = 56
	title.Color = "#e6edf3"
	sc.Add(title)

	commits := scene.NewStatWidget(scene.Rect{X: 60, Y: 150, W: 200, H: 190}, scene.MetricCommits, s.Login)
	commits.Value = s.TotalCommits
	commits.Label = "commits"
	commits.FontSize = 44
	sc.Add(commits)

	followers := scene.NewStatWidget(scene.Rect{X: 300, Y: 150, W: 200, H: 190}, scene.MetricFollowers, s.Login)
	followers.Value = s.Followers
	followers.Label = "followers"
	followers.Color = "#58a6ff"
	followers.FontSize = 44
	sc.Add(followers)

	stars := scene.NewStatWidget(scene.Rect{X: 540, Y: 150, W: 200, H: 190}, scene.MetricStars, s.Login)
	stars.Value = s.Stars
	stars.Label = "stars"
	stars.Color = "#e3b341"
	stars.FontSize = 44
	sc.Add(stars)

	return sc
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
