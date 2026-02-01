package main

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v3"

	"github.com/xpzouying/xiaohongshu-mcp/cookies"
	apppublish "github.com/xpzouying/xiaohongshu-mcp/internal/app/publish"
	browserplaywright "github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser/playwright"
	infraconfig "github.com/xpzouying/xiaohongshu-mcp/internal/infra/config"
	"github.com/xpzouying/xiaohongshu-mcp/internal/interfaces/wiring"
)

func buildPublishUsecase(cfg *infraconfig.Config, selectors map[string]string, headless bool) (*apppublish.Usecase, error) {
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
	return wiring.BuildPublishUsecase(cfg, selectors, engine)
}

func loadPublishUsecase(headless bool) (*apppublish.Usecase, error) {
	configPath := os.Getenv("XHS_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := infraconfig.LoadFromFile(configPath)
	if err != nil {
		return nil, err
	}

	// 优先查找采集器生成的选择器文件
	selectorsPath := os.Getenv("XHS_SELECTORS_PATH")
	if selectorsPath == "" {
		// 按优先级自动查找
		candidates := []string{
			"selectors_discovered_pages_creator.yaml", // 优先：采集器生成
			"selectors_discovered_pages.yaml",         // 次优：采集器生成（旧命名）
			"config.yaml",                             // 备用：传统配置
		}

		for _, candidate := range candidates {
			if _, err := os.Stat(candidate); err == nil {
				selectorsPath = candidate
				logrus.Infof("📂 找到选择器文件: %s", selectorsPath)
				break
			}
		}

		if selectorsPath == "" {
			selectorsPath = configPath // 最终回退到config.yaml
		}
	}

	selectors, err := loadPublishSelectors(selectorsPath)
	if err != nil {
		return nil, err
	}
	return buildPublishUsecase(cfg, selectors, headless)
}

func initPublishUsecase(headless bool) *apppublish.Usecase {
	usecase, err := loadPublishUsecase(headless)
	if err != nil {
		logrus.Warnf("初始化发布用例失败: %v", err)
		return nil
	}
	return usecase
}

type publishSelectorConfig struct {
	Selectors struct {
		Publish struct {
			UploadInput     string `yaml:"upload_input"`
			TitleInput      string `yaml:"title_input"`
			ContentEditorQL string `yaml:"content_editor_ql"`
			SubmitButton    string `yaml:"submit_button"`
			SaveDraftButton string `yaml:"save_draft_button"`
			UploadingMask   string `yaml:"uploading_mask"`
			UploadingClass  string `yaml:"uploading_class"`
			UploadPreview   string `yaml:"upload_preview"`
			UploadingToast  string `yaml:"uploading_toast"`
		} `yaml:"publish"`
	} `yaml:"selectors"`
}

// 采集器生成的YAML格式
type collectedSelectorsConfig struct {
	Version   string `yaml:"version"`
	Generated string `yaml:"generated"`
	Pages     map[string]struct {
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
	} `yaml:"pages"`
}

func loadPublishSelectors(path string) (map[string]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// 尝试识别文件格式：先尝试采集器格式
	var collected collectedSelectorsConfig
	if err := yaml.Unmarshal(data, &collected); err == nil && collected.Pages != nil {
		logrus.Info("📦 检测到采集器生成的选择器文件，正在提取发布页面选择器...")
		return extractPublishSelectorsFromCollected(&collected)
	}

	// 回退到旧格式（config.yaml）
	var cfg publishSelectorConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	logrus.Info("📄 使用传统config.yaml格式的选择器")
	return map[string]string{
		"upload_input":    cfg.Selectors.Publish.UploadInput,
		"title_input":     cfg.Selectors.Publish.TitleInput,
		"content":         cfg.Selectors.Publish.ContentEditorQL,
		"submit":          cfg.Selectors.Publish.SubmitButton,
		"save_draft":      cfg.Selectors.Publish.SaveDraftButton,
		"uploading_mask":  cfg.Selectors.Publish.UploadingMask,
		"uploading_class": cfg.Selectors.Publish.UploadingClass,
		"upload_preview":  cfg.Selectors.Publish.UploadPreview,
		"uploading_toast": cfg.Selectors.Publish.UploadingToast,
	}, nil
}

func extractPublishSelectorsFromCollected(collected *collectedSelectorsConfig) (map[string]string, error) {
	// 查找发布页面（publish_publish）
	publishPage, ok := collected.Pages["publish_publish"]
	if !ok {
		return nil, errors.New("采集文件中未找到 publish_publish 页面")
	}

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

	// 从输入框中提取选择器
	for _, inp := range publishPage.Inputs {
		// 文件上传
		if inp.Type == "file" && selectors["upload_input"] == "" {
			selectors["upload_input"] = inp.Selector
			logrus.Infof("  ✓ upload_input: %s", inp.Selector)
		}

		// 标题输入框（智能提取：优先使用placeholder，备用class）
		if contains(inp.Placeholder, "标题") {
			// 优先方案：基于placeholder（最稳定）
			preciseSelector := fmt.Sprintf("input[placeholder*=\"%s\"]", "标题")
			selectors["title_input"] = preciseSelector
			logrus.Infof("  ✓ title_input: %s (智能提取: placeholder)", preciseSelector)
			logrus.Infof("    备用选择器: %s", inp.Selector)
		}

		// 内容编辑器（通过class识别）
		if containsAny(inp.Classes, []string{"tiptap", "ProseMirror"}) {
			// 优先方案：精确class组合
			if contains(inp.Selector, "tiptap") && contains(inp.Selector, "ProseMirror") {
				selectors["content"] = inp.Selector
			} else {
				// 备用方案：role=textbox
				selectors["content"] = "[role='textbox']"
			}
			logrus.Infof("  ✓ content: %s (富文本编辑器)", selectors["content"])
		}
	}

	// 从按钮中提取选择器
	for _, btn := range publishPage.Buttons {
		// 发布按钮（智能提取：优先文本匹配）
		if contains(btn.Text, "发布") && selectors["submit"] == "" {
			// 优先方案：基于文本（最稳定）
			// Playwright支持: button:has-text("发布")
			// 或使用备用方案: class
			if containsAny(btn.Classes, []string{"publishBtn"}) {
				// 使用class选择器（更精确）
				selectors["submit"] = "button.publishBtn"
				logrus.Infof("  ✓ submit: button.publishBtn (text: %s)", btn.Text)
			} else {
				// 回退到通用选择器
				selectors["submit"] = btn.Selector
				logrus.Infof("  ✓ submit: %s (text: %s)", btn.Selector, btn.Text)
			}
		}

		// 暂存按钮
		if contains(btn.Text, "暂存") {
			selectors["save_draft"] = btn.Selector
			logrus.Infof("  ✓ save_draft: %s (text: %s)", btn.Selector, btn.Text)
		}
	}

	// 验证必需的选择器
	required := []string{"upload_input", "title_input", "content", "submit"}
	for _, key := range required {
		if selectors[key] == "" {
			logrus.Warnf("  ⚠️  缺少选择器: %s", key)
		}
	}

	return selectors, nil
}

func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || len(s) > 0 && len(substr) > 0 && containsIgnoreCase(s, substr))
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || strings.Contains(strings.ToLower(s), strings.ToLower(substr)))
}

func containsAny(slice []string, items []string) bool {
	for _, item := range items {
		for _, s := range slice {
			if s == item {
				return true
			}
		}
	}
	return false
}
