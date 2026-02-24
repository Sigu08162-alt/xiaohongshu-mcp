package ports

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// ── Publish ──────────────────────────────────────────────────────────────────

type PublishGateway interface {
	PublishImage(ctx context.Context, content publish.ImageContent) error
	PublishVideo(ctx context.Context, content publish.VideoContent) error
	SaveImageDraft(ctx context.Context, content publish.ImageContent) error
	SaveVideoDraft(ctx context.Context, content publish.VideoContent) error
}

type SelectorStore interface {
	Load() (map[string]string, error)
	Save(selectors map[string]string) error
	Snapshot() (string, error)
	Rollback(snapshot string) error
}

// ── Feed ─────────────────────────────────────────────────────────────────────

type FeedGateway interface {
	ListFeeds(ctx context.Context) ([]xiaohongshu.Feed, error)
	GetFeedDetail(ctx context.Context, feedID, xsecToken string) (*xiaohongshu.FeedDetailResponse, error)
	DeleteFeed(ctx context.Context, feedID, xsecToken string) error
	ShareFeed(ctx context.Context, feedID, xsecToken string) (string, error)
	GetMyFeeds(ctx context.Context, userID string, limit int) ([]xiaohongshu.Feed, error)
	SearchFeeds(ctx context.Context, keyword string, filters xiaohongshu.FilterOption) ([]xiaohongshu.Feed, error)
}

// ── Interaction ───────────────────────────────────────────────────────────────

type InteractionGateway interface {
	LikeFeed(ctx context.Context, feedID, xsecToken string, unlike bool) error
	FavoriteFeed(ctx context.Context, feedID, xsecToken string, unfavorite bool) error
	PostComment(ctx context.Context, feedID, xsecToken, content string) error
	DeleteComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error
	LikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string, unlike bool) error
	ReplyComment(ctx context.Context, feedID, xsecToken, commentID, userID, content string) error
}

// ── User ──────────────────────────────────────────────────────────────────────

type UserGateway interface {
	FollowUser(ctx context.Context, userID, xsecToken string, unfollow bool) error
	GetUserProfile(ctx context.Context, userID, xsecToken string) (*xiaohongshu.UserProfileResponse, error)
	GetMyProfile(ctx context.Context) (*xiaohongshu.UserProfileResponse, error)
	GetMyStats(ctx context.Context) (map[string]any, error)
}

// ── Analytics ─────────────────────────────────────────────────────────────────

type AnalyticsGateway interface {
	GetContentAnalytics(ctx context.Context, limit int, sortBy, sortOrder string) (any, error)
	GetFanAnalytics(ctx context.Context, period string) (any, error)
}

// ── Login ─────────────────────────────────────────────────────────────────────

type LoginGateway interface {
	CheckLoginStatus(ctx context.Context) (map[string]any, error)
	GetLoginQRCode(ctx context.Context) (map[string]any, error)
}
