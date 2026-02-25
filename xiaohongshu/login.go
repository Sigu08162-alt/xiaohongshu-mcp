package xiaohongshu

import (
	"context"
	"log/slog"
	"time"

	"github.com/pkg/errors"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

type LoginAction struct {
	page    browser.Page
	polling polling.Module
}

func NewLogin(page browser.Page, pollingModule polling.Module) *LoginAction {
	return &LoginAction{page: page, polling: pollingModule}
}

// LoginStatusResult holds the result of a login status check.
type LoginStatusResult struct {
	LoggedIn bool
	Nickname string
}

func (a *LoginAction) CheckLoginStatus(ctx context.Context) (*LoginStatusResult, error) {
	pp := a.page.WithContext(ctx)
	if err := pp.Goto("https://www.xiaohongshu.com/explore"); err != nil {
		return nil, errors.Wrap(err, "导航失败")
	}
	if err := pp.WaitLoad(); err != nil {
		return nil, errors.Wrap(err, "等待页面加载失败")
	}

	if err := polling.SleepDelay(a.polling, "wait_1000ms"); err != nil {
		return nil, err
	}

	exists, err := pp.Has(`.main-container .user .link-wrapper .channel`)
	if err != nil {
		return nil, errors.Wrap(err, "check login status failed")
	}
	if !exists {
		return &LoginStatusResult{LoggedIn: false}, nil
	}

	// 尝试从 __INITIAL_STATE__ 获取真实用户名
	nickname := ""
	raw, evalErr := pp.Eval(`() => {
		try {
			return window.__INITIAL_STATE__?.user?.userInfo?.basicInfo?.nickname || "";
		} catch(e) { return ""; }
	}`)
	if evalErr == nil {
		if s, ok := raw.(string); ok {
			nickname = s
		}
	}

	// fallback: 从侧边栏用户头像 title 属性读取昵称
	if nickname == "" {
		raw2, evalErr2 := pp.Eval(`() => {
			try {
				const el = document.querySelector('.user .avatar') || document.querySelector('.user-info .nickname');
				return el ? (el.getAttribute('title') || el.textContent || "") : "";
			} catch(e) { return ""; }
		}`)
		if evalErr2 == nil {
			if s, ok := raw2.(string); ok {
				nickname = s
			}
		}
	}

	return &LoginStatusResult{LoggedIn: true, Nickname: nickname}, nil
}

func (a *LoginAction) Login(ctx context.Context) error {
	pp := a.page.WithContext(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	if err := pp.Goto("https://www.xiaohongshu.com/explore"); err != nil {
		return errors.Wrap(err, "导航失败")
	}
	if err := pp.WaitLoad(); err != nil {
		return errors.Wrap(err, "等待页面加载失败")
	}

	// 等待一小段时间让页面完全加载
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return err
	}

	// 检查是否已经登录
	if exists, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		// 已经登录，直接返回
		return nil
	}

	// 提示用户扫描二维码（浏览器窗口已打开，二维码弹窗应自动出现）
	slog.Info("请在浏览器窗口中扫描二维码登录（等待最多120秒）...")

	// 使用 WaitForLogin 轮询等待用户扫码完成（最多120秒）
	waitCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()

	if !a.WaitForLogin(waitCtx) {
		return errors.New("等待登录超时（120秒），请重试")
	}

	return nil
}

func (a *LoginAction) FetchQrcodeImage(ctx context.Context) (string, bool, error) {
	pp := a.page.WithContext(ctx)

	// 导航到小红书首页，这会触发二维码弹窗
	if err := pp.Goto("https://www.xiaohongshu.com/explore"); err != nil {
		return "", false, errors.Wrap(err, "导航失败")
	}
	if err := pp.WaitLoad(); err != nil {
		return "", false, errors.Wrap(err, "等待页面加载失败")
	}

	// 等待一小段时间让页面完全加载
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return "", false, err
	}

	// 检查是否已经登录
	if exists, _ := pp.Has(".main-container .user .link-wrapper .channel"); exists {
		return "", true, nil
	}

	// 获取二维码图片
	elem, err := pp.Element(".login-container .qrcode-img")
	if err != nil {
		return "", false, errors.Wrap(err, "找不到二维码元素")
	}

	src, err := elem.Attribute("src")
	if err != nil {
		return "", false, errors.Wrap(err, "get qrcode src failed")
	}
	if len(src) == 0 {
		return "", false, errors.New("qrcode src is empty")
	}

	return src, false, nil
}

func (a *LoginAction) WaitForLogin(ctx context.Context) bool {
	pp := a.page.WithContext(ctx)
	interval, err := a.polling.Interval()
	if err != nil {
		return false
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return false
		case <-ticker.C:
			el, err := pp.Element(".main-container .user .link-wrapper .channel")
			if err == nil && el != nil {
				return true
			}
		}
	}
}
