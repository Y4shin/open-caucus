package routing

type ImportSpec struct {
	Package string
	Alias   string
}

type Imports struct {
	Imports []ImportSpec
}

const imports string = `{{ if .Imports -}}
import (
{{- range .Imports }}
	{{ if .Alias }}{{ .Alias }} {{end}}"{{ .Package }}"
{{- end }}
)
{{- end }}`

func GetImports(config *RouteConfig) Imports {
	// Collect all unique template extraImports
	extraImports := collectImports(config)

	// Check if we have any SSE routes (they need bytes and fmt)
	hasSSE := false
	for _, route := range config.Routes {
		for _, method := range route.Methods {
			if method.SSE {
				hasSSE = true
				break
			}
		}
		if hasSSE {
			break
		}
	}

	imports := []ImportSpec{}
	if hasSSE {
		imports = append(imports, ImportSpec{Package: "bytes"})
		imports = append(imports, ImportSpec{Package: "fmt"})
	}
	if len(config.Routes) > 0 {
		imports = append(imports, ImportSpec{Package: "context"})
		imports = append(imports, ImportSpec{Package: "github.com/a-h/templ"})
	}
	imports = append(imports, ImportSpec{Package: "net/http"})
	imports = append(imports, ImportSpec{Package: "strings"})

	for alias, pkg := range extraImports {
		imports = append(imports, ImportSpec{Package: pkg, Alias: alias})
	}
	return Imports{Imports: imports}
}
