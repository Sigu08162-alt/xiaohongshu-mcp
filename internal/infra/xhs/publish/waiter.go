package publish

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"log/slog"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

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

// waitForCompletion waits for publish or draft-save to complete.
func (g *Gateway) waitForCompletion(page browser.Page, isPublish bool) error {
	actionName := "草稿保存"
	if isPublish {
		actionName = "发布"
	}

	uploadSelectors := resolveUploadSelectors(g.cfg.Selectors)
	polling := g.pollingFor(isPublish)
	slog.Info("等待 完成（检查URL变化和成功标志）...", "arg1", actionName)
	maxWait, err := getTimeout(polling)
	if err != nil {
		return fmt.Errorf("%s wait timeout: %w", actionName, err)
	}
	checkInterval, err := getInterval(polling)
	if err != nil {
		return fmt.Errorf("%s wait interval: %w", actionName, err)
	}
	startTime := time.Now()

	savePageState := func(reason string) {
		timestamp := time.Now().Unix()
		screenshotPath := fmt.Sprintf("debug_%s_%d.png", reason, timestamp)
		page.Screenshot(screenshotPath)
		slog.Info("📸 已保存截图:", "arg1", screenshotPath)
		htmlPath := fmt.Sprintf("debug_%s_%d.html", reason, timestamp)
		if html, err := page.HTML("html"); err == nil {
			os.WriteFile(htmlPath, []byte(html), 0644)
			slog.Info("📄 已保存页面HTML:", "arg1", htmlPath)
		}
	}

	lastLogTime := startTime
	for time.Since(startTime) < maxWait {
		currentURL := page.URL()
		if time.Since(lastLogTime) >= 5*time.Second {
			slog.Info("等待中", "action", actionName, "elapsed_s", time.Since(startTime).Seconds(), "url", currentURL)
			lastLogTime = time.Now()
		}

		if isPublish {
			if strings.Contains(currentURL, "published=true") {
				slog.Info("✅ 发布成功！URL已更新为发布完成状态")
				return nil
			}
			if strings.Contains(currentURL, "/creator/post") || strings.Contains(currentURL, "/creator/content") {
				slog.Info("✅ 发布成功！页面已跳转到内容管理")
				return nil
			}
			if !strings.Contains(currentURL, "/publish/publish") {
				slog.Info("✅ 发布成功！页面已离开发布页")
				savePageState("success_redirect")
				return nil
			}
			for _, sel := range []string{"text=发布成功", "text=发送成功", "text=已发布", ".success-toast", ".toast-success"} {
				if ok, _ := page.Has(sel); ok {
					slog.Info("✅ 发布成功！检测到成功提示 ( )", "arg1", sel)
					savePageState("success")
					return nil
				}
			}
		} else {
			if strings.Contains(currentURL, "/user/") || strings.Contains(currentURL, "draft") {
				slog.Info("✅ 草稿保存成功！已跳转到草稿列表或创作者中心")
				return nil
			}
			if ok, _ := page.Has(".success-message, .toast-success, text=保存成功"); ok {
				slog.Info("✅ 草稿保存成功！检测到成功提示")
				return nil
			}
		}

		// uploading toast — recoverable, keep waiting
		for _, sel := range splitSelectors(uploadSelectors.UploadingToast) {
			if ok, _ := page.Has(sel); ok {
				if text, err := page.Text(sel); err == nil && strings.Contains(text, "上传中") {
					slog.Warn("⏳ 检测到上传中提示，继续等待:", "arg1", strings.TrimSpace(text))
					time.Sleep(checkInterval)
					continue
				}
			}
		}

		// captcha check
		for _, sel := range []string{".captcha", ".verify-code", "text=滑动验证", "text=点击验证", "iframe[src*='captcha']"} {
			if ok, _ := page.Has(sel); ok {
				slog.Warn("⚠️ 检测到验证码，可能触发了反机器人检测")
				savePageState("captcha")
				g.capturePageComponents(page, "captcha")
				return fmt.Errorf("%s失败: 检测到验证码，请手动完成验证", actionName)
			}
		}

		// error check
		for _, sel := range []string{".error-message", ".toast-error", "text=发布失败", "text=提交失败", "[class*='error']"} {
			if ok, _ := page.Has(sel); ok {
				if errText, err := page.Text(sel); err == nil && errText != "" {
					slog.Error("❌ 失败：", "arg1", actionName, "arg2", errText)
					savePageState("error")
					g.capturePageComponents(page, "error")
					return fmt.Errorf("%s失败: %s", actionName, errText)
				}
			}
		}

		time.Sleep(checkInterval)
	}

	finalURL := page.URL()
	slog.Warn("⚠️ 超时：60秒内未检测到成功标志", "arg1", actionName)
	slog.Warn("最终URL:", "arg1", finalURL)
	savePageState("timeout")
	g.capturePageComponents(page, "timeout")
	return fmt.Errorf("%s超时：60秒内未检测到成功，最终URL: %s", actionName, finalURL)
}

// capturePageComponents collects page component info for debug purposes.
func (g *Gateway) capturePageComponents(page browser.Page, reason string) {
	slog.Info("🔍 开始采集页面组件信息（原因: ）...", "arg1", reason)

	jsCode := `() => {
		const result = {
			timestamp: new Date().toISOString(),
			url: window.location.href,
			buttons: [],
			inputs: [],
			containers: []
		};
		const buttons = document.querySelectorAll('button');
		buttons.forEach((btn, idx) => {
			const text = btn.textContent?.trim() || '';
			if (text.includes('发布') || text.includes('暂存') ||
			    text.includes('提交') || text.includes('草稿') ||
			    text.includes('取消') || text.includes('确定')) {
				const classes = btn.className ? btn.className.split(' ').filter(c => c) : [];
				const computedStyle = window.getComputedStyle(btn);
				result.buttons.push({
					index: idx, text, id: btn.id || '',
					classes, mainClass: classes[0] || '',
					selector: btn.className ? 'button.' + classes[0] : 'button',
					type: btn.type || '', disabled: btn.disabled,
					visible: btn.offsetParent !== null,
					display: computedStyle.display, opacity: computedStyle.opacity,
					position: { top: btn.offsetTop, left: btn.offsetLeft, width: btn.offsetWidth, height: btn.offsetHeight },
					attributes: { ariaLabel: btn.getAttribute('aria-label') || '', role: btn.getAttribute('role') || '', dataTestId: btn.getAttribute('data-test-id') || '' }
				});
			}
		});
		const inputs = document.querySelectorAll('input, textarea, [contenteditable="true"]');
		inputs.forEach((input, idx) => {
			const computedStyle = window.getComputedStyle(input);
			result.inputs.push({
				index: idx, tagName: input.tagName.toLowerCase(), type: input.type || '',
				id: input.id || '', classes: input.className ? input.className.split(' ').filter(c => c) : [],
				placeholder: input.placeholder || input.getAttribute('data-placeholder') || '',
				value: input.value || input.textContent || '',
				contentEditable: input.contentEditable, visible: input.offsetParent !== null,
				display: computedStyle.display
			});
		});
		const containers = document.querySelectorAll('.upload-content, .creator-tab, .edit-container, .bottom');
		containers.forEach((container, idx) => {
			result.containers.push({ index: idx, classes: container.className ? container.className.split(' ').filter(c => c) : [], visible: container.offsetParent !== null, childCount: container.children.length });
		});
		return result;
	}`

	info, err := page.Eval(jsCode)
	if err != nil {
		slog.Warn("采集组件信息失败:", "arg1", err)
		return
	}

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("debug_components_%s_%d.json", reason, timestamp)
	if data, err := json.MarshalIndent(info, "", "  "); err == nil {
		if err := os.WriteFile(filename, data, 0644); err == nil {
			slog.Info("📄 已保存组件信息到:", "arg1", filename)
		}
	}

	if infoMap, ok := info.(map[string]interface{}); ok {
		if buttons, ok := infoMap["buttons"].([]interface{}); ok {
			slog.Info("📊 发现 个相关按钮:", "arg1", len(buttons))
			for _, btn := range buttons {
				if m, ok := btn.(map[string]interface{}); ok {
					slog.Info("- 文本: \" \" | 选择器: | 可见: | 禁用:", "arg1", m["text"], "arg2", m["selector"], "arg3", m["visible"], "arg4", m["disabled"])
				}
			}
		}
		if inputs, ok := infoMap["inputs"].([]interface{}); ok {
			slog.Info("📊 发现 个输入框", "arg1", len(inputs))
		}
		if containers, ok := infoMap["containers"].([]interface{}); ok {
			slog.Info("📊 发现 个关键容器", "arg1", len(containers))
		}
	}
	slog.Info("✅ 组件信息采集完成")
}
