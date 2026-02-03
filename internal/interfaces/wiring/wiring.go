package wiring

import (
	apppublish "github.com/vmxmy/xiaohongshu-mcp/internal/app/publish"
	domainpublish "github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/config"
	xhspublish "github.com/vmxmy/xiaohongshu-mcp/internal/infra/xhs/publish"
	"github.com/vmxmy/xiaohongshu-mcp/pkg/downloader"
)

func BuildPublishUsecase(cfg *config.Config, selectors map[string]string, engine browser.Engine) (*apppublish.Usecase, error) {
	gw, err := xhspublish.NewGateway(xhspublish.Config{
		PublishImageURL: cfg.URLs.Creator.PublishImage,
		PublishVideoURL: cfg.URLs.Creator.PublishVideo,
		Selectors:       selectors,
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
