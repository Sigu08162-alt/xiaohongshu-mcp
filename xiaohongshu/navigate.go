package xiaohongshu

import (
	"context"
	"fmt"
	"time"

	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
)

type NavigateAction struct {
	page browser.Page
}

func NewNavigate(page browser.Page) *NavigateAction {
	return &NavigateAction{page: page}
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
	page := n.page.WithContext(ctx)

	// 首先导航到探索页面以确保登录状态
	if err := n.ToExplorePage(ctx); err != nil {
		return err
	}

	// 等待 DOM 稳定
	if err := page.WaitDOMStable(time.Second, 0.1); err != nil {
		return err
	}

	// 从 __INITIAL_STATE__ 获取当前用户ID
	userID, err := page.Eval(`() => {
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

	userIDStr, ok := userID.(string)
	if !ok || userIDStr == "" {
		return fmt.Errorf("无法获取用户ID")
	}

	// 直接导航到个人主页
	profileURL := "https://www.xiaohongshu.com/user/profile/" + userIDStr
	if err := page.Goto(profileURL); err != nil {
		return err
	}

	// 等待导航完成
	if err := page.WaitLoad(); err != nil {
		return err
	}

	return nil
}
