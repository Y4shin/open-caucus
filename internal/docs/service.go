package docs

import (
	"fmt"
	"io"
	"io/fs"
	"mime"
	"path"
	"slices"
	"strings"
	"unicode"

	"github.com/blevesearch/bleve/v2"
	"github.com/blevesearch/bleve/v2/search/query"
)

type Service struct {
	docs    map[string]map[string]*document
	assets  map[string][]assetCandidate
	assetsF fs.FS
	index   bleve.Index
	tree    *treeDirNode
}

func Load(contentFS fs.FS, assetsFS fs.FS) (*Service, error) {
	assetCatalog, err := scanAssets(assetsFS)
	if err != nil {
		return nil, fmt.Errorf("scan docs assets: %w", err)
	}

	docs, err := loadDocuments(contentFS)
	if err != nil {
		return nil, fmt.Errorf("load docs content: %w", err)
	}
	if len(docs) == 0 {
		return nil, fmt.Errorf("no localized markdown files found")
	}
	tree, err := buildNavigationTree(docs)
	if err != nil {
		return nil, fmt.Errorf("validate docs structure: %w", err)
	}

	index, err := buildIndex(docs)
	if err != nil {
		return nil, fmt.Errorf("build docs search index: %w", err)
	}

	return &Service{
		docs:    docs,
		assets:  assetCatalog,
		assetsF: assetsFS,
		index:   index,
		tree:    tree,
	}, nil
}

func (s *Service) Close() error {
	if s == nil || s.index == nil {
		return nil
	}
	return s.index.Close()
}

func (s *Service) Render(ref DocRef, variant VariantContext) (RenderedDoc, error) {
	if s == nil {
		return RenderedDoc{}, ErrNotFound
	}
	_, localized, err := s.resolveDocument(ref.Path)
	if err != nil {
		return RenderedDoc{}, ErrNotFound
	}
	ref.Heading = normalizeHeading(ref.Heading)

	v := normalizeVariant(variant)
	doc := localized[v.Language]
	if doc == nil {
		doc = localized[DefaultLocale]
	}
	if doc == nil {
		return RenderedDoc{}, ErrNotFound
	}

	html := doc.HTMLTemplate
	for _, key := range doc.MediaKeys {
		resolved, resErr := s.ResolveAsset(key, v)
		if resErr != nil {
			continue
		}
		html = strings.ReplaceAll(html, mediaPlaceholderPrefix+key, "/docs/assets/"+resolved)
	}

	if ref.Heading != "" && !sectionExists(doc.Sections, ref.Heading) {
		return RenderedDoc{}, ErrNotFound
	}

	return RenderedDoc{
		Path:     doc.Path,
		Locale:   doc.Locale,
		Title:    doc.Title,
		Heading:  ref.Heading,
		HTML:     html,
		Sections: append([]Section(nil), doc.Sections...),
	}, nil
}

func (s *Service) resolveDocument(rawPath string) (string, map[string]*document, error) {
	if s == nil {
		return "", nil, ErrNotFound
	}
	pathKey, err := normalizeLogicalPath(rawPath)
	if err != nil {
		return "", nil, ErrNotFound
	}
	localized, ok := s.docs[pathKey]
	if ok {
		return pathKey, localized, nil
	}
	if pathKey != "index" {
		indexPath := path.Join(pathKey, "index")
		localized, ok = s.docs[indexPath]
		if ok {
			return indexPath, localized, nil
		}
	}
	return "", nil, ErrNotFound
}

func (s *Service) ResolveAsset(normalizedKey string, variant VariantContext) (string, error) {
	if s == nil {
		return "", ErrNotFound
	}
	key, err := normalizeAssetKey(normalizedKey)
	if err != nil {
		return "", ErrNotFound
	}
	candidates := s.assets[key]
	if len(candidates) == 0 {
		return "", ErrNotFound
	}

	v := normalizeVariant(variant)
	best := candidates[0]
	bestScore := scoreCandidate(best, v)
	for _, candidate := range candidates[1:] {
		score := scoreCandidate(candidate, v)
		if score > bestScore || (score == bestScore && candidate.Path < best.Path) {
			best = candidate
			bestScore = score
		}
	}
	return best.Path, nil
}

func (s *Service) Search(rawQuery string, locale string, limit int) ([]SearchHit, error) {
	if s == nil || s.index == nil {
		return nil, ErrNotFound
	}
	rawQuery = strings.TrimSpace(rawQuery)
	if rawQuery == "" {
		return []SearchHit{}, nil
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	mainQuery := bleve.NewQueryStringQuery(rawQuery)
	searchQuery := buildSearchQuery(mainQuery, rawQuery, normalizeLocale(locale))

	req := bleve.NewSearchRequestOptions(searchQuery, limit, 0, false)
	req.Fields = []string{"ref", "path", "heading", "title", "body", "locale"}

	result, err := s.index.Search(req)
	if err != nil {
		return nil, fmt.Errorf("search docs index: %w", err)
	}

	hits := make([]SearchHit, 0, len(result.Hits))
	for _, hit := range result.Hits {
		body := fieldString(hit.Fields, "body")
		hits = append(hits, SearchHit{
			Ref:     fieldString(hit.Fields, "ref"),
			Path:    fieldString(hit.Fields, "path"),
			Heading: fieldString(hit.Fields, "heading"),
			Title:   fieldString(hit.Fields, "title"),
			Locale:  fieldString(hit.Fields, "locale"),
			Snippet: buildSnippet(body, rawQuery),
			Score:   hit.Score,
		})
	}
	return hits, nil
}

func buildSearchQuery(mainQuery query.Query, rawQuery, locale string) query.Query {
	searchQuery := mainQuery
	if termQuery := buildTermSearchQuery(rawQuery); termQuery != nil {
		searchQuery = bleve.NewDisjunctionQuery(mainQuery, termQuery)
	}
	if locale != "" {
		localeQuery := bleve.NewTermQuery(locale)
		localeQuery.SetField("locale")
		searchQuery = bleve.NewConjunctionQuery(searchQuery, localeQuery)
	}
	return searchQuery
}

func buildTermSearchQuery(rawQuery string) query.Query {
	terms := searchTerms(rawQuery)
	if len(terms) == 0 {
		return nil
	}

	perTerm := make([]query.Query, 0, len(terms))
	for _, term := range terms {
		fieldQueries := make([]query.Query, 0, 6)
		for _, field := range []string{"title", "body"} {
			match := bleve.NewMatchQuery(term)
			match.SetField(field)
			match.SetBoost(2.0)
			fieldQueries = append(fieldQueries, match)

			prefix := bleve.NewPrefixQuery(term)
			prefix.SetField(field)
			prefix.SetBoost(1.2)
			fieldQueries = append(fieldQueries, prefix)

			// Keep typo matching conservative: only for longer terms and with a required exact prefix.
			if len(term) >= 4 {
				fuzzy := bleve.NewFuzzyQuery(term)
				fuzzy.SetField(field)
				fuzzy.SetFuzziness(1)
				fuzzy.SetPrefix(2)
				fuzzy.SetBoost(0.7)
				fieldQueries = append(fieldQueries, fuzzy)
			}
		}

		perTerm = append(perTerm, bleve.NewDisjunctionQuery(fieldQueries...))
	}

	if len(perTerm) == 1 {
		return perTerm[0]
	}
	return bleve.NewConjunctionQuery(perTerm...)
}

func searchTerms(rawQuery string) []string {
	rawQuery = strings.TrimSpace(strings.ToLower(rawQuery))
	if rawQuery == "" {
		return nil
	}

	terms := make([]string, 0, 4)
	seen := make(map[string]struct{}, 4)
	var b strings.Builder
	flush := func() {
		if b.Len() < 2 {
			b.Reset()
			return
		}
		term := b.String()
		b.Reset()
		if _, ok := seen[term]; ok {
			return
		}
		seen[term] = struct{}{}
		terms = append(terms, term)
	}

	for _, r := range rawQuery {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		flush()
	}
	flush()
	return terms
}

func (s *Service) ReadAsset(assetPath string) ([]byte, string, error) {
	if s == nil {
		return nil, "", ErrNotFound
	}
	clean, err := normalizeAssetKey(assetPath)
	if err != nil {
		return nil, "", ErrNotFound
	}
	f, err := s.assetsF.Open(clean)
	if err != nil {
		return nil, "", ErrNotFound
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		return nil, "", fmt.Errorf("read docs asset %q: %w", clean, err)
	}
	contentType := mime.TypeByExtension(path.Ext(clean))
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	return data, contentType, nil
}

func scanAssets(assetsFS fs.FS) (map[string][]assetCandidate, error) {
	catalog := make(map[string][]assetCandidate)
	err := fs.WalkDir(assetsFS, ".", func(name string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		clean, err := normalizeAssetKey(name)
		if err != nil {
			return nil
		}
		normalized, candidate := classifyAsset(clean)
		catalog[normalized] = append(catalog[normalized], candidate)
		return nil
	})
	if err != nil {
		return nil, err
	}
	for key, candidates := range catalog {
		slices.SortFunc(candidates, func(a, b assetCandidate) int {
			switch {
			case a.Path < b.Path:
				return -1
			case a.Path > b.Path:
				return 1
			default:
				return 0
			}
		})
		catalog[key] = candidates
	}
	return catalog, nil
}

func fieldString(fields map[string]any, key string) string {
	if fields == nil {
		return ""
	}
	value, ok := fields[key]
	if !ok || value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprintf("%v", value)
}

func sectionExists(sections []Section, heading string) bool {
	for _, section := range sections {
		if section.ID == heading {
			return true
		}
	}
	return false
}

func scoreCandidate(candidate assetCandidate, variant VariantContext) int {
	if candidate.Language == variant.Language && candidate.Theme == variant.Theme && candidate.Device == variant.Device {
		return 600
	}
	if candidate.Language == variant.Language && candidate.Theme == variant.Theme {
		return 500
	}
	if candidate.Language == variant.Language && candidate.Device == variant.Device {
		return 400
	}
	if candidate.Language == variant.Language {
		return 300
	}
	if candidate.Theme == variant.Theme && candidate.Device == variant.Device {
		return 200
	}
	if candidate.Language == "" && candidate.Theme == "" && candidate.Device == "" {
		return 100
	}
	return 90
}
