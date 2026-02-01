package main

import (
	"os"
	"testing"

	infraconfig "github.com/xpzouying/xiaohongshu-mcp/internal/infra/config"
)

func TestBuildPublishUsecase_ValidConfig(t *testing.T) {
	cfg := &infraconfig.Config{}
	cfg.URLs.Creator.PublishImage = "https://example.com/publish?target=image"
	cfg.URLs.Creator.PublishVideo = "https://example.com/publish?target=video"
	cfg.Limits.MaxTags = 10
	cfg.Limits.MinImages = 1
	cfg.Limits.MaxImages = 9
	selectors := map[string]string{
		"upload_input": "input[type=file]",
		"title_input":  "input[name=title]",
		"content":      "div.editor",
		"submit":       "button[type=submit]",
	}

	uc, err := buildPublishUsecase(cfg, selectors, true)
	if err != nil || uc == nil {
		t.Fatalf("expected usecase, err=%v", err)
	}
}

func TestLoadPublishSelectors(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	data := []byte("selectors:\n  publish:\n    upload_input: \"input[type=file]\"\n    title_input: \"input[name=title]\"\n    content_editor_ql: \"div.editor\"\n    submit_button: \"button[type=submit]\"\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	selectors, err := loadPublishSelectors(path)
	if err != nil {
		t.Fatalf("load selectors err: %v", err)
	}
	if selectors["upload_input"] == "" || selectors["submit"] == "" {
		t.Fatalf("expected selectors to be loaded")
	}
}

func TestLoadPublishUsecase(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	data := []byte("urls:\n  creator:\n    publish_image: \"https://example.com/publish?target=image\"\n    publish_video: \"https://example.com/publish?target=video\"\nlimits:\n  max_tags: 10\n  min_images: 1\n  max_images: 9\nselectors:\n  publish:\n    upload_input: \"input[type=file]\"\n    title_input: \"input[name=title]\"\n    content_editor_ql: \"div.editor\"\n    submit_button: \"button[type=submit]\"\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("XHS_CONFIG_PATH", path)
	uc, err := loadPublishUsecase(true)
	if err != nil || uc == nil {
		t.Fatalf("expected usecase, err=%v", err)
	}
}

func TestInitPublishUsecase_ReturnsUsecase(t *testing.T) {
	path := t.TempDir() + "/config.yaml"
	data := []byte("urls:\n  creator:\n    publish_image: \"https://example.com/publish?target=image\"\n    publish_video: \"https://example.com/publish?target=video\"\nlimits:\n  max_tags: 10\n  min_images: 1\n  max_images: 9\nselectors:\n  publish:\n    upload_input: \"input[type=file]\"\n    title_input: \"input[name=title]\"\n    content_editor_ql: \"div.editor\"\n    submit_button: \"button[type=submit]\"\n")
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	t.Setenv("XHS_CONFIG_PATH", path)
	uc := initPublishUsecase(true)
	if uc == nil {
		t.Fatalf("expected usecase")
	}
}

func TestExtractPublishSelectorsFromCollected_FindsPublishImagePage(t *testing.T) {
	collected := &collectedSelectorsConfig{
		Pages: map[string]struct {
			PageName string `yaml:"page_name"`
			URL      string `yaml:"url"`
			Buttons  []struct {
				Text     string   `yaml:"text"`
				Selector string   `yaml:"selector"`
				Classes  []string `yaml:"classes"`
			} `yaml:"buttons"`
			Inputs []struct {
				Text        string   `yaml:"text"`
				Selector    string   `yaml:"selector"`
				Placeholder string   `yaml:"placeholder"`
				Type        string   `yaml:"type"`
				TagName     string   `yaml:"tag_name"`
				Classes     []string `yaml:"classes"`
			} `yaml:"inputs"`
		}{
			"publish_image": {
				PageName: "发布图文",
				URL:      "https://creator.xiaohongshu.com/publish/publish?source=official&target=image",
				Buttons: []struct {
					Text     string   `yaml:"text"`
					Selector string   `yaml:"selector"`
					Classes  []string `yaml:"classes"`
				}{
					{Text: "发布", Selector: "button.d-button.publishBtn", Classes: []string{"d-button", "publishBtn"}},
				},
				Inputs: []struct {
					Text        string   `yaml:"text"`
					Selector    string   `yaml:"selector"`
					Placeholder string   `yaml:"placeholder"`
					Type        string   `yaml:"type"`
					TagName     string   `yaml:"tag_name"`
					Classes     []string `yaml:"classes"`
				}{
					{Selector: "input", Type: "file", TagName: "input"},
					{Selector: "input[placeholder*=\"标题\"]", Placeholder: "标题", TagName: "input"},
					{Selector: "div.tiptap.ProseMirror", TagName: "div", Classes: []string{"tiptap", "ProseMirror"}},
				},
			},
		},
	}

	selectors, err := extractPublishSelectorsFromCollected(collected)
	if err != nil {
		t.Fatalf("expected selectors, got err: %v", err)
	}
	if selectors["upload_input"] != "input[type=\"file\"]" {
		t.Fatalf("expected upload_input to be input[type=\"file\"], got: %s", selectors["upload_input"])
	}
	if selectors["upload_input"] == "" || selectors["title_input"] == "" || selectors["content"] == "" || selectors["submit"] == "" {
		t.Fatalf("expected publish selectors to be extracted, got: %+v", selectors)
	}
}

func TestLoadPublishUsecase_FallbackFromInvalidCollected(t *testing.T) {
	tmp := t.TempDir()
	origWD, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(tmp); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Chdir(origWD)
	})

	configPath := tmp + "/config.yaml"
	configData := []byte("urls:\n  creator:\n    publish_image: \"https://example.com/publish?target=image\"\n    publish_video: \"https://example.com/publish?target=video\"\nlimits:\n  max_tags: 10\n  min_images: 1\n  max_images: 9\npolling:\n  publish:\n    timeout_ms: 60000\n    interval_ms: 500\n    max_retries: 3\n    delays:\n      page_stable_ms: 3000\n      pre_submit_render_ms: 2000\n      post_content_render_ms: 2000\n      scroll_into_view_wait_ms: 500\n      click_retry_wait_ms: 2000\n      tag_editor_ready_ms: 1000\n      tag_arrow_step_ms: 10\n      tag_after_enter_ms: 1000\n      tag_hash_delay_ms: 200\n      tag_char_delay_ms: 50\n      tag_after_text_ms: 1000\n      tag_suggestion_click_ms: 200\n      tag_after_tag_ms: 500\n  draft:\n    timeout_ms: 60000\n    interval_ms: 500\n    max_retries: 3\n    delays:\n      page_stable_ms: 3000\n      post_content_render_ms: 2000\n      scroll_into_view_wait_ms: 500\n      click_retry_wait_ms: 2000\n      tag_editor_ready_ms: 1000\n      tag_arrow_step_ms: 10\n      tag_after_enter_ms: 1000\n      tag_hash_delay_ms: 200\n      tag_char_delay_ms: 50\n      tag_after_text_ms: 1000\n      tag_suggestion_click_ms: 200\n      tag_after_tag_ms: 500\n  video:\n    timeout_ms: 90000\n    interval_ms: 500\n    max_retries: 3\n    delays:\n      wait_300000ms: 300000\n      wait_3000ms: 3000\n      wait_1000ms: 1000\n      post_content_render_ms: 2000\n      scroll_into_view_wait_ms: 500\n      draft_save_wait_ms: 3000\n  interaction:\n    timeout_ms: 30000\n    interval_ms: 500\n    max_retries: 3\n    delays:\n      wait_600000ms: 600000\n      wait_60000ms: 60000\n      wait_300000ms: 300000\n      wait_10000ms: 10000\n      wait_800ms: 800\n      wait_5000ms: 5000\n      wait_3000ms: 3000\n      wait_2000ms: 2000\n      wait_1000ms: 1000\n      wait_500ms: 500\n      wait_300ms: 300\n      wait_200ms: 200\n      wait_100ms: 100\n      human_delay_min_ms: 300\n      human_delay_max_ms: 700\n      reaction_time_min_ms: 300\n      reaction_time_max_ms: 800\n      hover_time_min_ms: 100\n      hover_time_max_ms: 300\n      read_time_min_ms: 500\n      read_time_max_ms: 1200\n      short_read_min_ms: 600\n      short_read_max_ms: 1200\n      scroll_wait_min_ms: 100\n      scroll_wait_max_ms: 200\n      post_scroll_min_ms: 300\n      post_scroll_max_ms: 500\n      scroll_slow_min_ms: 1200\n      scroll_slow_max_ms: 1500\n      scroll_normal_min_ms: 600\n      scroll_normal_max_ms: 800\n      scroll_fast_min_ms: 300\n      scroll_fast_max_ms: 400\n  analytics:\n    timeout_ms: 30000\n    interval_ms: 500\n    max_retries: 3\n    delays:\n      wait_300000ms: 300000\n      wait_60000ms: 60000\n      wait_5000ms: 5000\n      wait_2000ms: 2000\n      wait_1000ms: 1000\n      wait_500ms: 500\n  auth:\n    timeout_ms: 60000\n    interval_ms: 500\n    max_retries: 3\n    delays:\n      wait_2000ms: 2000\n      wait_1000ms: 1000\nselectors:\n  publish:\n    upload_input: \"input[type=file]\"\n    title_input: \"input[name=title]\"\n    content_editor_ql: \"div.editor\"\n    submit_button: \"button[type=submit]\"\n")
	if err := os.WriteFile(configPath, configData, 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	invalidCollected := []byte("version: 1.0.0\npages:\n  analytics:\n    page_name: 内容分析\n    url: https://creator.xiaohongshu.com/statistics/data-analysis?source=official\n    buttons: []\n    inputs: []\n")
	if err := os.WriteFile("selectors_discovered_pages_creator.yaml", invalidCollected, 0644); err != nil {
		t.Fatalf("write selectors: %v", err)
	}

	t.Setenv("XHS_CONFIG_PATH", configPath)

	uc, err := loadPublishUsecase(true)
	if err != nil || uc == nil {
		t.Fatalf("expected fallback to config selectors, err=%v", err)
	}
}
