package routing

import "fmt"

type RouteParam struct {
	Slug string
	Name string
}

type RouteSSEEvent struct {
	Name         string
	TypeName     string
	TemplateFunc string
}

type RouteHandler struct {
	Name         string
	IsSSE        bool
	Params       []RouteParam
	SSEEvents    []RouteSSEEvent
	TemplateFunc string
}

type RouteHandlers struct {
	Routes []RouteHandler
}

const routeHandlers string = `
{{- range .Routes }}
{{ template "RouteHandler" . }}
{{- end }}
`

const routeHandler string = `func (rt *Router) handle{{ .Name }}(w http.ResponseWriter, r *http.Request) {
	{{- if .Params }}
	params := RouteParams{
		{{- range .Params }}
		{{ .Name }}: r.PathValue("{{ .Slug }}"),
		{{- end }}
	}
	{{ end }}
	{{- if .IsSSE }}
	eventChan, err := rt.handler.{{ .Name }}(r.Context(), r{{ if .Params }}, params{{ end }})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, \"streaming not supported\", http.StatusInternalServerError)
		return
	}

	for {
	select {
		case <-r.Context().Done():
			return
		case evt, ok := <-eventChan:
			if !ok {
				return
			}
			switch e := evt.(type) {
			{{- range .SSEEvents }}
			{{ template "RouteSSEEvent" . }}
			{{- end }}
			}	
		}
	}
	{{- else }}
	input, meta, err := rt.handler.{{ .Name }}(r.Context(), r{{ if .Params }}, params{{ end }})
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	// Set cookies and headers
	if meta != nil {
		for _, cookie := range meta.Cookies {
			http.SetCookie(w, cookie)
		}
		for key, value := range meta.Headers {
			w.Header().Set(key, value)
		}
	}

	// Handle redirect
	if meta != nil && meta.Redirect != nil {
		http.Redirect(w, r, meta.Redirect.Location, meta.Redirect.StatusCode)
		return
	}
	
	{{ .TemplateFunc }}(*input).Render(r.Context(), w)
	{{- end }}
}`

const routeSSEEvent string = `
			case {{ .TypeName }}:
				var buf bytes.Buffer
				{{ .TemplateFunc }}(e.Data).Render(r.Context(), &buf)
				fmt.Fprintf(w, "event: {{ .Name }}\ndata: %s\n\n", buf.String())
				flusher.Flush()
`

func GetRouteHandlers(config *RouteConfig) RouteHandlers {
	extraImports := collectImports(config)
	res := RouteHandlers{}
	for _, route := range config.Routes {
		for _, method := range route.Methods {
			routeRes := RouteHandler{
				Name:  method.Handler,
				IsSSE: method.SSE,
			}
			for _, param := range ExtractPathParams(route.Path) {
				routeRes.Params = append(routeRes.Params, RouteParam{Name: ToPascalCase(param), Slug: param})
			}

			if method.SSE {
				for _, event := range method.Events {
					eventTypeName := fmt.Sprintf("%sEvent", event.Name)
					templateAlias := getTemplateAlias(event.Template.Package, extraImports)
					templateFunc := fmt.Sprintf("%s.%s", templateAlias, event.Template.Type)

					// Convert event name to kebab-case for SSE event type
					eventName := toKebabCase(event.Name)
					eventRes := RouteSSEEvent{
						Name:         eventName,
						TypeName:     eventTypeName,
						TemplateFunc: templateFunc,
					}
					routeRes.SSEEvents = append(routeRes.SSEEvents, eventRes)
				}
			} else {
				templateAlias := getTemplateAlias(method.Template.Package, extraImports)
				routeRes.TemplateFunc = fmt.Sprintf("%s.%s", templateAlias, method.Template.Type)
			}
			res.Routes = append(res.Routes, routeRes)
		}
	}

	return res
}
