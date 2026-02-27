package scripts

import (
	"context"
	"fmt"
	"strings"

	"github.com/Y4shin/conference-tool/tools/docscapture"
)

func exampleScreenshotAdminLoginScript() Script {
	return Script{
		Name:        "example.screenshot-admin-login",
		Description: "Still screenshot of /admin/login",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			if _, err := env.Seeder().CreateAdminAccount(ctx, "demo-admin", "demo-password", "Demo Admin"); err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()
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
		Description: "Animated GIF of successful admin login with seeded credentials",
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			if _, err := env.Seeder().CreateAdminAccount(ctx, "demo-admin", "demo-password", "Demo Admin"); err != nil {
				return "", err
			}
			common.BaseURL = env.BaseURL()

			scaleWidth := 960
			submitInput := ""
			if variant.Device == docscapture.DeviceMobile {
				scaleWidth = 420
				// Example script branching point: mobile flows can use a dedicated submit button.
				submitInput = "button[type=submit]"
			}

			result, err := docscapture.RunGIFExample(docscapture.GIFExampleOptions{
				Common:        common,
				Variant:       variant,
				StartPath:     "/",
				LoginPath:     "/admin/login",
				Username:      "demo-admin",
				Password:      "demo-password",
				SubmitInput:   submitInput,
				Submit:        true,
				WaitForPath:   "/admin",
				OutputName:    fmt.Sprintf("example-login-flow.%s.gif", variantSuffix(variant)),
				FPS:           8,
				ScaleWidth:    scaleWidth,
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

func exampleScreenshotAdminLoginOAuthScript() Script {
	return Script{
		Name:        "example.screenshot-admin-login-oauth",
		Description: "Still screenshot of /admin/login with OAuth provider enabled in self-hosted env",
		Environment: EnvironmentOptions{
			EnablePassword:        true,
			EnableOAuth:           true,
			OAuthProvisioningMode: "auto_create",
			OAuthGroupsClaim:      "groups",
		},
		Run: func(ctx context.Context, common docscapture.CommonOptions, variant docscapture.Variant, env *docscapture.Environment) (string, error) {
			common.BaseURL = env.BaseURL()
			return docscapture.RunScreenshotExample(docscapture.ScreenshotExampleOptions{
				Common:     common,
				Variant:    variant,
				Path:       "/admin/login",
				OutputName: fmt.Sprintf("example-admin-login-oauth.%s.png", variantSuffix(variant)),
				FullPage:   true,
			})
		},
	}
}

func variantSuffix(variant docscapture.Variant) string {
	parts := []string{string(variant.Language), string(variant.Theme), string(variant.Device)}
	return strings.Join(parts, ".")
}
