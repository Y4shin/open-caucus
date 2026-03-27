package apihttp

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/docs"
)

type docsHandler struct {
	service *docs.Service
}

func NewDocsPageHandler(service *docs.Service) http.Handler {
	return &docsHandler{service: service}
}

func (h *docsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.service == nil {
		writeJSONError(w, http.StatusServiceUnavailable, "documentation service is unavailable")
		return
	}

	docPath := strings.TrimSpace(r.PathValue("docPath"))
	if docPath == "" {
		docPath = "index"
	}
	heading := strings.TrimSpace(r.URL.Query().Get("heading"))
	variant := docs.VariantFromRequest(r.Context(), r)

	rendered, err := h.service.Render(docs.DocRef{Path: docPath, Heading: heading}, variant)
	if err != nil {
		if errors.Is(err, docs.ErrNotFound) {
			writeJSONError(w, http.StatusNotFound, "document not found")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, "failed to render docs page")
		return
	}

	navigation, _ := h.service.Navigation(docPath, variant.Language)

	writeJSON(w, http.StatusOK, map[string]any{
		"path":         rendered.Path,
		"locale":       rendered.Locale,
		"title":        rendered.Title,
		"heading":      rendered.Heading,
		"html":         rendered.HTML,
		"path_display": navigation.PathDisplay,
		"crumbs":       navigation.Crumbs,
		"tree":         navigation.Nodes,
	})
}

func NewDocsSearchHandler(service *docs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			writeJSONError(w, http.StatusServiceUnavailable, "documentation search is unavailable")
			return
		}

		query := strings.TrimSpace(r.URL.Query().Get("q"))
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
		hits, err := service.Search(query, variant.Language, limit)
		if err != nil {
			writeJSONError(w, http.StatusInternalServerError, fmt.Sprintf("docs search failed: %v", err))
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"query": query,
			"hits":  hits,
		})
	}
}

func NewDocsAssetHandler(service *docs.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if service == nil {
			http.NotFound(w, r)
			return
		}

		body, contentType, err := service.ReadAsset(r.PathValue("assetPath"))
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
