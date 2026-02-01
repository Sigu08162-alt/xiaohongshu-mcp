package xiaohongshu

import (
	"context"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/polling"
)

// CommentLikeAction 评论点赞操作
type CommentLikeAction struct {
	page    browser.Page
	polling polling.Module
}

// NewCommentLikeAction 创建评论点赞操作实例
func NewCommentLikeAction(page browser.Page, pollingModule polling.Module) (*CommentLikeAction, error) {
	return &CommentLikeAction{page: page, polling: pollingModule}, nil
}

// LikeComment 点赞评论，如果已点赞则跳过
func (c *CommentLikeAction) LikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	return c.perform(ctx, feedID, xsecToken, commentID, userID, true)
}

// UnlikeComment 取消点赞评论，如果未点赞则跳过
func (c *CommentLikeAction) UnlikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	return c.perform(ctx, feedID, xsecToken, commentID, userID, false)
}

// perform 执行点赞/取消点赞操作
func (c *CommentLikeAction) perform(ctx context.Context, feedID, xsecToken, commentID, userID string, targetLiked bool) error {
	timeout, err := c.polling.Delay("wait_300000ms")
	if err != nil {
		return err
	}
	page := c.page.WithContext(ctx).WithTimeout(timeout)

	url := makeFeedDetailURL(feedID, xsecToken)
	actionName := "点赞评论"
	if !targetLiked {
		actionName = "取消点赞评论"
	}

	logrus.Infof("打开 feed 详情页进行%s: %s", actionName, url)

	// 导航到详情页
	if err := page.Goto(url); err != nil {
		return fmt.Errorf("导航失败: %w", err)
	}

	// 等待 __INITIAL_STATE__ 中的笔记数据加载，而不是等待 DOM 稳定
	// Feed 详情页有动态内容（评论实时加载、推荐内容），DOM 可能永远不会稳定
	maxWait, err := c.polling.Timeout()
	if err != nil {
		return err
	}
	checkInterval, err := c.polling.Interval()
	if err != nil {
		return err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasNoteData, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.note &&
			       window.__INITIAL_STATE__.note.noteDetailMap !== undefined;
		}`)
		if err == nil && hasNoteData == true {
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待500ms确保笔记数据完全加载
	if err := polling.SleepDelay(c.polling, "wait_500ms"); err != nil {
		return err
	}

	// 检查页面是否可访问
	if err := checkPageAccessible(page, c.polling); err != nil {
		return err
	}

	// 等待评论容器加载
	if err := polling.SleepDelay(c.polling, "wait_2000ms"); err != nil {
		return err
	}

	// 查找评论元素
	commentEl, err := findCommentElement(page, c.polling, commentID, userID)
	if err != nil {
		return fmt.Errorf("无法找到评论: %w", err)
	}

	// 滚动到评论位置
	logrus.Info("滚动到评论位置...")
	if err := commentEl.ScrollIntoView(); err != nil {
		logrus.Warnf("滚动失败: %v", err)
	}
	if err := polling.SleepDelay(c.polling, "wait_1000ms"); err != nil {
		return err
	}

	// 查找评论的点赞按钮
	likeBtn, err := c.findCommentLikeButton(commentEl)
	if err != nil {
		return fmt.Errorf("无法找到评论点赞按钮: %w", err)
	}

	// 获取当前点赞状态
	liked, err := c.getCommentLikeState(likeBtn)
	if err != nil {
		logrus.Warnf("获取点赞状态失败: %v，继续尝试点击", err)
	} else {
		// 检查是否需要操作
		if targetLiked && liked {
			logrus.Infof("评论已点赞，跳过操作")
			return nil
		}
		if !targetLiked && !liked {
			logrus.Infof("评论未点赞，跳过取消点赞操作")
			return nil
		}
	}

	// 点击点赞按钮
	logrus.Infof("点击%s按钮...", actionName)
	if err := likeBtn.Click(); err != nil {
		logrus.Warnf("点击失败: %v，尝试使用 JS 点击", err)

		// 备用方案：使用 JavaScript 点击
		_, err = commentEl.Eval(`(commentEl) => {
			const likeBtn = commentEl.querySelector('.like, [class*="like"]');
			if (likeBtn) {
				likeBtn.click();
				return true;
			}
			return false;
		}`)

		if err != nil {
			return fmt.Errorf("无法点击点赞按钮: %w", err)
		}
	}

	if err := polling.SleepDelay(c.polling, "wait_2000ms"); err != nil {
		return err
	}

	// 验证操作结果
	liked, err = c.getCommentLikeState(likeBtn)
	if err != nil {
		logrus.Warnf("验证%s状态失败: %v", actionName, err)
		return nil
	}

	if liked == targetLiked {
		logrus.Infof("评论%s成功", actionName)
		return nil
	}

	logrus.Warnf("评论%s可能未成功，状态未变化", actionName)
	return nil
}

// findCommentLikeButton 查找评论的点赞按钮
func (c *CommentLikeAction) findCommentLikeButton(commentEl browser.Element) (browser.Element, error) {
	// 尝试多个选择器
	selectors := []string{
		".like",
		".interactions .like",
		"[class*='like']",
		".right .like",
	}

	for _, sel := range selectors {
		elem, err := commentEl.Element(sel)
		if err == nil && elem != nil {
			logrus.Infof("找到评论点赞按钮: %s", sel)
			return elem, nil
		}
	}

	return nil, fmt.Errorf("所有选择器都失败")
}

// getCommentLikeState 获取评论点赞状态
func (c *CommentLikeAction) getCommentLikeState(likeBtn browser.Element) (bool, error) {
	// 方法1: 检查 class 是否包含 active/liked
	class, err := likeBtn.Attribute("class")
	if err == nil && class != "" {
		if contains(class, "active") || contains(class, "liked") {
			logrus.Info("从 class 判断: 已点赞")
			return true, nil
		}
		logrus.Info("从 class 判断: 未点赞")
		return false, nil
	}

	// 方法2: 检查子元素的样式
	html, err := likeBtn.HTML()
	if err == nil {
		if contains(html, "active") || contains(html, "liked") {
			logrus.Info("从 HTML 判断: 已点赞")
			return true, nil
		}
		logrus.Info("从 HTML 判断: 未点赞")
		return false, nil
	}

	return false, fmt.Errorf("无法获取点赞状态")
}

// contains 检查字符串是否包含子串
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

// findSubstring 在字符串中查找子串
func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
