package routing

import (
	"fmt"
	"sort"
	"strings"
)

type RegisterRoute struct {
	Verb        string
	Path        string
	HandlerFunc string
	Middleware  []string
	IsSSE       bool
}

type RegisterRoutes struct {
	MiddlewareGroups []MiddlewareGroup
	Routes           []RegisterRoute
}

const registerRoutes string = `// RegisterRoutes installs all routes into the HTTP server
func (rt *Router) RegisterRoutes() http.Handler {
	// Define middleware groups from YAML
	{{- if .MiddlewareGroups }}
	groups := []middlewareGroup{
		{{- range .MiddlewareGroups}}
			{{ template "MiddlewareGroup" . }},
		{{- end }}
	}
	{{- else }}
	groups := []middlewareGroup{}
	{{- end }}
	{{- if not .Routes }}
	
	_ = groups //suppress unused variable warning when no routes defined
	{{- end }}

	{{ range .Routes }}
	{{ template "RegisterRoute" . }}
	{{- end }}

	return rt.mux
}`

const middlewareGroup string = `{prefix: "{{ .Prefix }}", middleware: []string{
{{- range $i, $v := .Middleware}}{{ if $i }}, {{ end }}"{{ $v }}"{{ end -}}
} }`

const registerRoute string = `
	rt.mux.HandleFunc("{{ .Verb }} {{ .Path }}", rt.wrapMiddleware(
		{{ .HandlerFunc }},
		getMiddlewareForPath("{{ .Path }}", groups),
		[]string{ {{ range $i, $v := .Middleware}}{{ if $i }}, {{ end }}"{{ $v }}"{{ end }} },
		{{ .IsSSE }},
	))
`

func GetRegisterRoutes(config *RouteConfig) RegisterRoutes {

	sortedGroups := make([]MiddlewareGroup, len(config.MiddlewareGroups))
	copy(sortedGroups, config.MiddlewareGroups)
	sort.Slice(sortedGroups, func(i, j int) bool {
		return len(sortedGroups[i].Prefix) > len(sortedGroups[j].Prefix)
	})
	res := RegisterRoutes{
		MiddlewareGroups: sortedGroups,
	}
	for _, route := range config.Routes {
		for _, method := range route.Methods {
			route := RegisterRoute{
				Verb:        strings.ToUpper(method.Verb),
				Path:        route.Path,
				HandlerFunc: fmt.Sprintf("rt.handle%s", method.Handler),
				Middleware:  method.Middleware,
				IsSSE:       method.SSE,
			}

			res.Routes = append(res.Routes, route)

		}
	}

	return res
}
