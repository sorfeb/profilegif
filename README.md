# profilegif

Animated GitHub-stats GIF for your profile README. One Go binary serves the htmx UI **and** the
`/gif` embed endpoint — no Node, no separate frontend, deploys anywhere.

## Run locally

```sh
go run .
# open http://localhost:8080
```

`/gif` returns **501 Not Implemented** until the core pipeline is implemented.

## Layout

```
main.go                  HTTP entry point (htmx UI + /preview + /gif)
web/index.html           htmx front page (embedded into the binary via //go:embed)
internal/gifmaker/       CORE LIBRARY — Fetch / Render / Encode
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
- `PORT` — injected by the host (defaults to 8080).
- `GITHUB_TOKEN` — a read-only PAT for the GitHub API (you add this in step 2).
