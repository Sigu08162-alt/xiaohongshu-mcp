package feed_test

import (
	"context"
	"errors"
	"testing"

	"github.com/vmxmy/xiaohongshu-mcp/internal/app/feed"
	"github.com/vmxmy/xiaohongshu-mcp/internal/app/testkit"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

func TestListFeeds_ReturnsFeedsFromGateway(t *testing.T) {
	gw := &testkit.FakeFeedGateway{
		Feeds: []xiaohongshu.Feed{{ID: "feed-1"}, {ID: "feed-2"}},
	}
	uc := &feed.Usecase{Gateway: gw}

	feeds, err := uc.ListFeeds(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(feeds) != 2 {
		t.Errorf("expected 2 feeds, got %d", len(feeds))
	}
}

func TestListFeeds_PropagatesError(t *testing.T) {
	gw := &testkit.FakeFeedGateway{Err: errors.New("network error")}
	uc := &feed.Usecase{Gateway: gw}

	_, err := uc.ListFeeds(context.Background())
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestSearchFeeds_PassesKeyword(t *testing.T) {
	gw := &testkit.FakeFeedGateway{
		Feeds: []xiaohongshu.Feed{{ID: "feed-1"}},
	}
	uc := &feed.Usecase{Gateway: gw}

	_, err := uc.SearchFeeds(context.Background(), "普吉岛", xiaohongshu.FilterOption{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.SearchCalls != 1 {
		t.Errorf("expected 1 search call, got %d", gw.SearchCalls)
	}
	if gw.LastKeyword != "普吉岛" {
		t.Errorf("expected keyword '普吉岛', got '%s'", gw.LastKeyword)
	}
}

func TestDeleteFeed_PassesFeedID(t *testing.T) {
	gw := &testkit.FakeFeedGateway{}
	uc := &feed.Usecase{Gateway: gw}

	err := uc.DeleteFeed(context.Background(), "feed-123", "token-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.DeletedID != "feed-123" {
		t.Errorf("expected deleted ID 'feed-123', got '%s'", gw.DeletedID)
	}
}

func TestShareFeed_ReturnsURL(t *testing.T) {
	gw := &testkit.FakeFeedGateway{ShareURL: "https://xhs.com/share/123"}
	uc := &feed.Usecase{Gateway: gw}

	url, err := uc.ShareFeed(context.Background(), "feed-123", "token-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if url != "https://xhs.com/share/123" {
		t.Errorf("unexpected URL: %s", url)
	}
}
