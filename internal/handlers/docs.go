package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Y4shin/conference-tool/internal/docs"
	"github.com/Y4shin/conference-tool/internal/routes"
	"github.com/Y4shin/conference-tool/internal/templates"
)

func (h *Handler) buildDocsElementInput(ctx context.Context, r *http.Request, docPath, heading string) (*templates.DocsElementInput, error) {
	input := &templates.DocsElementInput{
		Path:  docPath,
		Title: "Documentation",
	}
	if h.DocsService == nil {
		input.Error = "Documentation service is unavailable."
		return input, nil
	}

	rendered, err := h.DocsService.Render(docs.DocRef{Path: docPath, Heading: heading}, docs.VariantFromRequest(ctx, r))
	if err != nil {
		if errors.Is(err, docs.ErrNotFound) {
			input.NotFound = true
			return input, nil
		}
		return nil, fmt.Errorf("render docs page: %w", err)
	}

	input.Path = rendered.Path
	input.Locale = rendered.Locale
	input.Title = rendered.Title
	input.Heading = rendered.Heading
	input.Body = templates.RawHTML{HTML: rendered.HTML}
	return input, nil
}

func (h *Handler) DocsPage(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.DocsElementInput, *routes.ResponseMeta, error) {
	heading := strings.TrimSpace(r.URL.Query().Get("heading"))
	input, err := h.buildDocsElementInput(ctx, r, params.DocPath, heading)
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

func (h *Handler) DocsPageOOB(ctx context.Context, r *http.Request, params routes.RouteParams) (*templates.DocsElementInput, *routes.ResponseMeta, error) {
	heading := strings.TrimSpace(r.URL.Query().Get("heading"))
	input, err := h.buildDocsElementInput(ctx, r, params.DocPath, heading)
	if err != nil {
		return nil, nil, err
	}
	return input, nil, nil
}

func (h *Handler) DocsSearch(ctx context.Context, r *http.Request) (*templates.DocsSearchResultsInput, *routes.ResponseMeta, error) {
	input := &templates.DocsSearchResultsInput{
		Query: strings.TrimSpace(r.URL.Query().Get("q")),
	}
	if h.DocsService == nil {
		input.Error = "Documentation search is unavailable."
		return input, nil, nil
	}
	if input.Query == "" {
		return input, nil, nil
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

	variant := docs.VariantFromRequest(ctx, r)
	hits, err := h.DocsService.Search(input.Query, variant.Language, limit)
	if err != nil {
		return nil, nil, fmt.Errorf("docs search: %w", err)
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
	return input, nil, nil
}

func (h *Handler) ServeDocsAsset(w http.ResponseWriter, r *http.Request, params routes.RouteParams) error {
	if h.DocsService == nil {
		http.NotFound(w, r)
		return nil
	}
	body, contentType, err := h.DocsService.ReadAsset(params.AssetPath)
	if err != nil {
		if errors.Is(err, docs.ErrNotFound) {
			http.NotFound(w, r)
			return nil
		}
		return fmt.Errorf("serve docs asset %q: %w", params.AssetPath, err)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=600")
	if _, err := w.Write(body); err != nil {
		return fmt.Errorf("write docs asset response: %w", err)
	}
	return nil
}
