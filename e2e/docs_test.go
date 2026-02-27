//go:build e2e

package e2e_test

import (
	"strings"
	"testing"
)

func TestDocsElementAndOOBRoute(t *testing.T) {
	ts := newTestServer(t)
	page := newPage(t)
	adminLogin(t, page, ts.URL)

	if _, err := page.Goto(ts.URL + "/docs/index"); err != nil {
		t.Fatalf("goto docs index: %v", err)
	}
	content, err := page.Content()
	if err != nil {
		t.Fatalf("get docs page content: %v", err)
	}
	if !strings.Contains(content, "id=\"app-docs-target\"") {
		t.Fatalf("expected docs element payload, url=%s content=%s", page.URL(), content)
	}

	if _, err := page.Goto(ts.URL + "/docs/oob/index"); err != nil {
		t.Fatalf("goto docs oob index: %v", err)
	}
	content, err = page.Content()
	if err != nil {
		t.Fatalf("get docs oob content: %v", err)
	}
	if !strings.Contains(content, "hx-swap-oob=\"outerHTML\"") {
		t.Fatalf("expected oob docs payload to include hx-swap-oob, got: %s", content)
	}
}
