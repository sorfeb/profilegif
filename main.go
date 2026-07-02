package main

import (
	"context"
	"embed"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/sorfeb/profilegif/internal/gifmaker"
)

//go:embed web/index.html
var webFS embed.FS

var indexTmpl = template.Must(template.ParseFS(webFS, "web/index.html"))

func main() {
	mux := http.NewServeMux()
	// Go 1.22+ ServeMux understands method+path patterns — no framework needed.
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	mux.HandleFunc("GET /", handleIndex)
	mux.HandleFunc("GET /preview", handlePreview)
	mux.HandleFunc("GET /gif", handleGif)

	port := os.Getenv("PORT") // PaaS hosts inject this — never hardcode it.
	if port == "" {
		port = "8080"
	}
	log.Printf("listening on :%s", port)
	if err := http.ListenAndServe(":"+port, mux); err != nil {
		log.Fatal(err)
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

// handleGif is the embed endpoint. The real work lives in internal/gifmaker — yours to build.
func handleGif(w http.ResponseWriter, r *http.Request) {
	user := r.URL.Query().Get("user")
	if user == "" {
		http.Error(w, "missing ?user=", http.StatusBadRequest)
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// TODO(step 4): set Cache-Control so GitHub's Camo proxy caches this and you stay free:
	// w.Header().Set("Cache-Control", "public, max-age=21600")
	w.Header().Set("Content-Type", "image/gif")

	token := os.Getenv("GITHUB_TOKEN")
	if err := gifmaker.Generate(ctx, w, user, token); err != nil {
		// Until you implement gifmaker, this returns 501 — that's expected.
		http.Error(w, err.Error(), http.StatusNotImplemented)
	}
}
