package templrender

import (
	"strings"
	"testing"

	"github.com/Y4shin/conference-tool/internal/locale"
	"github.com/Y4shin/conference-tool/internal/templates"
)

func TestRenderKnownTemplate(t *testing.T) {
	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}

	out, err := Render("LoginPageTemplate", ContextProfile{}, &templates.LoginPageInput{
		PasswordEnabled: true,
	})
	if err != nil {
		t.Fatalf("render login page: %v", err)
	}

	html := string(out)
	if !strings.Contains(html, "<fieldset") {
		t.Fatalf("expected fieldset in rendered output, got: %s", html)
	}
	if !strings.Contains(html, "Conference-Tool") {
		t.Fatalf("expected app shell in rendered output, got: %s", html)
	}
}

func TestRenderUnknownTemplateFails(t *testing.T) {
	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}

	_, err := Render("DoesNotExist", ContextProfile{}, struct{}{})
	if err == nil || !strings.Contains(err.Error(), "unknown component") {
		t.Fatalf("expected unknown component error, got %v", err)
	}
}
