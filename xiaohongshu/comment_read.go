package xiaohongshu

import (
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

// findCommentElement 查找指定评论元素（参考 feed_detail.go 的滚动逻辑）
func findCommentElement(page browser.Page, pollingModule polling.Module, commentID, userID string) (browser.Element, error) {
	logrus.Infof("开始查找评论 - commentID: %s, userID: %s", commentID, userID)

	const maxAttempts = 100
	scrollInterval, err := pollingModule.Delay("wait_800ms")
	if err != nil {
		return nil, err
	}

	// 先滚动到评论区
	if err := scrollToCommentsArea_browser(page, pollingModule); err != nil {
		return nil, err
	}
	if err := polling.SleepDelay(pollingModule, "wait_1000ms"); err != nil {
		return nil, err
	}

	var lastCommentCount = 0
	stagnantChecks := 0

	logrus.Infof("开始循环查找，最大尝试次数: %d", maxAttempts)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		logrus.Infof("=== 查找尝试 %d/%d ===", attempt+1, maxAttempts)

		// === 1. 检查是否到达底部 ===
		atEnd, err := checkEndContainer_browser(page, pollingModule)
		if err != nil {
			return nil, err
		}
		if atEnd {
			logrus.Info("已到达评论底部，未找到目标评论")
			break
		}

		// === 2. 获取当前评论数量 ===
		currentCount, err := getCommentCount_browser(page, pollingModule)
		if err != nil {
			return nil, err
		}
		logrus.Infof("当前评论数: %d", currentCount)

		if currentCount != lastCommentCount {
			logrus.Infof("✓ 评论数增加: %d -> %d", lastCommentCount, currentCount)
			lastCommentCount = currentCount
			stagnantChecks = 0
		} else {
			stagnantChecks++
			if stagnantChecks%5 == 0 {
				logrus.Infof("评论数停滞 %d 次", stagnantChecks)
			}
		}

		// === 3. 停滞检测 ===
		if stagnantChecks >= 10 {
			logrus.Info("评论数量停滞超过10次，可能已加载完所有评论")
			break
		}

		// === 4. 先滚动到最后一个评论（触发懒加载）===
		if currentCount > 0 {
			logrus.Infof("滚动到最后一个评论（共 %d 条）", currentCount)

			timeout, err := pollingModule.Delay("wait_2000ms")
			if err != nil {
				return nil, err
			}
			elements, err := page.WithTimeout(timeout).Elements(".parent-comment, .comment-item, .comment")
			if err == nil && len(elements) > 0 {
				lastComment := elements[len(elements)-1]
				err := lastComment.ScrollIntoView()
				if err != nil {
					logrus.Warnf("滚动到最后一个评论失败: %v", err)
				}
			} else {
				logrus.Warnf("未找到评论元素: %v", err)
			}
			if err := polling.SleepDelay(pollingModule, "wait_300ms"); err != nil {
				return nil, err
			}
		}

		// === 5. 继续向下滚动 ===
		logrus.Infof("继续向下滚动...")
		_, err = page.Eval(`() => { window.scrollBy(0, window.innerHeight * 0.8); return true; }`)
		if err != nil {
			logrus.Warnf("滚动失败: %v", err)
		}
		if err := polling.SleepDelay(pollingModule, "wait_500ms"); err != nil {
			return nil, err
		}

		// === 6. 滚动后立即查找（边滚动边查找）===
		if commentID != "" {
			selector := fmt.Sprintf("#comment-%s", commentID)
			logrus.Infof("尝试通过 commentID 查找: %s", selector)

			timeout, err := pollingModule.Delay("wait_2000ms")
			if err != nil {
				return nil, err
			}
			el, err := page.WithTimeout(timeout).Element(selector)
			if err == nil && el != nil {
				logrus.Infof("✓ 通过 commentID 找到评论: %s (尝试 %d 次)", commentID, attempt+1)
				return el, nil
			}
			logrus.Infof("未找到 commentID (2秒超时)")
		}

		if userID != "" {
			logrus.Infof("尝试通过 userID 查找: %s", userID)

			timeout, err := pollingModule.Delay("wait_2000ms")
			if err != nil {
				return nil, err
			}
			elements, err := page.WithTimeout(timeout).Elements(".comment-item, .comment, .parent-comment")
			if err == nil && len(elements) > 0 {
				logrus.Infof("找到 %d 个评论元素", len(elements))
				for i, el := range elements {
					userEl, err := el.Element(fmt.Sprintf(`[data-user-id="%s"]`, userID))
					if err == nil && userEl != nil {
						logrus.Infof("✓ 通过 userID 在第 %d 个元素中找到评论: %s (尝试 %d 次)", i+1, userID, attempt+1)
						return el, nil
					}
				}
				logrus.Infof("在 %d 个元素中未找到匹配的 userID", len(elements))
			} else {
				logrus.Infof("获取评论元素失败或超时: %v", err)
			}
		}

		logrus.Infof("本次尝试未找到目标评论，继续下一轮...")

		// === 7. 等待内容加载 ===
		time.Sleep(scrollInterval)
	}

	return nil, fmt.Errorf("未找到评论 (commentID: %s, userID: %s), 尝试次数: %d", commentID, userID, maxAttempts)
}

// scrollToCommentsArea_browser 滚动到评论区 (browser 版本)
func scrollToCommentsArea_browser(page browser.Page, pollingModule polling.Module) error {
	logrus.Info("滚动到评论区...")

	has, _ := page.Has(".comments-container")
	if has {
		_ = page.ScrollIntoView(".comments-container")
	}
	if err := polling.SleepDelay(pollingModule, "wait_500ms"); err != nil {
		return err
	}

	_ = page.ScrollBy(0, 100)
	return nil
}

// getCommentCount_browser 获取当前评论数量 (browser 版本)
func getCommentCount_browser(page browser.Page, pollingModule polling.Module) (int, error) {
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		return 0, err
	}
	elements, err := page.WithTimeout(timeout).Elements(".parent-comment")
	if err != nil {
		return 0, nil
	}
	return len(elements), nil
}

// checkEndContainer_browser 检查是否到达评论底部 (browser 版本)
func checkEndContainer_browser(page browser.Page, pollingModule polling.Module) (bool, error) {
	timeout, err := pollingModule.Delay("wait_2000ms")
	if err != nil {
		return false, err
	}
	has, err := page.WithTimeout(timeout).Has(".end-container")
	if err != nil || !has {
		return false, nil
	}

	text, err := page.Text(".end-container")
	if err != nil {
		return false, nil
	}

	return text == "没有更多了" || text == "已经到底了", nil
}

// checkPageAccessible_browser 检查页面是否可访问 (browser 版本)
func checkPageAccessible_browser(page browser.Page, pollingModule polling.Module) error {
	if err := polling.SleepDelay(pollingModule, "wait_500ms"); err != nil {
		return err
	}

	has, err := page.Has(".access-wrapper, .error-wrapper, .not-found-wrapper, .blocked-wrapper")
	if err != nil || !has {
		return nil
	}

	text, err := page.Text(".access-wrapper, .error-wrapper, .not-found-wrapper, .blocked-wrapper")
	if err != nil {
		return nil
	}

	if text == "" {
		return nil
	}

	logrus.Warnf("页面不可访问: %s", text)
	return fmt.Errorf("页面不可访问: %s", text)
}
