package main

import (
	"context"
	"encoding/json"
	"flag"
	"time"

	playwrightgo "github.com/playwright-community/playwright-go"
	"log/slog"
	"github.com/vmxmy/xiaohongshu-mcp/configs"
	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser/playwright"
	infraconfig "github.com/vmxmy/xiaohongshu-mcp/internal/infra/config"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

func main() {
	var (
		binPath string // 浏览器二进制文件路径
	)
	flag.StringVar(&binPath, "bin", "", "浏览器二进制文件路径")
	flag.Parse()

	// 登录的时候，需要界面，所以不能无头模式
	engine := newBrowserEngine()
	if err := engine.Start(); err != nil {
		slog.Error("failed to start browser:", "arg1", err)
	}
	defer engine.Close()

	page, err := engine.NewPage()
	if err != nil {
		slog.Error("failed to create page:", "arg1", err)
	}
	defer page.Close()

	pollingModule, err := loadAuthPollingModule()
	if err != nil {
		slog.Error("加载轮询配置失败:", "arg1", err)
	}
	action := xiaohongshu.NewLogin(page, pollingModule)

	status, err := action.CheckLoginStatus(context.Background())
	if err != nil {
		slog.Error("failed to check login status:", "arg1", err)
	}

	slog.Info("当前登录状态:", "arg1", status)

	if status {
		return
	}

	// 开始登录流程
	slog.Info("开始登录流程...")
	if err = action.Login(context.Background()); err != nil {
		slog.Error("登录失败:", "arg1", err)
	} else {
		if err := saveCookies(page); err != nil {
			slog.Error("failed to save cookies:", "arg1", err)
		}
	}

	// 再次检查登录状态确认成功
	status, err = action.CheckLoginStatus(context.Background())
	if err != nil {
		slog.Error("failed to check login status after login:", "arg1", err)
	}

	if status {
		slog.Info("登录成功！")
	} else {
		slog.Error("登录流程完成但仍未登录")
	}

}

func loadAuthPollingModule() (polling.Module, error) {
	cfg := infraconfig.DefaultConfig()
	return polling.Module{
		TimeoutMs:  cfg.Polling.Auth.TimeoutMs,
		IntervalMs: cfg.Polling.Auth.IntervalMs,
		MaxRetries: cfg.Polling.Auth.MaxRetries,
		Delays:     cfg.Polling.Auth.Delays,
	}, nil
}

// newBrowserEngine 创建 Playwright 浏览器引擎
func newBrowserEngine() browser.Engine {
	cfg := playwright.DefaultConfig()
	cfg.Headless = configs.IsHeadless()
	cfg.CookiePath = cookies.GetCookiesFilePath()
	cfg.ActionTimeout = 30 * time.Second
	cfg.NavigationTimeout = 60 * time.Second

	return playwright.New(cfg)
}

func saveCookies(page browser.Page) error {
	// 将 browser.Page 转换为 Playwright 的具体实现，获取 context
	type contextGetter interface {
		GetContext() playwrightgo.BrowserContext
	}

	pg, ok := page.(contextGetter)
	if !ok {
		slog.Warn("无法获取 Playwright context，跳过保存 cookies")
		return nil
	}

	ctx := pg.GetContext()
	if ctx == nil {
		return nil
	}

	cks, err := ctx.Cookies()
	if err != nil {
		return err
	}

	data, err := json.Marshal(cks)
	if err != nil {
		return err
	}

	cookieLoader := cookies.NewLoadCookie(cookies.GetCookiesFilePath())
	return cookieLoader.SaveCookies(data)
}
