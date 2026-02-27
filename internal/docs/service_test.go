package docs

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestLocalizedFallbackAndRender(t *testing.T) {
	svc, err := Load(
		fstest.MapFS{
			"guides/setup.en.md":   &fstest.MapFile{Data: []byte("# Setup\n\nEnglish content.")},
			"guides/setup.de.md":   &fstest.MapFile{Data: []byte("# Einrichtung\n\nDeutscher Inhalt.")},
			"guides/only-en.en.md": &fstest.MapFile{Data: []byte("# English Only\n\nOnly english content.")},
		},
		fstest.MapFS{},
	)
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	rendered, err := svc.Render(DocRef{Path: "guides/setup"}, VariantContext{Language: "de"})
	if err != nil {
		t.Fatalf("render de setup: %v", err)
	}
	if rendered.Locale != "de" {
		t.Fatalf("expected de locale variant, got %q", rendered.Locale)
	}
	if !strings.Contains(rendered.HTML, "Deutscher Inhalt") {
		t.Fatalf("expected german html content, got: %s", rendered.HTML)
	}

	fallback, err := svc.Render(DocRef{Path: "guides/only-en"}, VariantContext{Language: "de"})
	if err != nil {
		t.Fatalf("render fallback setup: %v", err)
	}
	if fallback.Locale != "en" {
		t.Fatalf("expected english fallback, got %q", fallback.Locale)
	}
}

func TestMediaVariantResolutionInRenderedHTML(t *testing.T) {
	svc, err := Load(
		fstest.MapFS{
			"guides/capture.en.md": &fstest.MapFile{Data: []byte("# Capture\n\n![Sample](../../assets/captures/demo.en.light.desktop.png)\n")},
		},
		fstest.MapFS{
			"captures/demo.en.light.desktop.png": &fstest.MapFile{Data: []byte("png")},
			"captures/demo.en.dark.desktop.png":  &fstest.MapFile{Data: []byte("png")},
			"captures/demo.de.dark.mobile.png":   &fstest.MapFile{Data: []byte("png")},
		},
	)
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	rendered, err := svc.Render(DocRef{Path: "guides/capture"}, VariantContext{Language: "de", Theme: "dark", Device: "mobile"})
	if err != nil {
		t.Fatalf("render capture: %v", err)
	}
	if !strings.Contains(rendered.HTML, "/docs/assets/captures/demo.de.dark.mobile.png") {
		t.Fatalf("expected resolved exact variant url in html, got: %s", rendered.HTML)
	}
}

func TestSearchFilteredByLocale(t *testing.T) {
	svc, err := Load(
		fstest.MapFS{
			"search/topic.en.md": &fstest.MapFile{Data: []byte("# Topic\n\nBlue whale reference only in english.")},
			"search/topic.de.md": &fstest.MapFile{Data: []byte("# Thema\n\nBlauer Wal nur auf deutsch.")},
		},
		fstest.MapFS{},
	)
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	hits, err := svc.Search("Blauer", "de", 10)
	if err != nil {
		t.Fatalf("search de locale: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected hits for de query")
	}
	for _, hit := range hits {
		if hit.Locale != "de" {
			t.Fatalf("expected locale-filtered de hit, got locale=%q", hit.Locale)
		}
	}

	enHits, err := svc.Search("Blauer", "en", 10)
	if err != nil {
		t.Fatalf("search en locale: %v", err)
	}
	if len(enHits) != 0 {
		t.Fatalf("expected zero english hits for german-only query, got %d", len(enHits))
	}
}

func TestSearchMatchesPrefixTerms(t *testing.T) {
	svc, err := Load(
		fstest.MapFS{
			"search/prefix.en.md": &fstest.MapFile{Data: []byte("# Prefix\n\nGetting started guide for setup flow.")},
		},
		fstest.MapFS{},
	)
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	hits, err := svc.Search("Star", "en", 10)
	if err != nil {
		t.Fatalf("search prefix: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected prefix query to match 'started'")
	}
}

func TestSearchMatchesConservativeFuzzyTerms(t *testing.T) {
	svc, err := Load(
		fstest.MapFS{
			"search/fuzzy.en.md": &fstest.MapFile{Data: []byte("# Fuzzy\n\nGetting started guide for setup flow.")},
		},
		fstest.MapFS{},
	)
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	hits, err := svc.Search("Starrted", "en", 10)
	if err != nil {
		t.Fatalf("search fuzzy: %v", err)
	}
	if len(hits) == 0 {
		t.Fatalf("expected fuzzy query to match 'started'")
	}

	caseHits, err := svc.Search("sTaR", "en", 10)
	if err != nil {
		t.Fatalf("search mixed-case prefix: %v", err)
	}
	if len(caseHits) == 0 {
		t.Fatalf("expected mixed-case query to match case-insensitively")
	}
}
