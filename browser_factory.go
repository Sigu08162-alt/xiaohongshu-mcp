package main

import (
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/configs"
	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser/playwright"
)

// newBrowserEngine 创建 Playwright 浏览器引擎
func newBrowserEngine() browser.Engine {
	cfg := playwright.DefaultConfig()
	cfg.Headless = configs.IsHeadless()
	cfg.CookiePath = cookies.GetCookiesFilePath()
	// 降低全局操作超时从30秒到10秒，重要操作通过 WithTimeout 单独控制
	cfg.ActionTimeout = 10 * time.Second
	cfg.NavigationTimeout = 60 * time.Second

	return playwright.New(cfg)
}

// withBrowserPage 执行需要浏览器页面的操作的通用函数
func withBrowserPage(fn func(browser.Page) error) error {
	engine := newBrowserEngine()
	if err := engine.Start(); err != nil {
		return err
	}
	defer engine.Close()

	page, err := engine.NewPage()
	if err != nil {
		return err
	}
	defer page.Close()

	return fn(page)
}
