package main

import (
	"os"
	"testing"
)

func TestLoadDiscoveredPages_AllowsTextOnlyPage(t *testing.T) {
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

	pages, _, err := loadDiscoveredPages(tmpFile.Name())
	if err != nil {
		t.Fatalf("loadDiscoveredPages failed: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("expected 1 page, got %d", len(pages))
	}
}

func TestBuildMenuClickTargets_IncludesTextFallback(t *testing.T) {
	page := DiscoveredPage{
		Text:    "发布笔记",
		Trigger: "div#menu",
		Item:    "div#menu > span",
	}

	targets := buildMenuClickTargets(page)
	if len(targets) < 2 {
		t.Fatalf("expected multiple click targets, got %d", len(targets))
	}
	if targets[len(targets)-1] != `text="发布笔记"` {
		t.Fatalf("expected text fallback at end, got %q", targets[len(targets)-1])
	}
}
