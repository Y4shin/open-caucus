package routing

import "sort"

type RouteParams struct {
	Params []string
}

const routeParams string = `
// RouteParams contains extracted parameters from the URL path
type RouteParams struct {
{{- range .Params }}
	{{ . }} string
{{- end }}
}
`

func GetRouteParams(config *RouteConfig) RouteParams {
	// Collect all unique path parameters
	paramSet := make(map[string]bool)
	for _, route := range config.Routes {
		params := ExtractPathParams(route.Path)
		for _, param := range params {
			paramSet[param] = true
		}
	}

	if len(paramSet) == 0 {
		// No parameters, skip struct generation
		return RouteParams{Params: []string{}}
	}

	// Sort for deterministic output
	var params []string
	for param := range paramSet {
		params = append(params, ToPascalCase(param))
	}
	sort.Strings(params)
	return RouteParams{Params: params}
}
