package routing

import "strings"

type BuilderRouteVerb struct {
	Verb        string
	Handler     string
	QueryParams []string
}

type BuilderRoute struct {
	Path       string
	PathReturn string
	StructName string
	Verbs      []BuilderRouteVerb
}

type RouteGroup struct {
	Params          []string
	ConstructorArgs string
	Routes          []BuilderRoute
	HasLocale       bool
	LocaleAlias     string
}

type RouteBuilders struct {
	Groups         []RouteGroup
	HasQueryParams bool
}

const routeBuilders string = `// Type-safe route builders
type Routes struct{}

var Route Routes

{{ range .Groups }}
{{ template "RouteGroup" . }}
{{ end -}}
`

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

func GetRouteBuilders(config *RouteConfig, localePackage string) RouteBuilders {
	groups := make(map[string]*RouteGroup)
	hasQueryParams := false
	localeAlias := localePackage
	if idx := strings.LastIndex(localePackage, "/"); idx >= 0 {
		localeAlias = localePackage[idx+1:]
	}

	for _, route := range config.Routes {
		params := ExtractPathParams(route.Path)
		// Create a key based on the path structure
		key := route.Path
		for _, param := range params {
			key = strings.Replace(key, "{"+param+"}", "{}", 1)
			key = strings.Replace(key, "{"+param+"...}", "{}", 1)
		}

		if groups[key] == nil {
			pascalParams := make([]string, len(params))
			for i, p := range params {
				pascalParams[i] = ToPascalCase(p)
			}

			// Build constructor args like "slug string, userId string"
			var args []string
			for _, p := range params {
				args = append(args, strings.ToLower(ToPascalCase(p))+" string")
			}

			groups[key] = &RouteGroup{
				Params:          pascalParams,
				ConstructorArgs: strings.Join(args, ", "),
				HasLocale:       localePackage != "",
				LocaleAlias:     localeAlias,
			}
		}
		pathReturn := route.Path
		for _, param := range params {
			pathReturn = strings.Replace(pathReturn, "{"+param+"}", "\" + r."+ToPascalCase(param)+" + \"", 1)
			pathReturn = strings.Replace(pathReturn, "{"+param+"...}", "\" + r."+ToPascalCase(param)+" + \"", 1)
		}
		pathReturn = strings.Replace(pathReturn, " + \"\"", "", -1)
		routeRes := BuilderRoute{
			Path:       route.Path,
			PathReturn: pathReturn,
			StructName: pathToStructName(route.Path) + "Route",
		}
		for _, method := range route.Methods {
			if len(method.QueryParams) > 0 {
				hasQueryParams = true
			}
			routeRes.Verbs = append(routeRes.Verbs, BuilderRouteVerb{
				Verb:        method.Verb,
				Handler:     method.Handler,
				QueryParams: method.QueryParams,
			})
		}
		groups[key].Routes = append(groups[key].Routes, routeRes)
	}

	// Convert map to slice
	var result []RouteGroup
	for _, group := range groups {
		result = append(result, *group)
	}

	return RouteBuilders{Groups: result, HasQueryParams: hasQueryParams}
}
