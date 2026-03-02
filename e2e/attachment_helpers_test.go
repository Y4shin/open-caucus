//go:build e2e

package e2e_test

import (
	"fmt"
	"os"
	"testing"
)

func agendaPointToolsURL(baseURL, slug, meetingID, agendaPointID string) string {
	return fmt.Sprintf("%s/committee/%s/meeting/%s/agenda-point/%s/tools", baseURL, slug, meetingID, agendaPointID)
}

func writeTempPDF(t *testing.T) string {
	t.Helper()

	file, err := os.CreateTemp(t.TempDir(), "upload-*.pdf")
	if err != nil {
		t.Fatalf("create temp pdf: %v", err)
	}
	defer file.Close()

	const minimalPDF = "%PDF-1.4\n1 0 obj<</Type/Catalog/Pages 2 0 R>>endobj\n2 0 obj<</Type/Pages/Count 1/Kids[3 0 R]>>endobj\n3 0 obj<</Type/Page/Parent 2 0 R/MediaBox[0 0 200 200]/Contents 4 0 R>>endobj\n4 0 obj<</Length 37>>stream\nBT /F1 12 Tf 20 100 Td (E2E PDF) Tj ET\nendstream\nendobj\ntrailer<</Root 1 0 R>>\n%%EOF\n"
	if _, err := file.WriteString(minimalPDF); err != nil {
		t.Fatalf("write temp pdf: %v", err)
	}

	return file.Name()
}

func mustReadFile(t *testing.T, path string) []byte {
	t.Helper()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read file %q: %v", path, err)
	}
	return data
}
