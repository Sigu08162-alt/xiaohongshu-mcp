package analytics

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/ports"
)

type Usecase struct {
	gateway ports.AnalyticsGateway
}

func New(gateway ports.AnalyticsGateway) *Usecase {
	return &Usecase{gateway: gateway}
}

func (u *Usecase) GetContentAnalytics(ctx context.Context, limit int, sortBy, sortOrder string) (any, error) {
	return u.gateway.GetContentAnalytics(ctx, limit, sortBy, sortOrder)
}

func (u *Usecase) GetFanAnalytics(ctx context.Context, period string) (any, error) {
	return u.gateway.GetFanAnalytics(ctx, period)
}
