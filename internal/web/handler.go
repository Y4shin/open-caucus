// Package web serves the embedded SvelteKit SPA build.
package web

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:build
var buildFS embed.FS

// NewSPAHandler returns an HTTP handler that serves the embedded SPA.
// Static assets are served directly; all other paths fall back to index.html
// to support client-side routing.
func NewSPAHandler() http.Handler {
	build, err := fs.Sub(buildFS, "build")
	if err != nil {
		panic("web: failed to create sub-filesystem: " + err.Error())
	}
	return &spaHandler{
		fileServer: http.FileServer(http.FS(build)),
		buildFS:    buildFS,
	}
}

type spaHandler struct {
	fileServer http.Handler
	buildFS    embed.FS
}

func (h *spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Try to serve the exact file from the embedded filesystem.
	build, _ := fs.Sub(h.buildFS, "build")
	path := strings.TrimPrefix(r.URL.Path, "/")
	if path == "" {
		path = "."
	}

	f, err := build.Open(path)
	if err == nil {
		stat, sterr := f.Stat()
		_ = f.Close()
		if sterr == nil && !stat.IsDir() {
			h.fileServer.ServeHTTP(w, r)
			return
		}
	}

	// Fall back to index.html for SPA client-side routing.
	http.ServeFileFS(w, r, h.buildFS, "build/index.html")
}
