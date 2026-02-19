# Step 2 – Update the code generator for locale-aware path builders

Three files in `tools/routing/` need changes.

---

## `tools/routing/routebuilders.go`

### 1. Extend `RouteGroup` struct

Add `HasLocale bool` and `LocaleAlias string` fields:

```go
type RouteGroup struct {
	Params          []string
	ConstructorArgs string
	Routes          []BuilderRoute
	HasLocale       bool   // true when a locale package was supplied to the generator
	LocaleAlias     string // import alias, derived from the package path's last segment, e.g. "locale"
}
```

### 2. Update `GetRouteBuilders` signature

```go
// was: func GetRouteBuilders(config *RouteConfig) RouteBuilders
func GetRouteBuilders(config *RouteConfig, localePackage string) RouteBuilders {
```

Inside, derive the alias from the package path:

```go
localeAlias := localePackage
if idx := strings.LastIndex(localePackage, "/"); idx >= 0 {
    localeAlias = localePackage[idx+1:]
}
```

Set the fields on every new `RouteGroup`:

```go
groups[key] = &RouteGroup{
    Params:          pascalParams,
    ConstructorArgs: strings.Join(args, ", "),
    HasLocale:       localePackage != "",
    LocaleAlias:     localeAlias,
}
```

### 3. Replace the `routeGroup` template constant

The new template stores `$hasLocale` and `$localeAlias` at the top (accessible inside all nested `range` blocks) and conditionally emits the extra parameters.

```go
const routeGroup string = `
{{- $hasLocale := .HasLocale }}
{{- $localeAlias := .LocaleAlias }}
{{- if not .Params }}
{{- range .Routes }}
{{- $path := .Path }}
{{- range .Verbs }}
{{- if $hasLocale }}
func (Routes) {{ .Handler }}{{ .Verb | lower | capitalize }}(ctx context.Context, override string) string {
	return {{ $localeAlias }}.PathPrefix(ctx, override) + "{{ $path }}"
}
{{- else }}
func (Routes) {{ .Handler }}{{ .Verb | lower | capitalize }}() string {
	return "{{ $path }}"
}
{{- end }}
{{- if .QueryParams }}

type {{ .Handler }}{{ .Verb | lower | capitalize }}QueryParams struct {
{{- range .QueryParams }}
	{{ . | toPascalCase }} string
{{- end }}
}
{{- if $hasLocale }}

func (Routes) {{ .Handler }}{{ .Verb | lower | capitalize }}WithQuery(ctx context.Context, override string, q {{ .Handler }}{{ .Verb | lower | capitalize }}QueryParams) string {
	path := {{ $localeAlias }}.PathPrefix(ctx, override) + "{{ $path }}"
	var qparts []string
{{- range .QueryParams }}
	if q.{{ . | toPascalCase }} != "" {
		qparts = append(qparts, "{{ . }}="+url.QueryEscape(q.{{ . | toPascalCase }}))
	}
{{- end }}
	if len(qparts) > 0 {
		path += "?" + strings.Join(qparts, "&")
	}
	return path
}
{{- else }}

func (Routes) {{ .Handler }}{{ .Verb | lower | capitalize }}WithQuery(q {{ .Handler }}{{ .Verb | lower | capitalize }}QueryParams) string {
	path := "{{ $path }}"
	var qparts []string
{{- range .QueryParams }}
	if q.{{ . | toPascalCase }} != "" {
		qparts = append(qparts, "{{ . }}="+url.QueryEscape(q.{{ . | toPascalCase }}))
	}
{{- end }}
	if len(qparts) > 0 {
		path += "?" + strings.Join(qparts, "&")
	}
	return path
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- else }}
{{- $params := .Params }}
{{- $constructorArgs := .ConstructorArgs }}
{{- range .Routes }}
type {{ .StructName }} struct {
	{{- range $params }}
	{{ . }} string
	{{- end }}
}
{{- $structName := .StructName }}
{{- $pathReturn := .PathReturn }}

func New{{ $structName }}({{ $constructorArgs }}) *{{ $structName }} {
	return &{{ $structName }}{
		{{- range $params }}
		{{ . }}: {{ . | lower }},
		{{- end }}
	}
}
{{- range .Verbs }}
{{- if $hasLocale }}

func (r *{{ $structName }}) {{ .Handler }}{{ .Verb | lower | capitalize }}(ctx context.Context, override string) string {
	return {{ $localeAlias }}.PathPrefix(ctx, override) + "{{ $pathReturn }}"
}
{{- else }}

func (r *{{ $structName }}) {{ .Handler }}{{ .Verb | lower | capitalize }}() string {
	return "{{ $pathReturn }}"
}
{{- end }}
{{- if .QueryParams }}

type {{ .Handler }}{{ .Verb | lower | capitalize }}QueryParams struct {
{{- range .QueryParams }}
	{{ . | toPascalCase }} string
{{- end }}
}
{{- if $hasLocale }}

func (r *{{ $structName }}) {{ .Handler }}{{ .Verb | lower | capitalize }}WithQuery(ctx context.Context, override string, q {{ .Handler }}{{ .Verb | lower | capitalize }}QueryParams) string {
	path := r.{{ .Handler }}{{ .Verb | lower | capitalize }}(ctx, override)
	var qparts []string
{{- range .QueryParams }}
	if q.{{ . | toPascalCase }} != "" {
		qparts = append(qparts, "{{ . }}="+url.QueryEscape(q.{{ . | toPascalCase }}))
	}
{{- end }}
	if len(qparts) > 0 {
		path += "?" + strings.Join(qparts, "&")
	}
	return path
}
{{- else }}

func (r *{{ $structName }}) {{ .Handler }}{{ .Verb | lower | capitalize }}WithQuery(q {{ .Handler }}{{ .Verb | lower | capitalize }}QueryParams) string {
	path := r.{{ .Handler }}{{ .Verb | lower | capitalize }}()
	var qparts []string
{{- range .QueryParams }}
	if q.{{ . | toPascalCase }} != "" {
		qparts = append(qparts, "{{ . }}="+url.QueryEscape(q.{{ . | toPascalCase }}))
	}
{{- end }}
	if len(qparts) > 0 {
		path += "?" + strings.Join(qparts, "&")
	}
	return path
}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
{{- end }}
`
```

---

## `tools/routing/generator.go`

### 1. Add `LocalePackage` to `PathsGenData`

```go
type PathsGenData struct {
	RouteBuilders RouteBuilders
	StaticAssets  StaticAssets
	PackageName   string
	LocalePackage string // e.g. "github.com/Y4shin/conference-tool/internal/locale"
}
```

### 2. Update the `pathsGen` template constant

Replace the conditional imports block so it handles the locale package:

```go
const pathsGen string = `// Code generated by route-codegen. DO NOT EDIT.

package {{ .PackageName }}
{{- if or .LocalePackage .RouteBuilders.HasQueryParams }}

import (
{{- if .LocalePackage }}
	"context"
	"{{ .LocalePackage }}"
{{- end }}
{{- if .RouteBuilders.HasQueryParams }}
	"net/url"
	"strings"
{{- end }}
)
{{- end }}

{{ template "RouteBuilders" .RouteBuilders }}
{{ template "StaticPathMethods" .StaticAssets }}
`
```

### 3. Update `GeneratePaths` signature

```go
// was: func GeneratePaths(config *RouteConfig, packageName string, staticFiles []StaticFileInfo) (string, error)
func GeneratePaths(config *RouteConfig, packageName string, staticFiles []StaticFileInfo, localePackage string) (string, error) {
	data := PathsGenData{
		PackageName:   packageName,
		RouteBuilders: GetRouteBuilders(config, localePackage),
		StaticAssets:  StaticAssets{Files: staticFiles, HasFiles: len(staticFiles) > 0},
		LocalePackage: localePackage,
	}
	// rest unchanged
```

---

## `tools/routing/cmd/route-codegen/main.go`

Add the new CLI flag and thread it through:

```go
localePackage = flag.String("locale-package", "", "Import path of the locale package (enables locale-aware path builders)")
```

Update the `GeneratePaths` call:

```go
pathsCode, err := routing.GeneratePaths(config, pathsPackage, staticFiles, *localePackage)
```
