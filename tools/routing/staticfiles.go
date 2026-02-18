package routing

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// StaticFileInfo describes a single file discovered in the static directory.
type StaticFileInfo struct {
	FileName   string // e.g. "htmx.min.js" or "ext/htmx-ext-sse.min.js"
	MethodName string // e.g. "HtmxMinJs" or "ExtHtmxExtSseMinJs"
	URLPath    string // e.g. "/static/htmx.min.js"
}

// StaticAssets holds all static files discovered for code generation.
type StaticAssets struct {
	Files    []StaticFileInfo
	HasFiles bool
}

// ScanStaticDir walks dir recursively and returns a StaticFileInfo for every
// file found. Directories are skipped. Returns nil, nil when dir does not exist.
func ScanStaticDir(dir string) ([]StaticFileInfo, error) {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, nil
	}

	var files []StaticFileInfo
	err := fs.WalkDir(os.DirFS(dir), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Normalise to forward slashes (important on Windows)
		relPath := filepath.ToSlash(path)
		files = append(files, StaticFileInfo{
			FileName:   relPath,
			MethodName: staticFileMethodName(relPath),
			URLPath:    "/static/" + relPath,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

// staticFileMethodName converts a slash-separated relative path to a PascalCase
// Go method name. Each path segment and each dot/dash/underscore-separated word
// within a segment contributes a capitalised word.
//
//	"htmx.min.js"              → "HtmxMinJs"
//	"htmx-ext-sse.min.js"      → "HtmxExtSseMinJs"
//	"ext/htmx-ext-sse.min.js"  → "ExtHtmxExtSseMinJs"
func staticFileMethodName(relPath string) string {
	var result strings.Builder
	for _, segment := range strings.Split(relPath, "/") {
		for _, word := range strings.FieldsFunc(segment, func(r rune) bool {
			return r == '.' || r == '-' || r == '_'
		}) {
			result.WriteString(ToPascalCase(word))
		}
	}
	return result.String()
}

const staticAssetsVar string = `{{- if .HasFiles }}

//go:embed static
var staticFS embed.FS
{{- end }}`

const staticPathMethods string = `{{- range .Files }}

func (Routes) {{ .MethodName }}() string {
	return "{{ .URLPath }}"
}
{{- end }}`
