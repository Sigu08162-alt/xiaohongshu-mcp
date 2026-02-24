package interaction

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/ports"
)

type Usecase struct {
	Gateway ports.InteractionGateway
}

func (u *Usecase) LikeFeed(ctx context.Context, feedID, xsecToken string) error {
	return u.Gateway.LikeFeed(ctx, feedID, xsecToken, false)
}

func (u *Usecase) UnlikeFeed(ctx context.Context, feedID, xsecToken string) error {
	return u.Gateway.LikeFeed(ctx, feedID, xsecToken, true)
}

func (u *Usecase) FavoriteFeed(ctx context.Context, feedID, xsecToken string) error {
	return u.Gateway.FavoriteFeed(ctx, feedID, xsecToken, false)
}

func (u *Usecase) UnfavoriteFeed(ctx context.Context, feedID, xsecToken string) error {
	return u.Gateway.FavoriteFeed(ctx, feedID, xsecToken, true)
}

func (u *Usecase) PostComment(ctx context.Context, feedID, xsecToken, content string) error {
	return u.Gateway.PostComment(ctx, feedID, xsecToken, content)
}

func (u *Usecase) ReplyComment(ctx context.Context, feedID, xsecToken, commentID, userID, content string) error {
	return u.Gateway.ReplyComment(ctx, feedID, xsecToken, commentID, userID, content)
}

func (u *Usecase) DeleteComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	return u.Gateway.DeleteComment(ctx, feedID, xsecToken, commentID, userID)
}

func (u *Usecase) LikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	return u.Gateway.LikeComment(ctx, feedID, xsecToken, commentID, userID, false)
}

func (u *Usecase) UnlikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	return u.Gateway.LikeComment(ctx, feedID, xsecToken, commentID, userID, true)
}
