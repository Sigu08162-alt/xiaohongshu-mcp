package main

import (
	"os"
	"testing"
)

func TestLoadPagesFromFile_DoesNotSkipCategories(t *testing.T) {
	content := `version: 1.0.0
generated: "2026-02-02T00:00:00+08:00"
home_page: https://example.com
links:
  测试页面:
    text: 测试页面
    url: https://example.com/page
    description: 测试页面
    category: other
`

	tmpFile, err := os.CreateTemp("", "discovered_pages_*.yaml")
	if err != nil {
		t.Fatalf("create temp file failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("close temp file failed: %v", err)
	}

	pages, _, err := loadPagesFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("loadPagesFromFile failed: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
}

func TestLoadPagesFromFile_AllowsTextOnlyPage(t *testing.T) {
	content := `version: 1.0.0
generated: "2026-02-02T00:00:00+08:00"
home_page: https://example.com
links:
  发布笔记:
    text: 发布笔记
    url: ""
`

	tmpFile, err := os.CreateTemp("", "discovered_pages_text_only_*.yaml")
	if err != nil {
		t.Fatalf("create temp file failed: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatalf("write temp file failed: %v", err)
	}
	if err := tmpFile.Close(); err != nil {
		t.Fatalf("close temp file failed: %v", err)
	}

	pages, _, err := loadPagesFromFile(tmpFile.Name())
	if err != nil {
		t.Fatalf("loadPagesFromFile failed: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
	if pages[0].Name != "发布笔记" {
		t.Fatalf("expected page name 发布笔记, got %q", pages[0].Name)
	}
}

func TestBuildMenuClickTargets_IncludesTextFallback(t *testing.T) {
	page := PageDefinition{
		Name:            "发布笔记",
		Text:            "发布笔记",
		TriggerSelector: "div#menu",
		ItemSelector:    "div#menu > span",
	}

	targets := buildMenuClickTargets(page)
	if len(targets) < 2 {
		t.Fatalf("expected multiple click targets, got %d", len(targets))
	}
	if targets[len(targets)-1] != `text="发布笔记"` {
		t.Fatalf("expected text fallback at end, got %q", targets[len(targets)-1])
	}
}
