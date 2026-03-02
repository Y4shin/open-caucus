package docs

import (
	"bytes"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

var (
	localizedDocPattern = regexp.MustCompile(`^(.+)\.([a-z]{2})\.md$`)
	variantFilePattern  = regexp.MustCompile(`^(?i)(.+)\.([a-z]{2})\.(light|dark)(?:\.(desktop|mobile))?\.([a-z0-9]+)$`)
)

const mediaPlaceholderPrefix = "docasset://"

func loadDocuments(contentFS fs.FS) (map[string]map[string]*document, error) {
	docs := make(map[string]map[string]*document)
	err := fs.WalkDir(contentFS, ".", func(name string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(strings.ToLower(name), ".md") {
			return nil
		}

		cleanPath := path.Clean(strings.TrimPrefix(name, "/"))
		m := localizedDocPattern.FindStringSubmatch(cleanPath)
		if m == nil {
			return fmt.Errorf("docs markdown file %q must follow <path>.<locale>.md", cleanPath)
		}
		logicalPath, err := normalizeLogicalPath(m[1])
		if err != nil {
			return fmt.Errorf("invalid docs logical path %q: %w", cleanPath, err)
		}
		locale := normalizeLocale(m[2])
		if locale == "" {
			return fmt.Errorf("unsupported locale in docs file %q", cleanPath)
		}

		raw, err := fs.ReadFile(contentFS, cleanPath)
		if err != nil {
			return fmt.Errorf("read docs markdown %q: %w", cleanPath, err)
		}

		doc, err := parseLocalizedMarkdown(cleanPath, logicalPath, locale, raw)
		if err != nil {
			return fmt.Errorf("parse docs markdown %q: %w", cleanPath, err)
		}

		if docs[logicalPath] == nil {
			docs[logicalPath] = make(map[string]*document)
		}
		docs[logicalPath][locale] = doc
		return nil
	})
	if err != nil {
		return nil, err
	}
	return docs, nil
}

func parseLocalizedMarkdown(sourcePath, logicalPath, locale string, source []byte) (*document, error) {
	titles, markdownBody, err := extractTitleFrontMatter(source)
	if err != nil {
		return nil, err
	}

	md := goldmark.New(
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)

	root := md.Parser().Parse(text.NewReader(markdownBody))

	type headingRef struct {
		node *ast.Heading
		idx  int
	}
	headingRefs := make([]headingRef, 0)
	sections := make([]Section, 0)
	slugCounts := make(map[string]int)
	mediaSet := make(map[string]bool)
	mediaKeys := make([]string, 0)

	err = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		heading, ok := node.(*ast.Heading)
		if ok {
			title := strings.TrimSpace(extractText(node, markdownBody))
			if title == "" {
				title = "section"
			}
			slug := uniqueSlug(slugify(title), slugCounts)
			heading.SetAttributeString("id", []byte(slug))
			headingRefs = append(headingRefs, headingRef{node: heading, idx: len(sections)})
			sections = append(sections, Section{
				ID:    slug,
				Title: title,
				Level: heading.Level,
			})
			return ast.WalkContinue, nil
		}

		image, ok := node.(*ast.Image)
		if ok {
			normalized, hasAsset := normalizeMediaDestination(string(image.Destination), sourcePath)
			if hasAsset {
				placeholder := mediaPlaceholderPrefix + normalized
				image.Destination = []byte(placeholder)
				if !mediaSet[normalized] {
					mediaSet[normalized] = true
					mediaKeys = append(mediaKeys, normalized)
				}
			}
		}

		return ast.WalkContinue, nil
	})
	if err != nil {
		return nil, err
	}

	headingIndexByNode := make(map[*ast.Heading]int, len(headingRefs))
	for _, ref := range headingRefs {
		headingIndexByNode[ref.node] = ref.idx
	}
	currentSection := -1
	_ = ast.Walk(root, func(node ast.Node, entering bool) (ast.WalkStatus, error) {
		heading, ok := node.(*ast.Heading)
		if ok {
			if entering {
				currentSection = headingIndexByNode[heading]
			}
			return ast.WalkContinue, nil
		}
		if !entering || currentSection < 0 {
			return ast.WalkContinue, nil
		}
		switch n := node.(type) {
		case *ast.Text:
			textChunk := strings.TrimSpace(string(n.Segment.Value(markdownBody)))
			if textChunk != "" {
				if sections[currentSection].Text != "" {
					sections[currentSection].Text += " "
				}
				sections[currentSection].Text += textChunk
			}
		case *ast.CodeSpan:
			textChunk := strings.TrimSpace(string(n.Text(markdownBody)))
			if textChunk != "" {
				if sections[currentSection].Text != "" {
					sections[currentSection].Text += " "
				}
				sections[currentSection].Text += textChunk
			}
		}
		return ast.WalkContinue, nil
	})

	var rendered bytes.Buffer
	if err := md.Renderer().Render(&rendered, markdownBody, root); err != nil {
		return nil, fmt.Errorf("render markdown to html: %w", err)
	}

	title := strings.TrimSpace(titles[locale])
	if title == "" {
		title = strings.TrimSpace(titles[DefaultLocale])
	}
	if title == "" {
		title = path.Base(logicalPath)
	}

	return &document{
		Path:         logicalPath,
		Locale:       locale,
		Title:        title,
		Titles:       titles,
		HTMLTemplate: rendered.String(),
		Sections:     sections,
		MediaKeys:    mediaKeys,
	}, nil
}

func extractTitleFrontMatter(source []byte) (map[string]string, []byte, error) {
	normalized := string(source)
	normalized = strings.TrimPrefix(normalized, "\ufeff")
	normalized = strings.ReplaceAll(normalized, "\r\n", "\n")

	const marker = "---\n"
	if !strings.HasPrefix(normalized, marker) {
		return nil, nil, fmt.Errorf("missing YAML frontmatter with title-en/title-de")
	}

	remaining := normalized[len(marker):]
	frontmatterEnd := strings.Index(remaining, "\n---\n")
	if frontmatterEnd < 0 {
		return nil, nil, fmt.Errorf("unterminated YAML frontmatter")
	}

	var raw map[string]any
	if err := yaml.Unmarshal([]byte(remaining[:frontmatterEnd]), &raw); err != nil {
		return nil, nil, fmt.Errorf("invalid YAML frontmatter: %w", err)
	}

	titles := map[string]string{
		"en": strings.TrimSpace(fmt.Sprintf("%v", raw["title-en"])),
		"de": strings.TrimSpace(fmt.Sprintf("%v", raw["title-de"])),
	}
	for _, locale := range []string{"en", "de"} {
		if titles[locale] == "" || titles[locale] == "<nil>" {
			return nil, nil, fmt.Errorf("frontmatter must include non-empty title-%s", locale)
		}
	}

	body := remaining[frontmatterEnd+len("\n---\n"):]
	return titles, []byte(body), nil
}

func extractText(node ast.Node, source []byte) string {
	var b strings.Builder
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		switch t := n.(type) {
		case *ast.Text:
			b.Write(t.Segment.Value(source))
			b.WriteByte(' ')
		case *ast.CodeSpan:
			b.Write(t.Text(source))
			b.WriteByte(' ')
		}
		return ast.WalkContinue, nil
	})
	return strings.TrimSpace(b.String())
}

func normalizeMediaDestination(destination, sourcePath string) (string, bool) {
	destination = strings.TrimSpace(destination)
	if destination == "" {
		return "", false
	}
	if strings.HasPrefix(destination, "http://") || strings.HasPrefix(destination, "https://") || strings.HasPrefix(destination, "data:") {
		return "", false
	}

	clean := strings.TrimPrefix(path.Clean(strings.TrimSpace(destination)), "/")
	if strings.HasPrefix(clean, "docs/assets/") {
		clean = strings.TrimPrefix(clean, "docs/assets/")
	} else if strings.HasPrefix(clean, "assets/") {
		clean = strings.TrimPrefix(clean, "assets/")
	} else {
		resolved := resolveDocRelativeAssetPath(clean, sourcePath)
		if resolved == "" {
			return "", false
		}
		clean = resolved
	}

	clean, err := normalizeAssetKey(clean)
	if err != nil {
		return "", false
	}
	normalized, _ := classifyAsset(clean)
	return normalized, true
}

func resolveDocRelativeAssetPath(destination, sourcePath string) string {
	if destination == "" {
		return ""
	}
	sourceDir := path.Dir(strings.TrimPrefix(path.Clean("/"+sourcePath), "/"))
	repoRelative := path.Clean(path.Join("content", sourceDir, destination))
	if strings.HasPrefix(repoRelative, "assets/") {
		return strings.TrimPrefix(repoRelative, "assets/")
	}
	return ""
}

func classifyAsset(assetPath string) (string, assetCandidate) {
	dir, file := path.Split(assetPath)
	m := variantFilePattern.FindStringSubmatch(file)
	candidate := assetCandidate{Path: assetPath}
	if m == nil {
		return assetPath, candidate
	}
	base := m[1]
	candidate.Language = normalizeLocale(m[2])
	candidate.Theme = normalizeTheme(m[3])
	candidate.Device = normalizeDevice(m[4])
	ext := strings.ToLower(m[5])
	normalizedFile := base + "." + ext
	return path.Join(dir, normalizedFile), candidate
}

func normalizeLogicalPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" || p == "." || p == "/" {
		return "index", nil
	}
	clean := strings.TrimPrefix(path.Clean("/"+p), "/")
	if clean == "." || clean == "" {
		return "index", nil
	}
	if strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("path traversal not allowed")
	}
	return clean, nil
}

func normalizeAssetKey(assetPath string) (string, error) {
	assetPath = strings.TrimSpace(strings.TrimPrefix(assetPath, "/"))
	if assetPath == "" {
		return "", fmt.Errorf("empty asset path")
	}
	clean := path.Clean(assetPath)
	if clean == "." || strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("invalid asset path")
	}
	return clean, nil
}

func normalizeLocale(locale string) string {
	locale = strings.ToLower(strings.TrimSpace(locale))
	switch locale {
	case "en", "de":
		return locale
	default:
		return ""
	}
}

func normalizeTheme(theme string) string {
	theme = strings.ToLower(strings.TrimSpace(theme))
	switch theme {
	case ThemeDark:
		return ThemeDark
	case ThemeLight:
		return ThemeLight
	default:
		return ThemeLight
	}
}

func normalizeDevice(device string) string {
	device = strings.ToLower(strings.TrimSpace(device))
	switch device {
	case DeviceMobile:
		return DeviceMobile
	case DeviceDesktop:
		return DeviceDesktop
	default:
		return DeviceDesktop
	}
}

func normalizeVariant(v VariantContext) VariantContext {
	lang := normalizeLocale(v.Language)
	if lang == "" {
		lang = DefaultLocale
	}
	return VariantContext{
		Language: lang,
		Theme:    normalizeTheme(v.Theme),
		Device:   normalizeDevice(v.Device),
	}
}

func normalizeHeading(h string) string {
	return strings.TrimSpace(strings.TrimPrefix(h, "#"))
}

func uniqueSlug(base string, counts map[string]int) string {
	if base == "" {
		base = "section"
	}
	count := counts[base]
	counts[base] = count + 1
	if count == 0 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, count+1)
}

func slugify(input string) string {
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return ""
	}
	var b strings.Builder
	lastDash := false
	for _, r := range input {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	slug := strings.Trim(b.String(), "-")
	if slug == "" {
		return "section"
	}
	return slug
}

func buildSnippet(body, query string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	lowerBody := strings.ToLower(body)
	lowerQuery := strings.ToLower(strings.TrimSpace(query))
	if lowerQuery == "" {
		if len(body) > 220 {
			return body[:220] + "..."
		}
		return body
	}
	idx := strings.Index(lowerBody, lowerQuery)
	if idx < 0 {
		if len(body) > 220 {
			return body[:220] + "..."
		}
		return body
	}
	start := idx - 90
	if start < 0 {
		start = 0
	}
	end := idx + len(lowerQuery) + 90
	if end > len(body) {
		end = len(body)
	}
	snippet := strings.TrimSpace(body[start:end])
	if start > 0 {
		snippet = "..." + snippet
	}
	if end < len(body) {
		snippet += "..."
	}
	return snippet
}
