// Package interaction implements the ports.InteractionGateway interface.
package interaction

import (
	"context"
	"fmt"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// Gateway implements ports.InteractionGateway.
type Gateway struct {
	pageFactory func() (browser.Page, func(), error)
	polling     polling.Module
}

// New creates a new interaction Gateway.
func New(pageFactory func() (browser.Page, func(), error), pollingModule polling.Module) *Gateway {
	return &Gateway{pageFactory: pageFactory, polling: pollingModule}
}

func (g *Gateway) withPage(ctx context.Context, fn func(browser.Page) error) error {
	page, cleanup, err := g.pageFactory()
	if err != nil {
		return fmt.Errorf("interaction gateway: create page: %w", err)
	}
	defer cleanup()
	return fn(page.WithContext(ctx))
}

// LikeFeed likes or unlikes a feed.
func (g *Gateway) LikeFeed(ctx context.Context, feedID, xsecToken string, unlike bool) error {
	return g.withPage(ctx, func(page browser.Page) error {
		if unlike {
			action := xiaohongshu.NewLikeAction(page, g.polling)
			return action.Unlike(ctx, feedID, xsecToken)
		}
		action := xiaohongshu.NewLikeAction(page, g.polling)
		return action.Like(ctx, feedID, xsecToken)
	})
}

// FavoriteFeed favorites or unfavorites a feed.
func (g *Gateway) FavoriteFeed(ctx context.Context, feedID, xsecToken string, unfavorite bool) error {
	return g.withPage(ctx, func(page browser.Page) error {
		if unfavorite {
			action := xiaohongshu.NewFavoriteAction(page, g.polling)
			return action.Unfavorite(ctx, feedID, xsecToken)
		}
		action := xiaohongshu.NewFavoriteAction(page, g.polling)
		return action.Favorite(ctx, feedID, xsecToken)
	})
}

// PostComment posts a comment on a feed.
func (g *Gateway) PostComment(ctx context.Context, feedID, xsecToken, content string) error {
	return g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewCommentFeedAction(page, g.polling)
		if err != nil {
			return err
		}
		return action.PostComment(ctx, feedID, xsecToken, content)
	})
}

// DeleteComment deletes a comment.
func (g *Gateway) DeleteComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	return g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewDeleteAction(page, g.polling)
		if err != nil {
			return err
		}
		return action.DeleteComment(ctx, feedID, xsecToken, commentID, userID)
	})
}

// LikeComment likes or unlikes a comment.
func (g *Gateway) LikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string, unlike bool) error {
	return g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewCommentLikeAction(page, g.polling)
		if err != nil {
			return err
		}
		if unlike {
			return action.UnlikeComment(ctx, feedID, xsecToken, commentID, userID)
		}
		return action.LikeComment(ctx, feedID, xsecToken, commentID, userID)
	})
}

// ReplyComment replies to a comment.
func (g *Gateway) ReplyComment(ctx context.Context, feedID, xsecToken, commentID, userID, content string) error {
	return g.withPage(ctx, func(page browser.Page) error {
		action, err := xiaohongshu.NewCommentFeedAction(page, g.polling)
		if err != nil {
			return err
		}
		return action.ReplyToComment(ctx, feedID, xsecToken, commentID, userID, content)
	})
}
