package publish

import (
	"context"
	"errors"
	"fmt"
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

	// 等待��布完成 - 通过URL变化来验证
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
	if err := g.engine.Start(); err != nil {
		return err
	}
	defer g.engine.Close()

	page, err := g.engine.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	if err := page.Goto(g.cfg.PublishImageURL); err != nil {
		return fmt.Errorf("save image draft goto url: %w", err)
	}
	// 等待上传输入框可见
	if err := page.WaitVisible(g.cfg.Selectors["upload_input"]); err != nil {
		return fmt.Errorf("save image draft wait upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
	}
	if err := page.SetFiles(g.cfg.Selectors["upload_input"], content.ImagePaths); err != nil {
		return fmt.Errorf("save image draft upload_input(%s): %w", g.cfg.Selectors["upload_input"], err)
	}
	// 等待标题输入框可见（图片上传后才出现）
	if err := page.WaitVisible(g.cfg.Selectors["title_input"]); err != nil {
		return fmt.Errorf("save image draft wait title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	if err := page.Fill(g.cfg.Selectors["title_input"], content.Title); err != nil {
		return fmt.Errorf("save image draft title_input(%s): %w", g.cfg.Selectors["title_input"], err)
	}
	// 等待内容编辑器可见
	if err := page.WaitVisible(g.cfg.Selectors["content"]); err != nil {
		return fmt.Errorf("save image draft wait content(%s): %w", g.cfg.Selectors["content"], err)
	}
	if err := page.Fill(g.cfg.Selectors["content"], content.Content); err != nil {
		return fmt.Errorf("save image draft content(%s): %w", g.cfg.Selectors["content"], err)
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
		return fmt.Errorf("save image draft save_draft(%s): %w", saveDraftSelector, err)
	}
	logrus.Info("已点击暂存按钮")

	// 等待草稿保存完成
	time.Sleep(3 * time.Second)

	return nil
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
