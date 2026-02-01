package xiaohongshu

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/polling"
)

type NavigateAction struct {
	page    browser.Page
	polling polling.Module
}

func NewNavigate(page browser.Page, pollingModule polling.Module) *NavigateAction {
	return &NavigateAction{page: page, polling: pollingModule}
}

func (n *NavigateAction) ToExplorePage(ctx context.Context) error {
	page := n.page.WithContext(ctx)

	// 导航到探索页面
	if err := page.Goto("https://www.xiaohongshu.com/explore"); err != nil {
		return err
	}

	// 等待页面加载完成
	if err := page.WaitLoad(); err != nil {
		return err
	}

	// 等待主要元素出现
	_, err := page.Element(`div#app`)
	if err != nil {
		return err
	}

	return nil
}

func (n *NavigateAction) ToProfilePage(ctx context.Context) error {
	return n.ToProfilePageWithUserID(ctx, "")
}

func (n *NavigateAction) ToProfilePageWithUserID(ctx context.Context, userID string) error {
	page := n.page.WithContext(ctx)

	// 首先导航到探索页面以确保登录状态
	if err := n.ToExplorePage(ctx); err != nil {
		return err
	}

	// 等待 __INITIAL_STATE__ 加载，而不是等待 DOM 稳定
	// 探索页面有动态内容，DOM 可能永远不会稳定
	maxWait, err := n.polling.Timeout()
	if err != nil {
		return err
	}
	checkInterval, err := n.polling.Interval()
	if err != nil {
		return err
	}
	startTime := time.Now()

	for time.Since(startTime) < maxWait {
		hasState, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ && window.__INITIAL_STATE__.user !== undefined;
		}`)
		if err == nil && hasState == true {
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待500ms确保数据完全加载
	if err := polling.SleepDelay(n.polling, "wait_500ms"); err != nil {
		return err
	}

	// 从 __INITIAL_STATE__ 获取当前用户ID
	currentUserID, err := page.Eval(`() => {
		if (window.__INITIAL_STATE__ && window.__INITIAL_STATE__.user) {
			const userInfo = window.__INITIAL_STATE__.user.userInfo;
			// Vue3的ref，需要访问.value或._value
			const actualUserInfo = userInfo?.value || userInfo?._value || userInfo?._rawValue;
			if (actualUserInfo && actualUserInfo.userId) {
				return actualUserInfo.userId;
			}
		}
		return null;
	}`)
	if err != nil {
		return err
	}

	currentUserIDStr, ok := currentUserID.(string)
	if !ok || currentUserIDStr == "" {
		return fmt.Errorf("无法获取用户ID")
	}

	resolvedUserID := resolveProfileUserID(currentUserIDStr, userID)
	// 直接导航到个人主页
	profileURL := buildProfileURL(resolvedUserID)
	if err := page.Goto(profileURL); err != nil {
		return err
	}
	if currentURL, err := page.Eval(`() => location.href`); err == nil {
		logrus.WithField("url", currentURL).Info("导航到个人主页完成")
	}

	// 等待导航完成
	if err := page.WaitLoad(); err != nil {
		return err
	}

	// 等待个人主页的 __INITIAL_STATE__ 加载，而不是等待 DOM 稳定
	// 个人主页可能有动态内容（笔记推荐、实时更新），DOM 可能永远不会稳定
	maxWait, err = n.polling.Timeout()
	if err != nil {
		return err
	}
	checkInterval, err = n.polling.Interval()
	if err != nil {
		return err
	}
	startTime = time.Now()

	for time.Since(startTime) < maxWait {
		hasState, err := page.Eval(`() => {
			return window.__INITIAL_STATE__ &&
			       window.__INITIAL_STATE__.user &&
			       window.__INITIAL_STATE__.user.userPageData !== undefined;
		}`)
		if err == nil && hasState == true {
			break
		}
		time.Sleep(checkInterval)
	}

	// 额外等待500ms确保数据完全加载
	if err := polling.SleepDelay(n.polling, "wait_500ms"); err != nil {
		return err
	}

	return nil
}

func resolveProfileUserID(currentUserID, requestedUserID string) string {
	if strings.TrimSpace(requestedUserID) != "" {
		return strings.TrimSpace(requestedUserID)
	}
	return currentUserID
}

func buildProfileURL(userID string) string {
	return "https://www.xiaohongshu.com/user/profile/" + userID
}
