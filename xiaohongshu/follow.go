package xiaohongshu

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

// FollowAction 关注操作
type FollowAction struct {
	page    browser.Page
	polling polling.Module
}

// NewFollowAction 创建关注操作实例
func NewFollowAction(page browser.Page, pollingModule polling.Module) (*FollowAction, error) {
	return &FollowAction{page: page, polling: pollingModule}, nil
}

// Follow 关注用户，如果已关注则跳过
func (f *FollowAction) Follow(ctx context.Context, userID, xsecToken string) error {
	return f.perform(ctx, userID, xsecToken, true)
}

// Unfollow 取关用户，如果未关注则跳过
func (f *FollowAction) Unfollow(ctx context.Context, userID, xsecToken string) error {
	return f.perform(ctx, userID, xsecToken, false)
}

// perform 执行关注/取关操作
func (f *FollowAction) perform(ctx context.Context, userID, xsecToken string, targetFollowed bool) error {
	timeout, err := f.polling.Delay("wait_60000ms")
	if err != nil {
		return err
	}
	page := f.page.WithContext(ctx).WithTimeout(timeout)

	// 构建用户主页 URL
	url := makeUserProfileURL(userID, xsecToken)
	actionName := "关注"
	if !targetFollowed {
		actionName = "取关"
	}

	logrus.Infof("打开用户主页进行%s: %s", actionName, url)

	// 导航到用户主页
	if err := page.Goto(url); err != nil {
		return fmt.Errorf("导航失败: %w", err)
	}

	// 等待 __INITIAL_STATE__ 中的用户数据加载，而不是等待 DOM 稳定
	// 用户主页有动态内容（笔记推荐、实时更新、无限滚动），DOM 可能永远不会稳定
	maxWait, err := f.polling.Timeout()
	if err != nil {
		return err
	}
	checkInterval, err := f.polling.Interval()
	if err != nil {
		return err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasUserData, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.user &&
			       window.__INITIAL_STATE__.user.userPageData !== undefined;
		}`)
		if err == nil && hasUserData == true {
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待500ms确保用户数据完全加载
	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	// 检查页面是否可访问
	if err := checkPageAccessible(page, f.polling); err != nil {
		return err
	}

	// 获取当前关注状态
	followed, err := f.getFollowState(page)
	if err != nil {
		logrus.Warnf("获取关注状态失败: %v，继续尝试点击", err)
	} else {
		// 检查是否需要操作
		if targetFollowed && followed {
			logrus.Infof("用户 %s 已关注，跳过操作", userID)
			return nil
		}
		if !targetFollowed && !followed {
			logrus.Infof("用户 %s 未关注，跳过取关操作", userID)
			return nil
		}
	}

	// 查找关注按钮
	followBtn, err := f.findFollowButton(page)
	if err != nil {
		return fmt.Errorf("未找到关注按钮: %w", err)
	}

	// 滚动到按钮位置
	logrus.Info("滚动到关注按钮...")
	if err := followBtn.ScrollIntoView(); err != nil {
		logrus.Warnf("滚动失败: %v", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	// 等待按钮可见
	logrus.Info("等待关注按钮可见...")
	if err := followBtn.WaitVisible(); err != nil {
		logrus.Warnf("等待可见失败: %v", err)
	}
	if err := polling.SleepDelay(f.polling, "wait_500ms"); err != nil {
		return err
	}

	// 点击按钮
	logrus.Infof("点击%s按钮...", actionName)
	if err := followBtn.Click(); err != nil {
		logrus.Warnf("点击失败: %v，尝试使用 JS 点击", err)

		// 备用方案：使用 JavaScript 点击
		_, err = page.Eval(`() => {
			const buttons = Array.from(document.querySelectorAll('button'));
			const followBtn = buttons.find(btn =>
				btn.textContent.includes('关注') ||
				btn.textContent.includes('已关注') ||
				btn.className.includes('follow')
			);
			if (followBtn) {
				followBtn.click();
				return true;
			}
			return false;
		}`)

		if err != nil {
			return fmt.Errorf("无法点击关注按钮: %w", err)
		}
	}

	if err := polling.SleepDelay(f.polling, "wait_2000ms"); err != nil {
		return err
	}

	// 验证操作结果
	followed, err = f.getFollowState(page)
	if err != nil {
		logrus.Warnf("验证%s状态失败: %v", actionName, err)
		return nil
	}

	if followed == targetFollowed {
		logrus.Infof("用户 %s %s成功", userID, actionName)
		return nil
	}

	logrus.Warnf("用户 %s %s可能未成功，状态未变化", userID, actionName)
	return nil
}

// findFollowButton 查找关注按钮
func (f *FollowAction) findFollowButton(page browser.Page) (browser.Element, error) {
	// 尝试多个选择器
	selectors := []string{
		"button.follow-button",
		"button[class*='follow']",
		".user-info button",
		".user-card button",
	}

	for _, sel := range selectors {
		timeout, err := f.polling.Delay("wait_3000ms")
		if err != nil {
			return nil, err
		}
		elem, err := page.WithTimeout(timeout).Element(sel)
		if err == nil && elem != nil {
			logrus.Infof("找到关注按钮: %s", sel)
			return elem, nil
		}
	}

	// 尝试通过文本查找
	buttons, err := page.Elements("button")
	if err == nil {
		for _, btn := range buttons {
			text, _ := btn.Text()
			if text == "关注" || text == "已关注" || text == "+ 关注" {
				logrus.Info("通过文本找到关注按钮")
				return btn, nil
			}
		}
	}

	return nil, fmt.Errorf("所有选择器都失败")
}

// getFollowState 获取当前关注状态
func (f *FollowAction) getFollowState(page browser.Page) (bool, error) {
	// 方法1: 从 __INITIAL_STATE__ 读取
	result, err := page.Eval(`() => {
		if (window.__INITIAL_STATE__ &&
		    window.__INITIAL_STATE__.user &&
		    window.__INITIAL_STATE__.user.userPageData) {
			const userData = window.__INITIAL_STATE__.user.userPageData;
			if (userData.basicInfo) {
				return JSON.stringify({
					followed: userData.basicInfo.followed || false
				});
			}
		}
		return "";
	}`)
	if err != nil {
		logrus.Warnf("Eval 失败: %v", err)
	}

	if resultStr, ok := result.(string); ok && resultStr != "" {
		var state struct {
			Followed bool `json:"followed"`
		}
		if err := json.Unmarshal([]byte(resultStr), &state); err == nil {
			logrus.Infof("从页面状态读取关注状态: %v", state.Followed)
			return state.Followed, nil
		}
	}

	// 方法2: 从按钮文本判断
	buttons, err := page.Elements("button")
	if err == nil {
		for _, btn := range buttons {
			text, _ := btn.Text()
			if text == "已关注" {
				logrus.Info("从按钮文本判断: 已关注")
				return true, nil
			}
			if text == "关注" || text == "+ 关注" {
				logrus.Info("从按钮文本判断: 未关注")
				return false, nil
			}
		}
	}

	return false, fmt.Errorf("无法获取关注状态")
}
