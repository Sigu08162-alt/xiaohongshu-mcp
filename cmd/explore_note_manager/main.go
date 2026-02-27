//go:build ignore

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {
	cookiesPath := os.Getenv("COOKIES_PATH")
	if cookiesPath == "" {
		cookiesPath = "./cookies.json"
	}

	raw, err := os.ReadFile(cookiesPath)
	if err != nil {
		panic(err)
	}

	var rawCookies []map[string]interface{}
	if err := json.Unmarshal(raw, &rawCookies); err != nil {
		panic(err)
	}

	pw, err := playwright.Run()
	if err != nil {
		panic(err)
	}
	defer pw.Stop()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
	})
	if err != nil {
		panic(err)
	}
	defer browser.Close()

	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{Width: 1920, Height: 1080},
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	})
	if err != nil {
		panic(err)
	}

	// 设置 cookies
	var pwCookies []playwright.OptionalCookie
	for _, c := range rawCookies {
		name, _ := c["name"].(string)
		value, _ := c["value"].(string)
		domain, _ := c["domain"].(string)
		path, _ := c["path"].(string)
		if path == "" {
			path = "/"
		}
		pwCookies = append(pwCookies, playwright.OptionalCookie{
			Name:   name,
			Value:  value,
			Domain: playwright.String(domain),
			Path:   playwright.String(path),
		})
	}
	if err := ctx.AddCookies(pwCookies); err != nil {
		panic(err)
	}

	page, err := ctx.NewPage()
	if err != nil {
		panic(err)
	}

	fmt.Println("=== 打开创作者中心笔记管理页 ===")
	if _, err := page.Goto("https://creator.xiaohongshu.com/new/note-manager", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	}); err != nil {
		fmt.Println("导航失败:", err)
	}
	time.Sleep(3 * time.Second)

	// 截图
	page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String("/tmp/note_manager_loaded.png")})
	fmt.Println("截图已保存: /tmp/note_manager_loaded.png")
	fmt.Println("当前 URL:", page.URL())
	title, _ := page.Title()
	fmt.Println("页面标题:", title)

	// 打印笔记列表区域 HTML
	result, err := page.Evaluate(`() => {
		const selectors = [
			'.note-list', '.note-manager', '[class*="note-item"]',
			'[class*="content-item"]', '[class*="publish"]',
			'.main', 'main', '#app'
		];
		for (const sel of selectors) {
			const el = document.querySelector(sel);
			if (el) return '[' + sel + '] found:\n' + el.innerHTML.slice(0, 8000);
		}
		return 'body:\n' + document.body.innerHTML.slice(0, 8000);
	}`)
	if err == nil {
		fmt.Println("\n=== 页面 HTML 结构 ===")
		fmt.Println(result)
	}

	// 列出所有可见按钮和操作元素
	btns, err := page.Evaluate(`() => {
		return [...document.querySelectorAll('button, [role="button"], [class*="btn"], [class*="operate"], [class*="action"], [class*="more"], [class*="delete"]')]
			.filter(el => el.offsetParent !== null)
			.map(el => 'tag=' + el.tagName + ' class="' + el.className + '" text="' + (el.innerText||'').trim().slice(0,30) + '"')
			.join('\n');
	}`)
	if err == nil {
		fmt.Println("\n=== 可见按钮/操作元素 ===")
		fmt.Println(btns)
	}
}
