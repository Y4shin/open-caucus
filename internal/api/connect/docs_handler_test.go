package apiconnect

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"testing/fstest"

	connect "connectrpc.com/connect"

	docsv1 "github.com/Y4shin/conference-tool/gen/go/conference/docs/v1"
	"github.com/Y4shin/conference-tool/internal/docs"
	"github.com/Y4shin/conference-tool/internal/locale"
)

func TestDocsServiceGetPage(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	client := newCombinedTestClient(t, ts)

	resp, err := client.docs.GetPage(context.Background(), connect.NewRequest(&docsv1.GetPageRequest{
		Path: "index",
	}))
	if err != nil {
		t.Fatalf("get docs page: %v", err)
	}
	if resp.Msg.GetPage().GetPath() != "index" {
		t.Fatalf("unexpected docs path: %q", resp.Msg.GetPage().GetPath())
	}
	if resp.Msg.GetPage().GetTitle() == "" {
		t.Fatal("expected docs page title")
	}
}

func TestDocsServiceSearch(t *testing.T) {
	ts := newCombinedAPITestServer(t)
	client := newCombinedTestClient(t, ts)

	resp, err := client.docs.Search(context.Background(), connect.NewRequest(&docsv1.SearchRequest{
		Query: "login",
		Limit: 5,
	}))
	if err != nil {
		t.Fatalf("search docs: %v", err)
	}
	if len(resp.Msg.GetHits()) == 0 {
		t.Fatal("expected docs search hits")
	}
}

func TestDocsHandlerGetPage_ReturnsNotFoundForMissingDocument(t *testing.T) {
	svc, err := docs.Load(connectTestDocsFS(fstest.MapFS{}), fstest.MapFS{})
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	handler := NewDocsHandler(svc)
	_, err = handler.GetPage(context.Background(), connect.NewRequest(&docsv1.GetPageRequest{
		Path: "missing",
	}))
	if err == nil {
		t.Fatal("expected not found error")
	}
	if connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("expected not found code, got %v", connect.CodeOf(err))
	}
}

func TestDocsHandlerGetPage_UsesAcceptLanguageForLocalizedNavigation(t *testing.T) {
	svc, err := docs.Load(connectTestDocsFS(fstest.MapFS{
		"guides/index.en.md": docTestFile("Guides", "Leitfaeden", "# Guides\n\nGuide index."),
		"guides/index.de.md": docTestFile("Guides", "Leitfaeden", "# Leitfaeden\n\nUebersicht."),
		"guides/setup.en.md": docTestFile("Setup", "Einrichtung", "# Setup\n\nEnglish content."),
		"guides/setup.de.md": docTestFile("Setup", "Einrichtung", "# Einrichtung\n\nDeutscher Inhalt."),
	}), fstest.MapFS{})
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	handler := NewDocsHandler(svc)
	req := connect.NewRequest(&docsv1.GetPageRequest{Path: "guides/setup"})

	resp, err := handler.GetPage(locale.WithLocale(context.Background(), "de"), req)
	if err != nil {
		t.Fatalf("get localized page: %v", err)
	}
	page := resp.Msg.GetPage()
	if page.GetLocale() != "de" {
		t.Fatalf("expected german locale, got %q", page.GetLocale())
	}
	if page.GetTitle() != "Einrichtung" {
		t.Fatalf("expected german title, got %q", page.GetTitle())
	}
	if page.GetPathDisplay() != "Leitfaeden / Einrichtung" {
		t.Fatalf("unexpected path display: %q", page.GetPathDisplay())
	}
	if len(page.GetCrumbs()) != 2 {
		t.Fatalf("expected 2 breadcrumbs, got %d", len(page.GetCrumbs()))
	}
	if len(page.GetTree()) == 0 {
		t.Fatal("expected navigation tree")
	}
}

func TestDocsHandlerSearch_UsesAcceptLanguageAndLimit(t *testing.T) {
	svc, err := docs.Load(connectTestDocsFS(fstest.MapFS{
		"search/index.en.md":  docTestFile("Search", "Suche", "# Search\n\nSearch index."),
		"search/index.de.md":  docTestFile("Search", "Suche", "# Suche\n\nIndex."),
		"search/topic.en.md":  docTestFile("Topic", "Thema", "# Topic\n\nBlue whale reference only in english."),
		"search/topic.de.md":  docTestFile("Topic", "Thema", "# Thema\n\nBlauer Wal nur auf deutsch."),
		"search/second.en.md": docTestFile("Second", "Zweite", "# Second\n\nBluebird entry in english."),
		"search/second.de.md": docTestFile("Second", "Zweite", "# Zweite\n\nBlauer Vogel nur auf deutsch."),
	}), fstest.MapFS{})
	if err != nil {
		t.Fatalf("load docs service: %v", err)
	}
	defer svc.Close()

	handler := NewDocsHandler(svc)
	req := connect.NewRequest(&docsv1.SearchRequest{
		Query: "Blauer",
		Limit: 1,
	})

	resp, err := handler.Search(locale.WithLocale(context.Background(), "de"), req)
	if err != nil {
		t.Fatalf("search localized docs: %v", err)
	}
	if len(resp.Msg.GetHits()) != 1 {
		t.Fatalf("expected limited hit count of 1, got %d", len(resp.Msg.GetHits()))
	}
	if resp.Msg.GetHits()[0].GetLocale() != "de" {
		t.Fatalf("expected german hit locale, got %q", resp.Msg.GetHits()[0].GetLocale())
	}
}

func connectTestDocsFS(extra fstest.MapFS) fstest.MapFS {
	out := fstest.MapFS{
		"index.en.md": docTestFile("Documentation", "Dokumentation", "# Home\n\nWelcome."),
		"index.de.md": docTestFile("Documentation", "Dokumentation", "# Start\n\nWillkommen."),
	}
	for name, file := range extra {
		out[name] = file
	}
	return out
}

func docTestFile(titleEN, titleDE, body string) *fstest.MapFile {
	content := fmt.Sprintf("---\ntitle-en: %q\ntitle-de: %q\n---\n\n%s\n", titleEN, titleDE, body)
	return &fstest.MapFile{Data: []byte(strings.TrimSpace(content) + "\n")}
}
