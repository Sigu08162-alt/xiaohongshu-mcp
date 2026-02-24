package xiaohongshu

import (
	"fmt"
	"time"

	"log/slog"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

// findCommentElement 查找指定评论元素（参考 feed_detail.go 的滚动逻辑）
func findCommentElement(page browser.Page, pollingModule polling.Module, commentID, userID string) (browser.Element, error) {
	slog.Info("开始查找评论 - commentID: , userID:", "arg1", commentID, "arg2", userID)

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

	slog.Info("开始循环查找，最大尝试次数:", "arg1", maxAttempts)

	for attempt := 0; attempt < maxAttempts; attempt++ {
		slog.Info("=== 查找尝试 / ===", "arg1", attempt+1, "arg2", maxAttempts)

		// === 1. 检查是否到达底部 ===
		atEnd, err := checkEndContainer_browser(page, pollingModule)
		if err != nil {
			return nil, err
		}
		if atEnd {
			slog.Info("已到达评论底部，未找到目标评论")
			break
		}

		// === 2. 获取当前评论数量 ===
		currentCount, err := getCommentCount_browser(page, pollingModule)
		if err != nil {
			return nil, err
		}
		slog.Info("当前评论数:", "arg1", currentCount)

		if currentCount != lastCommentCount {
			slog.Info("✓ 评论数增加: ->", "arg1", lastCommentCount, "arg2", currentCount)
			lastCommentCount = currentCount
			stagnantChecks = 0
		} else {
			stagnantChecks++
			if stagnantChecks%5 == 0 {
				slog.Info("评论数停滞 次", "arg1", stagnantChecks)
			}
		}

		// === 3. 停滞检测 ===
		if stagnantChecks >= 10 {
			slog.Info("评论数量停滞超过10次，可能已加载完所有评论")
			break
		}

		// === 4. 先滚动到最后一个评论（触发懒加载）===
		if currentCount > 0 {
			slog.Info("滚动到最后一个评论（共 条）", "arg1", currentCount)

			timeout, err := pollingModule.Delay("wait_2000ms")
			if err != nil {
				return nil, err
			}
			elements, err := page.WithTimeout(timeout).Elements(".parent-comment, .comment-item, .comment")
			if err == nil && len(elements) > 0 {
				lastComment := elements[len(elements)-1]
				err := lastComment.ScrollIntoView()
				if err != nil {
					slog.Warn("滚动到最后一个评论失败:", "arg1", err)
				}
			} else {
				slog.Warn("未找到评论元素:", "arg1", err)
			}
			if err := polling.SleepDelay(pollingModule, "wait_300ms"); err != nil {
				return nil, err
			}
		}

		// === 5. 继续向下滚动 ===
		slog.Info("继续向下滚动...")
		_, err = page.Eval(`() => { window.scrollBy(0, window.innerHeight * 0.8); return true; }`)
		if err != nil {
			slog.Warn("滚动失败:", "arg1", err)
		}
		if err := polling.SleepDelay(pollingModule, "wait_500ms"); err != nil {
			return nil, err
		}

		// === 6. 滚动后立即查找（边滚动边查找）===
		if commentID != "" {
			selector := fmt.Sprintf("#comment-%s", commentID)
			slog.Info("尝试通过 commentID 查找:", "arg1", selector)

			timeout, err := pollingModule.Delay("wait_2000ms")
			if err != nil {
				return nil, err
			}
			el, err := page.WithTimeout(timeout).Element(selector)
			if err == nil && el != nil {
				slog.Info("✓ 通过 commentID 找到评论: (尝试 次)", "arg1", commentID, "arg2", attempt+1)
				return el, nil
			}
			slog.Info("未找到 commentID (2秒超时)")
		}

		if userID != "" {
			slog.Info("尝试通过 userID 查找:", "arg1", userID)

			timeout, err := pollingModule.Delay("wait_2000ms")
			if err != nil {
				return nil, err
			}
			elements, err := page.WithTimeout(timeout).Elements(".comment-item, .comment, .parent-comment")
			if err == nil && len(elements) > 0 {
				slog.Info("找到 个评论元素", "arg1", len(elements))
				for i, el := range elements {
					userEl, err := el.Element(fmt.Sprintf(`[data-user-id="%s"]`, userID))
					if err == nil && userEl != nil {
						slog.Info("✓ 通过 userID 在第 个元素中找到评论: (尝试 次)", "arg1", i+1, "arg2", userID, "arg3", attempt+1)
						return el, nil
					}
				}
				slog.Info("在 个元素中未找到匹配的 userID", "arg1", len(elements))
			} else {
				slog.Info("获取评论元素失败或超时:", "arg1", err)
			}
		}

		slog.Info("本次尝试未找到目标评论，继续下一轮...")

		// === 7. 等待内容加载 ===
		time.Sleep(scrollInterval)
	}

	return nil, fmt.Errorf("未找到评论 (commentID: %s, userID: %s), 尝试次数: %d", commentID, userID, maxAttempts)
}

// scrollToCommentsArea_browser 滚动到评论区 (browser 版本)
func scrollToCommentsArea_browser(page browser.Page, pollingModule polling.Module) error {
	slog.Info("滚动到评论区...")

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

	slog.Warn("页面不可访问:", "arg1", text)
	return fmt.Errorf("页面不可访问: %s", text)
}
