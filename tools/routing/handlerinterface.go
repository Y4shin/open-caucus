package routing

import (
	"fmt"
)

type HandlerInterfaceMethod struct {
	IsSSE              bool
	IsRaw              bool
	Name               string
	HasParams          bool
	EventInterfaceName string
	InputType          string
}

type HandlerInterface struct {
	Methods []HandlerInterfaceMethod
}

const handlerInterface string = `// Handler interface with all route methods
type Handler interface {
{{- range .Methods }}
	{{ template "HandlerInterfaceMethod" . }}
{{- end }}
}`

const handlerInterfaceMethod string = `{{ .Name }}(
{{- if .IsSSE -}}
ctx context.Context, r *http.Request{{ if .HasParams }}, params RouteParams{{ end }}) (<-chan {{ .EventInterfaceName }}, error)
{{- else if .IsRaw -}}
w http.ResponseWriter, r *http.Request{{ if .HasParams }}, params RouteParams{{ end }}) error
{{- else -}}
ctx context.Context, r *http.Request{{ if .HasParams }}, params RouteParams{{ end }}) (*{{ .InputType }}, *ResponseMeta, error)
{{- end -}}
`

func GetHandlerInterface(config *RouteConfig) HandlerInterface {
	extraImports := collectImports(config)
	res := HandlerInterface{}

	for _, route := range config.Routes {
		for _, method := range route.Methods {
			methodRes := HandlerInterfaceMethod{
				HasParams: len(ExtractPathParams(route.Path)) > 0,
				IsSSE:     method.SSE,
				IsRaw:     method.Raw,
				Name:      method.Handler,
			}

			if method.SSE {
				methodRes.EventInterfaceName = getEventInterfaceName(&method)
			} else if !method.Raw {
				templateAlias := getTemplateAlias(method.Template.Package, extraImports)
				methodRes.InputType = fmt.Sprintf("%s.%s", templateAlias, method.Template.InputType)
			}
			res.Methods = append(res.Methods, methodRes)
		}
	}

	return res
}
