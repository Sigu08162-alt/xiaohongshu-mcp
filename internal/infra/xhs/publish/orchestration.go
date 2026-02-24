package publish

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
)

func (g *Gateway) PublishImage(ctx context.Context, content publish.ImageContent) error {
	logrus.Info(strings.Repeat("=", 60))
	logrus.Info("🚀 开始图文发布流程")
	logrus.Info(strings.Repeat("=", 60))
	logrus.Infof("  - 标题: %s", content.Title)
	logrus.Infof("  - 图片数量: %d", len(content.ImagePaths))
	return g.publishOrSaveCommon(ctx, content, true)
}

func (g *Gateway) SaveImageDraft(ctx context.Context, content publish.ImageContent) error {
	logrus.Info(strings.Repeat("=", 60))
	logrus.Info("💾 开始保存草稿流程")
	logrus.Info(strings.Repeat("=", 60))
	logrus.Infof("  - 标题: %s", content.Title)
	logrus.Infof("  - 图片数量: %d", len(content.ImagePaths))
	return g.publishOrSaveCommon(ctx, content, false)
}

// publishOrSaveCommon is the shared flow for both PublishImage and SaveImageDraft.
func (g *Gateway) publishOrSaveCommon(ctx context.Context, content publish.ImageContent, isPublish bool) error {
	actionName := "保存草稿"
	if isPublish {
		actionName = "发布"
	}

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
		return fmt.Errorf("%s goto url: %w", actionName, err)
	}

	if err := sleepDelay(g.pollingFor(isPublish), "page_stable_ms"); err != nil {
		return fmt.Errorf("%s page_stable_ms: %w", actionName, err)
	}

	resolver := g.newResolver(page)

	uploadSelector := resolveOrFallback(resolver, "publish_upload", g.cfg.Selectors["upload_input"])
	logrus.Infof("⏳ 等待上传输入框可见 (选择器: %s)...", uploadSelector)
	if err := page.WaitVisible(uploadSelector); err != nil {
		return fmt.Errorf("%s wait upload_input(%s): %w", actionName, uploadSelector, err)
	}

	beforeUploadURL := page.URL()
	if !strings.Contains(beforeUploadURL, "target=image") {
		return fmt.Errorf("上传前URL异常: %s", beforeUploadURL)
	}

	logrus.Infof("📤 开始上传图片 (共%d张)...", len(content.ImagePaths))
	if err := page.SetFiles(uploadSelector, content.ImagePaths); err != nil {
		return fmt.Errorf("%s upload_input(%s): %w", actionName, uploadSelector, err)
	}

	afterUploadURL := page.URL()
	if !strings.Contains(afterUploadURL, "target=image") {
		screenshotPath := fmt.Sprintf("debug_after_upload_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("上传后URL异常: %s", afterUploadURL)
	}

	titleSelector := resolveOrFallback(resolver, "publish_title", g.cfg.Selectors["title_input"])
	if err := page.WaitVisible(titleSelector); err != nil {
		return fmt.Errorf("%s wait title_input(%s): %w", actionName, titleSelector, err)
	}
	if err := page.Fill(titleSelector, content.Title); err != nil {
		return fmt.Errorf("%s title_input(%s): %w", actionName, titleSelector, err)
	}

	contentSelector := resolveOrFallback(resolver, "publish_content", g.cfg.Selectors["content"])
	if err := page.WaitVisible(contentSelector); err != nil {
		return fmt.Errorf("%s wait content(%s): %w", actionName, contentSelector, err)
	}
	if err := page.Fill(contentSelector, content.Content); err != nil {
		return fmt.Errorf("%s content(%s): %w", actionName, contentSelector, err)
	}

	if len(content.Tags) > 0 {
		logrus.Infof("🏷️  添加标签 (共%d个)...", len(content.Tags))
		if err := inputTags(page, content.Tags, g.pollingFor(isPublish)); err != nil {
			logrus.Warnf("⚠️ 标签添加失败: %v (继续)", err)
		}
	}

	if err := sleepDelay(g.pollingFor(isPublish), "post_content_render_ms"); err != nil {
		return fmt.Errorf("%s post_content_render_ms: %w", actionName, err)
	}

	beforeClickURL := page.URL()
	if !strings.Contains(beforeClickURL, "target=image") {
		screenshotPath := fmt.Sprintf("debug_before_click_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("点击前URL异常: %s", beforeClickURL)
	}

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

	var buttonSelector string
	if isPublish {
		buttonSelector = resolveOrFallback(resolver, "publish_submit", g.cfg.Selectors["submit"])
	} else {
		buttonSelector = resolveOrFallback(resolver, "publish_save_draft", g.cfg.Selectors["save_draft"])
	}
	logrus.Infof("选择器: %s", buttonSelector)

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
		return fmt.Errorf("%s失败: %w", actionName, err)
	}

	if err := page.WaitVisible(buttonSelector); err != nil {
		logrus.Warnf("等待按钮可见失败: %v (继续尝试)", err)
	}

	if !isPublish {
		if err := page.ScrollIntoView(buttonSelector); err != nil {
			logrus.Warnf("滚动到按钮失败: %v", err)
		}
		if err := sleepDelay(g.pollingFor(isPublish), "scroll_into_view_wait_ms"); err != nil {
			return fmt.Errorf("%s scroll_into_view_wait_ms: %w", actionName, err)
		}
	}

	clickErr := page.Click(buttonSelector)
	if clickErr != nil {
		logrus.Warnf("常规点击失败: %v，尝试强制点击", clickErr)
		if err := sleepDelay(g.pollingFor(isPublish), "click_retry_wait_ms"); err != nil {
			return fmt.Errorf("%s click_retry_wait_ms: %w", actionName, err)
		}
		if forceErr := page.ClickForce(buttonSelector); forceErr != nil {
			var jsCode string
			if isPublish {
				jsCode = `() => {
					const btn = Array.from(document.querySelectorAll('button')).find(b => {
						const t = b.textContent.trim();
						return (t.includes('发布') || t.includes('提交')) && !t.includes('暂存') && !b.disabled;
					});
					if (btn) { btn.click(); return true; } return false;
				}`
			} else {
				jsCode = `() => {
					const btn = Array.from(document.querySelectorAll('button')).find(b => {
						const t = b.textContent.trim();
						return (t.includes('暂存') || t.includes('草稿')) && !b.disabled;
					});
					if (btn) { btn.click(); return true; } return false;
				}`
			}
			clicked, err := page.Eval(jsCode)
			if err != nil || clicked != true {
				g.capturePageComponents(page, "click_failed")
				return fmt.Errorf("所有点击方式都失败: 常规=%v, 强制=%v, JS=%v", clickErr, forceErr, err)
			}
		}
	}

	afterClickURL := page.URL()
	if strings.Contains(afterClickURL, "target=video") {
		screenshotPath := fmt.Sprintf("debug_wrong_button_%d.png", time.Now().Unix())
		page.Screenshot(screenshotPath)
		return fmt.Errorf("点击的按钮错误，跳转到视频页面: %s", afterClickURL)
	}

	return g.waitForCompletion(page, isPublish)
}

func (g *Gateway) PublishVideo(ctx context.Context, content publish.VideoContent) error {
	return g.videoFlow(ctx, content, true)
}

func (g *Gateway) SaveVideoDraft(ctx context.Context, content publish.VideoContent) error {
	return g.videoFlow(ctx, content, false)
}

func (g *Gateway) videoFlow(ctx context.Context, content publish.VideoContent, isPublish bool) error {
	actionName := "保存视频草稿"
	if isPublish {
		actionName = "发布视频"
	}

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
		return fmt.Errorf("%s goto url: %w", actionName, err)
	}

	resolver := g.newResolver(page)

	uploadSelector := resolveOrFallback(resolver, "publish_upload", g.cfg.Selectors["upload_input"])
	if err := page.WaitVisible(uploadSelector); err != nil {
		return fmt.Errorf("%s wait upload_input(%s): %w", actionName, uploadSelector, err)
	}
	if err := page.SetFiles(uploadSelector, []string{content.VideoPath}); err != nil {
		return fmt.Errorf("%s upload_input(%s): %w", actionName, uploadSelector, err)
	}

	titleSelector := resolveOrFallback(resolver, "publish_title", g.cfg.Selectors["title_input"])
	if err := page.WaitVisible(titleSelector); err != nil {
		return fmt.Errorf("%s wait title_input(%s): %w", actionName, titleSelector, err)
	}
	if err := page.Fill(titleSelector, content.Title); err != nil {
		return fmt.Errorf("%s title_input(%s): %w", actionName, titleSelector, err)
	}

	contentSelector := resolveOrFallback(resolver, "publish_content", g.cfg.Selectors["content"])
	if err := page.WaitVisible(contentSelector); err != nil {
		return fmt.Errorf("%s wait content(%s): %w", actionName, contentSelector, err)
	}
	if err := page.Fill(contentSelector, content.Content); err != nil {
		return fmt.Errorf("%s content(%s): %w", actionName, contentSelector, err)
	}

	if err := sleepDelay(g.cfg.VideoPolling, "post_content_render_ms"); err != nil {
		return fmt.Errorf("%s post_content_render_ms: %w", actionName, err)
	}

	if isPublish {
		submitSelector := resolveOrFallback(resolver, "publish_submit", g.cfg.Selectors["submit"])
		logrus.Infof("选择器: %s", submitSelector)
		if err := page.WaitVisible(submitSelector); err != nil {
			screenshotPath := fmt.Sprintf("debug_submit_not_found_%d.png", time.Now().Unix())
			page.Screenshot(screenshotPath)
			return fmt.Errorf("发布按钮未找到: selector=%s, err=%v", submitSelector, err)
		}
		if err := page.Click(submitSelector); err != nil {
			return fmt.Errorf("点击发布按钮失败: %w", err)
		}
		return g.waitForCompletion(page, true)
	}

	// save draft path
	saveDraftSelector := resolveOrFallback(resolver, "publish_save_draft", g.cfg.Selectors["save_draft"])
	logrus.Infof("准备点击暂存按钮: %s", saveDraftSelector)
	if err := page.ScrollIntoView(saveDraftSelector); err != nil {
		logrus.Warnf("滚动到暂存按钮失败: %v", err)
	}
	if err := sleepDelay(g.cfg.VideoPolling, "scroll_into_view_wait_ms"); err != nil {
		return fmt.Errorf("%s scroll_into_view_wait_ms: %w", actionName, err)
	}
	if err := page.ClickForce(saveDraftSelector); err != nil {
		return fmt.Errorf("%s save_draft(%s): %w", actionName, saveDraftSelector, err)
	}
	if err := sleepDelay(g.cfg.VideoPolling, "draft_save_wait_ms"); err != nil {
		return fmt.Errorf("%s draft_save_wait_ms: %w", actionName, err)
	}
	return nil
}
