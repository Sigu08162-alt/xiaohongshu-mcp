package wiring

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"log/slog"
	"gopkg.in/yaml.v3"

	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	apppublish "github.com/vmxmy/xiaohongshu-mcp/internal/app/publish"
	browserplaywright "github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser/playwright"
	infraconfig "github.com/vmxmy/xiaohongshu-mcp/internal/infra/config"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/selector"
)

// ── Named types (replaces anonymous structs) ──────────────────────────────────

// CollectedButton represents a button element captured by the selector collector.
type CollectedButton struct {
	Text     string   `yaml:"text"`
	Selector string   `yaml:"selector"`
	Classes  []string `yaml:"classes"`
}

// CollectedInput represents an input element captured by the selector collector.
type CollectedInput struct {
	Text        string   `yaml:"text"`
	Selector    string   `yaml:"selector"`
	Placeholder string   `yaml:"placeholder"`
	Type        string   `yaml:"type"`
	TagName     string   `yaml:"tag_name"`
	Classes     []string `yaml:"classes"`
}

// CollectedPage represents a page captured by the selector collector.
type CollectedPage struct {
	PageName string            `yaml:"page_name"`
	URL      string            `yaml:"url"`
	Buttons  []CollectedButton `yaml:"buttons"`
	Inputs   []CollectedInput  `yaml:"inputs"`
}

// collectedSelectorsConfig is the top-level YAML structure from the collector.
type collectedSelectorsConfig struct {
	Version   string                    `yaml:"version"`
	Generated string                    `yaml:"generated"`
	Pages     map[string]CollectedPage  `yaml:"pages"`
}

// ── Bootstrap functions ───────────────────────────────────────────────────────

// BuildPublishUsecaseFromConfig creates a publish usecase from config + selectors file path.
func BuildPublishUsecaseFromConfig(cfg *infraconfig.Config, selectors map[string]string, headless bool) (*apppublish.Usecase, error) {
	if cfg == nil {
		return nil, errors.New("config missing")
	}
	engineCfg := browserplaywright.DefaultConfig()
	engineCfg.Headless = headless
	engineCfg.CookiePath = cookies.GetCookiesFilePath()
	if cfg.Timeouts.Navigate > 0 {
		engineCfg.NavigationTimeout = time.Duration(cfg.Timeouts.Navigate) * time.Second
	}
	if cfg.Timeouts.ElementWait > 0 {
		engineCfg.ActionTimeout = time.Duration(cfg.Timeouts.ElementWait) * time.Second
	}
	if cfg.Timeouts.ImageUpload > 0 {
		uploadTimeout := time.Duration(cfg.Timeouts.ImageUpload) * time.Second
		if uploadTimeout > engineCfg.ActionTimeout {
			engineCfg.ActionTimeout = uploadTimeout
		}
	}
	engine := browserplaywright.New(engineCfg)

	var selectorCfg *selector.SelectorConfig
	if hl, err := selector.NewHotLoader("configs/selectors.yaml", 30*time.Second); err != nil {
		slog.Warn("自适应选择器配置加载失败: (使用静态选择器)", "arg1", err)
	} else {
		selectorCfg = hl.Get()
		// HotLoader runs in background; its lifecycle is tied to the process.
		// Call hl.Stop() on graceful shutdown if needed.
	}

	return BuildPublishUsecase(cfg, selectors, selectorCfg, engine)
}

// LoadPublishUsecase auto-discovers selector files and builds the publish usecase.
func LoadPublishUsecase(headless bool) (*apppublish.Usecase, error) {
	cfg := infraconfig.DefaultConfig()

	if selectorsPath := os.Getenv("XHS_SELECTORS_PATH"); selectorsPath != "" {
		selectors, err := LoadPublishSelectors(selectorsPath)
		if err != nil {
			return nil, err
		}
		return BuildPublishUsecaseFromConfig(cfg, selectors, headless)
	}

	candidates := []string{
		"selectors_discovered_pages_creator.yaml",
		"selectors_discovered_pages_fixed.yaml",
		"selectors_discovered_pages.yaml",
	}

	var lastErr error
	for _, candidate := range candidates {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		slog.Info("📂 找到选择器文件:", "arg1", candidate)
		selectors, err := LoadPublishSelectors(candidate)
		if err != nil {
			lastErr = err
			slog.Warn("选择器加载失败( ):", "arg1", candidate, "arg2", err)
			continue
		}
		return BuildPublishUsecaseFromConfig(cfg, selectors, headless)
	}

	if lastErr != nil {
		return nil, lastErr
	}

	slog.Info("未找到采集器选择器文件，使用自适应选择器")
	return BuildPublishUsecaseFromConfig(cfg, map[string]string{}, headless)
}

// InitPublishUsecase loads the publish usecase, logging a warning on failure.
func InitPublishUsecase(headless bool) *apppublish.Usecase {
	usecase, err := LoadPublishUsecase(headless)
	if err != nil {
		slog.Warn("初始化发布用例失败:", "arg1", err)
		return nil
	}
	return usecase
}

// LoadPublishSelectors reads a collector YAML file and extracts publish selectors.
func LoadPublishSelectors(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var collected collectedSelectorsConfig
	if err := yaml.Unmarshal(data, &collected); err != nil || collected.Pages == nil {
		return nil, fmt.Errorf("无效的选择器文件格式: %w", err)
	}

	slog.Info("📦 检测到采集器生成的选择器文件，正在提取发布页面选择器...")
	return ExtractPublishSelectorsFromCollected(&collected)
}

// ExtractPublishSelectorsFromCollected extracts publish-page selectors from collected data.
func ExtractPublishSelectorsFromCollected(collected *collectedSelectorsConfig) (map[string]string, error) {
	pageKey, publishPage, ok := findPublishPage(collected.Pages)
	if !ok {
		return nil, errors.New("采集文件中未找到发布页面")
	}
	slog.Info("📌 发布页面命中:", "arg1", pageKey)

	selectors := map[string]string{
		"upload_input":    "",
		"title_input":     "",
		"content":         "",
		"submit":          "",
		"save_draft":      "",
		"uploading_mask":  "",
		"uploading_class": "",
		"upload_preview":  "",
		"uploading_toast": "",
	}

	for _, inp := range publishPage.Inputs {
		if inp.Type == "file" && selectors["upload_input"] == "" {
			s := strings.TrimSpace(inp.Selector)
			if s == "" || s == "input" || !strings.Contains(s, "type") {
				s = `input[type="file"]`
			}
			selectors["upload_input"] = s
			slog.Info("✓ upload_input:", "arg1", s)
		}
		if containsStr(inp.Placeholder, "标题") {
			s := fmt.Sprintf(`input[placeholder*="%s"]`, "标题")
			selectors["title_input"] = s
			slog.Info("✓ title_input:", "arg1", s)
		}
		if containsAnyStr(inp.Classes, []string{"tiptap", "ProseMirror"}) {
			if containsStr(inp.Selector, "tiptap") && containsStr(inp.Selector, "ProseMirror") {
				selectors["content"] = inp.Selector
			} else {
				selectors["content"] = "[role='textbox']"
			}
			slog.Info("✓ content:", "arg1", selectors["content"])
		}
	}

	for _, btn := range publishPage.Buttons {
		if containsStr(btn.Text, "发布") && selectors["submit"] == "" {
			if containsAnyStr(btn.Classes, []string{"publishBtn"}) {
				selectors["submit"] = "button.publishBtn"
			} else {
				selectors["submit"] = btn.Selector
			}
			slog.Info("✓ submit:", "arg1", selectors["submit"])
		}
		if containsStr(btn.Text, "暂存") {
			selectors["save_draft"] = btn.Selector
			slog.Info("✓ save_draft:", "arg1", btn.Selector)
		}
	}

	for _, key := range []string{"upload_input", "title_input", "content", "submit"} {
		if selectors[key] == "" {
			slog.Warn("⚠️ 缺少选择器:", "arg1", key)
		}
	}

	return selectors, nil
}

// findPublishPage locates the publish page entry in the collected pages map.
func findPublishPage(pages map[string]CollectedPage) (string, CollectedPage, bool) {
	for _, key := range []string{"publish_publish", "publish_image", "publish_video", "publish_article"} {
		if page, ok := pages[key]; ok {
			return key, page, true
		}
	}

	bestScore := -1
	bestKey := ""
	var bestPage CollectedPage

	for key, page := range pages {
		score := 0
		if strings.Contains(page.URL, "/publish/publish") {
			score += 5
		}
		if strings.Contains(page.URL, "target=image") {
			score += 3
		}
		if strings.Contains(page.URL, "target=video") || strings.Contains(page.URL, "target=article") {
			score += 2
		}
		if containsStr(key, "发布") || containsStr(page.PageName, "发布") {
			score += 2
		}
		if containsStr(key, "图文") || containsStr(page.PageName, "图文") {
			score++
		}
		if hasFileInput(page.Inputs) {
			score += 3
		}
		if hasPublishButton(page.Buttons) {
			score += 3
		}
		if score > bestScore {
			bestScore = score
			bestKey = key
			bestPage = page
		}
	}

	if bestScore <= 0 {
		return "", bestPage, false
	}
	return bestKey, bestPage, true
}

func hasFileInput(inputs []CollectedInput) bool {
	for _, inp := range inputs {
		if strings.ToLower(strings.TrimSpace(inp.Type)) == "file" {
			return true
		}
	}
	return false
}

func hasPublishButton(buttons []CollectedButton) bool {
	for _, btn := range buttons {
		if containsStr(btn.Text, "发布") {
			return true
		}
	}
	return false
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

func containsAnyStr(slice []string, items []string) bool {
	for _, item := range items {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
	}
	return false
}
