# CLAUDE.md — profilegif

## Working with this user
The user is a **React / web developer, new to Go**. When explaining Go concepts, code, or
tooling, **use JavaScript / React analogies** wherever they help, and don't assume Go
idioms are familiar. Prefer showing the JS equivalent alongside the Go one.

### Go-for-JS-devs cheat sheet
| Go | JS / React world |
|---|---|
| `go.mod` | `package.json` (deps + versions) |
| `go.sum` | `package-lock.json` / `yarn.lock` (integrity hashes) |
| `go get <pkg>` | `npm install <pkg>` |
| `go mod tidy` | `npm install` + prune (reconcile deps with imports) |
| module cache (`$GOPATH/pkg/mod`) | pnpm's global store (one shared cache, no per-project `node_modules`) |
| `go build` → single `.exe` | a bundler that also bundles the runtime — deploy = copy one file |
| `interface` | a TypeScript `interface` / duck typing (implemented implicitly, no `implements`) |
| struct + methods | a class without inheritance (composition instead) |
| goroutine / channel | async task / a typed message queue between them |
| `err != nil` checks | explicit error returns instead of `try/catch` |
| `internal/` package | not importable outside the module — like a private/unexported module |

## Project
`profilegif` — generates an animated GitHub-stats GIF **and** is an interactive terminal
editor. One Go binary, no Node.

- `profilegif` (no args) / `profilegif serve` → the htmx web server + `/gif` embed endpoint
  (delivery mechanism: an image URL you drop in a README; GitHub's Camo proxy caches it).
- `profilegif edit` → mouse-driven TUI editor (drag/resize elements on a hybrid canvas).

### Architecture (renderer-agnostic, 3 layers)
- `internal/scene` — pure document model (elements, z-order, JSON). Bridge between TUI & web.
- `internal/render` — scene → real pixel frames → GIF (fogleman/gg + stdlib image/gif).
- `internal/termimg` — image → terminal string. **Only** layer that knows terminal pixels
  (half-block truecolor now; Sixel/Kitty can implement the same interface later).
- `internal/tui` — Bubble Tea editor (Model/Update/View + mouse).
- `internal/gifmaker` — GitHub Fetch + a default Scene builder; delegates rendering to above.

Full plan: `C:\Users\Soros\.ccs\instances\kakai\plans\ancient-nibbling-garden.md`

### Conventions
- No web frameworks (stdlib `net/http` + Go 1.22 ServeMux); no cobra (stdlib arg dispatch).
- Keep `internal/gifmaker`, `internal/scene`, `internal/render` free of HTTP/TUI assumptions.
- All deps are pure Go so the Dockerfile's `CGO_ENABLED=0` static build stays valid.
