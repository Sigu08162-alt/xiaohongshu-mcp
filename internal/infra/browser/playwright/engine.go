package playwright

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

type Config struct {
	Headless          bool
	ActionTimeout     time.Duration
	NavigationTimeout time.Duration
	CookiePath        string
	ViewportWidth     int
	ViewportHeight    int
}

func DefaultConfig() Config {
	width := getEnvInt("XHS_VIEWPORT_WIDTH", 1920)
	height := getEnvInt("XHS_VIEWPORT_HEIGHT", 1080)
	return Config{
		Headless:       true,
		ViewportWidth:  width,
		ViewportHeight: height,
	}
}

type Engine struct {
	cfg     Config
	pw      *playwright.Playwright
	browser playwright.Browser
}

func New(cfg Config) *Engine {
	return &Engine{cfg: cfg}
}

func (e *Engine) Start() error {
	// 尝试运行playwright，如果驱动未安装则自动安装
	pw, err := playwright.Run()
	if err != nil {
		// 检查是否是驱动未安装的错误
		if strings.Contains(err.Error(), "driver not installed") || strings.Contains(err.Error(), "please install") {
			// 自动安装驱动
			if installErr := playwright.Install(); installErr != nil {
				return fmt.Errorf("自动安装playwright驱动失败: %w", installErr)
			}
			// 重试运行
			pw, err = playwright.Run()
			if err != nil {
				return wrapPlaywrightError(err)
			}
		} else {
			return wrapPlaywrightError(err)
		}
	}
	b, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(e.cfg.Headless),
	})
	if err != nil {
		_ = pw.Stop()
		return err
	}
	e.pw = pw
	e.browser = b
	return nil
}

func (e *Engine) NewPage() (browser.Page, error) {
	if e.browser == nil {
		return nil, errors.New("browser not started")
	}

	// 创建上下文选项，设置视口大小（确保无头模式下有足够大的视口）
	viewportWidth := e.cfg.ViewportWidth
	viewportHeight := e.cfg.ViewportHeight
	if viewportWidth <= 0 {
		viewportWidth = 1920
	}
	if viewportHeight <= 0 {
		viewportHeight = 1080
	}
	contextOptions := playwright.BrowserNewContextOptions{
		Viewport: &playwright.Size{
			Width:  viewportWidth,
			Height: viewportHeight,
		},
		// 设置 User-Agent，避免被检测为自动化
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
	}

	if e.cfg.CookiePath != "" {
		ctx, err := e.browser.NewContext(contextOptions)
		if err != nil {
			return nil, err
		}
		cookies, err := loadCookies(e.cfg.CookiePath)
		if err != nil {
			_ = ctx.Close()
			return nil, err
		}
		if len(cookies) > 0 {
			if err := ctx.AddCookies(cookies); err != nil {
				_ = ctx.Close()
				return nil, err
			}
		}
		p, err := ctx.NewPage()
		if err != nil {
			_ = ctx.Close()
			return nil, err
		}
		applyTimeouts(p, e.cfg)
		return newPage(p, ctx), nil
	}

	ctx, err := e.browser.NewContext(contextOptions)
	if err != nil {
		return nil, err
	}
	p, err := ctx.NewPage()
	if err != nil {
		_ = ctx.Close()
		return nil, err
	}
	applyTimeouts(p, e.cfg)
	return newPage(p, ctx), nil
}

func (e *Engine) Close() error {
	if e.browser != nil {
		_ = e.browser.Close()
	}
	if e.pw != nil {
		return e.pw.Stop()
	}
	return nil
}

func applyTimeouts(p playwright.Page, cfg Config) {
	if cfg.ActionTimeout > 0 {
		p.SetDefaultTimeout(float64(cfg.ActionTimeout.Milliseconds()))
	}
	if cfg.NavigationTimeout > 0 {
		p.SetDefaultNavigationTimeout(float64(cfg.NavigationTimeout.Milliseconds()))
	}
}

func wrapPlaywrightError(err error) error {
	if err == nil {
		return nil
	}
	if strings.Contains(err.Error(), "install the driver") {
		return fmt.Errorf("playwright driver not installed; run: go run github.com/playwright-community/playwright-go/cmd/playwright install: %w", err)
	}
	return err
}

func getEnvInt(name string, defaultValue int) int {
	raw := strings.TrimSpace(os.Getenv(name))
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value <= 0 {
		return defaultValue
	}
	return value
}
