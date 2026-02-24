//go:build integration

package main

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// ── 冒烟测试 ─────────────────────────────────────────────

func TestIntegration_CheckLoginStatus(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.CheckLoginStatus(ctx)
	require.NoError(t, err)
	assert.True(t, resp.IsLoggedIn, "应已登录")
	t.Logf("登录用户: %s", resp.Username)
}

// ── 读操作测试 ────────────────────────────────────────────

func TestIntegration_ListFeeds(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.ListFeeds(ctx)
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Feeds)
	t.Logf("首页 Feed: %d 条", len(resp.Feeds))
}

func TestIntegration_SearchFeeds(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	resp, err := svc.SearchFeeds(ctx, "普吉岛")
	require.NoError(t, err)
	assert.NotEmpty(t, resp.Feeds)
	t.Logf("搜索'普吉岛': %d 条结果", len(resp.Feeds))
}

func TestIntegration_GetMyStats(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stats, err := svc.GetMyStats(ctx)
	require.NoError(t, err)
	t.Logf("粉丝: %d, 关注: %d, 获赞收藏: %d, 笔记: %d",
		stats.FollowerCount, stats.FollowCount, stats.LikedCount, stats.NoteCount)
}

func TestIntegration_GetMyFeeds(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	feeds, err := svc.GetMyFeeds(ctx, 10, "")
	require.NoError(t, err)
	assert.NotEmpty(t, feeds)
	t.Logf("我的笔记: %d 条", len(feeds))
}

func TestIntegration_GetContentAnalytics(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	analytics, err := svc.GetContentAnalytics(ctx, 5, xiaohongshu.SortByViews, xiaohongshu.SortDesc)
	require.NoError(t, err)
	t.Logf("内容分析: %d 条笔记", len(analytics.Notes))
	for _, n := range analytics.Notes {
		t.Logf("  - %s: 曝光=%d 观看=%d 点赞=%d", n.Title, n.Exposure, n.Views, n.Likes)
	}
}
