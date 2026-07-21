# profilegif

Animated GitHub-stats GIF for your profile README — with an **interactive terminal editor**.
One Go binary is both a web service (the `/gif` embed endpoint) and a mouse-driven TUI where
you drag & resize elements on a canvas. No Node, no separate frontend, deploys anywhere.

## Two front-ends, one core

```sh
go run .                     # or: go run . serve  → web server on :8080
go run . edit                # interactive TUI editor (drag/resize elements, export a GIF)
```

### The editor (`profilegif edit`)

A hybrid canvas rendered right in your terminal using truecolor half-blocks:

```sh
go run . edit                       # open a sample composition
go run . edit scenes/example.json   # edit a saved scene
go run . edit -user octocat         # start from live stats (needs GITHUB_TOKEN)
go run . edit -user me -mock        # sample stats, no network/token
```

| Key | Action | | Key | Action |
|---|---|---|---|---|
| **drag** / arrows | move element | | `t` `i` `g` `b` | add text / image / stat / background |
| **drag corner** | resize | | `↵` | edit selected element's text/path |
| `tab` | cycle selection | | `[` `]` | send back / bring forward |
| `space` | play/pause preview | | `s` `e` | save scene JSON / export GIF |
| `d` | delete | | `q` | quit |

Composition is a **renderer-agnostic scene model** (`internal/scene`) → real pixels
(`internal/render`) → terminal cells (`internal/termimg`). Half-blocks work in any modern
terminal; a Sixel/Kitty high-fidelity backend can drop in later behind the same interface.

### The web server (`profilegif serve`)

```sh
go run .            # open http://localhost:8080
```

- `GET /gif?user=<login>` — the default GitHub-stats GIF (needs `GITHUB_TOKEN`, or run with
  `PROFILEGIF_MOCK=1` for sample data). Paste `![](https://yourhost/gif?user=you)` into your
  README; GitHub's Camo proxy caches it so most views never hit your server.
- `GET /gif?scene=<name>` — renders a scene you authored in the editor and saved to
  `./scenes/<name>.json`. This is the bridge: **compose in the TUI, serve on the web.**

## Layout

```
main.go                  entry point + subcommand dispatch (serve | edit)
web/index.html           htmx front page (embedded via //go:embed)
scenes/                  saved scene JSON files (served via /gif?scene=)
internal/scene/          document model — elements, z-order, JSON (TUI↔web bridge)
internal/render/         scene → pixel frames → GIF (fogleman/gg + image/gif)
internal/termimg/        image → terminal half-block string (swappable renderer)
internal/tui/            Bubble Tea editor (Model/Update/View + mouse)
internal/gifmaker/       GitHub Fetch + the default stats composition
Dockerfile               host-agnostic container
```

## Hosting (free, no surprise bills)

Deliberately **not** GCP/Cloud Run: GCP has **no hard spending cap** — budget alerts only notify,
they don't stop charges. For a fear-free free host, pick one that hard-stops or is always-free:

| Host | Card needed? | Notes |
|---|---|---|
| **Oracle Cloud Always Free** | yes (but never charges Always-Free shapes) | Always-on VM, no cold start, most compute. More ops. |
| **Koyeb free** | no | Hard-capped PaaS, runs this container. Some cold start. |
| **Render free** | no | Sleeps after 15 min idle (~50s cold). Fine because Camo caches the image. |

The `Cache-Control` header (step 4) makes GitHub's Camo proxy cache the GIF, so most README
views never hit your server — that's what keeps you inside any free tier.

## Env
- `PORT` — injected by the host (defaults to 8080; `serve -port` overrides locally).
- `GITHUB_TOKEN` — a read-only PAT for the GitHub GraphQL API (followers, commit
  contributions, and summed stargazers).
- `PROFILEGIF_MOCK=1` — skip the API and use deterministic sample data (great for local dev,
  demos, and CI). The editor's `-mock` flag sets this for you.
