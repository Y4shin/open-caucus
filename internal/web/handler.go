// Package web serves the embedded SvelteKit SPA build.
package web

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"strings"
)

//go:embed all:build
var buildFS embed.FS

// supportedLocales is the set of locales that the SPA supports.
// "en" is the build-time default baked into index.html.
var supportedLocales = map[string]bool{"en": true, "de": true}

// NewSPAHandler returns an HTTP handler that serves the embedded SPA.
// Static assets are served directly; all other paths fall back to index.html
// to support client-side routing.
//
// When serving index.html, the handler reads the PARAGLIDE_LOCALE cookie and
// substitutes the html[lang] attribute so that the server-rendered shell
// reflects the user's selected locale on each page load.
func NewSPAHandler() http.Handler {
	build, err := fs.Sub(buildFS, "build")
	if err != nil {
		panic("web: failed to create sub-filesystem: " + err.Error())
	}
	indexHTML, err := buildFS.ReadFile("build/index.html")
	if err != nil {
		panic("web: failed to read build/index.html: " + err.Error())
	}
	return &spaHandler{
		fileServer: http.FileServer(http.FS(build)),
		buildFS:    buildFS,
		indexHTML:  indexHTML,
	}
}

type spaHandler struct {
	fileServer http.Handler
	buildFS    embed.FS
	indexHTML  []byte
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
	// Inject the locale from the PARAGLIDE_LOCALE cookie so that the
	// html[lang] attribute matches the user's selected locale.
	html := h.localizedIndex(r)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Content-Length", fmt.Sprintf("%d", len(html)))
	_, _ = w.Write(html)
}

// localizedIndex returns index.html with the html[lang] attribute replaced to
// match the PARAGLIDE_LOCALE cookie, if present and supported.
func (h *spaHandler) localizedIndex(r *http.Request) []byte {
	cookie, err := r.Cookie("PARAGLIDE_LOCALE")
	if err != nil || !supportedLocales[cookie.Value] || cookie.Value == "en" {
		return h.indexHTML
	}
	locale := cookie.Value
	// The built index.html always has lang="en" (the build-time default).
	// Replace it with the user's locale.
	return bytes.Replace(h.indexHTML, []byte(`lang="en"`), []byte(`lang="`+locale+`"`), 1)
}
