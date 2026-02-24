package testkit

import (
	"context"

	"github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// ── Publish ───────────────────────────────────────────────────────────────────

type FakePublishGateway struct {
	ImageCalls int
	VideoCalls int
	LastImage  publish.ImageContent
	LastVideo  publish.VideoContent
	Err        error
}

func (f *FakePublishGateway) PublishImage(ctx context.Context, content publish.ImageContent) error {
	f.ImageCalls++
	f.LastImage = content
	return f.Err
}

func (f *FakePublishGateway) PublishVideo(ctx context.Context, content publish.VideoContent) error {
	f.VideoCalls++
	f.LastVideo = content
	return f.Err
}

func (f *FakePublishGateway) SaveImageDraft(ctx context.Context, content publish.ImageContent) error {
	f.ImageCalls++
	f.LastImage = content
	return f.Err
}

func (f *FakePublishGateway) SaveVideoDraft(ctx context.Context, content publish.VideoContent) error {
	f.VideoCalls++
	f.LastVideo = content
	return f.Err
}

// ── Selector ──────────────────────────────────────────────────────────────────

type FakeSelectorStore struct {
	Selectors map[string]string
}

func (f *FakeSelectorStore) Load() (map[string]string, error) {
	return f.Selectors, nil
}

func (f *FakeSelectorStore) Save(selectors map[string]string) error {
	f.Selectors = selectors
	return nil
}

func (f *FakeSelectorStore) Snapshot() (string, error) {
	return "snapshot", nil
}

func (f *FakeSelectorStore) Rollback(snapshot string) error {
	return nil
}

// ── Feed ──────────────────────────────────────────────────────────────────────

type FakeFeedGateway struct {
	Feeds       []xiaohongshu.Feed
	FeedDetail  *xiaohongshu.FeedDetailResponse
	ShareURL    string
	Err         error
	DeletedID   string
	SearchCalls int
	LastKeyword string
}

func (f *FakeFeedGateway) ListFeeds(ctx context.Context) ([]xiaohongshu.Feed, error) {
	return f.Feeds, f.Err
}

func (f *FakeFeedGateway) GetFeedDetail(ctx context.Context, feedID, xsecToken string) (*xiaohongshu.FeedDetailResponse, error) {
	return f.FeedDetail, f.Err
}

func (f *FakeFeedGateway) DeleteFeed(ctx context.Context, feedID, xsecToken string) error {
	f.DeletedID = feedID
	return f.Err
}

func (f *FakeFeedGateway) ShareFeed(ctx context.Context, feedID, xsecToken string) (string, error) {
	return f.ShareURL, f.Err
}

func (f *FakeFeedGateway) GetMyFeeds(ctx context.Context, userID string, limit int) ([]xiaohongshu.Feed, error) {
	return f.Feeds, f.Err
}

func (f *FakeFeedGateway) SearchFeeds(ctx context.Context, keyword string, filters xiaohongshu.FilterOption) ([]xiaohongshu.Feed, error) {
	f.SearchCalls++
	f.LastKeyword = keyword
	return f.Feeds, f.Err
}

// ── Interaction ───────────────────────────────────────────────────────────────

type FakeInteractionGateway struct {
	Err          error
	LikeCalls    int
	CommentCalls int
	LastComment  string
}

func (f *FakeInteractionGateway) LikeFeed(ctx context.Context, feedID, xsecToken string, unlike bool) error {
	f.LikeCalls++
	return f.Err
}

func (f *FakeInteractionGateway) FavoriteFeed(ctx context.Context, feedID, xsecToken string, unfavorite bool) error {
	return f.Err
}

func (f *FakeInteractionGateway) PostComment(ctx context.Context, feedID, xsecToken, content string) error {
	f.CommentCalls++
	f.LastComment = content
	return f.Err
}

func (f *FakeInteractionGateway) DeleteComment(ctx context.Context, feedID, xsecToken, commentID, userID string) error {
	return f.Err
}

func (f *FakeInteractionGateway) LikeComment(ctx context.Context, feedID, xsecToken, commentID, userID string, unlike bool) error {
	return f.Err
}

func (f *FakeInteractionGateway) ReplyComment(ctx context.Context, feedID, xsecToken, commentID, userID, content string) error {
	return f.Err
}

// ── User ──────────────────────────────────────────────────────────────────────

type FakeUserGateway struct {
	Err         error
	Profile     *xiaohongshu.UserProfileResponse
	Stats       map[string]any
	FollowCalls int
}

func (f *FakeUserGateway) FollowUser(ctx context.Context, userID, xsecToken string, unfollow bool) error {
	f.FollowCalls++
	return f.Err
}

func (f *FakeUserGateway) GetUserProfile(ctx context.Context, userID, xsecToken string) (*xiaohongshu.UserProfileResponse, error) {
	return f.Profile, f.Err
}

func (f *FakeUserGateway) GetMyProfile(ctx context.Context) (*xiaohongshu.UserProfileResponse, error) {
	return f.Profile, f.Err
}

func (f *FakeUserGateway) GetMyStats(ctx context.Context) (map[string]any, error) {
	return f.Stats, f.Err
}

// ── Analytics ─────────────────────────────────────────────────────────────────

type FakeAnalyticsGateway struct {
	Err             error
	ContentAnalytics *xiaohongshu.ContentAnalytics
	FanAnalytics    *xiaohongshu.FanAnalytics
}

func (f *FakeAnalyticsGateway) GetContentAnalytics(ctx context.Context, limit int, sortBy, sortOrder string) (*xiaohongshu.ContentAnalytics, error) {
	return f.ContentAnalytics, f.Err
}

func (f *FakeAnalyticsGateway) GetFanAnalytics(ctx context.Context, period string) (*xiaohongshu.FanAnalytics, error) {
	return f.FanAnalytics, f.Err
}

// ── Login ─────────────────────────────────────────────────────────────────────

type FakeLoginGateway struct {
	Err        error
	StatusData map[string]any
	QRCodeData map[string]any
}

func (f *FakeLoginGateway) CheckLoginStatus(ctx context.Context) (map[string]any, error) {
	return f.StatusData, f.Err
}

func (f *FakeLoginGateway) GetLoginQRCode(ctx context.Context) (map[string]any, error) {
	return f.QRCodeData, f.Err
}
