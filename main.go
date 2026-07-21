package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/sorfeb/profilegif/internal/gifmaker"
	"github.com/sorfeb/profilegif/internal/render"
	"github.com/sorfeb/profilegif/internal/scene"
	"github.com/sorfeb/profilegif/internal/tui"
)

//go:embed web/index.html
var webFS embed.FS

var indexTmpl = template.Must(template.ParseFS(webFS, "web/index.html"))

// profilegif has two front-ends over one shared core (internal/gifmaker + scene/render):
//
//	profilegif            → same as "serve" (backwards-compatible default)
//	profilegif serve      → the htmx web server + /gif embed endpoint (delivery)
//	profilegif edit       → the interactive TUI editor (authoring)
//
// Plain stdlib arg dispatch — no cobra — matching the project's "no framework" ethos.
func main() {
	cmd := "serve"
	args := os.Args[1:]
	if len(args) > 0 {
		cmd, args = args[0], args[1:]
	}

	switch cmd {
	case "serve":
		runServe(args)
	case "edit":
		runEdit(args)
	case "help", "-h", "--help":
		usage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "profilegif: unknown command %q\n\n", cmd)
		usage(os.Stderr)
		os.Exit(2)
	}
}

func usage(w io.Writer) {
	fmt.Fprint(w, `profilegif — animated GitHub-stats GIFs, served or hand-composed.

usage:
  profilegif [serve]   start the web server + /gif embed endpoint (default)
  profilegif edit      launch the interactive TUI editor
  profilegif help      show this message
`)
}

// runServe is the original web server: the htmx UI plus /preview and /gif.
func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	// PORT env still wins by default (PaaS hosts inject it); -port overrides for local runs.
	addr := fs.String("port", os.Getenv("PORT"), "port to listen on (default $PORT or 8080)")
	fs.Parse(args)

	port := *addr
	if port == "" {
		port = "8080"
	}

	mux := http.NewServeMux()
	// Go 1.22+ ServeMux understands method+path patterns — no framework needed.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("GET /preview", handlePreview)
	mux.HandleFunc("GET /gif", handleGif)

	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
	}
}

// runEdit launches the interactive terminal editor. Sources for the starting scene, in order
// of precedence: -user (fetch GitHub stats), a scene-file argument (load it), else sample data.
//
//	profilegif edit                    open a sample composition
//	profilegif edit scenes/foo.json    edit an existing scene
//	profilegif edit -user octocat      start from octocat's live stats (needs GITHUB_TOKEN)
//	profilegif edit -user me -mock     start from sample stats, no network/token
func runEdit(args []string) {
	fs := flag.NewFlagSet("edit", flag.ExitOnError)
	user := fs.String("user", "", "fetch this GitHub user's stats as the starting scene")
	mock := fs.Bool("mock", false, "use sample data instead of calling the GitHub API")
	fs.Parse(args)

	if *mock {
		os.Setenv("PROFILEGIF_MOCK", "1")
	}

	var (
		sc   *scene.Scene
		path string
	)
	switch {
	case *user != "":
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		stats, err := gifmaker.Fetch(ctx, *user, os.Getenv("GITHUB_TOKEN"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "profilegif edit: %v\n", err)
			os.Exit(1)
		}
		sc = gifmaker.DefaultScene(stats)

	case fs.NArg() > 0:
		path = fs.Arg(0)
		loaded, err := scene.Load(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "profilegif edit: %v\n", err)
			os.Exit(1)
		}
		sc = loaded

	default:
		// Sample composition so the editor always opens with something to arrange.
		sc = gifmaker.DefaultScene(gifmaker.Stats{
			Login: "sorfeb", TotalCommits: 4237, Followers: 128, Stars: 512,
		})
	}

	if err := tui.Run(sc, path); err != nil {
		fmt.Fprintf(os.Stderr, "profilegif edit: %v\n", err)
		os.Exit(1)
	}
}

func handleIndex(w http.ResponseWriter, r *http.Request) {
	if err := indexTmpl.Execute(w, nil); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handlePreview returns an HTML fragment (an <img>) that htmx swaps into the page.
// The browser then loads /gif on its own. This is the htmx pattern: return HTML, not the image.
func handlePreview(w http.ResponseWriter, r *http.Request) {
	user := template.HTMLEscapeString(r.URL.Query().Get("user"))
	fmt.Fprintf(w, `<img src="/gif?user=%s" alt="GitHub stats for %s">`, user, user)
}

// handleGif is the embed endpoint. Two modes:
//
//   - ?scene=<name>  renders a composition authored in the TUI editor (from ./scenes/<name>.json).
//   - ?user=<login>  renders the default GitHub-stats composition for that user.
//
// The scene path is sanitized to its base name so the query can't traverse the filesystem.
func handleGif(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// TODO(step 4): set Cache-Control so GitHub's Camo proxy caches this and you stay free:
	// w.Header().Set("Cache-Control", "public, max-age=21600")

	if name := r.URL.Query().Get("scene"); name != "" {
		serveScene(w, name)
		return
	}

	user := r.URL.Query().Get("user")
	if user == "" {
		http.Error(w, "missing ?user= or ?scene=", http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Type", "image/gif")
	token := os.Getenv("GITHUB_TOKEN")
	if err := gifmaker.Generate(ctx, w, user, token); err != nil {
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}

// serveScene renders a saved scene JSON from the ./scenes directory as a GIF.
func serveScene(w http.ResponseWriter, name string) {
	// Only allow a bare file name — strip any directory components to prevent path traversal.
	safe := filepath.Base(name)
	if !strings.HasSuffix(safe, ".json") {
		safe += ".json"
	}
	sc, err := scene.Load(filepath.Join("scenes", safe))
	if err != nil {
		http.Error(w, "scene not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "image/gif")
	if err := render.EncodeScene(w, sc); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
