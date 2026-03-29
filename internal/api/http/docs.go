package apihttp

import (
	"errors"
	"net/http"
	"path"
	"strconv"
	"strings"

	"github.com/a-h/templ"
	"github.com/Y4shin/conference-tool/internal/docs"
	"github.com/Y4shin/conference-tool/internal/templates"
)

type docsHandler struct {
	service *docs.Service
}

func newDocsElementInput(r *http.Request, service *docs.Service, docPath, heading string) (*templates.DocsElementInput, error) {
	variant := docs.VariantFromRequest(r.Context(), r)
	input := &templates.DocsElementInput{
		Path:  docPath,
		Title: "Documentation",
	}
	if service == nil {
		input.Error = "Documentation service is unavailable."
		return input, nil
	}

	if navigation, err := service.Navigation(docPath, variant.Language); err == nil {
		input.PathDisplay = navigation.PathDisplay
		input.PathCrumbs = make([]templates.DocsPathCrumb, 0, len(navigation.Crumbs))
		for _, crumb := range navigation.Crumbs {
			input.PathCrumbs = append(input.PathCrumbs, templates.DocsPathCrumb{
				Title:   crumb.Title,
				Path:    crumb.Path,
				Current: crumb.Current,
			})
		}
		input.Tree = mapDocsNavigationNodes(navigation.Nodes)
	}

	rendered, err := service.Render(docs.DocRef{
		Path:    docPath,
		Heading: heading,
	}, variant)
	if err != nil {
		if errors.Is(err, docs.ErrNotFound) {
			input.NotFound = true
			return input, nil
		}
		return nil, err
	}

	input.Path = rendered.Path
	input.Locale = rendered.Locale
	input.Title = rendered.Title
	input.Heading = rendered.Heading
	input.Body = templates.RawHTML{HTML: rendered.HTML}
	return input, nil
}

func mapDocsNavigationNodes(nodes []docs.NavNode) []templates.DocsTreeNode {
	out := make([]templates.DocsTreeNode, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, templates.DocsTreeNode{
			Title:    node.Title,
			Path:     node.Path,
			Current:  node.Current,
			Expanded: node.Expanded,
			Children: mapDocsNavigationNodes(node.Children),
		})
	}
	return out
}

func renderTemplComponent(w http.ResponseWriter, r *http.Request, component templ.Component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := component.Render(r.Context(), w); err != nil {
		http.Error(w, "failed to render docs page", http.StatusInternalServerError)
	}
}

func docsPathFromRequestPath(reqPath, prefix string) string {
	trimmed := strings.TrimPrefix(reqPath, prefix)
	trimmed = strings.TrimPrefix(trimmed, "/")
	if trimmed == "" {
		return "index"
	}
	clean := strings.TrimPrefix(path.Clean("/"+trimmed), "/")
	if clean == "" || clean == "." {
		return "index"
	}
	return clean
}

func NewDocsAssetHandler(service *docs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			http.NotFound(w, r)
			return
		}

		assetPath := r.PathValue("assetPath")
		if assetPath == "" {
			assetPath = strings.TrimPrefix(r.URL.Path, "/docs/assets/")
			assetPath = strings.TrimPrefix(assetPath, "docs/assets/")
		}

		body, contentType, err := service.ReadAsset(assetPath)
		if err != nil {
			if errors.Is(err, docs.ErrNotFound) {
				http.NotFound(w, r)
				return
			}
			http.Error(w, "failed to read docs asset", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", contentType)
		w.Header().Set("Cache-Control", "public, max-age=600")
		_, _ = w.Write(body)
	}
}

func NewDocsPageHandler(service *docs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docPath := docsPathFromRequestPath(r.URL.Path, "/docs")
		heading := strings.TrimSpace(r.URL.Query().Get("heading"))
		input, err := newDocsElementInput(r, service, docPath, heading)
		if err != nil {
			http.Error(w, "failed to load docs page", http.StatusInternalServerError)
			return
		}
		renderTemplComponent(w, r, templates.DocsElement(*input))
	}
}

func NewDocsPageOOBHandler(service *docs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		docPath := docsPathFromRequestPath(r.URL.Path, "/docs/oob")
		heading := strings.TrimSpace(r.URL.Query().Get("heading"))
		input, err := newDocsElementInput(r, service, docPath, heading)
		if err != nil {
			http.Error(w, "failed to load docs page", http.StatusInternalServerError)
			return
		}
		renderTemplComponent(w, r, templates.DocsElementOOB(*input))
	}
}

func NewDocsSearchHandler(service *docs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		input := templates.DocsSearchResultsInput{
			Query: strings.TrimSpace(r.URL.Query().Get("q")),
		}
		if service == nil {
			input.Error = "Documentation search is unavailable."
			renderTemplComponent(w, r, templates.DocsSearchResults(input))
			return
		}
		if input.Query == "" {
			renderTemplComponent(w, r, templates.DocsSearchResults(input))
			return
		}

		limit := 10
		if rawLimit := strings.TrimSpace(r.URL.Query().Get("limit")); rawLimit != "" {
			if parsed, err := strconv.Atoi(rawLimit); err == nil && parsed > 0 {
				if parsed > 50 {
					parsed = 50
				}
				limit = parsed
			}
		}

		variant := docs.VariantFromRequest(r.Context(), r)
		hits, err := service.Search(input.Query, variant.Language, limit)
		if err != nil {
			input.Error = "Documentation search failed."
			renderTemplComponent(w, r, templates.DocsSearchResults(input))
			return
		}

		input.Hits = make([]templates.DocsSearchHit, 0, len(hits))
		for _, hit := range hits {
			input.Hits = append(input.Hits, templates.DocsSearchHit{
				Ref:     hit.Ref,
				Path:    hit.Path,
				Heading: hit.Heading,
				Title:   hit.Title,
				Snippet: hit.Snippet,
				Score:   hit.Score,
			})
		}
		renderTemplComponent(w, r, templates.DocsSearchResults(input))
	}
}
