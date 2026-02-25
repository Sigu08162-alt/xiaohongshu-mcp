//go:build integration

package integration

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser/playwright"
	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	"os"
)

// TestDOMDebug_DeleteButton 调试删除按钮的真实 DOM 结构
// 运行: COOKIES_PATH=./cookies.json go test -tags integration -run TestDOMDebug_DeleteButton -v -timeout 120s
func TestDOMDebug_DeleteButton(t *testing.T) {
	feedID := "699e7a9e000000002603c023"
	xsecToken := "ABXh8X-YqIw5-Nvk1NxFfQpZO5ki3I6UZ00vbWtu5trDM="

	cookiePath := cookies.GetCookiesFilePath()
	if _, err := os.Stat(cookiePath); os.IsNotExist(err) {
		t.Skipf("cookies.json not found at %s", cookiePath)
	}

	cfg := playwright.DefaultConfig()
	cfg.Headless = true
	cfg.CookiePath = cookiePath
	cfg.NavigationTimeout = 30 * time.Second
	cfg.ActionTimeout = 15 * time.Second

	engine := playwright.New(cfg)
	if err := engine.Start(); err != nil {
		t.Fatalf("启动浏览器失败: %v", err)
	}
	defer engine.Close()

	page, err := engine.NewPage()
	if err != nil {
		t.Fatalf("创建页面失败: %v", err)
	}

	url := fmt.Sprintf("https://www.xiaohongshu.com/explore/%s?xsec_token=%s&xsec_source=pc_feed", feedID, xsecToken)
	t.Logf("访问: %s", url)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	p := page.WithContext(ctx).WithTimeout(30 * time.Second)

	if err := p.Goto(url); err != nil {
		t.Logf("导航警告: %v", err)
	}
	time.Sleep(4 * time.Second)

	// 截图（打开前）
	_ = p.Screenshot("/tmp/dom_before.png")
	t.Log("截图已保存: /tmp/dom_before.png")

	// 打印所有 button
	t.Log("\n=== 所有 button 元素 ===")
	btnResult, _ := p.Eval(`() => {
		return [...document.querySelectorAll('button')].map(b => ({
			text: b.innerText?.trim().slice(0, 50),
			class: b.className,
			ariaLabel: b.getAttribute('aria-label'),
			dataTestId: b.getAttribute('data-testid'),
			html: b.outerHTML.slice(0, 200),
		}));
	}`)
	t.Logf("buttons: %v", btnResult)

	// 打印含 more/menu/operate/icon 的元素
	t.Log("\n=== class 含 more/menu/operate/icon 的元素 ===")
	moreResult, _ := p.Eval(`() => {
		return [...document.querySelectorAll('*')]
			.filter(e => e.className && typeof e.className === 'string' &&
				/more|operate|menu|icon|ellipsis|dot/i.test(e.className))
			.slice(0, 30)
			.map(e => ({
				tag: e.tagName,
				class: e.className,
				text: e.innerText?.trim().slice(0, 30),
				ariaLabel: e.getAttribute('aria-label'),
				html: e.outerHTML.slice(0, 200),
			}));
	}`)
	t.Logf("more/menu elements: %v", moreResult)

	// 尝试点击 .menu-icon-btn
	t.Log("\n=== 尝试点击 .menu-icon-btn ===")
	clickResult, _ := p.Eval(`() => {
		const el = document.querySelector('.menu-icon-btn');
		if (el) { el.click(); return 'FOUND and clicked: ' + el.outerHTML.slice(0, 150); }
		return 'NOT FOUND: .menu-icon-btn';
	}`)
	t.Logf("click result: %v", clickResult)

	time.Sleep(2 * time.Second)
	_ = p.Screenshot("/tmp/dom_after_click.png")
	t.Log("点击后截图: /tmp/dom_after_click.png")

	// 打印点击后含"删除"文字的元素
	t.Log("\n=== 点击后含'删除'文字的元素 ===")
	deleteResult, _ := p.Eval(`() => {
		return [...document.querySelectorAll('*')]
			.filter(e => e.innerText && e.innerText.trim() === '删除')
			.map(e => ({
				tag: e.tagName,
				class: e.className,
				ariaLabel: e.getAttribute('aria-label'),
				html: e.outerHTML.slice(0, 300),
			}));
	}`)
	t.Logf("delete elements: %v", deleteResult)

	// 打印点击后所有可见文本
	t.Log("\n=== 点击后菜单可见文本（li/div/span/button） ===")
	menuResult, _ := p.Eval(`() => {
		return [...document.querySelectorAll('li, [role="menuitem"], [class*="item"], [class*="menu"]')]
			.filter(e => e.innerText?.trim().length > 0)
			.slice(0, 20)
			.map(e => ({
				tag: e.tagName,
				class: e.className,
				text: e.innerText.trim().slice(0, 50),
				html: e.outerHTML.slice(0, 200),
			}));
	}`)
	t.Logf("menu items: %v", menuResult)
}
