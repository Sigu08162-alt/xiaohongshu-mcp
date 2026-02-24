package feed

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/ports"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

type Usecase struct {
	Gateway ports.FeedGateway
}

func (u *Usecase) ListFeeds(ctx context.Context) ([]xiaohongshu.Feed, error) {
	return u.Gateway.ListFeeds(ctx)
}

func (u *Usecase) SearchFeeds(ctx context.Context, keyword string, filters xiaohongshu.FilterOption) ([]xiaohongshu.Feed, error) {
	return u.Gateway.SearchFeeds(ctx, keyword, filters)
}

func (u *Usecase) GetFeedDetail(ctx context.Context, feedID, xsecToken string) (*xiaohongshu.FeedDetailResponse, error) {
	return u.Gateway.GetFeedDetail(ctx, feedID, xsecToken)
}

func (u *Usecase) DeleteFeed(ctx context.Context, feedID, xsecToken string) error {
	return u.Gateway.DeleteFeed(ctx, feedID, xsecToken)
}

func (u *Usecase) ShareFeed(ctx context.Context, feedID, xsecToken string) (string, error) {
	return u.Gateway.ShareFeed(ctx, feedID, xsecToken)
}

func (u *Usecase) GetMyFeeds(ctx context.Context, userID string, limit int) ([]xiaohongshu.Feed, error) {
	return u.Gateway.GetMyFeeds(ctx, userID, limit)
}
