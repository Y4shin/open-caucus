package docs

import "errors"

const (
	DefaultLocale = "en"
	ThemeLight    = "light"
	ThemeDark     = "dark"
	DeviceDesktop = "desktop"
	DeviceMobile  = "mobile"
)

var ErrNotFound = errors.New("docs: not found")

type DocRef struct {
	Path    string
	Heading string
}

type VariantContext struct {
	Language string
	Theme    string
	Device   string
}

type Section struct {
	ID    string
	Title string
	Level int
	Text  string
}

type RenderedDoc struct {
	Path     string
	Locale   string
	Title    string
	Heading  string
	HTML     string
	Sections []Section
}

type SearchHit struct {
	Ref     string
	Path    string
	Heading string
	Title   string
	Locale  string
	Snippet string
	Score   float64
}

type document struct {
	Path         string
	Locale       string
	Title        string
	HTMLTemplate string
	Sections     []Section
	MediaKeys    []string
}

type assetCandidate struct {
	Path     string
	Language string
	Theme    string
	Device   string
}
