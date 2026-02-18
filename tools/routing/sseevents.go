package routing

import "fmt"

type SSEEventKind struct {
	EventName          string
	EventTypeName      string
	InputType          string
	EventInterfaceName string
}

type SSEEventType struct {
	HandlerName        string
	EventInterfaceName string
	EventKinds         []SSEEventKind
}

type SSEEventTypes struct {
	EventTypes []SSEEventType
}

const sseEventTypes string = `
{{- range .EventTypes}}
{{ template "SSEEventType" .}}
{{ end -}}
`

const sseEventType string = `// {{ .EventInterfaceName }} is the union type for all {{ .HandlerName }} SSE events
type {{ .EventInterfaceName }} interface {
	Is{{ .EventInterfaceName }}()
}

{{ range .EventKinds -}}
{{ template "SSEEventKind" .}}


{{ end -}}
`

const sseEventKind string = `// {{ .EventTypeName }} represents a {{ .EventName }} event
type {{ .EventTypeName }} struct {
	Data {{ .InputType }}
}

func (e {{ .EventTypeName }}) Is{{ .EventInterfaceName }}() {}
`

func GetSSEEventTypes(config *RouteConfig) SSEEventTypes {
	generatedHandlers := make(map[string]bool)
	extraImports := collectImports(config)
	res := SSEEventTypes{}
	for _, route := range config.Routes {
		for _, method := range route.Methods {
			if !method.SSE || len(method.Events) == 0 {
				continue
			}

			// Skip if we've already generated for this handler
			if generatedHandlers[method.Handler] {
				continue
			}
			generatedHandlers[method.Handler] = true

			eventInterfaceName := fmt.Sprintf("%sEvent", method.Handler)
			eventType := SSEEventType{
				HandlerName:        method.Handler,
				EventInterfaceName: eventInterfaceName,
			}

			for _, event := range method.Events {
				eventTypeName := fmt.Sprintf("%sEvent", event.Name)
				templateAlias := getTemplateAlias(event.Template.Package, extraImports)
				inputType := fmt.Sprintf("%s.%s", templateAlias, event.Template.InputType)

				eventKind := SSEEventKind{
					EventName:          event.Name,
					EventTypeName:      eventTypeName,
					InputType:          inputType,
					EventInterfaceName: eventInterfaceName,
				}

				eventType.EventKinds = append(eventType.EventKinds, eventKind)

			}
			res.EventTypes = append(res.EventTypes, eventType)
		}
	}
	return res
}
