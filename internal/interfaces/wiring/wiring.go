package wiring

import (
	"fmt"

	appanalytics "github.com/vmxmy/xiaohongshu-mcp/internal/app/analytics"
	appfeed "github.com/vmxmy/xiaohongshu-mcp/internal/app/feed"
	appinteraction "github.com/vmxmy/xiaohongshu-mcp/internal/app/interaction"
	apppublish "github.com/vmxmy/xiaohongshu-mcp/internal/app/publish"
	appuser "github.com/vmxmy/xiaohongshu-mcp/internal/app/user"
	domainpublish "github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/config"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/selector"
	infraanalytics "github.com/vmxmy/xiaohongshu-mcp/internal/infra/xhs/analytics"
	infrafeed "github.com/vmxmy/xiaohongshu-mcp/internal/infra/xhs/feed"
	infrainteraction "github.com/vmxmy/xiaohongshu-mcp/internal/infra/xhs/interaction"
	infralogin "github.com/vmxmy/xiaohongshu-mcp/internal/infra/xhs/login"
	xhspublish "github.com/vmxmy/xiaohongshu-mcp/internal/infra/xhs/publish"
	infrauser "github.com/vmxmy/xiaohongshu-mcp/internal/infra/xhs/user"
	"github.com/vmxmy/xiaohongshu-mcp/pkg/downloader"
)

func BuildPublishUsecase(cfg *config.Config, selectors map[string]string, selectorCfg *selector.SelectorConfig, engine browser.Engine) (*apppublish.Usecase, error) {
	gw, err := xhspublish.NewGateway(xhspublish.Config{
		PublishImageURL: cfg.URLs.Creator.PublishImage,
		PublishVideoURL: cfg.URLs.Creator.PublishVideo,
		Selectors:       selectors,
		SelectorCfg:     selectorCfg,
		PublishPolling: xhspublish.PollingModule{
			TimeoutMs:  cfg.Polling.Publish.TimeoutMs,
			IntervalMs: cfg.Polling.Publish.IntervalMs,
			MaxRetries: cfg.Polling.Publish.MaxRetries,
			Delays:     cfg.Polling.Publish.Delays,
		},
		DraftPolling: xhspublish.PollingModule{
			TimeoutMs:  cfg.Polling.Draft.TimeoutMs,
			IntervalMs: cfg.Polling.Draft.IntervalMs,
			MaxRetries: cfg.Polling.Draft.MaxRetries,
			Delays:     cfg.Polling.Draft.Delays,
		},
		VideoPolling: xhspublish.PollingModule{
			TimeoutMs:  cfg.Polling.Video.TimeoutMs,
			IntervalMs: cfg.Polling.Video.IntervalMs,
			MaxRetries: cfg.Polling.Video.MaxRetries,
			Delays:     cfg.Polling.Video.Delays,
		},
	}, engine)
	if err != nil {
		return nil, err
	}
	return &apppublish.Usecase{
		Gateway: gw,
		Limits: domainpublish.Limits{
			MaxTags:   cfg.Limits.MaxTags,
			MinImages: cfg.Limits.MinImages,
			MaxImages: cfg.Limits.MaxImages,
		},
		ImageProcessor: downloader.NewImageProcessor(),
	}, nil
}

// pageFactoryFromEngine creates a page factory from an engine factory func.
// Each call starts a fresh engine, creates a page, and returns a cleanup func
// that closes both the page and the engine.
func pageFactoryFromEngine(engineFactory func() browser.Engine) func() (browser.Page, func(), error) {
	return func() (browser.Page, func(), error) {
		engine := engineFactory()
		if err := engine.Start(); err != nil {
			return nil, nil, fmt.Errorf("start browser engine: %w", err)
		}
		page, err := engine.NewPage()
		if err != nil {
			_ = engine.Close()
			return nil, nil, fmt.Errorf("new browser page: %w", err)
		}
		cleanup := func() {
			_ = page.Close()
			_ = engine.Close()
		}
		return page, cleanup, nil
	}
}

// BuildFeedUsecase wires the feed infra gateway and app usecase.
func BuildFeedUsecase(engineFactory func() browser.Engine, pollingModule polling.Module) *appfeed.Usecase {
	gw := infrafeed.New(pageFactoryFromEngine(engineFactory), pollingModule)
	return &appfeed.Usecase{Gateway: gw}
}

// BuildInteractionUsecase wires the interaction infra gateway and app usecase.
func BuildInteractionUsecase(engineFactory func() browser.Engine, pollingModule polling.Module) *appinteraction.Usecase {
	gw := infrainteraction.New(pageFactoryFromEngine(engineFactory), pollingModule)
	return &appinteraction.Usecase{Gateway: gw}
}

// BuildUserUsecase wires the user infra gateway and app usecase.
func BuildUserUsecase(engineFactory func() browser.Engine, pollingModule polling.Module) *appuser.Usecase {
	gw := infrauser.New(pageFactoryFromEngine(engineFactory), pollingModule)
	return &appuser.Usecase{Gateway: gw}
}

// BuildAnalyticsUsecase wires the analytics infra gateway and app usecase.
func BuildAnalyticsUsecase(engineFactory func() browser.Engine, pollingModule polling.Module) *appanalytics.Usecase {
	gw := infraanalytics.New(pageFactoryFromEngine(engineFactory), pollingModule)
	return &appanalytics.Usecase{Gateway: gw}
}

// BuildLoginUsecase wires the login infra gateway.
func BuildLoginUsecase(engineFactory func() browser.Engine, pollingModule polling.Module) *infralogin.Gateway {
	return infralogin.New(pageFactoryFromEngine(engineFactory), pollingModule)
}
