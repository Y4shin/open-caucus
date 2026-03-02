package docs

import (
	"fmt"
	"strings"
	"testing"
	"testing/fstest"
)

func TestLocalizedFallbackAndRender(t *testing.T) {
	svc, err := Load(
		withRootDocs(fstest.MapFS{
			"guides/index.en.md":   docFile("Guides", "Leitfaeden", "# Guides\n\nGuide index."),
			"guides/index.de.md":   docFile("Guides", "Leitfaeden", "# Leitfaeden\n\nUebersicht."),
			"guides/setup.en.md":   docFile("Setup", "Einrichtung", "# Setup\n\nEnglish content."),
			"guides/setup.de.md":   docFile("Setup", "Einrichtung", "# Einrichtung\n\nDeutscher Inhalt."),
			"guides/only-en.en.md": docFile("English Only", "Nur Englisch", "# English Only\n\nOnly english content."),
		}),
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

func TestRenderResolvesDirectoryIndexDocument(t *testing.T) {
	svc, err := Load(
		withRootDocs(fstest.MapFS{
			"committees/index.en.md": docFile("Committees", "Ausschuesse", "# Committees\n\nOverview."),
			"committees/index.de.md": docFile("Committees", "Ausschuesse", "# Ausschuesse\n\nUebersicht."),
		}),
		fstest.MapFS{},
	)
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	rendered, err := svc.Render(DocRef{Path: "committees"}, VariantContext{Language: "de"})
	if err != nil {
		t.Fatalf("render committees directory path: %v", err)
	}
	if rendered.Path != "committees/index" {
		t.Fatalf("expected committees/index path, got %q", rendered.Path)
	}
	if rendered.Title != "Ausschuesse" {
		t.Fatalf("expected german index title, got %q", rendered.Title)
	}
}

func TestMediaVariantResolutionInRenderedHTML(t *testing.T) {
	svc, err := Load(
		withRootDocs(fstest.MapFS{
			"guides/index.en.md":   docFile("Guides", "Leitfaeden", "# Guides\n\nGuide index."),
			"guides/index.de.md":   docFile("Guides", "Leitfaeden", "# Leitfaeden\n\nUebersicht."),
			"guides/capture.en.md": docFile("Capture", "Aufnahme", "# Capture\n\n![Sample](../../assets/captures/demo.en.light.desktop.png)\n"),
		}),
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
		withRootDocs(fstest.MapFS{
			"search/index.en.md": docFile("Search", "Suche", "# Search\n\nSearch index."),
			"search/index.de.md": docFile("Search", "Suche", "# Suche\n\nIndex."),
			"search/topic.en.md": docFile("Topic", "Thema", "# Topic\n\nBlue whale reference only in english."),
			"search/topic.de.md": docFile("Topic", "Thema", "# Thema\n\nBlauer Wal nur auf deutsch."),
		}),
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
		withRootDocs(fstest.MapFS{
			"search/index.en.md":  docFile("Search", "Suche", "# Search\n\nSearch index."),
			"search/prefix.en.md": docFile("Prefix", "Praefix", "# Prefix\n\nGetting started guide for setup flow."),
		}),
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
		withRootDocs(fstest.MapFS{
			"search/index.en.md": docFile("Search", "Suche", "# Search\n\nSearch index."),
			"search/fuzzy.en.md": docFile("Fuzzy", "Unscharf", "# Fuzzy\n\nGetting started guide for setup flow."),
		}),
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

func TestLoadFailsWhenFrontmatterTitlesAreMissing(t *testing.T) {
	_, err := Load(
		fstest.MapFS{
			"index.en.md": &fstest.MapFile{Data: []byte("---\ntitle-en: Home\n---\n\n# Home\n")},
		},
		fstest.MapFS{},
	)
	if err == nil {
		t.Fatalf("expected load failure when title-de is missing")
	}
	if !strings.Contains(err.Error(), "frontmatter must include non-empty title-de") {
		t.Fatalf("expected title-de validation error, got: %v", err)
	}
}

func TestLoadFailsWhenDirectoryIndexIsMissing(t *testing.T) {
	_, err := Load(
		withRootDocs(fstest.MapFS{
			"guides/setup.en.md": docFile("Setup", "Einrichtung", "# Setup\n\nBody."),
		}),
		fstest.MapFS{},
	)
	if err == nil {
		t.Fatalf("expected load failure when guides/index is missing")
	}
	if !strings.Contains(err.Error(), `docs directory "guides" is missing index markdown`) {
		t.Fatalf("expected missing index validation error, got: %v", err)
	}
}

func TestNavigationBuildsLocalizedCrumbsAndOrderedTree(t *testing.T) {
	svc, err := Load(
		withRootDocs(fstest.MapFS{
			"committees/index.en.md":       docFile("Committees", "Ausschuesse", "# Committees\n\nOverview."),
			"committees/index.de.md":       docFile("Committees", "Ausschuesse", "# Ausschuesse\n\nUebersicht."),
			"committees/10-wrapup.en.md":   docFile("Wrap Up", "Abschluss", "# Wrap Up\n\nBody."),
			"committees/10-wrapup.de.md":   docFile("Wrap Up", "Abschluss", "# Abschluss\n\nBody."),
			"committees/01-intro.en.md":    docFile("Introduction", "Einfuehrung", "# Intro\n\nBody."),
			"committees/01-intro.de.md":    docFile("Introduction", "Einfuehrung", "# Einfuehrung\n\nBody."),
			"committees/02-rules.en.md":    docFile("Rules", "Regeln", "# Rules\n\nBody."),
			"committees/02-rules.de.md":    docFile("Rules", "Regeln", "# Regeln\n\nBody."),
			"resources/index.en.md":        docFile("Resources", "Ressourcen", "# Resources\n\nOverview."),
			"resources/index.de.md":        docFile("Resources", "Ressourcen", "# Ressourcen\n\nUebersicht."),
			"resources/01-checklist.en.md": docFile("Checklist", "Checkliste", "# Checklist\n\nBody."),
			"resources/01-checklist.de.md": docFile("Checklist", "Checkliste", "# Checkliste\n\nBody."),
		}),
		fstest.MapFS{},
	)
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	nav, err := svc.Navigation("committees/10-wrapup", "de")
	if err != nil {
		t.Fatalf("build navigation: %v", err)
	}

	if nav.PathDisplay != "Ausschuesse / Abschluss" {
		t.Fatalf("unexpected path display: %q", nav.PathDisplay)
	}
	if len(nav.Crumbs) != 2 {
		t.Fatalf("expected 2 breadcrumbs, got %d", len(nav.Crumbs))
	}
	if nav.Crumbs[0].Title != "Ausschuesse" || nav.Crumbs[0].Path != "committees" {
		t.Fatalf("unexpected first breadcrumb: %+v", nav.Crumbs[0])
	}
	if nav.Crumbs[1].Title != "Abschluss" || nav.Crumbs[1].Path != "committees/10-wrapup" || !nav.Crumbs[1].Current {
		t.Fatalf("unexpected second breadcrumb: %+v", nav.Crumbs[1])
	}

	if len(nav.Nodes) < 2 {
		t.Fatalf("expected root and directory nodes, got %d", len(nav.Nodes))
	}
	if nav.Nodes[0].Path != "index" {
		t.Fatalf("expected first root node to be index, got %q", nav.Nodes[0].Path)
	}

	committeesNode := findNodeByPath(nav.Nodes, "committees")
	if committeesNode == nil {
		t.Fatalf("expected committees node in tree")
	}
	if !committeesNode.Expanded {
		t.Fatalf("expected committees directory to be expanded for current document")
	}

	gotChildPaths := make([]string, 0, len(committeesNode.Children))
	for _, child := range committeesNode.Children {
		gotChildPaths = append(gotChildPaths, child.Path)
	}
	wantChildPaths := []string{"committees/01-intro", "committees/02-rules", "committees/10-wrapup"}
	if strings.Join(gotChildPaths, ",") != strings.Join(wantChildPaths, ",") {
		t.Fatalf("unexpected child order. got=%v want=%v", gotChildPaths, wantChildPaths)
	}

	currentNode := findNodeByPath(committeesNode.Children, "committees/10-wrapup")
	if currentNode == nil || !currentNode.Current {
		t.Fatalf("expected current file node for committees/10-wrapup")
	}
}

func withRootDocs(extra fstest.MapFS) fstest.MapFS {
	out := fstest.MapFS{
		"index.en.md": docFile("Documentation", "Dokumentation", "# Home\n\nWelcome."),
		"index.de.md": docFile("Documentation", "Dokumentation", "# Start\n\nWillkommen."),
	}
	for name, file := range extra {
		out[name] = file
	}
	return out
}

func docFile(titleEN, titleDE, body string) *fstest.MapFile {
	content := fmt.Sprintf("---\ntitle-en: %q\ntitle-de: %q\n---\n\n%s\n", titleEN, titleDE, body)
	return &fstest.MapFile{Data: []byte(content)}
}

func findNodeByPath(nodes []NavNode, path string) *NavNode {
	for _, node := range nodes {
		if node.Path == path {
			copy := node
			return &copy
		}
	}
	return nil
}
