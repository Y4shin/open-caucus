package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Y4shin/conference-tool/internal/locale"
)

func executeRootForTest(t *testing.T, args []string) string {
	t.Helper()
	output := &bytes.Buffer{}
	rootCmd.SetOut(output)
	rootCmd.SetErr(output)
	rootCmd.SetArgs(args)
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("execute root command %v: %v\noutput:\n%s", args, err, output.String())
	}
	return output.String()
}

func TestDecodeRenderTemplatePayloadMalformedJSON(t *testing.T) {
	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}
	_, _, err := decodeRenderTemplatePayload("LoginPageTemplate", strings.NewReader("{"))
	if err == nil || !strings.Contains(err.Error(), "decode render payload") {
		t.Fatalf("expected malformed payload error, got %v", err)
	}
}

func TestRenderTemplateCommandReadsStdin(t *testing.T) {
	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}

	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()

	tmpFile, err := os.CreateTemp(t.TempDir(), "render-stdin-*.json")
	if err != nil {
		t.Fatalf("create temp stdin file: %v", err)
	}
	defer tmpFile.Close()
	if _, err := tmpFile.WriteString(`{"input":{"passwordEnabled":true}}`); err != nil {
		t.Fatalf("write temp stdin file: %v", err)
	}
	if _, err := tmpFile.Seek(0, 0); err != nil {
		t.Fatalf("rewind temp stdin file: %v", err)
	}
	os.Stdin = tmpFile

	output := executeRootForTest(t, []string{"render-template", "LoginPageTemplate"})
	if !strings.Contains(output, "<fieldset") {
		t.Fatalf("expected rendered html, got %s", output)
	}
}

func TestRenderTemplateCommandReadsInputFile(t *testing.T) {
	if err := locale.LoadTranslations(); err != nil {
		t.Fatalf("load translations: %v", err)
	}

	inputPath := filepath.Join(t.TempDir(), "input.json")
	if err := os.WriteFile(inputPath, []byte(`{"input":{"passwordEnabled":true}}`), 0o600); err != nil {
		t.Fatalf("write input file: %v", err)
	}

	output := executeRootForTest(t, []string{"render-template", "LoginPageTemplate", "--input", inputPath})
	if !strings.Contains(output, "<fieldset") {
		t.Fatalf("expected rendered html, got %s", output)
	}
}
