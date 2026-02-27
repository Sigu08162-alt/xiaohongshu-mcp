package main

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
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
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			"--disable-blink-features=AutomationControlled",
		},
	})
	if err != nil {
		panic(err)
	}
	defer browser.Close()

	ctx, err := browser.NewContext(playwright.BrowserNewContextOptions{
		Viewport:  &playwright.Size{Width: 1920, Height: 1080},
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

	// 拦截所有 API 请求
	apiCalls := []string{}
	apiResponses := map[string]string{}

	page.On("request", func(req playwright.Request) {
		url := req.URL()
		if strings.Contains(url, "xiaohongshu.com") && !strings.Contains(url, ".js") && !strings.Contains(url, ".css") && !strings.Contains(url, ".png") && !strings.Contains(url, ".jpg") {
			entry := fmt.Sprintf("[%s] %s", req.Method(), url)
			apiCalls = append(apiCalls, entry)
		}
	})

	page.On("response", func(resp playwright.Response) {
		url := resp.URL()
		if strings.Contains(url, "xiaohongshu.com") && !strings.Contains(url, ".js") && !strings.Contains(url, ".css") && !strings.Contains(url, ".png") && !strings.Contains(url, ".jpg") {
			body, err := resp.Text()
			status := resp.Status()
			if err == nil {
				preview := body
				if len(preview) > 300 {
					preview = preview[:300]
				}
				apiResponses[url] = fmt.Sprintf("status=%d body=%s", status, preview)
			}
		}
	})

	fmt.Println("=== 导航到创作者中心首页 ===")
	if _, err := page.Goto("https://creator.xiaohongshu.com/new/home?source=official", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		Timeout:   playwright.Float(20000),
	}); err != nil {
		fmt.Println("导航失败(忽略):", err)
	}
	time.Sleep(3 * time.Second)

	fmt.Println("当前 URL:", page.URL())
	title, _ := page.Title()
	fmt.Println("页面标题:", title)

	fmt.Println("\n=== 捕获到的 API 请求 ===")
	for _, c := range apiCalls {
		fmt.Println(c)
	}

	fmt.Println("\n=== API 响应详情 ===")
	for url, resp := range apiResponses {
		fmt.Printf("URL: %s\n  %s\n\n", url, resp)
	}

	// 截图
	page.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String("/tmp/creator_home.png")})
	fmt.Println("截图已保存: /tmp/creator_home.png")
}
