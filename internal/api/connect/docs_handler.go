package apiconnect

import (
	"context"
	"errors"

	connect "connectrpc.com/connect"

	docsv1 "github.com/Y4shin/conference-tool/gen/go/conference/docs/v1"
	docsv1connect "github.com/Y4shin/conference-tool/gen/go/conference/docs/v1/docsv1connect"
	"github.com/Y4shin/conference-tool/internal/docs"
)

type DocsHandler struct {
	docsv1connect.UnimplementedDocsServiceHandler
	service *docs.Service
}

func NewDocsHandler(service *docs.Service) *DocsHandler {
	return &DocsHandler{service: service}
}

func (h *DocsHandler) GetPage(ctx context.Context, req *connect.Request[docsv1.GetPageRequest]) (*connect.Response[docsv1.GetPageResponse], error) {
	variant := docs.VariantFromRequest(ctx, headerRequest(req.Header()))
	rendered, err := h.service.Render(docs.DocRef{
		Path:    req.Msg.Path,
		Heading: req.Msg.Heading,
	}, variant)
	if err != nil {
		if errors.Is(err, docs.ErrNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	navigation, _ := h.service.Navigation(rendered.Path, variant.Language)
	return connect.NewResponse(&docsv1.GetPageResponse{
		Page: &docsv1.DocsPage{
			Path:        rendered.Path,
			Locale:      rendered.Locale,
			Title:       rendered.Title,
			Heading:     rendered.Heading,
			Html:        rendered.HTML,
			PathDisplay: navigation.PathDisplay,
			Crumbs:      mapNavCrumbs(navigation.Crumbs),
			Tree:        mapNavNodes(navigation.Nodes),
		},
	}), nil
}

func (h *DocsHandler) Search(ctx context.Context, req *connect.Request[docsv1.SearchRequest]) (*connect.Response[docsv1.SearchResponse], error) {
	variant := docs.VariantFromRequest(ctx, headerRequest(req.Header()))
	limit := int(req.Msg.Limit)
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	hits, err := h.service.Search(req.Msg.Query, variant.Language, limit)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&docsv1.SearchResponse{
		Query: req.Msg.Query,
		Hits:  mapSearchHits(hits),
	}), nil
}

func mapNavCrumbs(crumbs []docs.NavCrumb) []*docsv1.NavCrumb {
	out := make([]*docsv1.NavCrumb, 0, len(crumbs))
	for _, crumb := range crumbs {
		out = append(out, &docsv1.NavCrumb{
			Title:   crumb.Title,
			Path:    crumb.Path,
			Current: crumb.Current,
		})
	}
	return out
}

func mapNavNodes(nodes []docs.NavNode) []*docsv1.NavNode {
	out := make([]*docsv1.NavNode, 0, len(nodes))
	for _, node := range nodes {
		out = append(out, &docsv1.NavNode{
			Title:    node.Title,
			Path:     node.Path,
			Current:  node.Current,
			Expanded: node.Expanded,
			Children: mapNavNodes(node.Children),
		})
	}
	return out
}

func mapSearchHits(hits []docs.SearchHit) []*docsv1.SearchHit {
	out := make([]*docsv1.SearchHit, 0, len(hits))
	for _, hit := range hits {
		out = append(out, &docsv1.SearchHit{
			Ref:     hit.Ref,
			Path:    hit.Path,
			Heading: hit.Heading,
			Title:   hit.Title,
			Locale:  hit.Locale,
			Snippet: hit.Snippet,
			Score:   hit.Score,
		})
	}
	return out
}
