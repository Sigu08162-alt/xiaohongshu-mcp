package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/playwright-community/playwright-go"
)

func main() {
	pw, _ := playwright.Run()
	defer pw.Stop()

	browser, _ := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	defer browser.Close()

	data, _ := os.ReadFile("/tmp/cookies.json")
	var rawCookies []map[string]interface{}
	json.Unmarshal(data, &rawCookies)

	ctx, _ := browser.NewContext(playwright.BrowserNewContextOptions{
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"),
	})

	var pwCookies []playwright.OptionalCookie
	for _, c := range rawCookies {
		name, _ := c["name"].(string)
		value, _ := c["value"].(string)
		domain, _ := c["domain"].(string)
		path, _ := c["path"].(string)
		httpOnly, _ := c["httpOnly"].(bool)
		secure, _ := c["secure"].(bool)
		pwCookies = append(pwCookies, playwright.OptionalCookie{
			Name: name, Value: value,
			Domain: playwright.String(domain), Path: playwright.String(path),
			HttpOnly: playwright.Bool(httpOnly), Secure: playwright.Bool(secure),
		})
	}
	ctx.AddCookies(pwCookies)

	page, _ := ctx.NewPage()

	// 方案A: 直接 goto，等待更长时间
	fmt.Println("=== 方案A: direct goto with longer wait ===")
	noteURL := "https://www.xiaohongshu.com/explore/699d460b000000000a03366d?xsec_token=ABVIZLoe7nWU09dUaBzsKdllrETDocbeb9hUj3enwvKSM=&xsec_source=pc_feed"
	page.Goto(noteURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(30000),
	})

	// 等待 Vue 路由完成
	for i := 0; i < 15; i++ {
		time.Sleep(1 * time.Second)
		url := page.URL()
		count, _ := page.Locator("#content-textarea, [class*='note-detail'], [class*='NoteDetail']").Count()
		fmt.Printf("  t+%ds: url=%s elements=%d\n", i+1, url, count)
		if count > 0 {
			fmt.Println("  -> Note loaded!")
			break
		}
	}

	// 方案B: 用 window.location 强制跳转
	fmt.Println("\n=== 方案B: window.location navigation ===")
	page.Evaluate(fmt.Sprintf(`() => { window.location.href = %q; }`, noteURL))
	time.Sleep(5 * time.Second)

	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)
		url := page.URL()
		count, _ := page.Locator("#content-textarea").Count()
		fmt.Printf("  t+%ds: url=%s textarea=%d\n", i+1, url, count)
		if count > 0 {
			break
		}
	}

	// dump what we have
	result, _ := page.Evaluate(`() => {
		const el = document.querySelector('#content-textarea');
		const noteEl = document.querySelector('[class*="note-detail"], [class*="NoteDetail"], .note-container');
		return {
			url: window.location.href,
			hasTextarea: !!el,
			hasNote: !!noteEl,
			noteClass: noteEl?.className?.substring(0,80) || '',
			allContentEditable: Array.from(document.querySelectorAll('[contenteditable]')).map(e => ({id:e.id, class:(e.className||'').substring(0,50)})),
		};
	}`)
	fmt.Println("\n=== Final state ===")
	fmt.Println(result)

	page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String("/tmp/xhs_note_b.png"), FullPage: playwright.Bool(false)})
	fmt.Println("Screenshot: /tmp/xhs_note_b.png")
}
