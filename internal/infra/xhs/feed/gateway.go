// Package feed implements the ports.FeedGateway interface using browser automation.
package feed

import (
	"context"
	"fmt"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// Gateway implements ports.FeedGateway.
type Gateway struct {
	pageFactory func() (browser.Page, func(), error)
	polling     polling.Module
}

// New creates a new feed Gateway.
func New(pageFactory func() (browser.Page, func(), error), pollingModule polling.Module) *Gateway {
	return &Gateway{pageFactory: pageFactory, polling: pollingModule}
}

func (g *Gateway) withPage(ctx context.Context, fn func(browser.Page) error) error {
	page, cleanup, err := g.pageFactory()
	if err != nil {
		return fmt.Errorf("feed gateway: create page: %w", err)
	}
	defer cleanup()
	return fn(page.WithContext(ctx))
}

// ListFeeds returns the home feed list.
func (g *Gateway) ListFeeds(ctx context.Context) ([]xiaohongshu.Feed, error) {
	var result []xiaohongshu.Feed
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewFeedsListAction(page, g.polling)
		if err != nil {
			return err
		}
		result, err = action.GetFeedsList(ctx)
		return err
	})
	return result, err
}

// GetFeedDetail returns detail for a single note.
func (g *Gateway) GetFeedDetail(ctx context.Context, feedID, xsecToken string) (*xiaohongshu.FeedDetailResponse, error) {
	var result *xiaohongshu.FeedDetailResponse
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewFeedDetailAction(page, g.polling)
		if err != nil {
			return err
		}
		detail, err := action.GetFeedDetail(ctx, feedID, xsecToken, false, xiaohongshu.CommentLoadConfig{})
		if err != nil {
			return err
		}
		result = detail
		return nil
	})
	return result, err
}

// DeleteFeed deletes a note owned by the current user.
func (g *Gateway) DeleteFeed(ctx context.Context, feedID, xsecToken string) error {
	return g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewDeleteAction(page, g.polling)
		if err != nil {
			return err
		}
		return action.DeleteFeed(ctx, feedID, xsecToken)
	})
}

// ShareFeed returns the share URL for a note.
func (g *Gateway) ShareFeed(ctx context.Context, feedID, xsecToken string) (string, error) {
	var link string
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewShareAction(page, g.polling)
		if err != nil {
			return err
		}
		var e error
		link, e = action.ShareFeed(ctx, feedID, xsecToken)
		return e
	})
	return link, err
}

// GetMyFeeds returns notes published by a user (or current user if userID is empty).
func (g *Gateway) GetMyFeeds(ctx context.Context, userID string, limit int) ([]xiaohongshu.Feed, error) {
	var result []xiaohongshu.Feed
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewDataAction(page, g.polling)
		if err != nil {
			return err
		}
		var e error
		result, e = action.GetMyFeeds(ctx, limit, userID)
		return e
	})
	return result, err
}

// SearchFeeds searches notes by keyword.
func (g *Gateway) SearchFeeds(ctx context.Context, keyword string, filters xiaohongshu.FilterOption) ([]xiaohongshu.Feed, error) {
	var result []xiaohongshu.Feed
	err := g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewSearchAction(page, g.polling)
		if err != nil {
			return err
		}
		var e error
		result, e = action.Search(ctx, keyword, filters)
		return e
	})
	return result, err
}
