package docs

import (
	"fmt"
	"strings"

	"github.com/blevesearch/bleve/v2"
)

type indexedDoc struct {
	Ref     string `json:"ref"`
	Path    string `json:"path"`
	Heading string `json:"heading"`
	Title   string `json:"title"`
	Body    string `json:"body"`
	Locale  string `json:"locale"`
}

func buildIndex(docs map[string]map[string]*document) (bleve.Index, error) {
	mapping := bleve.NewIndexMapping()
	docMapping := bleve.NewDocumentMapping()

	keywordField := bleve.NewKeywordFieldMapping()
	keywordField.Store = true
	keywordField.Index = true

	textField := bleve.NewTextFieldMapping()
	textField.Store = true
	textField.Index = true

	docMapping.AddFieldMappingsAt("ref", keywordField)
	docMapping.AddFieldMappingsAt("path", keywordField)
	docMapping.AddFieldMappingsAt("heading", keywordField)
	docMapping.AddFieldMappingsAt("locale", keywordField)
	docMapping.AddFieldMappingsAt("title", textField)
	docMapping.AddFieldMappingsAt("body", textField)

	mapping.DefaultAnalyzer = "standard"
	mapping.AddDocumentMapping("doc", docMapping)

	index, err := bleve.NewMemOnly(mapping)
	if err != nil {
		return nil, err
	}

	batch := index.NewBatch()
	for pathKey, localized := range docs {
		for locale, doc := range localized {
			if doc == nil {
				continue
			}
			allTextParts := make([]string, 0, len(doc.Sections))
			for _, section := range doc.Sections {
				sectionText := strings.TrimSpace(strings.TrimSpace(section.Title + " " + section.Text))
				if sectionText != "" {
					allTextParts = append(allTextParts, sectionText)
				}
				ref := pathKey + "#" + section.ID
				id := locale + "|" + ref
				batch.Index(id, indexedDoc{
					Ref:     ref,
					Path:    pathKey,
					Heading: section.ID,
					Title:   doc.Title,
					Body:    sectionText,
					Locale:  locale,
				})
			}
			batch.Index(locale+"|"+pathKey, indexedDoc{
				Ref:     pathKey,
				Path:    pathKey,
				Heading: "",
				Title:   doc.Title,
				Body:    strings.Join(allTextParts, " "),
				Locale:  locale,
			})
		}
	}

	if err := index.Batch(batch); err != nil {
		_ = index.Close()
		return nil, fmt.Errorf("populate bleve docs index: %w", err)
	}
	return index, nil
}
