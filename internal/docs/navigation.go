package docs

import (
	"fmt"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode"
)

var numberedSegmentPattern = regexp.MustCompile(`^(\d+)[-_]+(.+)$`)

type pathSegment struct {
	raw      string
	label    string
	order    int
	hasOrder bool
}

type treeDirNode struct {
	segment  pathSegment
	fullPath string
	indexDoc string
	dirs     map[string]*treeDirNode
	files    map[string]*treeFileNode
}

type treeFileNode struct {
	segment pathSegment
	docPath string
}

func buildNavigationTree(docs map[string]map[string]*document) (*treeDirNode, error) {
	root := &treeDirNode{
		fullPath: "",
		dirs:     make(map[string]*treeDirNode),
		files:    make(map[string]*treeFileNode),
	}
	if _, ok := docs["index"]; !ok {
		return nil, fmt.Errorf("docs root is missing index markdown")
	}
	root.indexDoc = "index"

	keys := make([]string, 0, len(docs))
	for key := range docs {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	for _, docPath := range keys {
		if docPath == "index" {
			continue
		}
		parts := strings.Split(docPath, "/")
		if len(parts) == 0 {
			continue
		}

		last := parts[len(parts)-1]
		if last == "index" {
			dirNode, err := ensureDirectory(root, parts[:len(parts)-1])
			if err != nil {
				return nil, err
			}
			dirNode.indexDoc = docPath
			continue
		}

		parent, err := ensureDirectory(root, parts[:len(parts)-1])
		if err != nil {
			return nil, err
		}
		if _, dirExists := parent.dirs[last]; dirExists {
			return nil, fmt.Errorf("path conflict: %q is both directory and file", path.Join(parent.fullPath, last))
		}
		parent.files[last] = &treeFileNode{
			segment: parsePathSegment(last),
			docPath: docPath,
		}
	}

	if err := validateDirectoryIndexes(root); err != nil {
		return nil, err
	}
	return root, nil
}

func ensureDirectory(root *treeDirNode, segments []string) (*treeDirNode, error) {
	current := root
	for _, segment := range segments {
		if segment == "" {
			continue
		}
		if _, fileExists := current.files[segment]; fileExists {
			return nil, fmt.Errorf("path conflict: %q is both file and directory", path.Join(current.fullPath, segment))
		}
		next := current.dirs[segment]
		if next == nil {
			nextFullPath := segment
			if current.fullPath != "" {
				nextFullPath = current.fullPath + "/" + segment
			}
			next = &treeDirNode{
				segment:  parsePathSegment(segment),
				fullPath: nextFullPath,
				dirs:     make(map[string]*treeDirNode),
				files:    make(map[string]*treeFileNode),
			}
			current.dirs[segment] = next
		}
		current = next
	}
	return current, nil
}

func validateDirectoryIndexes(dir *treeDirNode) error {
	if dir.fullPath != "" && dir.indexDoc == "" {
		return fmt.Errorf("docs directory %q is missing index markdown", dir.fullPath)
	}
	for _, child := range dir.dirs {
		if err := validateDirectoryIndexes(child); err != nil {
			return err
		}
	}
	return nil
}

func parsePathSegment(raw string) pathSegment {
	segment := pathSegment{raw: raw, label: humanizeSegment(raw)}
	matches := numberedSegmentPattern.FindStringSubmatch(raw)
	if len(matches) != 3 {
		return segment
	}
	order, err := strconv.Atoi(matches[1])
	if err != nil {
		return segment
	}
	segment.order = order
	segment.hasOrder = true
	segment.label = humanizeSegment(matches[2])
	return segment
}

func humanizeSegment(raw string) string {
	replacer := strings.NewReplacer("-", " ", "_", " ")
	raw = strings.TrimSpace(replacer.Replace(raw))
	if raw == "" {
		return "Untitled"
	}
	parts := strings.Fields(raw)
	for idx, part := range parts {
		parts[idx] = titleCaseWord(part)
	}
	return strings.Join(parts, " ")
}

func titleCaseWord(word string) string {
	if word == "" {
		return word
	}
	runes := []rune(strings.ToLower(word))
	runes[0] = unicode.ToUpper(runes[0])
	return string(runes)
}

func (s *Service) Navigation(rawPath, locale string) (Navigation, error) {
	if s == nil || s.tree == nil {
		return Navigation{}, ErrNotFound
	}

	language := normalizeLocale(locale)
	if language == "" {
		language = DefaultLocale
	}

	currentPath := ""
	if resolved, _, err := s.resolveDocument(rawPath); err == nil {
		currentPath = resolved
	}

	nodes := s.navigationNodes(s.tree, currentPath, language, true)
	crumbs, pathDisplay := s.navigationCrumbs(currentPath, language)

	return Navigation{
		PathDisplay: pathDisplay,
		Crumbs:      crumbs,
		Nodes:       nodes,
	}, nil
}

func (s *Service) navigationNodes(dir *treeDirNode, currentPath, language string, isRoot bool) []NavNode {
	type sortableNode struct {
		segment pathSegment
		title   string
		node    NavNode
	}

	items := make([]sortableNode, 0, len(dir.dirs)+len(dir.files))
	for _, child := range dir.dirs {
		title := s.localizedTitle(child.indexDoc, language)
		if title == "" {
			title = child.segment.label
		}
		items = append(items, sortableNode{
			segment: child.segment,
			title:   title,
			node: NavNode{
				Title:    title,
				Path:     docsRoutePath(child.indexDoc),
				Current:  child.indexDoc == currentPath,
				Expanded: child.indexDoc == currentPath || strings.HasPrefix(currentPath, child.fullPath+"/"),
				Children: s.navigationNodes(child, currentPath, language, false),
			},
		})
	}
	for _, file := range dir.files {
		title := s.localizedTitle(file.docPath, language)
		if title == "" {
			title = file.segment.label
		}
		items = append(items, sortableNode{
			segment: file.segment,
			title:   title,
			node: NavNode{
				Title:    title,
				Path:     docsRoutePath(file.docPath),
				Current:  file.docPath == currentPath,
				Expanded: false,
			},
		})
	}

	slices.SortFunc(items, func(a, b sortableNode) int {
		if a.segment.hasOrder && b.segment.hasOrder && a.segment.order != b.segment.order {
			if a.segment.order < b.segment.order {
				return -1
			}
			return 1
		}
		if a.segment.hasOrder != b.segment.hasOrder {
			if a.segment.hasOrder {
				return -1
			}
			return 1
		}
		aTitle := strings.ToLower(strings.TrimSpace(a.title))
		bTitle := strings.ToLower(strings.TrimSpace(b.title))
		switch {
		case aTitle < bTitle:
			return -1
		case aTitle > bTitle:
			return 1
		default:
			return strings.Compare(a.node.Path, b.node.Path)
		}
	})

	result := make([]NavNode, 0, len(items)+1)
	if isRoot {
		rootTitle := s.localizedTitle("index", language)
		if rootTitle == "" {
			rootTitle = "Documentation"
		}
		result = append(result, NavNode{
			Title:    rootTitle,
			Path:     docsRoutePath("index"),
			Current:  currentPath == "index",
			Expanded: currentPath == "index",
		})
	}
	for _, item := range items {
		result = append(result, item.node)
	}
	return result
}

func (s *Service) navigationCrumbs(currentPath, language string) ([]NavCrumb, string) {
	if currentPath == "" {
		return nil, ""
	}
	if currentPath == "index" {
		title := s.localizedTitle("index", language)
		if title == "" {
			title = "Documentation"
		}
		crumb := NavCrumb{Title: title, Path: docsRoutePath("index"), Current: true}
		return []NavCrumb{crumb}, title
	}

	parts := strings.Split(currentPath, "/")
	crumbs := make([]NavCrumb, 0, len(parts))

	if parts[len(parts)-1] == "index" {
		directoryParts := parts[:len(parts)-1]
		for i := 1; i <= len(directoryParts); i++ {
			docPath := strings.Join(directoryParts[:i], "/") + "/index"
			title := s.localizedTitle(docPath, language)
			if title == "" {
				title = humanizeSegment(directoryParts[i-1])
			}
			crumbs = append(crumbs, NavCrumb{Title: title, Path: docsRoutePath(docPath), Current: false})
		}
		if len(crumbs) > 0 {
			crumbs[len(crumbs)-1].Current = true
		}
	} else {
		directoryParts := parts[:len(parts)-1]
		for i := 1; i <= len(directoryParts); i++ {
			docPath := strings.Join(directoryParts[:i], "/") + "/index"
			title := s.localizedTitle(docPath, language)
			if title == "" {
				title = humanizeSegment(directoryParts[i-1])
			}
			crumbs = append(crumbs, NavCrumb{Title: title, Path: docsRoutePath(docPath), Current: false})
		}
		title := s.localizedTitle(currentPath, language)
		if title == "" {
			title = humanizeSegment(parts[len(parts)-1])
		}
		crumbs = append(crumbs, NavCrumb{Title: title, Path: docsRoutePath(currentPath), Current: true})
	}

	if len(crumbs) == 0 {
		return nil, ""
	}
	titles := make([]string, 0, len(crumbs))
	for _, crumb := range crumbs {
		titles = append(titles, crumb.Title)
	}
	return crumbs, strings.Join(titles, " / ")
}

func (s *Service) localizedTitle(docPath, language string) string {
	localized := s.docs[docPath]
	if len(localized) == 0 {
		return ""
	}

	preferred := localized[language]
	if preferred == nil {
		preferred = localized[DefaultLocale]
	}
	if preferred == nil {
		for _, candidate := range localized {
			preferred = candidate
			break
		}
	}
	if preferred == nil {
		return ""
	}

	title := strings.TrimSpace(preferred.Titles[language])
	if title != "" {
		return title
	}
	title = strings.TrimSpace(preferred.Titles[DefaultLocale])
	if title != "" {
		return title
	}
	return strings.TrimSpace(preferred.Title)
}

func docsRoutePath(docPath string) string {
	clean, err := normalizeLogicalPath(docPath)
	if err != nil {
		return "index"
	}
	if clean == "index" {
		return clean
	}
	if strings.HasSuffix(clean, "/index") {
		trimmed := strings.TrimSuffix(clean, "/index")
		if trimmed == "" {
			return "index"
		}
		return trimmed
	}
	return clean
}
