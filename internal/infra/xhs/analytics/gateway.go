// Package analytics implements the ports.AnalyticsGateway interface.
package analytics

import (
	"context"
	"fmt"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// Gateway implements ports.AnalyticsGateway.
type Gateway struct {
	pageFactory func() (browser.Page, func(), error)
	polling     polling.Module
}

// New creates a new analytics Gateway.
func New(pageFactory func() (browser.Page, func(), error), pollingModule polling.Module) *Gateway {
	return &Gateway{pageFactory: pageFactory, polling: pollingModule}
}

func (g *Gateway) withPage(ctx context.Context, fn func(browser.Page) error) error {
	page, cleanup, err := g.pageFactory()
	if err != nil {
		return fmt.Errorf("analytics gateway: create page: %w", err)
	}
	defer cleanup()
	return fn(page.WithContext(ctx))
}

// GetContentAnalytics returns per-note content analytics.
func (g *Gateway) GetContentAnalytics(ctx context.Context, limit int, sortBy, sortOrder string) (*xiaohongshu.ContentAnalytics, error) {
	var result *xiaohongshu.ContentAnalytics
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewDataAction(page, g.polling)
		if err != nil {
			return err
		}
		result, err = action.GetContentAnalytics(ctx, limit, xiaohongshu.SortField(sortBy), xiaohongshu.SortOrder(sortOrder))
		return err
	})
	return result, err
}

// GetFanAnalytics returns fan overview and demographics.
func (g *Gateway) GetFanAnalytics(ctx context.Context, period string) (*xiaohongshu.FanAnalytics, error) {
	var result *xiaohongshu.FanAnalytics
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewDataAction(page, g.polling)
		if err != nil {
			return err
		}
		result, err = action.GetFanAnalytics(ctx, period)
		return err
	})
	return result, err
}
