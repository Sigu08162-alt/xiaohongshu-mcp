package user

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/ports"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

type Usecase struct {
	gateway ports.UserGateway
}

func New(gateway ports.UserGateway) *Usecase {
	return &Usecase{gateway: gateway}
}

func (u *Usecase) FollowUser(ctx context.Context, userID, xsecToken string, unfollow bool) error {
	return u.gateway.FollowUser(ctx, userID, xsecToken, unfollow)
}

func (u *Usecase) GetUserProfile(ctx context.Context, userID, xsecToken string) (*xiaohongshu.UserProfileResponse, error) {
	return u.gateway.GetUserProfile(ctx, userID, xsecToken)
}

func (u *Usecase) GetMyStats(ctx context.Context) (map[string]any, error) {
	return u.gateway.GetMyStats(ctx)
}
