# Architecture & design decisions

This document explains *why* profilegif is built the way it is. It doubles as a learning
artifact: if you're coming from web/React and learning Go, the reasoning here is as useful as
the code.

## The one decision everything else follows from

**Separate the document from the pixels from the terminal.** Three layers, each ignorant of
the others:

```
scene  (what to draw)  →  render  (draw it to real pixels)  →  termimg  (put pixels on screen)
```

- `internal/scene` is pure data: elements, positions, z-order, JSON. It has no idea whether
  it will become a GIF or a terminal preview.
- `internal/render` turns a scene into `image.Image` frames and a GIF. It knows nothing about
  terminals or HTTP.
- `internal/termimg` is the *only* code that knows a terminal has cells and colors.

Why it matters: the same composition renders identically to an exported GIF, to the web
endpoint, and to the live terminal preview — because they all share `scene` + `render`. And
because `termimg` is the only terminal-aware layer, a higher-fidelity backend (Sixel, Kitty)
can be added as a second implementation of one interface, with **zero changes** to the editor
or the document model. (React analogy: `scene` is your serializable state, `render` is a pure
render-to-canvas function, and `termimg`/GIF are two different "DOM targets".)

This is the Go proverb *"clear is better than clever"* applied at the module level: boring,
one-directional dependencies beat a clever shared blob.

## Why Go

- **Single static binary.** `go build` produces one executable with the runtime baked in —
  deploy is `scp` one file. No `node_modules`, no interpreter to install. (The `Dockerfile` is
  ~15 lines because of this.)
- **Standard library does the heavy lifting.** HTTP server, `image/gif` encoding, and JSON are
  all stdlib. We added a web framework: none. A CLI framework: none. Go 1.22's `net/http`
  `ServeMux` handles method+path routing; `os.Args` handles subcommands.
- **Concurrency is cheap and safe-ish.** The web server handles requests in parallel for free;
  the constraint that pushed one design choice (fonts, below) came directly from that.

## Why Bubble Tea (the Elm Architecture)

The editor uses [Bubble Tea](https://github.com/charmbracelet/bubbletea), whose model is
literally The Elm Architecture — the same idea as a Redux reducer:

| Bubble Tea | React/Redux equivalent |
|---|---|
| `Model` | your component state / store |
| `Update(msg) → Model` | a reducer: `(state, action) → state` |
| `View() → string` | `render()` — but it returns the whole screen as a string |
| `Cmd` | a side-effect (like a thunk) the runtime runs for you |

Why this fits an editor: every interaction (keypress, mouse drag, animation tick) is just a
message; state changes are all funneled through `Update`, so there's one place to reason about
"what can change and when." No hidden mutation scattered across event handlers.

**Chosen over `tview`** (the other big Go TUI lib) because tview is widget-oriented (forms,
flexboxes) and fights you when you want a free-form draggable pixel canvas. Bubble Tea gives
low-level mouse events (`tea.MouseMsg` with press/motion/release), which is exactly what
drag-to-move and drag-corner-to-resize need.

## Why half-blocks for the terminal (and not Sixel yet)

A terminal is a grid of character cells, not pixels. The `▀` (upper-half-block) trick paints
the top half of a cell one color and the bottom half (the cell's background) another — so one
cell shows **two stacked pixels** in 24-bit color. It works in essentially every modern
terminal with no capability detection.

The alternative — Sixel/Kitty graphics protocols — gives real bitmaps (photographic quality)
but only works in *some* terminals and doesn't clip cleanly to a TUI layout. We chose
half-blocks **first** on purpose: portable, integrates natively with Bubble Tea's cell grid,
and makes drag/resize/animation trivial. Because `termimg.Renderer` is an interface, Sixel can
arrive later as a pure upgrade. (This is *"make the easy thing possible before the impressive
thing"* — ship the portable 80% now, keep the door open for the 20%.)

## A few decisions worth calling out

- **`StatWidget.Value` lives in the scene, not fetched at render time.** The renderer stays
  pure (scene in, pixels out); `gifmaker` populates the number after a GitHub fetch, and the
  editor uses sample values. Purity here is what makes `render` trivially testable.
- **Fonts are created per-draw, not cached globally.** A `font.Face` isn't safe for concurrent
  use, and the web server renders in parallel — a shared cached face would be a data race. So
  `render` creates faces per call. The *editor*, which is single-threaded, instead caches
  whole rasterized **frames** (`baseCache`) to stay fast. Same problem, two correct answers,
  because the concurrency context differs. Noticing that is the skill.
- **The scene JSON file is the bridge.** The TUI saves it; the web server loads it via
  `/gif?scene=`. Two programs, one file format, no shared process — the simplest possible
  integration.
- **Path traversal is closed at the edge.** `/gif?scene=` runs the input through
  `filepath.Base` so `?scene=../../secret` can't escape `./scenes`. Untrusted input meets a
  boundary the moment it arrives.

## Learning while building — the workflow behind this repo

A few principles (drawn from writing on
[deliberate practice for developers](https://www.redgreencode.com/deliberate-practice-for-software-developers/))
that shaped how this was built, and are worth stealing:

1. **Build the smallest thing that runs, then grow it.** This repo went in phases —
   scene model → rasterizer → terminal renderer → editor shell → mouse → authoring → fetch —
   and each phase *built and passed tests on its own*. You always have a working program to
   poke at, and a bug is localized to the phase you just added.
2. **Make feedback immediate and specific.** `go build ./... && go test ./... && go vet ./...`
   after every change. Fast, honest feedback is the entire engine of improvement; a slow or
   vague feedback loop is the main thing that stalls learning.
3. **Verify by *running it*, not just by testing.** Unit tests proved the logic; rendering a
   frame to a PNG and *looking at it* proved the design. Both matter — tests catch regressions,
   eyes catch "this is ugly / wrong-shaped".
4. **Consolidate by rewriting, don't chase tutorials.** The fastest way to *own* an idea (the
   Elm architecture, Go interfaces, image compositing) is to implement it once in a real
   project and then refactor it, not to watch five videos. Depth over novelty.
5. **Let constraints teach you.** The font-face concurrency issue and the half-block
   coordinate math weren't in any plan — they surfaced *because* the thing was real. Real
   projects generate the exact problems worth learning from; toy exercises rarely do.
6. **Write down the "why."** This file exists so the reasoning survives longer than the memory
   of writing it. For a public project it also invites better contributions — reviewers can
   argue with decisions instead of guessing them.

### Further reading

- [Effective Go](https://go.dev/doc/effective_go) — the canonical idioms
- [The Zen of Go](https://dave.cheney.net/2020/02/23/the-zen-of-go) — design principles
- [Bubble Tea](https://github.com/charmbracelet/bubbletea) — the TUI framework & its examples
- [Deliberate practice for software developers](https://www.redgreencode.com/deliberate-practice-for-software-developers/)
