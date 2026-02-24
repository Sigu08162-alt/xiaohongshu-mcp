package analytics

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/ports"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

type Usecase struct {
	Gateway ports.AnalyticsGateway
}

func (u *Usecase) GetContentAnalytics(ctx context.Context, limit int, sortBy, sortOrder string) (*xiaohongshu.ContentAnalytics, error) {
	return u.Gateway.GetContentAnalytics(ctx, limit, sortBy, sortOrder)
}

func (u *Usecase) GetFanAnalytics(ctx context.Context, period string) (*xiaohongshu.FanAnalytics, error) {
	return u.Gateway.GetFanAnalytics(ctx, period)
}
