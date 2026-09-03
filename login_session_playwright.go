package main

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"log/slog"
)

const (
	xhsLoginURL         = "https://www.xiaohongshu.com/explore"
	xhsCreatorHomeURL   = "https://creator.xiaohongshu.com/new/home?source=official"
	loginStatusSelector = ".main-container .user .link-wrapper .channel"
)

var qrSelectors = []string{
	".qrcode-img",
	".login-container .qrcode-img",
	".qrcode img",
	".qrcode canvas",
	".login-container canvas",
	".login-container img",
	"canvas",
}

const (
	qrFallbackRegex    = "二维码|扫码"
	securityHintRegexp = "安全认证|安全验证|风险验证|二次验证|安全校验|扫码验证|验证身份|保护账号安全|二维码.*失效"
	scanSuccessRegexp  = "扫码成功|登录成功|登录已成功|手机上确认|重新扫码"
	maxFrameDepth      = 6
)

var forceFullPageQRCode = strings.EqualFold(os.Getenv("XHS_FORCE_FULL_PAGE_QR"), "1") ||
	strings.EqualFold(os.Getenv("XHS_FORCE_FULL_PAGE_QR"), "true")

// qrElement 二维码元素接口
type qrElement interface {
	Screenshot() ([]byte, error)
	BoundingBox() (*browser.BoundingBox, error)
}

// qrFrame 二维码框架接口
type qrFrame interface {
	HasRegex(ctx context.Context, selector, jsRegex string) (bool, error)
	Element(ctx context.Context, selector string) (qrElement, error)
	ElementByRegex(ctx context.Context, selector, jsRegex string) (qrElement, error)
	Elements(ctx context.Context, selector string) ([]qrElement, error)
	Frames(ctx context.Context) ([]qrFrame, error)
}

// qrPage 二维码页面接口
type qrPage interface {
	qrFrame
	Navigate(ctx context.Context, url string) error
	WaitLoad(ctx context.Context) error
	Has(ctx context.Context, selector string) (bool, error)
	Close() error
}

type fullPageScreenshotter interface {
	ScreenshotFullPage(path string) error
}

// browserPageAdapter 将 browser.Page 适配为 qrPage 接口
type browserPageAdapter struct {
	page browser.Page
	ctx  playwright.BrowserContext
}

func (b *browserPageAdapter) Navigate(ctx context.Context, url string) error {
	return b.page.Goto(url)
}

func (b *browserPageAdapter) WaitLoad(ctx context.Context) error {
	return b.page.WaitLoad()
}

func (b *browserPageAdapter) Has(ctx context.Context, selector string) (bool, error) {
	return b.page.Has(selector)
}

func (b *browserPageAdapter) HasRegex(ctx context.Context, selector, jsRegex string) (bool, error) {
	return b.page.HasRegex(selector, jsRegex)
}

func (b *browserPageAdapter) Element(ctx context.Context, selector string) (qrElement, error) {
	el, err := b.page.Element(selector)
	if err != nil {
		return nil, err
	}
	return &browserElementAdapter{element: el}, nil
}

func (b *browserPageAdapter) Elements(ctx context.Context, selector string) ([]qrElement, error) {
	els, err := b.page.Elements(selector)
	if err != nil {
		return nil, err
	}
	result := make([]qrElement, 0, len(els))
	for _, el := range els {
		result = append(result, &browserElementAdapter{element: el})
	}
	return result, nil
}

func (b *browserPageAdapter) ScreenshotFullPage(path string) error {
	return b.page.ScreenshotFullPage(path)
}

func (b *browserPageAdapter) ElementByRegex(ctx context.Context, selector, jsRegex string) (qrElement, error) {
	el, err := b.page.ElementByRegex(selector, jsRegex)
	if err != nil {
		return nil, err
	}
	return &browserElementAdapter{element: el}, nil
}

func (b *browserPageAdapter) Frames(ctx context.Context) ([]qrFrame, error) {
	return framesForPage(ctx, b.page)
}

func (b *browserPageAdapter) Close() error {
	return b.page.Close()
}

// browserElementAdapter 将 browser.Element 适配为 qrElement 接口
type browserElementAdapter struct {
	element browser.Element
}

func (b *browserElementAdapter) Screenshot() ([]byte, error) {
	// Playwright 元素截图通过 Eval 实现
	bbox, err := b.element.BoundingBox()
	if err != nil {
		return nil, err
	}
	if bbox == nil {
		return nil, errors.New("element has no bounding box")
	}

	// 通过 Eval 执行截图
	result, err := b.element.Eval(`(el) => {
		const rect = el.getBoundingClientRect();
		const pad = 12;
		const createCanvas = (w, h) => {
			const canvas = document.createElement('canvas');
			canvas.width = Math.max(1, Math.floor(w));
			canvas.height = Math.max(1, Math.floor(h));
			const ctx = canvas.getContext('2d');
			ctx.fillStyle = '#fff';
			ctx.fillRect(0, 0, canvas.width, canvas.height);
			return { canvas, ctx };
		};

		// 如果是 canvas 元素，直接获取数据
		if (el.tagName === 'CANVAS') {
			const sw = el.width || rect.width || 1;
			const sh = el.height || rect.height || 1;
			const out = createCanvas(sw + pad * 2, sh + pad * 2);
			out.ctx.drawImage(el, pad, pad, Math.floor(sw), Math.floor(sh));
			return out.canvas.toDataURL('image/png');
		}

		// 如果是 img 元素，绘制到 canvas
		if (el.tagName === 'IMG') {
			// 使用图片原始尺寸，避免显示尺寸导致右/下边缘被裁切。
			const sw = el.naturalWidth || el.width || rect.width || 1;
			const sh = el.naturalHeight || el.height || rect.height || 1;
			const out = createCanvas(sw + pad * 2, sh + pad * 2);
			out.ctx.drawImage(el, pad, pad, Math.floor(sw), Math.floor(sh));
			return out.canvas.toDataURL('image/png');
		}

		// 其他元素使用 html2canvas 或返回错误
		return null;
	}`)
	if err != nil {
		return nil, err
	}

	// 如果返回的是 data URL，解析它
	if dataURL, ok := result.(string); ok && len(dataURL) > 0 {
		// data:image/png;base64,iVBORw0KG...
		if len(dataURL) > 22 && dataURL[:22] == "data:image/png;base64," {
			return base64.StdEncoding.DecodeString(dataURL[22:])
		}
	}

	return nil, errors.New("failed to capture element screenshot")
}

func (b *browserElementAdapter) BoundingBox() (*browser.BoundingBox, error) {
	return b.element.BoundingBox()
}

// browserFrameAdapter 将 browser.Page (frame) 适配为 qrFrame 接口
type browserFrameAdapter struct {
	frame browser.Page
}

func (b *browserFrameAdapter) HasRegex(ctx context.Context, selector, jsRegex string) (bool, error) {
	return b.frame.HasRegex(selector, jsRegex)
}

func (b *browserFrameAdapter) Element(ctx context.Context, selector string) (qrElement, error) {
	el, err := b.frame.Element(selector)
	if err != nil {
		return nil, err
	}
	return &browserElementAdapter{element: el}, nil
}

func (b *browserFrameAdapter) Elements(ctx context.Context, selector string) ([]qrElement, error) {
	els, err := b.frame.Elements(selector)
	if err != nil {
		return nil, err
	}
	result := make([]qrElement, 0, len(els))
	for _, el := range els {
		result = append(result, &browserElementAdapter{element: el})
	}
	return result, nil
}

func (b *browserFrameAdapter) ElementByRegex(ctx context.Context, selector, jsRegex string) (qrElement, error) {
	el, err := b.frame.ElementByRegex(selector, jsRegex)
	if err != nil {
		return nil, err
	}
	return &browserElementAdapter{element: el}, nil
}

func (b *browserFrameAdapter) Frames(ctx context.Context) ([]qrFrame, error) {
	return framesForPage(ctx, b.frame)
}

func framesForPage(ctx context.Context, page browser.Page) ([]qrFrame, error) {
	frames := []qrFrame{}
	if page == nil {
		return frames, nil
	}

	// 获取所有 iframe 元素
	iframes, err := page.Elements("iframe")
	if err != nil {
		return frames, err
	}

	// 对于每个 iframe，获取其 frame page
	for _, el := range iframes {
		framePage, err := el.Frame()
		if err != nil {
			continue
		}
		frames = append(frames, &browserFrameAdapter{frame: framePage})
	}
	return frames, nil
}

type playwrightLoginSession struct {
	engine            browser.Engine
	page              qrPage
	pwPage            *playwrightPageWrapper // 保存包含 context 的包装器
	saveCookies       func() error
	sleep             func(time.Duration)
	initialWebSession string
}

// playwrightPageWrapper 包装 browser.Page 并保存 playwright context
type playwrightPageWrapper struct {
	page browser.Page
	ctx  playwright.BrowserContext
}

func newPlaywrightLoginSession() (loginSession, error) {
	engine := newBrowserEngine()
	if err := engine.Start(); err != nil {
		return nil, err
	}

	page, err := engine.NewPage()
	if err != nil {
		_ = engine.Close()
		return nil, err
	}

	// 尝试从 page 中获取 playwright context
	var pwCtx playwright.BrowserContext
	type pwPageInterface interface {
		GetContext() playwright.BrowserContext
	}
	if pwPage, ok := page.(pwPageInterface); ok {
		pwCtx = pwPage.GetContext()
	}

	wrapper := &playwrightPageWrapper{
		page: page,
		ctx:  pwCtx,
	}

	return &playwrightLoginSession{
		engine:      engine,
		page:        &browserPageAdapter{page: page},
		pwPage:      wrapper,
		saveCookies: func() error { return saveCookiesFromWrapper(wrapper) },
		sleep:       time.Sleep,
	}, nil
}

func (s *playwrightLoginSession) Open(ctx context.Context) error {
	if s.page == nil {
		return errors.New("login page not initialized")
	}
	if err := s.page.Navigate(ctx, xhsLoginURL); err != nil {
		return err
	}
	if err := s.page.WaitLoad(ctx); err != nil {
		return err
	}
	if s.sleep != nil {
		s.sleep(2 * time.Second)
	}
	if s.pwPage != nil && s.pwPage.ctx != nil {
		cks, cookieErr := s.pwPage.ctx.Cookies()
		if cookieErr == nil {
			for _, cookie := range cks {
				if cookie.Name == "web_session" {
					s.initialWebSession = cookie.Value
					break
				}
			}
		}
	}
	return nil
}

func (s *playwrightLoginSession) LoggedIn(ctx context.Context) (bool, error) {
	if s.page == nil {
		return false, errors.New("login page not initialized")
	}

	// 方法1: 检查 DOM 选择器
	ok, err := s.page.Has(ctx, loginStatusSelector)
	loginVisible, loginErr := s.loginContainerVisible(ctx)
	slog.Info("login status selector check", "login_status_selector", loginStatusSelector, "login_status_match", ok, "login_status_err", err, "login_container_match", loginVisible, "login_container_err", loginErr)

	// 页面选择器检查异常时上抛，避免把页面异常误报成“未登录”。
	if err != nil {
		return false, err
	}

	// 命中登录态且登录弹窗不可见，直接认为已登录。
	if ok && (loginErr != nil || !loginVisible) {
		return true, nil
	}

	// 当登录弹窗可见时再短暂复查一次，降低 SPA 过渡态误判。
	if ok && loginErr == nil && loginVisible && s.sleep != nil {
		s.sleep(500 * time.Millisecond)
		recheckOK, recheckErr := s.page.Has(ctx, loginStatusSelector)
		recheckLoginVisible, recheckLoginErr := s.loginContainerVisible(ctx)
		slog.Info("login status selector recheck", "login_status_match", recheckOK, "login_status_err", recheckErr, "login_container_match", recheckLoginVisible, "login_container_err", recheckLoginErr)
		if recheckErr == nil && recheckOK && (recheckLoginErr != nil || !recheckLoginVisible) {
			return true, nil
		}
	}

	// Anonymous visitors also receive a1 and web_session cookies. After the QR
	// confirmation, give Xiaohongshu time to finish the server-side session,
	// then reload the authenticated page. Persist cookies only after the real
	// signed-in navigation entry appears; never accept the transient cookie pair.
	if loginErr == nil && loginVisible {
		scanConfirmed, scanErr := s.page.HasRegex(ctx, "body", scanSuccessRegexp)
		slog.Info("login modal stabilization gate", "scan_confirmed", scanConfirmed, "scan_err", scanErr)
		if scanErr != nil || !scanConfirmed {
			changed, cookieErr := s.authCookieChanged()
			slog.Info("login session cookie change check", "changed", changed, "error", cookieErr)
			if cookieErr == nil && changed {
				return true, nil
			}
			// The mobile app may complete authentication before the QR page
			// updates its text or cookies. Reload once and verify the real
			// signed-in navigation state instead of waiting on stale UI.
		}
		if s.sleep != nil {
			s.sleep(3 * time.Second)
		}
		if err := s.page.Navigate(ctx, xhsLoginURL); err != nil {
			return false, err
		}
		if err := s.page.WaitLoad(ctx); err != nil {
			return false, err
		}
		if s.sleep != nil {
			s.sleep(2 * time.Second)
		}
		stableOK, stableErr := s.page.Has(ctx, loginStatusSelector)
		stableLoginVisible, stableLoginErr := s.loginContainerVisible(ctx)
		slog.Info("login status after stabilization", "login_status_match", stableOK, "login_status_err", stableErr, "login_container_match", stableLoginVisible, "login_container_err", stableLoginErr)
		if stableErr != nil {
			return false, stableErr
		}
		if stableOK && (stableLoginErr != nil || !stableLoginVisible) {
			return true, nil
		}
		return false, nil
	}

	// 方法2: 检查关键 cookies（备用方案）
	if s.pwPage != nil && s.pwPage.ctx != nil {
		cookies, cookieErr := s.pwPage.ctx.Cookies()
		if cookieErr == nil {
			hasWebSession := false
			hasA1 := false
			for _, c := range cookies {
				if c.Name == "web_session" && len(c.Value) > 20 {
					hasWebSession = true
				}
				if c.Name == "a1" && len(c.Value) > 20 {
					hasA1 = true
				}
			}
			cookieLoggedIn := hasWebSession && hasA1
			slog.Info("login status cookie check", "has_web_session", hasWebSession, "has_a1", hasA1, "cookie_logged_in", cookieLoggedIn)
			if cookieLoggedIn {
				return true, nil
			}
		}
	}

	return false, nil
}

func (s *playwrightLoginSession) authCookieChanged() (bool, error) {
	if s.pwPage == nil || s.pwPage.ctx == nil {
		return false, errors.New("playwright context is nil")
	}
	cks, err := s.pwPage.ctx.Cookies()
	if err != nil {
		return false, err
	}
	webSession := ""
	hasA1 := false
	for _, cookie := range cks {
		switch cookie.Name {
		case "web_session":
			webSession = cookie.Value
		case "a1":
			hasA1 = len(cookie.Value) > 20
		}
	}
	return hasA1 && len(webSession) > 20 && webSession != s.initialWebSession, nil
}

func (s *playwrightLoginSession) loginContainerVisible(ctx context.Context) (bool, error) {
	if s.pwPage != nil && s.pwPage.page != nil {
		visible, err := s.pwPage.page.WithContext(ctx).IsVisible(".login-container")
		if err == nil {
			return visible, nil
		}
		slog.Warn("login container visibility check failed, fallback to exists check", "error", err)
	}
	return s.page.Has(ctx, ".login-container")
}

func (s *playwrightLoginSession) Nickname(ctx context.Context) (string, error) {
	if s.pwPage == nil || s.pwPage.page == nil {
		return "", errors.New("login page not initialized")
	}

	baseNickname, profilePath, baseErr := s.nicknameFromExplore(ctx)
	if !isPlaceholderNicknameMain(baseNickname) {
		return baseNickname, nil
	}

	profileNickname := ""
	profileTitle := ""
	if profilePath != "" {
		pn, pt, profileErr := s.nicknameFromProfile(ctx, profilePath)
		if profileErr != nil && baseErr == nil {
			baseErr = profileErr
		}
		profileNickname = pn
		profileTitle = pt
	}

	if preferred := selectPreferredNicknameMain(baseNickname, profileNickname, profileTitle); !isPlaceholderNicknameMain(preferred) {
		return preferred, nil
	}

	creatorNickname, creatorErr := s.nicknameFromCreatorHome(ctx)
	if creatorErr == nil && !isPlaceholderNicknameMain(creatorNickname) {
		return creatorNickname, nil
	}

	if baseErr == nil {
		baseErr = creatorErr
	}
	if baseErr != nil {
		return "", baseErr
	}
	return "", errors.New("nickname unavailable")
}

func (s *playwrightLoginSession) nicknameFromExplore(ctx context.Context) (nickname, profilePath string, err error) {
	pp := s.pwPage.page.WithContext(ctx)
	raw, err := pp.Eval(`() => {
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
	if err != nil {
		return "", "", fmt.Errorf("extract nickname on explore failed: %w", err)
	}
	nickname, profilePath, _ = parseNicknameEvalResult(raw)
	return nickname, profilePath, nil
}

func (s *playwrightLoginSession) nicknameFromProfile(ctx context.Context, profilePath string) (nickname, title string, err error) {
	profileURL := profilePath
	if strings.HasPrefix(profileURL, "/") {
		profileURL = "https://www.xiaohongshu.com" + profileURL
	}
	if err := s.page.Navigate(ctx, profileURL); err != nil {
		return "", "", fmt.Errorf("navigate profile page failed: %w", err)
	}
	if err := s.page.WaitLoad(ctx); err != nil {
		return "", "", fmt.Errorf("wait profile page load failed: %w", err)
	}
	if s.sleep != nil {
		s.sleep(1 * time.Second)
	}

	pp := s.pwPage.page.WithContext(ctx)
	raw, err := pp.Eval(`() => {
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
	if err != nil {
		return "", "", fmt.Errorf("extract nickname on profile failed: %w", err)
	}

	nickname, _, title = parseNicknameEvalResult(raw)
	return nickname, title, nil
}

func (s *playwrightLoginSession) nicknameFromCreatorHome(ctx context.Context) (string, error) {
	if err := s.page.Navigate(ctx, xhsCreatorHomeURL); err != nil {
		return "", fmt.Errorf("navigate creator home failed: %w", err)
	}
	if err := s.page.WaitLoad(ctx); err != nil {
		return "", fmt.Errorf("wait creator home load failed: %w", err)
	}
	if s.sleep != nil {
		s.sleep(1500 * time.Millisecond)
	}

	pp := s.pwPage.page.WithContext(ctx)
	raw, err := pp.Eval(`async () => {
		const pick = (vals) => {
			for (const v of vals) {
				if (typeof v === "string" && v.trim()) return v.trim();
			}
			return "";
		};
		const keySet = new Set(["nickname", "nickName", "userName", "username", "name", "displayName"]);
		const deepFind = (obj, depth = 0) => {
			if (depth > 5 || obj == null) return "";
			if (Array.isArray(obj)) {
				for (const item of obj) {
					const v = deepFind(item, depth + 1);
					if (v) return v;
				}
				return "";
			}
			if (typeof obj !== "object") return "";
			for (const [k, v] of Object.entries(obj)) {
				if (keySet.has(k) && typeof v === "string" && v.trim()) return v.trim();
			}
			for (const v of Object.values(obj)) {
				const found = deepFind(v, depth + 1);
				if (found) return found;
			}
			return "";
		};

		const domNickname = pick([
			document.querySelector(".user-name")?.textContent,
			document.querySelector('[class*="nickname"]')?.textContent,
			document.querySelector('[class*="user-name"]')?.textContent,
			document.querySelector('[class*="avatar"] + div')?.textContent,
		]);

		const stateNickname = pick([
			deepFind(window.__INITIAL_STATE__),
			deepFind(window.__NUXT__),
			deepFind(window.__NEXT_DATA__),
			deepFind(window.__STORE__),
			deepFind(window.__APOLLO_STATE__),
		]);
		if (stateNickname) {
			return { nickname: stateNickname, title: (document.title || "").trim() };
		}
		if (domNickname) {
			return { nickname: domNickname, title: (document.title || "").trim() };
		}

		const endpoints = [
			"/api/galaxy/creator/home/base_info",
			"/api/galaxy/creator/home/user_info",
			"/api/sns/web/v1/user/me",
			"/api/sns/web/v1/user/profile",
		];
		for (const endpoint of endpoints) {
			try {
				const resp = await fetch(endpoint, { credentials: "include" });
				if (!resp || !resp.ok) continue;
				const text = await resp.text();
				if (!text) continue;
				let data = null;
				try {
					data = JSON.parse(text);
				} catch (_) {
					continue;
				}
				const apiNickname = deepFind(data);
				if (apiNickname) {
					return { nickname: apiNickname, title: (document.title || "").trim() };
				}
			} catch (_) {}
		}
		return { nickname: "", title: (document.title || "").trim() };
	}`)
	if err != nil {
		return "", fmt.Errorf("extract nickname on creator home failed: %w", err)
	}

	nickname, _, title := parseNicknameEvalResult(raw)
	preferred := selectPreferredNicknameMain("", nickname, title)
	if isPlaceholderNicknameMain(preferred) {
		return "", errors.New("nickname unavailable on creator home")
	}
	return preferred, nil
}

func parseNicknameEvalResult(raw interface{}) (nickname, profilePath, title string) {
	m, ok := raw.(map[string]interface{})
	if !ok {
		return "", "", ""
	}
	if s, ok := m["nickname"].(string); ok {
		nickname = strings.TrimSpace(s)
	}
	if s, ok := m["profilePath"].(string); ok {
		profilePath = strings.TrimSpace(s)
	}
	if s, ok := m["title"].(string); ok {
		title = strings.TrimSpace(s)
	}
	return nickname, profilePath, title
}

func isPlaceholderNicknameMain(name string) bool {
	n := strings.TrimSpace(strings.ToLower(name))
	switch n {
	case "", "我", "我的", "me", "my":
		return true
	default:
		return false
	}
}

func nicknameFromTitleMain(title string) string {
	t := strings.TrimSpace(title)
	if t == "" {
		return ""
	}
	if !strings.HasSuffix(t, "- 小红书") {
		return ""
	}
	t = strings.TrimSuffix(t, "- 小红书")
	return strings.TrimSpace(t)
}

func selectPreferredNicknameMain(baseNickname, profileNickname, profileTitle string) string {
	base := strings.TrimSpace(baseNickname)
	if !isPlaceholderNicknameMain(base) {
		return base
	}
	profile := strings.TrimSpace(profileNickname)
	if !isPlaceholderNicknameMain(profile) {
		return profile
	}
	titleName := strings.TrimSpace(nicknameFromTitleMain(profileTitle))
	if !isPlaceholderNicknameMain(titleName) {
		return titleName
	}
	return ""
}

func (s *playwrightLoginSession) QRCode(ctx context.Context) (loginQRCode, error) {
	if s.page == nil {
		return loginQRCode{}, errors.New("login page not initialized")
	}
	stage := "login"
	if s.hasSecurityHint(ctx) {
		stage = "security"
	}

	slog.Info("login qrcode stage detect", "stage", stage)

	if forceFullPageQRCode {
		shotter, ok := s.page.(fullPageScreenshotter)
		if ok {
			img, shotErr := fullPageScreenshotBase64(shotter)
			if shotErr == nil {
				slog.Info("login qrcode captured", "source", "full_page_forced", "stage", stage)
				return loginQRCode{
					Image: img,
					Stage: stage,
				}, nil
			}
		}
	}

	el, err := s.findQRCodeElement(ctx, stage == "security")
	if err != nil {
		shotter, ok := s.page.(fullPageScreenshotter)
		if !ok {
			return loginQRCode{}, err
		}
		img, shotErr := fullPageScreenshotBase64(shotter)
		if shotErr != nil {
			return loginQRCode{}, err
		}
		slog.Info("login qrcode screenshot fallback", "source", "full_page")
		return loginQRCode{
			Image: img,
			Stage: stage,
		}, nil
	}

	img, err := el.Screenshot()
	if err != nil {
		return loginQRCode{}, err
	}
	if withQuietZone, qErr := ensurePNGQuietZone(img, 12); qErr == nil {
		img = withQuietZone
	} else {
		slog.Warn("ensure qrcode quiet zone failed", "error", qErr)
	}

	source := "login_qrcode"
	if stage == "security" {
		source = "security_qrcode"
	}
	slog.Info("login qrcode captured", "source", source, "stage", stage)

	return loginQRCode{
		Image: base64.StdEncoding.EncodeToString(img),
		Stage: stage,
	}, nil
}

// ensurePNGQuietZone guarantees a white border around QR images so decoders can
// reliably detect finder patterns even when the source image is tightly cropped.
func ensurePNGQuietZone(pngData []byte, minPadding int) ([]byte, error) {
	if len(pngData) == 0 {
		return pngData, errors.New("empty png data")
	}
	if minPadding <= 0 {
		minPadding = 12
	}

	img, err := png.Decode(bytes.NewReader(pngData))
	if err != nil {
		return nil, err
	}
	if !pngTouchesDarkEdge(img) {
		return pngData, nil
	}

	b := img.Bounds()
	w, h := b.Dx(), b.Dy()
	dst := image.NewRGBA(image.Rect(0, 0, w+minPadding*2, h+minPadding*2))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{color.White}, image.Point{}, draw.Src)
	draw.Draw(dst, image.Rect(minPadding, minPadding, minPadding+w, minPadding+h), img, b.Min, draw.Src)

	var out bytes.Buffer
	if err := png.Encode(&out, dst); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func pngTouchesDarkEdge(img image.Image) bool {
	b := img.Bounds()
	isDark := func(x, y int) bool {
		r, g, bb, _ := img.At(x, y).RGBA()
		rr := uint8(r >> 8)
		gg := uint8(g >> 8)
		bb8 := uint8(bb >> 8)
		return rr < 200 || gg < 200 || bb8 < 200
	}

	for x := b.Min.X; x < b.Max.X; x++ {
		if isDark(x, b.Min.Y) || isDark(x, b.Max.Y-1) {
			return true
		}
	}
	for y := b.Min.Y; y < b.Max.Y; y++ {
		if isDark(b.Min.X, y) || isDark(b.Max.X-1, y) {
			return true
		}
	}
	return false
}

func fullPageScreenshotBase64(page fullPageScreenshotter) (string, error) {
	tmp, err := os.CreateTemp("", "xhs-login-*.png")
	if err != nil {
		return "", err
	}
	path := tmp.Name()
	if err := tmp.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	defer os.Remove(path)
	if err := page.ScreenshotFullPage(path); err != nil {
		return "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

func (s *playwrightLoginSession) SaveCookies() error {
	if s.saveCookies == nil {
		return nil
	}
	return s.saveCookies()
}

func (s *playwrightLoginSession) Close() error {
	if s.page != nil {
		_ = s.page.Close()
	}
	if s.engine != nil {
		s.engine.Close()
	}
	return nil
}

func (s *playwrightLoginSession) hasSecurityHint(ctx context.Context) bool {
	ok, err := s.page.HasRegex(ctx, "body", securityHintRegexp)
	scanOK, scanErr := s.page.HasRegex(ctx, "body", scanSuccessRegexp)
	slog.Info("login qrcode security hint on page", "match", ok, "err", err, "scan_match", scanOK, "scan_err", scanErr, "scan_regex", scanSuccessRegexp, "security_re", securityHintRegexp)
	if err == nil && ok {
		return true
	}
	return s.frameHasSecurityHint(ctx, s.page, 0)
}

func (s *playwrightLoginSession) findQRCodeElement(ctx context.Context, preferFrames bool) (qrElement, error) {
	if preferFrames {
		if el, ok := s.findQRCodeElementInChildFrames(ctx, s.page); ok {
			slog.Info("login qrcode element found", "source", "frame")
			return el, nil
		}
	}

	for _, selector := range qrSelectors {
		el, err := s.findLargestElement(ctx, s.page, selector)
		if err == nil && el != nil {
			slog.Info("login qrcode element found", "source", "page", "selector", selector)
			return el, nil
		}
	}

	el, err := s.page.ElementByRegex(ctx, "div", qrFallbackRegex)
	if err == nil && el != nil {
		slog.Info("login qrcode element found", "source", "page_fallback")
		return el, nil
	}

	if !preferFrames {
		if el, ok := s.findQRCodeElementInChildFrames(ctx, s.page); ok {
			slog.Info("login qrcode element found", "source", "frame")
			return el, nil
		}
	}

	slog.Info("login qrcode element not found", "prefer_frames", preferFrames)
	return nil, errors.New("login qrcode element not found")
}

func (s *playwrightLoginSession) frameHasSecurityHint(ctx context.Context, frame qrFrame, depth int) bool {
	if depth >= maxFrameDepth {
		slog.Warn("login qrcode frame scan depth exceeded", "depth", depth)
		return false
	}
	frames, err := frame.Frames(ctx)
	slog.Debug("login qrcode scan frames", "count", len(frames), "err", err, "depth", depth)
	if err != nil {
		return false
	}
	for _, child := range frames {
		ok, err := child.HasRegex(ctx, "body", securityHintRegexp)
		scanOK, scanErr := child.HasRegex(ctx, "body", scanSuccessRegexp)
		slog.Debug("login qrcode security hint on frame", "match", ok, "err", err, "scan_match", scanOK, "scan_err", scanErr, "scan_regex", scanSuccessRegexp, "security_re", securityHintRegexp, "depth", depth)
		if err == nil && ok {
			return true
		}
		if s.frameHasSecurityHint(ctx, child, depth+1) {
			return true
		}
	}
	return false
}

func (s *playwrightLoginSession) findQRCodeElementInFrame(ctx context.Context, frame qrFrame, depth int) (qrElement, bool) {
	if depth >= maxFrameDepth {
		return nil, false
	}
	for _, selector := range qrSelectors {
		el, err := s.findLargestElement(ctx, frame, selector)
		if err == nil && el != nil {
			return el, true
		}
	}

	el, err := frame.ElementByRegex(ctx, "div", qrFallbackRegex)
	if err == nil && el != nil {
		return el, true
	}

	frames, err := frame.Frames(ctx)
	if err != nil {
		return nil, false
	}
	for _, child := range frames {
		if el, ok := s.findQRCodeElementInFrame(ctx, child, depth+1); ok {
			return el, true
		}
	}
	return nil, false
}

func (s *playwrightLoginSession) findQRCodeElementInChildFrames(ctx context.Context, frame qrFrame) (qrElement, bool) {
	frames, err := frame.Frames(ctx)
	if err != nil {
		return nil, false
	}
	for _, child := range frames {
		if el, ok := s.findQRCodeElementInFrame(ctx, child, 0); ok {
			return el, true
		}
	}
	return nil, false
}

func (s *playwrightLoginSession) findLargestElement(ctx context.Context, frame qrFrame, selector string) (qrElement, error) {
	els, err := frame.Elements(ctx, selector)
	if err != nil {
		return nil, err
	}
	best := pickLargestElement(els)
	if best != nil {
		return best, nil
	}
	return frame.Element(ctx, selector)
}

func pickLargestElement(els []qrElement) qrElement {
	var best qrElement
	var bestArea float64
	for _, el := range els {
		box, err := el.BoundingBox()
		if err != nil || box == nil {
			continue
		}
		area := box.Width * box.Height
		if area <= 0 {
			continue
		}
		if best == nil || area > bestArea {
			best = el
			bestArea = area
		}
	}
	return best
}

// saveCookiesFromWrapper 保存 Playwright 的 cookies
func saveCookiesFromWrapper(wrapper *playwrightPageWrapper) error {
	if wrapper == nil || wrapper.ctx == nil {
		return errors.New("playwright context is nil")
	}

	cks, err := wrapper.ctx.Cookies()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cks)
	if err != nil {
		return err
	}

	cookiePath := cookies.GetCookiesFilePath()
	if err := os.WriteFile(cookiePath, data, 0600); err != nil {
		return err
	}
	// WriteFile keeps the mode of an existing file, so enforce it explicitly.
	return os.Chmod(cookiePath, 0600)
}
