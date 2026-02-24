package user

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/ports"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

type Usecase struct {
	Gateway ports.UserGateway
}

func (u *Usecase) GetUserProfile(ctx context.Context, userID, xsecToken string) (*xiaohongshu.UserProfileResponse, error) {
	return u.Gateway.GetUserProfile(ctx, userID, xsecToken)
}

func (u *Usecase) FollowUser(ctx context.Context, userID, xsecToken string) error {
	return u.Gateway.FollowUser(ctx, userID, xsecToken, false)
}

func (u *Usecase) UnfollowUser(ctx context.Context, userID, xsecToken string) error {
	return u.Gateway.FollowUser(ctx, userID, xsecToken, true)
}

func (u *Usecase) GetMyProfile(ctx context.Context) (*xiaohongshu.UserProfileResponse, error) {
	return u.Gateway.GetMyProfile(ctx)
}

func (u *Usecase) GetMyStats(ctx context.Context) (map[string]any, error) {
	return u.Gateway.GetMyStats(ctx)
}
