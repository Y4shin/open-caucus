package routing

import (
	"strings"
	"testing"
)

func TestExtractPathParamsCatchAll(t *testing.T) {
	params := ExtractPathParams("/docs/{doc_path...}/asset/{asset_path...}")
	if len(params) != 2 {
		t.Fatalf("expected 2 params, got %d (%v)", len(params), params)
	}
	if params[0] != "doc_path" {
		t.Fatalf("expected first param doc_path, got %q", params[0])
	}
	if params[1] != "asset_path" {
		t.Fatalf("expected second param asset_path, got %q", params[1])
	}
}

func TestGenerateCatchAllRoute(t *testing.T) {
	cfg := &RouteConfig{
		Version: "1.0",
		Routes: []Route{
			{
				Path: "/docs/{doc_path...}",
				Methods: []RouteMethod{
					{
						Verb:    "GET",
						Handler: "DocsPage",
						Template: Template{
							Package:   "github.com/Y4shin/conference-tool/internal/templates",
							Type:      "DocsElement",
							InputType: "DocsElementInput",
						},
					},
				},
			},
		},
	}

	generated, err := Generate(cfg, "routes", nil)
	if err != nil {
		t.Fatalf("generate routes: %v", err)
	}
	if !strings.Contains(generated, `DocPath string`) {
		t.Fatalf("generated route params missing DocPath: %s", generated)
	}
	if !strings.Contains(generated, `r.PathValue("doc_path")`) {
		t.Fatalf("generated handler should extract catch-all path value by normalized name: %s", generated)
	}
}

func TestGeneratePathsCatchAllRoute(t *testing.T) {
	cfg := &RouteConfig{
		Version: "1.0",
		Routes: []Route{
			{
				Path:    "/docs/{doc_path...}",
				Methods: []RouteMethod{{Verb: "GET", Handler: "DocsPage", Template: Template{Package: "github.com/Y4shin/conference-tool/internal/templates", Type: "DocsElement", InputType: "DocsElementInput"}}},
			},
		},
	}

	generated, err := GeneratePaths(cfg, "paths", nil, "github.com/Y4shin/conference-tool/internal/locale")
	if err != nil {
		t.Fatalf("generate paths: %v", err)
	}
	if !strings.Contains(generated, `type DocsDocPathRoute struct`) {
		t.Fatalf("generated builder missing catch-all route struct: %s", generated)
	}
	if !strings.Contains(generated, `DocPath string`) {
		t.Fatalf("generated builder missing normalized catch-all field: %s", generated)
	}
}
