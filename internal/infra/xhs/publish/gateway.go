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
	"github.com/xpzouying/xiaohongshu-mcp/internal/domain/publish"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

var ErrNotReady = errors.New("publish not implemented")

type Config struct {
	PublishImageURL string
	PublishVideoURL string
	Selectors       map[string]string
}

type Gateway struct {
	cfg    Config
	engine browser.Engine
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
	time.Sleep(3 * time.Second)
	logrus.Info("✅ 页面稳定")

	// 等待上传输入框
	logrus.Infof("⏳ 等待上传输入框可见 (选择器: %s)...", g.cfg.Selectors["upload_input"])
	if err := page.WaitVisible(g.cfg.Selectors["upload_input"]); err != nil {
		logrus.Errorf("❌ 上传输入框未出现: %v", err)
		return fmt.Errorf("publish image wait upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
	}
	logrus.Info("✅ 上传输入框已可见")

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
	if err := page.SetFiles(g.cfg.Selectors["upload_input"], content.ImagePaths); err != nil {
		logrus.Errorf("❌ 图片上传失败: %v", err)
		return fmt.Errorf("publish image upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
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
	logrus.Infof("⏳ 等待标题输入框可见 (选择器: %s)...", g.cfg.Selectors["title_input"])
	if err := page.WaitVisible(g.cfg.Selectors["title_input"]); err != nil {
		logrus.Errorf("❌ 标题输入框未出现: %v", err)
		return fmt.Errorf("publish image wait title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	logrus.Info("✅ 标题输入框已可见")

	// 填写标题
	logrus.Infof("✍️ 填写标题: '%s'", content.Title)
	if err := page.Fill(g.cfg.Selectors["title_input"], content.Title); err != nil {
		logrus.Errorf("❌ 标题填写失败: %v", err)
		return fmt.Errorf("publish image title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	logrus.Info("✅ 标题填写完成")

	// 等待内容编辑器
	logrus.Infof("⏳ 等待内容编辑器可见 (选择器: %s)...", g.cfg.Selectors["content"])
	if err := page.WaitVisible(g.cfg.Selectors["content"]); err != nil {
		logrus.Errorf("❌ 内容编辑器未出现: %v", err)
		return fmt.Errorf("publish image wait content(%s): %w", g.cfg.Selectors["content"], err)
	}
	logrus.Info("✅ 内容编辑器已可见")

	// 填写内容
	logrus.Infof("✍️ 填写内容: '%s'", content.Content)
	if err := page.Fill(g.cfg.Selectors["content"], content.Content); err != nil {
		logrus.Errorf("❌ 内容填写失败: %v", err)
		return fmt.Errorf("publish image content(%s): %w", g.cfg.Selectors["content"], err)
	}
	logrus.Info("✅ 内容填写完成")

	// 输入标签（如果有）
	if len(content.Tags) > 0 {
		logrus.Infof("🏷️ 开始输入标签 (共%d个)...", len(content.Tags))
		if err := inputTags(page, content.Tags); err != nil {
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

	if len(content.MarkerTags) > 0 {
		if err := setMarkerTags(page, content.MarkerTags); err != nil {
			return fmt.Errorf("设置标记失败: %w", err)
		}
	}

	// 提交前等待
	logrus.Info("⏱️ 等待2秒让页面渲染完成...")
	time.Sleep(2 * time.Second)
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

	// 点���发布按钮
	submitSelector := g.cfg.Selectors["submit"]
	logrus.Infof("=== 准备点击发布按钮 ===")
	logrus.Infof("选择器: %s", submitSelector)

	// 等待按钮出现并可点击
	if err := page.WaitVisible(submitSelector); err != nil {
		logrus.Warnf("等待发布按钮可见失败: %v (继续尝试)", err)
	}

	// 使用普通点击（更可靠，能正确触发Vue事件）
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
	// 等待上传输入框可见
	if err := page.WaitVisible(g.cfg.Selectors["upload_input"]); err != nil {
		return fmt.Errorf("publish video wait upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
	}
	if err := page.SetFiles(g.cfg.Selectors["upload_input"], []string{content.VideoPath}); err != nil {
		return fmt.Errorf("publish video upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
	}
	// 等待标题输入框可见（视频上传后才出现）
	if err := page.WaitVisible(g.cfg.Selectors["title_input"]); err != nil {
		return fmt.Errorf("publish video wait title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	if err := page.Fill(g.cfg.Selectors["title_input"], content.Title); err != nil {
		return fmt.Errorf("publish video title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	// 等待内容编辑器可见
	if err := page.WaitVisible(g.cfg.Selectors["content"]); err != nil {
		return fmt.Errorf("publish video wait content(%s): %w", g.cfg.Selectors["content"], err)
	}
	if err := page.Fill(g.cfg.Selectors["content"], content.Content); err != nil {
		return fmt.Errorf("publish video content(%s): %w", g.cfg.Selectors["content"], err)
	}

	// 提交前短暂等待，确保内容已输入完成
	logrus.Info("内容填写完成，等待2秒让页面渲染完成...")
	time.Sleep(2 * time.Second)

	// 点击发布按钮
	submitSelector := g.cfg.Selectors["submit"]
	logrus.Infof("=== 准备点击发布按钮 ===")
	logrus.Infof("选择器: %s", submitSelector)

	// 等待按钮出现并可点击
	if err := page.WaitVisible(submitSelector); err != nil {
		logrus.Warnf("等待发布按钮可见失败: %v (继续尝试)", err)
	}

	// 使用普通点击（更可靠，能正确触发Vue事件）
	logrus.Info("点击发布按钮...")
	if err := page.Click(submitSelector); err != nil {
		return fmt.Errorf("点击发布按钮失败: %w", err)
	}
	logrus.Info("发布按钮已点击")

	// 等待发布完成 - 通过URL变化来验证
	logrus.Info("等待发布完成（检查URL变化）...")
	maxWait := 30 * time.Second
	checkInterval := 500 * time.Millisecond
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		currentURL := page.URL()
		logrus.Debugf("当前URL: %s", currentURL)

		// 检查URL是否包含 published=true，这是发布成功的标志
		if strings.Contains(currentURL, "published=true") {
			logrus.Info("✅ 发布成功！URL已更新为发布完成状态")
			logrus.Infof("发布完成URL: %s", currentURL)
			return nil
		}

		// 检查是否有错误消息
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
	// 等待上传输入框可见
	if err := page.WaitVisible(g.cfg.Selectors["upload_input"]); err != nil {
		return fmt.Errorf("save video draft wait upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
	}
	if err := page.SetFiles(g.cfg.Selectors["upload_input"], []string{content.VideoPath}); err != nil {
		return fmt.Errorf("save video draft upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
	}
	// 等待标题输入框可见（视频上传后才出现）
	if err := page.WaitVisible(g.cfg.Selectors["title_input"]); err != nil {
		return fmt.Errorf("save video draft wait title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	if err := page.Fill(g.cfg.Selectors["title_input"], content.Title); err != nil {
		return fmt.Errorf("save video draft title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	// 等待内容编辑器可见
	if err := page.WaitVisible(g.cfg.Selectors["content"]); err != nil {
		return fmt.Errorf("save video draft wait content(%s): %w", g.cfg.Selectors["content"], err)
	}
	if err := page.Fill(g.cfg.Selectors["content"], content.Content); err != nil {
		return fmt.Errorf("save video draft content(%s): %w", g.cfg.Selectors["content"], err)
	}

	// 等待页面渲染完成
	logrus.Info("内容填写完成，等待2秒...")
	time.Sleep(2 * time.Second)

	// 点击暂存按钮
	saveDraftSelector := g.cfg.Selectors["save_draft"]
	logrus.Infof("准备点击暂存按钮: %s", saveDraftSelector)

	// 滚动并强制点击
	if err := page.ScrollIntoView(saveDraftSelector); err != nil {
		logrus.Warnf("滚动到暂存按钮失败: %v", err)
	}
	time.Sleep(500 * time.Millisecond)

	if err := page.ClickForce(saveDraftSelector); err != nil {
		return fmt.Errorf("save video draft save_draft(%s): %w", saveDraftSelector, err)
	}
	logrus.Info("已点击暂存按钮")

	// 等待草稿保存完成
	time.Sleep(3 * time.Second)

	return nil
}

// inputTags 在内容编辑器中输入标签
func inputTags(page browser.Page, tags []string) error {
	if len(tags) == 0 {
		return nil
	}

	// 先获取内容编辑器元素
	contentElem, err := page.Element("[role=\"textbox\"]")
	if err != nil {
		return fmt.Errorf("未找到内容编辑器: %w", err)
	}

	time.Sleep(1 * time.Second)

	// 按下箭头键移动到底部
	for i := 0; i < 20; i++ {
		contentElem.Press("ArrowDown")
		time.Sleep(10 * time.Millisecond)
	}

	// 按两次回车换行
	contentElem.Press("Enter")
	contentElem.Press("Enter")

	time.Sleep(1 * time.Second)

	// 逐个输入标签
	for i, tag := range tags {
		tag = strings.TrimLeft(tag, "#")
		logrus.Infof("  [%d/%d] 输入标签: #%s", i+1, len(tags), tag)
		if err := inputTag(page, contentElem, tag); err != nil {
			logrus.Warnf("  ⚠️ 标签输入失败: %v", err)
			// 继续下一个标签
		}
	}

	return nil
}

// inputTag 输入单个标签
func inputTag(page browser.Page, contentElem browser.Element, tag string) error {
	// 输入 # 号
	contentElem.Input("#")
	time.Sleep(200 * time.Millisecond)

	// 逐字符输入标签
	for _, char := range tag {
		contentElem.Input(string(char))
		time.Sleep(50 * time.Millisecond)
	}

	time.Sleep(1 * time.Second)

	// 查找并点击标签联想选项
	topicContainer, err := page.Element("#creator-editor-topic-container")
	if err == nil && topicContainer != nil {
		firstItem, err := topicContainer.Element(".item")
		if err == nil && firstItem != nil {
			firstItem.Click()
			logrus.Infof("    ✅ 成功点击标签联想选项")
			time.Sleep(200 * time.Millisecond)
		} else {
			logrus.Warnf("    ⚠️ 未找到标签联想选项，直接输入空格")
			contentElem.Input(" ")
		}
	} else {
		logrus.Warnf("    ⚠️ 未找到标签联想下拉框，直接输入空格")
		contentElem.Input(" ")
	}

	time.Sleep(500 * time.Millisecond)
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
	time.Sleep(3 * time.Second)
	logrus.Info("✅ 页面稳定")

	// 等待上传输入框
	logrus.Infof("⏳ 等待上传输入框可见 (选择器: %s)...", g.cfg.Selectors["upload_input"])
	if err := page.WaitVisible(g.cfg.Selectors["upload_input"]); err != nil {
		logrus.Errorf("❌ 上传输入框未出现: %v", err)
		return fmt.Errorf("%s wait upload_input(%s): %w", actionName, g.cfg.Selectors["upload_input"], err)
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
	if err := page.SetFiles(g.cfg.Selectors["upload_input"], content.ImagePaths); err != nil {
		logrus.Errorf("❌ 图片上传失败: %v", err)
		return fmt.Errorf("%s upload_input(%s): %w", actionName, g.cfg.Selectors["upload_input"], err)
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
	logrus.Infof("⏳ 等待标题输入框可见 (选择器: %s)...", g.cfg.Selectors["title_input"])
	if err := page.WaitVisible(g.cfg.Selectors["title_input"]); err != nil {
		logrus.Errorf("❌ 标题输入框未出现: %v", err)
		return fmt.Errorf("%s wait title_input(%s): %w", actionName, g.cfg.Selectors["title_input"], err)
	}
	logrus.Info("✅ 标题输入框已可见")

	// 填写标题
	logrus.Infof("📝 填写标题: %s", content.Title)
	if err := page.Fill(g.cfg.Selectors["title_input"], content.Title); err != nil {
		logrus.Errorf("❌ 填写标题失败: %v", err)
		return fmt.Errorf("%s title_input(%s): %w", actionName, g.cfg.Selectors["title_input"], err)
	}
	logrus.Info("✅ 标题填写成功")

	// 等待内容编辑器
	logrus.Infof("⏳ 等待内容编辑器可见 (选择器: %s)...", g.cfg.Selectors["content"])
	if err := page.WaitVisible(g.cfg.Selectors["content"]); err != nil {
		logrus.Errorf("❌ 内容编辑器未出现: %v", err)
		return fmt.Errorf("%s wait content(%s): %w", actionName, g.cfg.Selectors["content"], err)
	}
	logrus.Info("✅ 内容编辑器已可见")

	// 填写正文
	logrus.Infof("📝 填写正文: %s", content.Content)
	if err := page.Fill(g.cfg.Selectors["content"], content.Content); err != nil {
		logrus.Errorf("❌ 填写正文失败: %v", err)
		return fmt.Errorf("%s content(%s): %w", actionName, g.cfg.Selectors["content"], err)
	}
	logrus.Info("✅ 正文填写成功")

	// 处理标签（如果有）
	if len(content.Tags) > 0 {
		logrus.Infof("🏷️  添加标签 (共%d个)...", len(content.Tags))
		if err := inputTags(page, content.Tags); err != nil {
			logrus.Warnf("⚠️ 标签添加失败: %v (继续)", err)
		} else {
			logrus.Info("✅ 标签添加成功")
		}
	}

	// 等待页面渲染完成
	logrus.Info("内容填写完成，等待2秒...")
	time.Sleep(2 * time.Second)

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
		buttonSelector = g.cfg.Selectors["submit"]
		buttonName = "发布按钮"
	} else {
		buttonSelector = g.cfg.Selectors["save_draft"]
		buttonName = "暂存按钮"
	}

	logrus.Infof("=== 准备点击%s ===", buttonName)
	logrus.Infof("选择器: %s", buttonSelector)

	// 等待按钮出现并可点击
	if err := page.WaitVisible(buttonSelector); err != nil {
		logrus.Warnf("等待%s可见失败: %v (继续尝试)", buttonName, err)
	}

	// 滚动到按钮（保存草稿按钮可能在底部）
	if !isPublish {
		if err := page.ScrollIntoView(buttonSelector); err != nil {
			logrus.Warnf("滚动到%s失败: %v", buttonName, err)
		}
		time.Sleep(500 * time.Millisecond)
	}

	// 点击按钮（先尝试常规点击，失败后使用JS点击兜底）
	logrus.Infof("点击%s...", buttonName)
	clickErr := page.Click(buttonSelector)
	if clickErr != nil {
		logrus.Warnf("常规点击%s失败: %v，等待2秒后尝试 JS 点击", buttonName, clickErr)
		time.Sleep(2 * time.Second)

		// 备用方案1：强制点击
		if forceErr := page.ClickForce(buttonSelector); forceErr != nil {
			logrus.Warnf("强制点击也失败: %v，尝试 JS 点击", forceErr)

			// 备用方案2：使用 JavaScript 点击（最可靠）
			clicked, err := page.Eval(`() => {
				const buttons = Array.from(document.querySelectorAll('button'));
				const submitBtn = buttons.find(btn =>
					btn.textContent.includes('发布') ||
					btn.textContent.includes('提交') ||
					btn.textContent.includes('暂存') ||
					btn.className.includes('submit')
				);
				if (submitBtn && !submitBtn.disabled) {
					submitBtn.click();
					return true;
				}
				return false;
			}`)

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

	logrus.Infof("等待%s完成（检查URL变化和成功标志）...", actionName)
	maxWait := 60 * time.Second // 增加到60秒，给小红书更多处理时间
	checkInterval := 500 * time.Millisecond
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

	for time.Since(startTime) < maxWait {
		currentURL := page.URL()
		logrus.Debugf("当前URL: %s", currentURL)

		// 检查成功标志
		if isPublish {
			// 发布成功：检查URL是否包含 published=true
			if strings.Contains(currentURL, "published=true") {
				logrus.Info("✅ 发布成功！URL已更新为发布完成状态")
				logrus.Infof("发布完成URL: %s", currentURL)
				return nil
			}

			// 新增：检查是否有成功Toast提示
			successSelectors := []string{
				"text=发布成功",
				"text=发送成功",
				".success-toast",
				".toast-success",
				"[class*='success']",
			}
			for _, selector := range successSelectors {
				if hasSuccess, _ := page.Has(selector); hasSuccess {
					logrus.Info("✅ 发布成功！检测到成功提示")
					savePageState("success")
					return nil
				}
			}

			// 新增：检查发布按钮是否变为"已发布"或消失
			if hasPublished, _ := page.Has("text=已发布"); hasPublished {
				logrus.Info("✅ 发布成功！按钮显示已发布")
				return nil
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
