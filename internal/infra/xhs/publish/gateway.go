package publish

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/selector"
)

var ErrNotReady = errors.New("publish not implemented")

type Config struct {
	PublishImageURL string
	PublishVideoURL string
	Selectors       map[string]string
	SelectorCfg     *selector.SelectorConfig
	PublishPolling  PollingModule
	DraftPolling    PollingModule
	VideoPolling    PollingModule
}

type Gateway struct {
	cfg    Config
	engine browser.Engine
}

type UploadStateSelectors struct {
	UploadingMask  string
	UploadingClass string
	UploadPreview  string
	UploadingToast string
}

type PollingModule struct {
	TimeoutMs  int
	IntervalMs int
	MaxRetries int
	Delays     map[string]int
}

func NewGateway(cfg Config, engine browser.Engine) (*Gateway, error) {
	if cfg.PublishImageURL == "" || cfg.PublishVideoURL == "" {
		return nil, errors.New("publish url missing")
	}
	if engine == nil {
		return nil, errors.New("engine missing")
	}
	logrus.Infof("🔧 Gateway配置:")
	logrus.Infof("  - 图文发布URL: %s", cfg.PublishImageURL)
	logrus.Infof("  - 视频发布URL: %s", cfg.PublishVideoURL)
	return &Gateway{cfg: cfg, engine: engine}, nil
}

func (g *Gateway) newResolver(page browser.Page) *selector.ElementResolver {
	if g.cfg.SelectorCfg != nil {
		return selector.NewElementResolver(g.cfg.SelectorCfg, page)
	}
	return nil
}

func resolveOrFallback(resolver *selector.ElementResolver, smartName, legacySelector string) string {
	if resolver != nil {
		if sel, err := resolver.Resolve(smartName); err == nil {
			return sel
		}
		logrus.Warnf("自适应解析失败: %s, 降级到静态配置: %s", smartName, legacySelector)
	}
	return legacySelector
}

func (g *Gateway) pollingFor(isPublish bool) PollingModule {
	if isPublish {
		return g.cfg.PublishPolling
	}
	return g.cfg.DraftPolling
}
