package main

import (
	"os"
	"strings"
	"testing"
)

func TestCategorizeLinks_MergeKeepsSelectors(t *testing.T) {
	links := []LinkInfo{
		{
			Text:    "发布笔记",
			URL:     "",
			Trigger: "trigger-1",
			Item:    "item-1",
		},
		{
			Text: "发布笔记",
			URL:  "",
		},
	}

	got := categorizeLinks(links)
	item, ok := got["发布笔记"]
	if !ok {
		t.Fatalf("expected key 发布笔记 to exist")
	}
	if item.Trigger != "trigger-1" {
		t.Fatalf("expected trigger selector preserved, got %q", item.Trigger)
	}
	if item.Item != "item-1" {
		t.Fatalf("expected item selector preserved, got %q", item.Item)
	}
}

func TestDiscoverPages_NoHardcodedKeywords(t *testing.T) {
	data, err := os.ReadFile("main.go")
	if err != nil {
		t.Fatalf("read main.go failed: %v", err)
	}
	content := string(data)
	keywords := []string{"发布笔记", "笔记管理", "数据看板", "内容分析", "粉丝数据"}
	for _, keyword := range keywords {
		if strings.Contains(content, keyword) {
			t.Fatalf("found hardcoded keyword %q in main.go", keyword)
		}
	}
}

func TestDetectCategory_AlwaysOther(t *testing.T) {
	link := LinkInfo{
		Text: "发布笔记",
		URL:  "https://creator.xiaohongshu.com/publish/publish",
	}
	if got := detectCategory(link); got != "other" {
		t.Fatalf("expected category other, got %q", got)
	}
}
