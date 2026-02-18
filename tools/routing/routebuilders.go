package routing

import "strings"

type BuilderRouteVerb struct {
	Verb    string
	Handler string
}

type BuilderRoute struct {
	Path       string
	PathReturn string
	StructName string
	Verbs      []BuilderRouteVerb
}

type RouteGroup struct {
	Params []string
	Routes []BuilderRoute
}

type RouteBuilders struct {
	Groups []RouteGroup
}

const routeBuilders string = `// Type-safe route builders
type Routes struct{}

var Route Routes

{{ range .Groups }}
{{ template "RouteGroup" . }}
{{ end -}}
`

const routeGroup string = `
{{- if not .Params }}
{{- range .Routes }}
{{- $path := .Path }}
{{- range .Verbs }}
func (Routes) {{ .Handler }}{{ .Verb | lower | capitalize }}() string {
	return "{{ $path }}"
}
{{- end }}
{{- end }}
{{- else }}
{{- $params := .Params }}
{{- range .Routes }}
type {{ .StructName }} struct {
	{{- range $params }}
	{{ . }} string
	{{- end }}
}
{{- $structName := .StructName }}
{{- $pathReturn := .PathReturn }}
{{- range .Verbs }}
func (r *{{ $structName }}) {{ .Handler }}{{ .Verb | lower | capitalize }}() string {
	return "{{ $pathReturn }}"
}

func (r *{{ $structName }}) {{ .Handler }}{{ .Verb | lower | capitalize }}URL() templ.SafeURL {
	return templ.URL(r.{{ .Handler }}{{ .Verb | lower | capitalize }}())
}
{{- end }}
{{- end }}
{{- end }}
`

func GetRouteBuilders(config *RouteConfig) RouteBuilders {
	groups := make(map[string]*RouteGroup)

	for _, route := range config.Routes {
		params := ExtractPathParams(route.Path)
		// Create a key based on the path structure
		key := route.Path
		for _, param := range params {
			key = strings.Replace(key, "{"+param+"}", "{}", 1)
		}

		if groups[key] == nil {
			groups[key] = &RouteGroup{
				Params: params,
			}
		}
		pathReturn := route.Path
		for _, param := range params {
			pathReturn = strings.Replace(pathReturn, "{"+param+"}", "\" + r."+param+" + \"", 1)
		}
		pathReturn = strings.Replace(pathReturn, " + \"\"", "", -1)
		routeRes := BuilderRoute{
			Path:       route.Path,
			PathReturn: pathReturn,
			StructName: pathToStructName(route.Path) + "Route",
		}
		for _, method := range route.Methods {
			routeRes.Verbs = append(routeRes.Verbs, BuilderRouteVerb{
				Verb:    method.Verb,
				Handler: method.Handler,
			})
		}
		groups[key].Routes = append(groups[key].Routes, routeRes)
	}

	// Convert map to slice
	var result []RouteGroup
	for _, group := range groups {
		result = append(result, *group)
	}

	return RouteBuilders{Groups: result}
}
