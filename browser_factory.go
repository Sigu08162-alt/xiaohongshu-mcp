package main

import (
	"context"
	"sync"
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/configs"
	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser/playwright"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser/pool"
)

// globalPool is the shared browser page pool for all MCP/API requests.
// It is initialised once on first use via sync.Once.
var (
	globalPool     *pool.BrowserPool
	globalPoolOnce sync.Once
)

// newBrowserEngine creates a configured Playwright engine (used by pool and login).
func newBrowserEngine() browser.Engine {
	cfg := playwright.DefaultConfig()
	cfg.Headless = configs.IsHeadless()
	cfg.CookiePath = cookies.GetCookiesFilePath()
	cfg.ActionTimeout = 10 * time.Second
	cfg.NavigationTimeout = 60 * time.Second
	return playwright.New(cfg)
}

// getPool returns the singleton BrowserPool, starting it on first call.
func getPool() *pool.BrowserPool {
	globalPoolOnce.Do(func() {
		globalPool = pool.New(newBrowserEngine(), 2)
	})
	return globalPool
}

// withBrowserPage acquires a page from the shared pool, runs fn, then releases
// the page back. On error the page is discarded so a fresh one is created next time.
func withBrowserPage(fn func(browser.Page) error) error {
	return getPool().WithPage(context.Background(), fn)
}
