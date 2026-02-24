package analytics

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/ports"
)

type Usecase struct {
	Gateway ports.AnalyticsGateway
}

func (u *Usecase) GetContentAnalytics(ctx context.Context, limit int, sortBy, sortOrder string) (any, error) {
	return u.Gateway.GetContentAnalytics(ctx, limit, sortBy, sortOrder)
}

func (u *Usecase) GetFanAnalytics(ctx context.Context, period string) (any, error) {
	return u.Gateway.GetFanAnalytics(ctx, period)
}
