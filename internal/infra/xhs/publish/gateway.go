package publish

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/selector"
)

var ErrNotReady = errors.New("publish not implemented")

type Config struct {
	PublishImageURL string
	PublishVideoURL string
	Selectors       map[string]string
	SelectorCfg     *selector.SelectorConfig // 自适应选择器配置（nil时降级到Selectors）
	PublishPolling  PollingModule
	DraftPolling    PollingModule
	VideoPolling    PollingModule
}

type Gateway struct {
	cfg    Config
	engine browser.Engine
}

type UploadStateSelectors struct {
	UploadingMask  string
	UploadingClass string
	UploadPreview  string
	UploadingToast string
}

type PollingModule struct {
	TimeoutMs  int
	IntervalMs int
	MaxRetries int
	Delays     map[string]int
}

func NewGateway(cfg Config, engine browser.Engine) (*Gateway, error) {
	if cfg.PublishImageURL == "" || cfg.PublishVideoURL == "" {
		return nil, errors.New("publish url missing")
	}
	if engine == nil {
		return nil, errors.New("engine missing")
	}
	// 输出配置信息
	logrus.Infof("🔧 Gateway配置:")
	logrus.Infof("  - 图文发布URL: %s", cfg.PublishImageURL)
	logrus.Infof("  - 视频发布URL: %s", cfg.PublishVideoURL)
	return &Gateway{cfg: cfg, engine: engine}, nil
}

// newResolver 创建页面级选择器解析器（SelectorCfg为nil时返回nil）
func (g *Gateway) newResolver(page browser.Page) *selector.ElementResolver {
	if g.cfg.SelectorCfg != nil {
		return selector.NewElementResolver(g.cfg.SelectorCfg, page)
	}
	return nil
}

// resolveOrFallback 优先用自适应解析，失败降级到静态配置
func resolveOrFallback(resolver *selector.ElementResolver, smartName, legacySelector string) string {
	if resolver != nil {
		if sel, err := resolver.Resolve(smartName); err == nil {
			return sel
		}
		logrus.Warnf("自适应解析失败: %s, 降级到静态配置: %s", smartName, legacySelector)
	}
	return legacySelector
}

func (g *Gateway) PublishImage(ctx context.Context, content publish.ImageContent) error {
	logrus.Info(strings.Repeat("=", 60))
	logrus.Info("🚀 开始图文发布流程")
	logrus.Info(strings.Repeat("=", 60))

	logrus.Info("📋 发布内容:")
	logrus.Infof("  - 标题: %s", content.Title)
	logrus.Infof("  - 内容: %s", content.Content)
	logrus.Infof("  - 图片数量: %d", len(content.ImagePaths))

	logrus.Info("🌐 启动浏览器引擎...")
	if err := g.engine.Start(); err != nil {
		logrus.Errorf("❌ 浏览器引擎启动失败: %v", err)
		return err
	}
	defer g.engine.Close()
	logrus.Info("✅ 浏览器引擎启动成功")

	logrus.Info("📄 创建新页面...")
	page, err := g.engine.NewPage()
	if err != nil {
		logrus.Errorf("❌ 创建页面失败: %v", err)
		return err
	}
	defer page.Close()
	logrus.Info("✅ 页面创建成功")

	// 访问发布页面
	logrus.Info("🔗 准备访问图文发布页面...")
	logrus.Infof("📍 目标URL: %s", g.cfg.PublishImageURL)
	if err := page.Goto(g.cfg.PublishImageURL); err != nil {
		logrus.Errorf("❌ 访问页面失败: %v", err)
		return fmt.Errorf("publish image goto url: %w", err)
	}
	logrus.Info("✅ 页面导航完成（可能正在重定向验证cookie）")

	// 等待页面完全稳定（允许cookie验证重定向完成）
	logrus.Info("⏳ 等待页面稳定（包括cookie验证重定向）...")
	if err := sleepDelay(g.cfg.PublishPolling, "page_stable_ms"); err != nil {
		return fmt.Errorf("publish image page_stable_ms: %w", err)
	}
	logrus.Info("✅ 页面稳定")

	// 创建自适应选择器解析器
	resolver := g.newResolver(page)

	// 等待上传输入框
	uploadSelector := resolveOrFallback(resolver, "publish_upload", g.cfg.Selectors["upload_input"])
	logrus.Infof("⏳ 等待上传输入框出现 (选择器: %s)...", uploadSelector)
	jsCheck := fmt.Sprintf(`() => document.querySelector('%s') !== null`, uploadSelector)
	if err := page.WaitForFunction(jsCheck, 60*time.Second); err != nil {
		actualURL := page.URL()
		logrus.Errorf("❌ 上传输入框未出现: %v", err)
		logrus.Errorf("📍 当前URL: %s", actualURL)
		screenshotPath := fmt.Sprintf("debug_upload_wait_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		logrus.Infof("📸 已保存截图: %s", screenshotPath)
		return fmt.Errorf("publish image wait upload_input(%s): %w", uploadSelector, err)
	}
	logrus.Info("✅ 上传输入框已就绪")

	// 上传图片前检查URL
	beforeUploadURL := page.URL()
	logrus.Infof("📍 上传前URL: %s", beforeUploadURL)
	if !strings.Contains(beforeUploadURL, "target=image") {
		logrus.Errorf("❌ 警告：准备上传时URL已变化")
		logrus.Errorf("📍 预期包含: target=image")
		logrus.Errorf("📍 实际URL: %s", beforeUploadURL)
		return fmt.Errorf("上传前URL异常: %s", beforeUploadURL)
	}

	// 上传图片
	logrus.Infof("📤 开始上传图片 (共%d张)...", len(content.ImagePaths))
	for i, path := range content.ImagePaths {
		logrus.Infof("  [%d/%d] %s", i+1, len(content.ImagePaths), path)
	}
	if err := page.SetFiles(uploadSelector, content.ImagePaths); err != nil {
		logrus.Errorf("❌ 图片上传失败: %v", err)
		return fmt.Errorf("publish image upload_input(%s): %w", uploadSelector, err)
	}
	logrus.Info("✅ 图片���传成功")

	// 上传图片后立即检查URL
	afterUploadURL := page.URL()
	logrus.Infof("📍 上传后URL: %s", afterUploadURL)
	if !strings.Contains(afterUploadURL, "target=image") {
		logrus.Errorf("❌ 严重错误：上传图片后URL变为视频页面！")
		logrus.Errorf("📍 预期包含: target=image")
		logrus.Errorf("📍 实际URL: %s", afterUploadURL)
		screenshotPath := fmt.Sprintf("debug_after_upload_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		logrus.Errorf("📸 已保存截图: %s", screenshotPath)
		return fmt.Errorf("上传后URL跳转到视频页面: %s", afterUploadURL)
	}
	logrus.Info("✅ 上传后URL仍在图文页面")

	// 等待标题输入框

	// 填写标题后检查URL
	afterTitleURL := page.URL()
	logrus.Infof("📍 填写标题后URL: %s", afterTitleURL)
	if !strings.Contains(afterTitleURL, "target=image") {
		logrus.Errorf("❌ 错误：填写标题后URL已变化")
		logrus.Errorf("📍 实际URL: %s", afterTitleURL)
		return fmt.Errorf("填写标题后URL异常: %s", afterTitleURL)
	}
	titleSelector := resolveOrFallback(resolver, "publish_title", g.cfg.Selectors["title_input"])
	logrus.Infof("⏳ 等待标题输入框可见 (选择器: %s)...", titleSelector)
	if err := page.WaitVisible(titleSelector); err != nil {
		logrus.Errorf("❌ 标题输入框未出现: %v", err)
		return fmt.Errorf("publish image wait title_input(%s): %w", titleSelector, err)
	}
	logrus.Info("✅ 标题输入框已可见")

	// 填写标题
	logrus.Infof("✍️ 填写标题: '%s'", content.Title)
	if err := page.Fill(titleSelector, content.Title); err != nil {
		logrus.Errorf("❌ 标题填写失败: %v", err)
		return fmt.Errorf("publish image title_input(%s): %w", titleSelector, err)
	}
	logrus.Info("✅ 标题填写完成")

	// 等待内容编辑器
	contentSelector := resolveOrFallback(resolver, "publish_content", g.cfg.Selectors["content"])
	logrus.Infof("⏳ 等待内容编辑器可见 (选择器: %s)...", contentSelector)
	if err := page.WaitVisible(contentSelector); err != nil {
		logrus.Errorf("❌ 内容编辑器未出现: %v", err)
		return fmt.Errorf("publish image wait content(%s): %w", contentSelector, err)
	}
	logrus.Info("✅ 内容编辑器已可见")

	// 填写内容
	logrus.Infof("✍️ 填写内容: '%s'", content.Content)
	if err := page.Fill(contentSelector, content.Content); err != nil {
		logrus.Errorf("❌ 内容填写失败: %v", err)
		return fmt.Errorf("publish image content(%s): %w", contentSelector, err)
	}
	logrus.Info("✅ 内容填写完成")

	// 输入标签（如果有）
	if len(content.Tags) > 0 {
		logrus.Infof("🏷️ 开始输入标签 (共%d个)...", len(content.Tags))
		if err := inputTags(page, content.Tags, g.cfg.PublishPolling); err != nil {
			logrus.Warnf("⚠️ 标签输入失败: %v", err)
			// 标签输入失败不影响发布，继续
		} else {
			logrus.Info("✅ 标签输入完成")
		}
	}

	// 填写内容后检查URL
	afterContentURL := page.URL()
	logrus.Infof("📍 填写内容后URL: %s", afterContentURL)
	if !strings.Contains(afterContentURL, "target=image") {
		logrus.Errorf("❌ 错误：填写内容后URL已变化")
		logrus.Errorf("📍 实际URL: %s", afterContentURL)
		screenshotPath := fmt.Sprintf("debug_after_content_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("填写内容后URL异常: %s", afterContentURL)
	}
	logrus.Info("✅ 填写内容后URL仍在图文页面")

	if content.Location != "" {
		if err := setLocation(page, content.Location); err != nil {
			return fmt.Errorf("设置地点失败: %w", err)
		}
	}

	validMarkers := publish.FilterMarkerTags(content.MarkerTags)
	if len(validMarkers) > 0 {
		if err := setMarkerTags(page, validMarkers); err != nil {
			return fmt.Errorf("设置标记失败: %w", err)
		}
	}

	// 提交前等待
	logrus.Info("⏱️ 等待页面渲染完成...")
	if err := sleepDelay(g.cfg.PublishPolling, "pre_submit_render_ms"); err != nil {
		return fmt.Errorf("publish image pre_submit_render_ms: %w", err)
	}
	logrus.Info("✅ 等待完成")

	// 点击发布按钮前最后检查URL
	beforeClickURL := page.URL()
	logrus.Infof("📍 点击发布按钮前URL: %s", beforeClickURL)
	if !strings.Contains(beforeClickURL, "target=image") {
		logrus.Errorf("❌ 严重错误：准备点击发布按钮时URL已变化！")
		logrus.Errorf("📍 实际URL: %s", beforeClickURL)
		screenshotPath := fmt.Sprintf("debug_before_click_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("点击前URL异常: %s", beforeClickURL)
	}

	// 点击发布按钮（自适应解析已包含降级逻辑）
	submitSelector := resolveOrFallback(resolver, "publish_submit", g.cfg.Selectors["submit"])
	logrus.Infof("=== 准备点击发布按钮 ===")
	logrus.Infof("选择器: %s", submitSelector)

	if err := page.WaitVisible(submitSelector); err != nil {
		logrus.Warnf("等待发布按钮可见失败: %v", err)
		screenshotPath := fmt.Sprintf("debug_submit_not_found_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("发布按钮未找到: selector=%s, err=%v", submitSelector, err)
	}

	logrus.Info("点击发布按钮...")
	if err := page.Click(submitSelector); err != nil {
		return fmt.Errorf("点击发布按钮失败: %w", err)
	}
	logrus.Info("发布按钮已点击")

	// 点击后立即检查URL
	afterClickURL := page.URL()
	logrus.Infof("📍 点击发布按钮后URL: %s", afterClickURL)
	if strings.Contains(afterClickURL, "target=video") {
		logrus.Errorf("❌ 严重错误：点击发布按钮后跳转到视频页面！")
		logrus.Errorf("📍 选择器: %s", submitSelector)
		logrus.Errorf("📍 点击后URL: %s", afterClickURL)
		screenshotPath := fmt.Sprintf("debug_wrong_button_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("点击的按钮错误，跳转到视频页面: %s", afterClickURL)
	}

	// 等待发布完成 - 使用公共的验证逻辑
	return g.waitForCompletion(page, true)
}

func (g *Gateway) PublishVideo(ctx context.Context, content publish.VideoContent) error {
	if err := g.engine.Start(); err != nil {
		return err
	}
	defer g.engine.Close()

	page, err := g.engine.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	if err := page.Goto(g.cfg.PublishVideoURL); err != nil {
		return fmt.Errorf("publish video goto url: %w", err)
	}

	// 创建自适应选择器解析器
	resolver := g.newResolver(page)

	// 等待上传输入框可见
	uploadSelector := resolveOrFallback(resolver, "publish_upload", g.cfg.Selectors["upload_input"])
	if err := page.WaitVisible(uploadSelector); err != nil {
		return fmt.Errorf("publish video wait upload_input(%s): %w", uploadSelector, err)
	}
	if err := page.SetFiles(uploadSelector, []string{content.VideoPath}); err != nil {
		return fmt.Errorf("publish video upload_input(%s): %w", uploadSelector, err)
	}
	// 等待标题输入框可见（视频上传后才出现）
	titleSelector := resolveOrFallback(resolver, "publish_title", g.cfg.Selectors["title_input"])
	if err := page.WaitVisible(titleSelector); err != nil {
		return fmt.Errorf("publish video wait title_input(%s): %w", titleSelector, err)
	}
	if err := page.Fill(titleSelector, content.Title); err != nil {
		return fmt.Errorf("publish video title_input(%s): %w", titleSelector, err)
	}
	// 等待内容编辑器可见
	contentSelector := resolveOrFallback(resolver, "publish_content", g.cfg.Selectors["content"])
	if err := page.WaitVisible(contentSelector); err != nil {
		return fmt.Errorf("publish video wait content(%s): %w", contentSelector, err)
	}
	if err := page.Fill(contentSelector, content.Content); err != nil {
		return fmt.Errorf("publish video content(%s): %w", contentSelector, err)
	}

	// 提交前短暂等待，确保内容已输入完成
	logrus.Info("内容填写完成，等待页面渲染...")
	if err := sleepDelay(g.cfg.VideoPolling, "post_content_render_ms"); err != nil {
		return fmt.Errorf("publish video post_content_render_ms: %w", err)
	}

	// 点击发布按钮
	submitSelector := resolveOrFallback(resolver, "publish_submit", g.cfg.Selectors["submit"])
	logrus.Infof("=== 准备点击发布按钮 ===")
	logrus.Infof("选择器: %s", submitSelector)

	if err := page.WaitVisible(submitSelector); err != nil {
		logrus.Warnf("等待发布按钮可见失败: %v", err)
		screenshotPath := fmt.Sprintf("debug_submit_not_found_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("发布按钮未找到: selector=%s, err=%v", submitSelector, err)
	}

	logrus.Info("点击发布按钮...")
	if err := page.Click(submitSelector); err != nil {
		return fmt.Errorf("点击发布按钮失败: %w", err)
	}
	logrus.Info("发布按钮已点击")

	// 等待发布完成 - 通过URL变化来验证
	logrus.Info("等待发布完成（检查URL变化）...")
	maxWait, err := getTimeout(g.cfg.VideoPolling)
	if err != nil {
		return fmt.Errorf("publish video wait timeout: %w", err)
	}
	checkInterval, err := getInterval(g.cfg.VideoPolling)
	if err != nil {
		return fmt.Errorf("publish video wait interval: %w", err)
	}
	startTime := time.Now()

	lastLogTime := startTime
	for time.Since(startTime) < maxWait {
		currentURL := page.URL()
		if time.Since(lastLogTime) >= 5*time.Second {
			logrus.Infof("⏳ 视频发布等待中 (%.0fs)，当前URL: %s", time.Since(startTime).Seconds(), currentURL)
			lastLogTime = time.Now()
		}

		if strings.Contains(currentURL, "published=true") {
			logrus.Info("✅ 发布成功！URL已更新为发布完成状态")
			logrus.Infof("发布完成URL: %s", currentURL)
			return nil
		}

		if strings.Contains(currentURL, "/creator/post") || strings.Contains(currentURL, "/creator/content") {
			logrus.Info("✅ 发布成功！页面已跳转到内容管理")
			logrus.Infof("发布完成URL: %s", currentURL)
			return nil
		}

		if !strings.Contains(currentURL, "/publish/publish") {
			logrus.Infof("✅ 发布成功！页面已离开发布页")
			logrus.Infof("发布完成URL: %s", currentURL)
			return nil
		}

		successSelectors := []string{
			"text=发布成功",
			"text=发送成功",
			"text=已发布",
		}
		for _, sel := range successSelectors {
			if hasSuccess, _ := page.Has(sel); hasSuccess {
				logrus.Infof("✅ 发布成功！检测到成功提示 (%s)", sel)
				return nil
			}
		}

		if hasError, _ := page.Has(".error-message"); hasError {
			if errText, err := page.Text(".error-message"); err == nil {
				logrus.Errorf("❌ 发布失败：%s", errText)
				return fmt.Errorf("发布失败: %s", errText)
			}
		}

		time.Sleep(checkInterval)
	}

	// 超时了，记录最终URL帮助调试
	finalURL := page.URL()
	logrus.Warnf("⚠️ 发布超时：30秒内未检测到发布成功")
	logrus.Warnf("最终URL: %s", finalURL)

	// 尝试截图保存当前状态
	screenshotPath := fmt.Sprintf("debug_timeout_%d.png", time.Now().Unix())
	if err := page.Screenshot(screenshotPath); err == nil {
		logrus.Infof("已保存超时时的截图: %s", screenshotPath)
	}

	return fmt.Errorf("发布超时：30秒内未检测到发布成功，最终URL: %s", finalURL)
}

func (g *Gateway) SaveImageDraft(ctx context.Context, content publish.ImageContent) error {
	logrus.Info(strings.Repeat("=", 60))
	logrus.Info("💾 开始保存草稿流程")
	logrus.Info(strings.Repeat("=", 60))

	logrus.Info("📋 草稿内容:")
	logrus.Infof("  - 标题: %s", content.Title)
	logrus.Infof("  - 内容: %s", content.Content)
	logrus.Infof("  - 图片数量: %d", len(content.ImagePaths))

	// 使用公共的发布流程，只是最后点击不同的按钮
	return g.publishOrSaveCommon(ctx, content, false)
}

func (g *Gateway) SaveVideoDraft(ctx context.Context, content publish.VideoContent) error {
	if err := g.engine.Start(); err != nil {
		return err
	}
	defer g.engine.Close()

	page, err := g.engine.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	if err := page.Goto(g.cfg.PublishVideoURL); err != nil {
		return fmt.Errorf("save video draft goto url: %w", err)
	}

	// 创建自适应选择器解析器
	resolver := g.newResolver(page)

	// 等待上传输入框可见
	uploadSelector := resolveOrFallback(resolver, "publish_upload", g.cfg.Selectors["upload_input"])
	if err := page.WaitVisible(uploadSelector); err != nil {
		return fmt.Errorf("save video draft wait upload_input(%s): %w", uploadSelector, err)
	}
	if err := page.SetFiles(uploadSelector, []string{content.VideoPath}); err != nil {
		return fmt.Errorf("save video draft upload_input(%s): %w", uploadSelector, err)
	}
	// 等待标题输入框可见（视频上传后才出现）
	titleSelector := resolveOrFallback(resolver, "publish_title", g.cfg.Selectors["title_input"])
	if err := page.WaitVisible(titleSelector); err != nil {
		return fmt.Errorf("save video draft wait title_input(%s): %w", titleSelector, err)
	}
	if err := page.Fill(titleSelector, content.Title); err != nil {
		return fmt.Errorf("save video draft title_input(%s): %w", titleSelector, err)
	}
	// 等待内容编辑器可见
	contentSelector := resolveOrFallback(resolver, "publish_content", g.cfg.Selectors["content"])
	if err := page.WaitVisible(contentSelector); err != nil {
		return fmt.Errorf("save video draft wait content(%s): %w", contentSelector, err)
	}
	if err := page.Fill(contentSelector, content.Content); err != nil {
		return fmt.Errorf("save video draft content(%s): %w", contentSelector, err)
	}

	// 等待页面渲染完成
	logrus.Info("内容填写完成，等待页��渲染...")
	if err := sleepDelay(g.cfg.VideoPolling, "post_content_render_ms"); err != nil {
		return fmt.Errorf("save video draft post_content_render_ms: %w", err)
	}

	// 点击暂存按钮
	saveDraftSelector := resolveOrFallback(resolver, "publish_save_draft", g.cfg.Selectors["save_draft"])
	logrus.Infof("准备点击暂存按钮: %s", saveDraftSelector)

	// 滚动并强制点击
	if err := page.ScrollIntoView(saveDraftSelector); err != nil {
		logrus.Warnf("滚动到暂存按钮失败: %v", err)
	}
	if err := sleepDelay(g.cfg.VideoPolling, "scroll_into_view_wait_ms"); err != nil {
		return fmt.Errorf("save video draft scroll_into_view_wait_ms: %w", err)
	}

	if err := page.ClickForce(saveDraftSelector); err != nil {
		return fmt.Errorf("save video draft save_draft(%s): %w", saveDraftSelector, err)
	}
	logrus.Info("已点击暂存按钮")

	// 等待草稿保存完成
	if err := sleepDelay(g.cfg.VideoPolling, "draft_save_wait_ms"); err != nil {
		return fmt.Errorf("save video draft draft_save_wait_ms: %w", err)
	}

	return nil
}

func resolveUploadSelectors(selectors map[string]string) UploadStateSelectors {
	return UploadStateSelectors{
		UploadingMask:  getSelectorOrDefault(selectors, "uploading_mask", ".mask.uploading"),
		UploadingClass: getSelectorOrDefault(selectors, "uploading_class", "[class*='uploading']"),
		UploadPreview:  getSelectorOrDefault(selectors, "upload_preview", "img.preview"),
		UploadingToast: getSelectorOrDefault(selectors, "uploading_toast", ".creator-publish-toast"),
	}
}

func getSelectorOrDefault(selectors map[string]string, key, defaultValue string) string {
	if selectors == nil {
		return defaultValue
	}
	if value, ok := selectors[key]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func getDelay(module PollingModule, key string) (time.Duration, error) {
	if module.Delays == nil {
		return 0, fmt.Errorf("polling delay missing: %s", key)
	}
	value, ok := module.Delays[key]
	if !ok || value <= 0 {
		return 0, fmt.Errorf("polling delay missing: %s", key)
	}
	return time.Duration(value) * time.Millisecond, nil
}

func getInterval(module PollingModule) (time.Duration, error) {
	if module.IntervalMs <= 0 {
		return 0, fmt.Errorf("polling interval missing")
	}
	return time.Duration(module.IntervalMs) * time.Millisecond, nil
}

func getTimeout(module PollingModule) (time.Duration, error) {
	if module.TimeoutMs <= 0 {
		return 0, fmt.Errorf("polling timeout missing")
	}
	return time.Duration(module.TimeoutMs) * time.Millisecond, nil
}

func splitSelectors(raw string) []string {
	items := strings.Split(raw, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		trimmed := strings.TrimSpace(item)
		if trimmed == "" {
			continue
		}
		out = append(out, trimmed)
	}
	return out
}

func sleepDelay(module PollingModule, key string) error {
	delay, err := getDelay(module, key)
	if err != nil {
		return err
	}
	time.Sleep(delay)
	return nil
}

func (g *Gateway) pollingFor(isPublish bool) PollingModule {
	if isPublish {
		return g.cfg.PublishPolling
	}
	return g.cfg.DraftPolling
}

func waitForUploadComplete(page browser.Page, selectors UploadStateSelectors, expectedCount int, maxWait, interval time.Duration) error {
	deadline := time.Now().Add(maxWait)
	uploadingSelectors := append(splitSelectors(selectors.UploadingMask), splitSelectors(selectors.UploadingClass)...)
	previewSelectors := splitSelectors(selectors.UploadPreview)

	for {
		isUploading := false
		for _, sel := range uploadingSelectors {
			if visible, _ := page.IsVisible(sel); visible {
				isUploading = true
				break
			}
		}

		countOk := true
		if expectedCount > 0 && len(previewSelectors) > 0 {
			countOk = false
			if v, err := page.Eval(`(selectors) => {
				let maxCount = 0;
				for (const sel of selectors) {
					const count = document.querySelectorAll(sel).length;
					if (count > maxCount) maxCount = count;
				}
				return maxCount;
			}`, previewSelectors); err == nil {
				switch n := v.(type) {
				case int:
					countOk = n >= expectedCount
				case int64:
					countOk = int(n) >= expectedCount
				case float64:
					countOk = int(n) >= expectedCount
				case float32:
					countOk = int(n) >= expectedCount
				}
			}
		}

		if !isUploading && countOk {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("图片上传中，请稍后")
		}
		time.Sleep(interval)
	}
}

// inputTags 在内容编辑器中输入标签
func inputTags(page browser.Page, tags []string, polling PollingModule) error {
	if len(tags) == 0 {
		return nil
	}

	// 先获取内容编辑器元素
	contentElem, err := page.Element("[role=\"textbox\"]")
	if err != nil {
		return fmt.Errorf("未找到内容编辑器: %w", err)
	}

	if err := sleepDelay(polling, "tag_editor_ready_ms"); err != nil {
		return err
	}

	// 按下箭头键移动到底部
	for i := 0; i < 20; i++ {
		contentElem.Press("ArrowDown")
		if err := sleepDelay(polling, "tag_arrow_step_ms"); err != nil {
			return err
		}
	}

	// 按两次回车换行
	contentElem.Press("Enter")
	contentElem.Press("Enter")

	if err := sleepDelay(polling, "tag_after_enter_ms"); err != nil {
		return err
	}

	// 逐个输入标签
	for i, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		logrus.Infof("  [%d/%d] 输入标签: #%s", i+1, len(tags), tag)
		if err := inputTag(page, contentElem, tag, polling); err != nil {
			logrus.Warnf("  ⚠️ 标签输入失败: %v", err)
			// 继续下一个标签
		}
	}

	return nil
}

// inputTag 输入单个标签
func inputTag(page browser.Page, contentElem browser.Element, tag string, polling PollingModule) error {
	// 输入 # 号
	contentElem.Input("#")
	if err := sleepDelay(polling, "tag_hash_delay_ms"); err != nil {
		return err
	}

	// 逐字符输入标签
	for _, char := range tag {
		contentElem.Input(string(char))
		if err := sleepDelay(polling, "tag_char_delay_ms"); err != nil {
			return err
		}
	}

	if err := sleepDelay(polling, "tag_after_text_ms"); err != nil {
		return err
	}

	// 查找并点击标签联想选项
	topicContainer, err := page.Element("#creator-editor-topic-container")
	if err == nil && topicContainer != nil {
		firstItem, err := topicContainer.Element(".item")
		if err == nil && firstItem != nil {
			firstItem.Click()
			logrus.Infof("    ✅ 成功点击标签联想选项")
			if err := sleepDelay(polling, "tag_suggestion_click_ms"); err != nil {
				return err
			}
		} else {
			logrus.Warnf("    ⚠️ 未找到标签联想选项，直接输入空格")
			contentElem.Input(" ")
		}
	} else {
		logrus.Warnf("    ⚠️ 未找到标签联想下拉框，直接输入空格")
		contentElem.Input(" ")
	}

	if err := sleepDelay(polling, "tag_after_tag_ms"); err != nil {
		return err
	}
	return nil
}

// publishOrSaveCommon 发布或保存草稿的公共流程
// isPublish: true=发布, false=保存草稿
func (g *Gateway) publishOrSaveCommon(ctx context.Context, content publish.ImageContent, isPublish bool) error {
	actionName := "保存草稿"
	if isPublish {
		actionName = "发布"
	}

	logrus.Info("🌐 启动浏览器引擎...")
	if err := g.engine.Start(); err != nil {
		logrus.Errorf("❌ 浏览器引擎启动失败: %v", err)
		return err
	}
	defer g.engine.Close()
	logrus.Info("✅ 浏览器引擎启动成功")

	logrus.Info("📄 创建新页面...")
	page, err := g.engine.NewPage()
	if err != nil {
		logrus.Errorf("❌ 创建页面失败: %v", err)
		return err
	}
	defer page.Close()
	logrus.Info("✅ 页面创建成功")

	// 访问发布页面
	logrus.Info("🔗 准备访问图文发布页面...")
	logrus.Infof("📍 目标URL: %s", g.cfg.PublishImageURL)
	if err := page.Goto(g.cfg.PublishImageURL); err != nil {
		logrus.Errorf("❌ 访问页面失败: %v", err)
		return fmt.Errorf("%s goto url: %w", actionName, err)
	}
	logrus.Info("✅ 页面导航完成（可能正在重定向验证cookie）")

	// 等待页面完全稳定
	logrus.Info("⏳ 等待页面稳定（包括cookie验证重定向）...")
	if err := sleepDelay(g.pollingFor(isPublish), "page_stable_ms"); err != nil {
		return fmt.Errorf("%s page_stable_ms: %w", actionName, err)
	}
	logrus.Info("✅ 页面稳定")

	// 创建自适应选择器解析器
	resolver := g.newResolver(page)

	// 等待上传输入框
	uploadSelector := resolveOrFallback(resolver, "publish_upload", g.cfg.Selectors["upload_input"])
	logrus.Infof("⏳ 等待上传输入框可见 (选择器: %s)...", uploadSelector)
	if err := page.WaitVisible(uploadSelector); err != nil {
		logrus.Errorf("❌ 上传输入框未出现: %v", err)
		return fmt.Errorf("%s wait upload_input(%s): %w", actionName, uploadSelector, err)
	}
	logrus.Info("✅ 上传输入框已可见")

	// 上传前检查URL
	beforeUploadURL := page.URL()
	logrus.Infof("📍 上传前URL: %s", beforeUploadURL)
	if !strings.Contains(beforeUploadURL, "target=image") {
		logrus.Errorf("❌ 警告：准备上传时URL已变化")
		logrus.Errorf("📍 预期包含: target=image")
		logrus.Errorf("📍 实际URL: %s", beforeUploadURL)
		return fmt.Errorf("上传前URL异常: %s", beforeUploadURL)
	}

	// 上传图片
	logrus.Infof("📤 开始上传图片 (共%d张)...", len(content.ImagePaths))
	for i, path := range content.ImagePaths {
		logrus.Infof("  [%d/%d] %s", i+1, len(content.ImagePaths), path)
	}
	if err := page.SetFiles(uploadSelector, content.ImagePaths); err != nil {
		logrus.Errorf("❌ 图片上传失败: %v", err)
		return fmt.Errorf("%s upload_input(%s): %w", actionName, uploadSelector, err)
	}
	logrus.Info("✅ 图片上传成功")

	// 上传后检查URL
	afterUploadURL := page.URL()
	logrus.Infof("📍 上传后URL: %s", afterUploadURL)
	if !strings.Contains(afterUploadURL, "target=image") {
		logrus.Errorf("❌ 严重错误：上传图片后URL变为视频页面！")
		logrus.Errorf("📍 预期包含: target=image")
		logrus.Errorf("📍 实际URL: %s", afterUploadURL)
		screenshotPath := fmt.Sprintf("debug_after_upload_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		logrus.Errorf("📸 已保存截图: %s", screenshotPath)
		return fmt.Errorf("上传后URL异常: %s", afterUploadURL)
	}

	// 等待标题输入框
	titleSelector := resolveOrFallback(resolver, "publish_title", g.cfg.Selectors["title_input"])
	logrus.Infof("⏳ 等待标题输入框可见 (选择器: %s)...", titleSelector)
	if err := page.WaitVisible(titleSelector); err != nil {
		logrus.Errorf("❌ 标题输入框未出现: %v", err)
		return fmt.Errorf("%s wait title_input(%s): %w", actionName, titleSelector, err)
	}
	logrus.Info("✅ 标题输入框已可见")

	// 填写标题
	logrus.Infof("📝 填写标题: %s", content.Title)
	if err := page.Fill(titleSelector, content.Title); err != nil {
		logrus.Errorf("❌ 填写标题失败: %v", err)
		return fmt.Errorf("%s title_input(%s): %w", actionName, titleSelector, err)
	}
	logrus.Info("✅ 标题填写成功")

	// 等待内容编辑器
	contentSelector := resolveOrFallback(resolver, "publish_content", g.cfg.Selectors["content"])
	logrus.Infof("⏳ 等待内容编辑器可见 (选择器: %s)...", contentSelector)
	if err := page.WaitVisible(contentSelector); err != nil {
		logrus.Errorf("❌ 内容编辑器未出现: %v", err)
		return fmt.Errorf("%s wait content(%s): %w", actionName, contentSelector, err)
	}
	logrus.Info("✅ 内容编辑器已可见")

	// 填写正文
	logrus.Infof("📝 填写正文: %s", content.Content)
	if err := page.Fill(contentSelector, content.Content); err != nil {
		logrus.Errorf("❌ 填写正文失败: %v", err)
		return fmt.Errorf("%s content(%s): %w", actionName, contentSelector, err)
	}
	logrus.Info("✅ 正文填写成功")

	// 处理标签（如果有）
	if len(content.Tags) > 0 {
		logrus.Infof("🏷️  添加标签 (共%d个)...", len(content.Tags))
		if err := inputTags(page, content.Tags, g.pollingFor(isPublish)); err != nil {
			logrus.Warnf("⚠️ 标签添加失败: %v (继续)", err)
		} else {
			logrus.Info("✅ 标签添加成功")
		}
	}

	// 等待页面渲染完成
	logrus.Info("内容填写完成，等待渲染...")
	if err := sleepDelay(g.pollingFor(isPublish), "post_content_render_ms"); err != nil {
		return fmt.Errorf("%s post_content_render_ms: %w", actionName, err)
	}

	// 点击前检查URL
	beforeClickURL := page.URL()
	logrus.Infof("📍 点击按钮前URL: %s", beforeClickURL)
	if !strings.Contains(beforeClickURL, "target=image") {
		logrus.Errorf("❌ 严重错误：准备点击按钮时URL已变化！")
		logrus.Errorf("📍 实际URL: %s", beforeClickURL)
		screenshotPath := fmt.Sprintf("debug_before_click_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("点击前URL异常: %s", beforeClickURL)
	}

	// 根据 isPublish 决定点击哪个按钮
	var buttonSelector string
	var buttonName string
	if isPublish {
		buttonSelector = resolveOrFallback(resolver, "publish_submit", g.cfg.Selectors["submit"])
		buttonName = "发布按钮"
	} else {
		buttonSelector = resolveOrFallback(resolver, "publish_save_draft", g.cfg.Selectors["save_draft"])
		buttonName = "暂存按钮"
	}

	logrus.Infof("=== 准备点击%s ===", buttonName)
	logrus.Infof("选择器: %s", buttonSelector)

	// 等待图片上传完成（条件等待）
	logrus.Info("检查图片上传状态...")
	uploadSelectors := resolveUploadSelectors(g.cfg.Selectors)
	uploadTimeout, err := getTimeout(g.pollingFor(isPublish))
	if err != nil {
		return fmt.Errorf("%s upload timeout: %w", actionName, err)
	}
	uploadInterval, err := getInterval(g.pollingFor(isPublish))
	if err != nil {
		return fmt.Errorf("%s upload interval: %w", actionName, err)
	}
	if err := waitForUploadComplete(page, uploadSelectors, len(content.ImagePaths), uploadTimeout, uploadInterval); err != nil {
		logrus.Errorf("❌ 图片仍在上传中，已超时: %v", err)
		return fmt.Errorf("%s失败: %w", actionName, err)
	}
	logrus.Info("✅ 图片上传完成")

	// 等待按钮出现并可点击
	if err := page.WaitVisible(buttonSelector); err != nil {
		logrus.Warnf("等待%s可见失败: %v (继续尝试)", buttonName, err)
	}

	// 滚动到按钮（保存草稿按钮可能在底部）
	if !isPublish {
		if err := page.ScrollIntoView(buttonSelector); err != nil {
			logrus.Warnf("滚动到%s失败: %v", buttonName, err)
		}
		if err := sleepDelay(g.pollingFor(isPublish), "scroll_into_view_wait_ms"); err != nil {
			return fmt.Errorf("%s scroll_into_view_wait_ms: %w", actionName, err)
		}
	}

	// 点击按钮（先尝试常规点击，失败后使用JS点击兜底）
	logrus.Infof("点击%s...", buttonName)
	clickErr := page.Click(buttonSelector)
	if clickErr != nil {
		logrus.Warnf("常规点击%s失败: %v，等待后尝试 JS 点击", buttonName, clickErr)
		if err := sleepDelay(g.pollingFor(isPublish), "click_retry_wait_ms"); err != nil {
			return fmt.Errorf("%s click_retry_wait_ms: %w", actionName, err)
		}

		// 备用方案1：强制点击
		if forceErr := page.ClickForce(buttonSelector); forceErr != nil {
			logrus.Warnf("强制点击也失败: %v，尝试 JS 点击", forceErr)

			// 备用方案2：使用 JavaScript 点击（最可靠）
			// 根据 isPublish 精确匹配按钮文本
			var jsCode string
			if isPublish {
				// 发布按钮：匹配"发布"但排除"暂存"
				jsCode = `() => {
					const buttons = Array.from(document.querySelectorAll('button'));
					const submitBtn = buttons.find(btn => {
						const text = btn.textContent.trim();
						return (text.includes('发布') || text.includes('提交')) &&
						       !text.includes('暂存') &&
						       !btn.disabled;
					});
					if (submitBtn) {
						submitBtn.click();
						return true;
					}
					return false;
				}`
			} else {
				// 暂存按钮：匹配"暂存"或"草稿"
				jsCode = `() => {
					const buttons = Array.from(document.querySelectorAll('button'));
					const draftBtn = buttons.find(btn => {
						const text = btn.textContent.trim();
						return (text.includes('暂存') || text.includes('草稿')) &&
						       !btn.disabled;
					});
					if (draftBtn) {
						draftBtn.click();
						return true;
					}
					return false;
				}`
			}

			clicked, err := page.Eval(jsCode)

			if err != nil || clicked != true {
				// 所有点击方式都失败，采集页面组件信息用于调试
				logrus.Error("❌ 所有点击方式都失败，开始采集页面组件信息...")
				g.capturePageComponents(page, "click_failed")
				return fmt.Errorf("所有点击方式都失败: 常规点击=%v, 强制点击=%v, JS点击=%v", clickErr, forceErr, err)
			}
			logrus.Info("✅ JS 点击成功")
		} else {
			logrus.Info("✅ 强制点击成功")
		}
	} else {
		logrus.Info("✅ 常规点击成功")
	}
	logrus.Infof("%s已点击", buttonName)

	// 点击后立即检查URL
	afterClickURL := page.URL()
	logrus.Infof("📍 点击%s后URL: %s", buttonName, afterClickURL)
	if strings.Contains(afterClickURL, "target=video") {
		logrus.Errorf("❌ 严重错误：点击%s后跳转到视频页面！", buttonName)
		logrus.Errorf("📍 选择器: %s", buttonSelector)
		logrus.Errorf("📍 点击后URL: %s", afterClickURL)
		screenshotPath := fmt.Sprintf("debug_wrong_button_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("点击的按钮错误，跳转到视频页面: %s", afterClickURL)
	}

	// 等待完成 - 使用公共的验证逻辑
	return g.waitForCompletion(page, isPublish)
}

// waitForCompletion 等待发布或保存草稿完成，包含成功验证、错误检测、超时保护、调试截图
func (g *Gateway) waitForCompletion(page browser.Page, isPublish bool) error {
	actionName := "草稿保存"
	if isPublish {
		actionName = "发布"
	}

	uploadSelectors := resolveUploadSelectors(g.cfg.Selectors)
	polling := g.pollingFor(isPublish)
	logrus.Infof("等待%s完成（检查URL变化和成功标志）...", actionName)
	maxWait, err := getTimeout(polling)
	if err != nil {
		return fmt.Errorf("%s wait timeout: %w", actionName, err)
	}
	checkInterval, err := getInterval(polling)
	if err != nil {
		return fmt.Errorf("%s wait interval: %w", actionName, err)
	}
	startTime := time.Now()

	// 保存页面HTML用于调试
	savePageState := func(reason string) {
		timestamp := time.Now().Unix()
		// 保存截图
		screenshotPath := fmt.Sprintf("debug_%s_%d.png", reason, timestamp)
		page.Screenshot(screenshotPath)
		logrus.Infof("📸 已保存截图: %s", screenshotPath)

		// 保存HTML
		htmlPath := fmt.Sprintf("debug_%s_%d.html", reason, timestamp)
		if html, err := page.HTML("html"); err == nil {
			os.WriteFile(htmlPath, []byte(html), 0644)
			logrus.Infof("📄 已保存页面HTML: %s", htmlPath)
		}
	}

	lastLogTime := startTime
	for time.Since(startTime) < maxWait {
		currentURL := page.URL()
		if time.Since(lastLogTime) >= 5*time.Second {
			logrus.Infof("⏳ %s等待中 (%.0fs)，当前URL: %s", actionName, time.Since(startTime).Seconds(), currentURL)
			lastLogTime = time.Now()
		}

		if isPublish {
			if strings.Contains(currentURL, "published=true") {
				logrus.Info("✅ 发布成功！URL已更新为发布完成状态")
				logrus.Infof("发布完成URL: %s", currentURL)
				return nil
			}

			if strings.Contains(currentURL, "/creator/post") || strings.Contains(currentURL, "/creator/content") {
				logrus.Info("✅ 发布成功！页面已跳转到内容管理")
				logrus.Infof("发布完成URL: %s", currentURL)
				return nil
			}

			if !strings.Contains(currentURL, "/publish/publish") {
				logrus.Infof("✅ 发布成功！页面已离开发布页")
				logrus.Infof("发布完成URL: %s", currentURL)
				savePageState("success_redirect")
				return nil
			}

			successSelectors := []string{
				"text=发布成功",
				"text=发送成功",
				"text=已发布",
				".success-toast",
				".toast-success",
			}
			for _, selector := range successSelectors {
				if hasSuccess, _ := page.Has(selector); hasSuccess {
					logrus.Infof("✅ 发布成功！检测到成功提示 (%s)", selector)
					savePageState("success")
					return nil
				}
			}
		} else {
			// 保存草稿成功：检查是否回到创作者中心或草稿列表
			if strings.Contains(currentURL, "/user/") || strings.Contains(currentURL, "draft") {
				logrus.Info("✅ 草稿保存成功！已跳转到草稿列表或创作者中心")
				logrus.Infof("保存后URL: %s", currentURL)
				return nil
			}

			// 检查是否有成功提示（Toast消息）
			if hasSuccess, _ := page.Has(".success-message, .toast-success, text=保存成功"); hasSuccess {
				logrus.Info("✅ 草稿保存成功！检测到成功提示")
				return nil
			}
		}

		// 上传中提示：属于可恢复状态，继续等待
		uploadingToastDetected := false
		for _, selector := range splitSelectors(uploadSelectors.UploadingToast) {
			if hasToast, _ := page.Has(selector); hasToast {
				if text, err := page.Text(selector); err == nil && strings.Contains(text, "上传中") {
					logrus.Warnf("⏳ 检测到上传中提示，继续等待: %s", strings.TrimSpace(text))
					uploadingToastDetected = true
					break
				}
			}
		}
		if uploadingToastDetected {
			time.Sleep(checkInterval)
			continue
		}

		// 检查是否有验证码
		captchaSelectors := []string{
			".captcha",
			".verify-code",
			"text=滑动验证",
			"text=点击验证",
			"iframe[src*='captcha']",
		}
		for _, selector := range captchaSelectors {
			if hasCaptcha, _ := page.Has(selector); hasCaptcha {
				logrus.Warn("⚠️ 检测到验证码，可能触发了反机器人检测")
				savePageState("captcha")
				g.capturePageComponents(page, "captcha")
				return fmt.Errorf("%s失败: 检测到验证码，请手动完成验证", actionName)
			}
		}

		// 检查是否有错误消息
		errorSelectors := []string{
			".error-message",
			".toast-error",
			"text=发布失败",
			"text=提交失败",
			"[class*='error']",
		}
		for _, selector := range errorSelectors {
			if hasError, _ := page.Has(selector); hasError {
				if errText, err := page.Text(selector); err == nil && errText != "" {
					logrus.Errorf("❌ %s失败：%s", actionName, errText)
					savePageState("error")
					g.capturePageComponents(page, "error")
					return fmt.Errorf("%s失败: %s", actionName, errText)
				}
			}
		}

		time.Sleep(checkInterval)
	}

	// 超时了，记录最终状态
	finalURL := page.URL()
	logrus.Warnf("⚠️ %s超时：60秒内未检测到成功标志", actionName)
	logrus.Warnf("最终URL: %s", finalURL)

	// 保存超时时的完整状态
	savePageState("timeout")

	// 采集页面真实组件信息用于调试
	g.capturePageComponents(page, "timeout")

	return fmt.Errorf("%s超时：60秒内未检测到成功，最终URL: %s", actionName, finalURL)
}

// capturePageComponents 采集页面真实组件信息，用于错误调试和选择器验证
func (g *Gateway) capturePageComponents(page browser.Page, reason string) {
	logrus.Infof("🔍 开始采集页面组件信息（原因: %s）...", reason)

	// JavaScript 代码：采集页面所有相关按钮和元素信息
	jsCode := `() => {
		const result = {
			timestamp: new Date().toISOString(),
			url: window.location.href,
			buttons: [],
			inputs: [],
			containers: []
		};

		// 1. 采集所有按钮信息
		const buttons = document.querySelectorAll('button');
		buttons.forEach((btn, idx) => {
			const text = btn.textContent?.trim() || '';
			// 只采集相关按钮（发布、暂存、提交等）
			if (text.includes('发布') || text.includes('暂存') ||
			    text.includes('提交') || text.includes('草稿') ||
			    text.includes('取消') || text.includes('确定')) {

				const classes = btn.className ? btn.className.split(' ').filter(c => c) : [];
				const computedStyle = window.getComputedStyle(btn);

				result.buttons.push({
					index: idx,
					text: text,
					id: btn.id || '',
					classes: classes,
					mainClass: classes[0] || '',
					selector: btn.className ? 'button.' + classes[0] : 'button',
					type: btn.type || '',
					disabled: btn.disabled,
					visible: btn.offsetParent !== null,
					display: computedStyle.display,
					opacity: computedStyle.opacity,
					position: {
						top: btn.offsetTop,
						left: btn.offsetLeft,
						width: btn.offsetWidth,
						height: btn.offsetHeight
					},
					attributes: {
						ariaLabel: btn.getAttribute('aria-label') || '',
						role: btn.getAttribute('role') || '',
						dataTestId: btn.getAttribute('data-test-id') || ''
					}
				});
			}
		});

		// 2. 采集输入框信息（标题、内容）
		const inputs = document.querySelectorAll('input, textarea, [contenteditable="true"]');
		inputs.forEach((input, idx) => {
			const computedStyle = window.getComputedStyle(input);
			result.inputs.push({
				index: idx,
				tagName: input.tagName.toLowerCase(),
				type: input.type || '',
				id: input.id || '',
				classes: input.className ? input.className.split(' ').filter(c => c) : [],
				placeholder: input.placeholder || input.getAttribute('data-placeholder') || '',
				value: input.value || input.textContent || '',
				contentEditable: input.contentEditable,
				visible: input.offsetParent !== null,
				display: computedStyle.display
			});
		});

		// 3. 采集关键容器信息
		const containers = document.querySelectorAll('.upload-content, .creator-tab, .edit-container, .bottom');
		containers.forEach((container, idx) => {
			result.containers.push({
				index: idx,
				classes: container.className ? container.className.split(' ').filter(c => c) : [],
				visible: container.offsetParent !== null,
				childCount: container.children.length
			});
		});

		return result;
	}`

	// 执行 JavaScript 采集信息
	info, err := page.Eval(jsCode)
	if err != nil {
		logrus.Warnf("采集组件信息失败: %v", err)
		return
	}

	// 保存到 JSON 文件
	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("debug_components_%s_%d.json", reason, timestamp)

	if data, err := json.MarshalIndent(info, "", "  "); err == nil {
		if err := os.WriteFile(filename, data, 0644); err == nil {
			logrus.Infof("📄 已保存组件信息到: %s", filename)
		}
	}

	// 解析并输出关键信息到日志
	if infoMap, ok := info.(map[string]interface{}); ok {
		// 输出按钮信息
		if buttons, ok := infoMap["buttons"].([]interface{}); ok {
			logrus.Infof("📊 发现 %d 个相关按钮:", len(buttons))
			for _, btn := range buttons {
				if btnMap, ok := btn.(map[string]interface{}); ok {
					text := btnMap["text"]
					selector := btnMap["selector"]
					visible := btnMap["visible"]
					disabled := btnMap["disabled"]
					logrus.Infof("  - 文本: \"%v\" | 选择器: %v | 可见: %v | 禁用: %v",
						text, selector, visible, disabled)
				}
			}
		}

		// 输出输入框信息
		if inputs, ok := infoMap["inputs"].([]interface{}); ok {
			logrus.Infof("📊 发现 %d 个输入框", len(inputs))
		}

		// 输出容器信息
		if containers, ok := infoMap["containers"].([]interface{}); ok {
			logrus.Infof("📊 发现 %d 个关键容器", len(containers))
		}
	}

	logrus.Info("✅ 组件信息采集完成")
}
