package gifmaker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"hash/fnv"
	"net/http"
	"os"
	"strings"
)

const githubGraphQL = "https://api.github.com/graphql"

// Fetch pulls stats for a GitHub user. Behavior:
//
//   - PROFILEGIF_MOCK=1 (or the --mock flag, which sets it) → deterministic sample data,
//     so the editor and /gif work with no network and no token.
//   - otherwise a token is required, and stats come from the GitHub GraphQL API.
//
// The token is passed in by the caller (read from GITHUB_TOKEN at the edge, not here).
func Fetch(ctx context.Context, login, token string) (Stats, error) {
	login = strings.TrimSpace(login)
	if login == "" {
		return Stats{}, fmt.Errorf("gifmaker: empty login")
	}
	if MockEnabled() {
		return MockStats(login), nil
	}
	if token == "" {
		return Stats{}, fmt.Errorf("gifmaker: no GITHUB_TOKEN set (or set PROFILEGIF_MOCK=1 for sample data)")
	}
	return fetchLive(ctx, login, token)
}

// MockEnabled reports whether sample-data mode is on.
func MockEnabled() bool {
	v := os.Getenv("PROFILEGIF_MOCK")
	return v == "1" || strings.EqualFold(v, "true")
}

// MockStats returns stable, non-zero pseudo-stats derived from the login, so previews look
// plausible and don't change between runs.
func MockStats(login string) Stats {
	h := fnv.New32a()
	_, _ = h.Write([]byte(login))
	s := h.Sum32()
	return Stats{
		Login:        login,
		TotalCommits: 800 + int(s%6000),
		Followers:    12 + int((s>>8)%800),
		Stars:        int((s >> 16) % 1500),
	}
}

// fetchLive queries the GitHub GraphQL API for followers, commit contributions, and the sum
// of stargazers across the user's own (non-fork) repositories.
func fetchLive(ctx context.Context, login, token string) (Stats, error) {
	const query = `query($login:String!){
	  user(login:$login){
	    followers{ totalCount }
	    contributionsCollection{ totalCommitContributions }
	    repositories(first:100, ownerAffiliations:OWNER, isFork:false){ nodes{ stargazerCount } }
	  }
	}`

	payload, err := json.Marshal(map[string]any{
		"query":     query,
		"variables": map[string]string{"login": login},
	})
	if err != nil {
		return Stats{}, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, githubGraphQL, bytes.NewReader(payload))
	if err != nil {
		return Stats{}, err
	}
	req.Header.Set("Authorization", "bearer "+token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return Stats{}, err
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
	case http.StatusOK:
		// fall through
	case http.StatusForbidden, http.StatusTooManyRequests:
		return Stats{}, fmt.Errorf("gifmaker: github rate limited (status %d)", resp.StatusCode)
	case http.StatusUnauthorized:
		return Stats{}, fmt.Errorf("gifmaker: github rejected the token (status 401)")
	default:
		return Stats{}, fmt.Errorf("gifmaker: github status %d", resp.StatusCode)
	}

	var out struct {
		Data struct {
			User *struct {
				Followers struct {
					TotalCount int
				}
				ContributionsCollection struct {
					TotalCommitContributions int
				}
				Repositories struct {
					Nodes []struct {
						StargazerCount int
					}
				}
			}
		}
		Errors []struct {
			Message string
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return Stats{}, fmt.Errorf("gifmaker: decode github response: %w", err)
	}
	if len(out.Errors) > 0 {
		return Stats{}, fmt.Errorf("gifmaker: github: %s", out.Errors[0].Message)
	}
	if out.Data.User == nil {
		return Stats{}, fmt.Errorf("gifmaker: user %q not found", login)
	}

	u := out.Data.User
	stars := 0
	for _, n := range u.Repositories.Nodes {
		stars += n.StargazerCount
	}
	return Stats{
		Login:        login,
		TotalCommits: u.ContributionsCollection.TotalCommitContributions,
		Followers:    u.Followers.TotalCount,
		Stars:        stars,
	}, nil
}
