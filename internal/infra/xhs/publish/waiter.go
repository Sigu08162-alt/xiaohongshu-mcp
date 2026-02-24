package publish

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
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

	savePageState := func(reason string) {
		timestamp := time.Now().Unix()
		screenshotPath := fmt.Sprintf("debug_%s_%d.png", reason, timestamp)
		page.Screenshot(screenshotPath)
		logrus.Infof("📸 已保存截图: %s", screenshotPath)
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
				return nil
			}
			if strings.Contains(currentURL, "/creator/post") || strings.Contains(currentURL, "/creator/content") {
				logrus.Info("✅ 发布成功！页面已跳转到内容管理")
				return nil
			}
			if !strings.Contains(currentURL, "/publish/publish") {
				logrus.Infof("✅ 发布成功！页面已离开发布页")
				savePageState("success_redirect")
				return nil
			}
			for _, sel := range []string{"text=发布成功", "text=发送成功", "text=已发布", ".success-toast", ".toast-success"} {
				if ok, _ := page.Has(sel); ok {
					logrus.Infof("✅ 发布成功！检测到成功提示 (%s)", sel)
					savePageState("success")
					return nil
				}
			}
		} else {
			if strings.Contains(currentURL, "/user/") || strings.Contains(currentURL, "draft") {
				logrus.Info("✅ 草稿保存成功！已跳转到草稿列表或创作者中心")
				return nil
			}
			if ok, _ := page.Has(".success-message, .toast-success, text=保存成功"); ok {
				logrus.Info("✅ 草稿保存成功！检测到成功提示")
				return nil
			}
		}

		// uploading toast — recoverable, keep waiting
		for _, sel := range splitSelectors(uploadSelectors.UploadingToast) {
			if ok, _ := page.Has(sel); ok {
				if text, err := page.Text(sel); err == nil && strings.Contains(text, "上传中") {
					logrus.Warnf("⏳ 检测到上传中提示，继续等待: %s", strings.TrimSpace(text))
					time.Sleep(checkInterval)
					continue
				}
			}
		}

		// captcha check
		for _, sel := range []string{".captcha", ".verify-code", "text=滑动验证", "text=点击验证", "iframe[src*='captcha']"} {
			if ok, _ := page.Has(sel); ok {
				logrus.Warn("⚠️ 检测到验证码，可能触发了反机器人检测")
				savePageState("captcha")
				g.capturePageComponents(page, "captcha")
				return fmt.Errorf("%s失败: 检测到验证码，请手动完成验证", actionName)
			}
		}

		// error check
		for _, sel := range []string{".error-message", ".toast-error", "text=发布失败", "text=提交失败", "[class*='error']"} {
			if ok, _ := page.Has(sel); ok {
				if errText, err := page.Text(sel); err == nil && errText != "" {
					logrus.Errorf("❌ %s失败：%s", actionName, errText)
					savePageState("error")
					g.capturePageComponents(page, "error")
					return fmt.Errorf("%s失败: %s", actionName, errText)
				}
			}
		}

		time.Sleep(checkInterval)
	}

	finalURL := page.URL()
	logrus.Warnf("⚠️ %s超时：60秒内未检测到成功标志", actionName)
	logrus.Warnf("最终URL: %s", finalURL)
	savePageState("timeout")
	g.capturePageComponents(page, "timeout")
	return fmt.Errorf("%s超时：60秒内未检测到成功，最终URL: %s", actionName, finalURL)
}

// capturePageComponents collects page component info for debug purposes.
func (g *Gateway) capturePageComponents(page browser.Page, reason string) {
	logrus.Infof("🔍 开始采集页面组件信息（原因: %s）...", reason)

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
		logrus.Warnf("采集组件信息失败: %v", err)
		return
	}

	timestamp := time.Now().Unix()
	filename := fmt.Sprintf("debug_components_%s_%d.json", reason, timestamp)
	if data, err := json.MarshalIndent(info, "", "  "); err == nil {
		if err := os.WriteFile(filename, data, 0644); err == nil {
			logrus.Infof("📄 已保存组件信息到: %s", filename)
		}
	}

	if infoMap, ok := info.(map[string]interface{}); ok {
		if buttons, ok := infoMap["buttons"].([]interface{}); ok {
			logrus.Infof("📊 发现 %d 个相关按钮:", len(buttons))
			for _, btn := range buttons {
				if m, ok := btn.(map[string]interface{}); ok {
					logrus.Infof("  - 文本: \"%v\" | 选择器: %v | 可见: %v | 禁用: %v", m["text"], m["selector"], m["visible"], m["disabled"])
				}
			}
		}
		if inputs, ok := infoMap["inputs"].([]interface{}); ok {
			logrus.Infof("📊 发现 %d 个输入框", len(inputs))
		}
		if containers, ok := infoMap["containers"].([]interface{}); ok {
			logrus.Infof("📊 发现 %d 个关键容器", len(containers))
		}
	}
	logrus.Info("✅ 组件信息采集完成")
}
