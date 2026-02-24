package wiring

import (
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
		ImageProcessor: downloader.NewImageProcessor(), // 注入图片处理器
	}, nil
}

// defaultPolling returns a sensible default polling module for non-publish operations.
func defaultPolling() polling.Module {
	return polling.Module{
		TimeoutMs:  30000,
		IntervalMs: 500,
		MaxRetries: 10,
		Delays:     map[string]int{"short": 500, "medium": 1000, "long": 2000},
	}
}

// pageFactory creates a factory function that opens a new browser page.
func pageFactory(engine browser.Engine) func() (browser.Page, func(), error) {
	return func() (browser.Page, func(), error) {
		page, err := engine.NewPage()
		if err != nil {
			return nil, nil, err
		}
		return page, func() { page.Close() }, nil
	}
}

// BuildFeedUsecase wires the feed infra gateway and app usecase.
func BuildFeedUsecase(engine browser.Engine) *appfeed.Usecase {
	gw := infrafeed.New(pageFactory(engine), defaultPolling())
	return &appfeed.Usecase{Gateway: gw}
}

// BuildInteractionUsecase wires the interaction infra gateway and app usecase.
func BuildInteractionUsecase(engine browser.Engine) *appinteraction.Usecase {
	gw := infrainteraction.New(pageFactory(engine), defaultPolling())
	return &appinteraction.Usecase{Gateway: gw}
}

// BuildUserUsecase wires the user infra gateway and app usecase.
func BuildUserUsecase(engine browser.Engine) *appuser.Usecase {
	gw := infrauser.New(pageFactory(engine), defaultPolling())
	return &appuser.Usecase{Gateway: gw}
}

// BuildAnalyticsUsecase wires the analytics infra gateway and app usecase.
func BuildAnalyticsUsecase(engine browser.Engine) *appanalytics.Usecase {
	gw := infraanalytics.New(pageFactory(engine), defaultPolling())
	return &appanalytics.Usecase{Gateway: gw}
}
