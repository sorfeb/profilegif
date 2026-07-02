// Package gifmaker is the CORE library: fetch GitHub stats, render frames, encode a GIF.
//
// Keep this package free of HTTP/CLI assumptions — it just takes inputs and writes bytes.
// Both entry points (the web server today, a CLI later) call into here. This separation is
// what makes "deploy to a frontend" an afternoon instead of a rewrite.
//
// Fetch, Render, and Encode are stubbed out below — see the TODOs.
package gifmaker

import (
	"context"
	"fmt"
	"io"
)

// Stats holds the GitHub data we animate into frames.
type Stats struct {
	Login        string
	TotalCommits int
	Followers    int
	// TODO(step 2): add Languages []LangStat, ContributionDays []int, Streak int, etc.
}

// Fetch pulls stats for a GitHub user.
//
// TODO(step 2): implement with the GitHub GraphQL API for the contribution calendar
// (query `user.contributionsCollection`) + REST for the rest. Use net/http; the token is
// passed in by the caller (read from an env var there, not here). Handle rate limits.
func Fetch(ctx context.Context, login, token string) (Stats, error) {
	return Stats{}, notImpl("Fetch")
}

// Render turns Stats into animated frames and writes an encoded GIF to w.
//
// TODO(step 3): `go get github.com/fogleman/gg`. For each frame draw with a gg.Context
// (bars growing, counters ticking up), convert the result to *image.Paletted, collect them,
// then image/gif.EncodeAll(w, &gif.GIF{Image: frames, Delay: delays, LoopCount: 0}).
func Render(w io.Writer, s Stats) error {
	return notImpl("Render")
}

// Generate is the one-call pipeline the HTTP handler (and a future CLI) use.
func Generate(ctx context.Context, w io.Writer, login, token string) error {
	s, err := Fetch(ctx, login, token)
	if err != nil {
		return fmt.Errorf("fetch %q: %w", login, err)
	}
	return Render(w, s)
}

func notImpl(fn string) error { return fmt.Errorf("gifmaker: %s not implemented yet", fn) }
