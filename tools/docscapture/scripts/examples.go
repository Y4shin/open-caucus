package scripts

import (
	"fmt"
	"strings"

	"github.com/Y4shin/conference-tool/tools/docscapture"
)

func exampleScreenshotAdminLoginScript() Script {
	return Script{
		Name:        "example.screenshot-admin-login",
		Description: "Still screenshot of /admin/login",
		Run: func(common docscapture.CommonOptions, variant docscapture.Variant) (string, error) {
			return docscapture.RunScreenshotExample(docscapture.ScreenshotExampleOptions{
				Common:     common,
				Variant:    variant,
				Path:       "/admin/login",
				OutputName: fmt.Sprintf("example-admin-login.%s.png", variantSuffix(variant)),
				FullPage:   true,
			})
		},
	}
}

func exampleGIFLoginFlowScript() Script {
	return Script{
		Name:        "example.gif-login-flow",
		Description: "Animated GIF of the login form typing flow",
		Run: func(common docscapture.CommonOptions, variant docscapture.Variant) (string, error) {
			result, err := docscapture.RunGIFExample(docscapture.GIFExampleOptions{
				Common:        common,
				Variant:       variant,
				StartPath:     "/",
				LoginPath:     "/admin/login",
				OutputName:    fmt.Sprintf("example-login-flow.%s.gif", variantSuffix(variant)),
				FPS:           8,
				ScaleWidth:    960,
				TypingDelayMS: 90,
				FinalPauseMS:  700,
			})
			if err != nil {
				return "", err
			}
			return result.GIFPath, nil
		},
	}
}

func variantSuffix(variant docscapture.Variant) string {
	parts := []string{string(variant.Language), string(variant.Theme)}
	return strings.Join(parts, ".")
}
