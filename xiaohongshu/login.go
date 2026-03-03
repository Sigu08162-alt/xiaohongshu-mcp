package xiaohongshu

import (
	"context"
	"log/slog"
	"strings"
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

	// Give the top bar time to hydrate; a single fast check is often flaky.
	if err := polling.SleepDelay(a.polling, "wait_2000ms"); err != nil {
		return nil, err
	}

	// 方法1: 检查 DOM 选择器
	exists, err := pp.Has(`.main-container .user .link-wrapper .channel`)
	if err != nil {
		return nil, errors.Wrap(err, "check login status failed")
	}
	if !exists {
		// Retry once to reduce transient false negatives during SPA rendering.
		if err := polling.SleepDelay(a.polling, "wait_1000ms"); err != nil {
			return nil, err
		}
		exists, err = pp.Has(`.main-container .user .link-wrapper .channel`)
		if err != nil {
			return nil, errors.Wrap(err, "check login status failed")
		}
	}
	
	// 方法2: 如果 DOM 选择器失败，检查关键 cookies
	if !exists {
		cookieCheck, cookieErr := pp.Eval(`() => {
			const cookies = document.cookie.split(';').reduce((acc, c) => {
				const [k, v] = c.trim().split('=');
				if (k && v) acc[k] = v;
				return acc;
			}, {});
			const hasWebSession = cookies.web_session && cookies.web_session.length > 20;
			const hasA1 = cookies.a1 && cookies.a1.length > 20;
			return hasWebSession && hasA1;
		}`)
		if cookieErr == nil {
			if loggedIn, ok := cookieCheck.(bool); ok && loggedIn {
				slog.Info("login status detected via cookies", "selector_failed", true, "cookie_check", true)
				// Cookie 检测通过，继续获取用户名
				exists = true
			}
		}
		if !exists {
			return &LoginStatusResult{LoggedIn: false}, nil
		}
	}

	nickname := ""
	profilePath := ""
	raw, evalErr := pp.Eval(`() => {
		try {
			const pick = (vals) => {
				for (const v of vals) {
					if (typeof v === "string" && v.trim()) return v.trim();
				}
				return "";
			};
			const link = document.querySelector('.main-container .user .link-wrapper a');
			const channel = document.querySelector('.main-container .user .link-wrapper .channel');
			return {
				nickname: pick([
					window.__INITIAL_STATE__?.user?.userPageData?.value?.basicInfo?.nickname,
					window.__INITIAL_STATE__?.user?.userPageData?.basicInfo?.nickname,
					window.__INITIAL_STATE__?.user?.basicInfo?.nickname,
					window.__INITIAL_STATE__?.user?.nickname,
					link?.getAttribute("title"),
					link?.textContent,
					channel?.textContent,
				]),
				profilePath: (link?.getAttribute("href") || "").trim(),
			};
		} catch(e) {
			return { nickname: "", profilePath: "" };
		}
	}`)
	if evalErr == nil {
		if m, ok := raw.(map[string]interface{}); ok {
			if s, ok := m["nickname"].(string); ok {
				nickname = strings.TrimSpace(s)
			}
			if s, ok := m["profilePath"].(string); ok {
				profilePath = strings.TrimSpace(s)
			}
		}
	}

	profileNickname := ""
	profileTitle := ""
	if isPlaceholderNickname(nickname) && profilePath != "" {
		profileURL := profilePath
		if strings.HasPrefix(profileURL, "/") {
			profileURL = "https://www.xiaohongshu.com" + profileURL
		}
		if err := pp.Goto(profileURL); err == nil {
			if err := pp.WaitLoad(); err == nil {
				if err := polling.SleepDelay(a.polling, "wait_1000ms"); err == nil {
					profileRaw, pErr := pp.Eval(`() => {
						try {
							const pick = (vals) => {
								for (const v of vals) {
									if (typeof v === "string" && v.trim()) return v.trim();
								}
								return "";
							};
							return {
								nickname: pick([
									window.__INITIAL_STATE__?.user?.userPageData?.value?.basicInfo?.nickname,
									window.__INITIAL_STATE__?.user?.userPageData?.basicInfo?.nickname,
									window.__INITIAL_STATE__?.user?.basicInfo?.nickname,
									window.__INITIAL_STATE__?.user?.nickname,
									document.querySelector(".user-name")?.textContent,
									document.querySelector('[class*="nickname"]')?.textContent,
									document.querySelector('[class*="user-name"]')?.textContent,
								]),
								title: (document.title || "").trim(),
							};
						} catch(e) {
							return { nickname: "", title: "" };
						}
					}`)
					if pErr == nil {
						if m, ok := profileRaw.(map[string]interface{}); ok {
							if s, ok := m["nickname"].(string); ok {
								profileNickname = strings.TrimSpace(s)
							}
							if s, ok := m["title"].(string); ok {
								profileTitle = strings.TrimSpace(s)
							}
						}
					}
				}
			}
		}
	}

	nickname = selectPreferredNickname(nickname, profileNickname, profileTitle)

	return &LoginStatusResult{LoggedIn: true, Nickname: nickname}, nil
}

func isPlaceholderNickname(name string) bool {
	n := strings.TrimSpace(strings.ToLower(name))
	switch n {
	case "", "我", "我的", "me", "my":
		return true
	default:
		return false
	}
}

func nicknameFromTitle(title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return ""
	}
	t = strings.TrimSuffix(t, "- 小红书")
	return strings.TrimSpace(t)
}

func selectPreferredNickname(baseNickname, profileNickname, profileTitle string) string {
	base := strings.TrimSpace(baseNickname)
	if !isPlaceholderNickname(base) {
		return base
	}
	profile := strings.TrimSpace(profileNickname)
	if !isPlaceholderNickname(profile) {
		return profile
	}
	titleName := strings.TrimSpace(nicknameFromTitle(profileTitle))
	if !isPlaceholderNickname(titleName) {
		return titleName
	}
	return ""
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
