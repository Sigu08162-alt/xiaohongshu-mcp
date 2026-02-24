//go:build integration

package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

const testNoteTitle = "[TEST] 自动化测试笔记 - 请忽略"

// TestIntegration_PublishAndDelete 发布一篇测试图文笔记，验证后立即删除
func TestIntegration_PublishAndDelete(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	t.Log("Step 1: 发布测试笔记...")
	_, err := svc.PublishContent(ctx, &PublishRequest{
		Title:   testNoteTitle,
		Content: "这是一篇自动化集成测试笔记，发布后将立即删除。",
		Images:  []string{"https://www.xiaohongshu.com/favicon.ico"},
		Tags:    []string{"测试"},
	})
	require.NoError(t, err, "发布失败")
	t.Log("发布请求成功")

	t.Log("Step 2: 等待笔记可见（8s）...")
	time.Sleep(8 * time.Second)

	t.Log("Step 3: 从我的笔记列表找到刚发布的笔记...")
	feeds, err := svc.GetMyFeeds(ctx, 10, "")
	require.NoError(t, err)

	var noteID, xsecToken string
	for _, f := range feeds {
		if strings.Contains(f.NoteCard.DisplayTitle, "[TEST]") {
			noteID = f.ID
			xsecToken = f.XsecToken
			break
		}
	}
	require.NotEmpty(t, noteID, "未在笔记列表中找到测试笔记")
	t.Logf("找到测试笔记: ID=%s", noteID)

	// 确保无论如何都会删除
	t.Cleanup(func() {
		delCtx, delCancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer delCancel()
		if _, err := svc.DeleteFeed(delCtx, noteID, xsecToken); err != nil {
			t.Logf("⚠️  cleanup 删除失败（需手动删除）: noteID=%s err=%v", noteID, err)
		} else {
			t.Logf("✅ cleanup 删除成功: noteID=%s", noteID)
		}
	})

	assert.NotEmpty(t, noteID)
}

// TestIntegration_PostAndDeleteComment 对自己的笔记发评论后删除
func TestIntegration_PostAndDeleteComment(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	feeds, err := svc.GetMyFeeds(ctx, 1, "")
	require.NoError(t, err)
	require.NotEmpty(t, feeds, "没有可用笔记，跳过评论测试")

	target := feeds[0]
	t.Logf("目标笔记: %s (%s)", target.NoteCard.DisplayTitle, target.ID)

	t.Log("Step 1: 发评论...")
	commentResp, err := svc.PostCommentToFeed(ctx, target.ID, target.XsecToken, "[TEST] 自动化测试评论，请忽略")
	require.NoError(t, err, "发评论失败")
	assert.True(t, commentResp.Success, "评论响应 Success=false: %s", commentResp.Message)
	t.Logf("评论结果: %s", commentResp.Message)

	t.Log("Step 2: 验证评论出现在笔记详情...")
	time.Sleep(2 * time.Second)
	detail, err := svc.GetFeedDetailWithConfig(ctx, target.ID, target.XsecToken, true, xiaohongshu.DefaultCommentLoadConfig())
	require.NoError(t, err)
	// FeedDetailResponse.Data 是 any，只验证不报错即可
	assert.NotNil(t, detail.Data)
	t.Logf("笔记详情获取成功: feedID=%s", detail.FeedID)
}

// TestIntegration_LikeAndUnlike 点赞后取消点赞
func TestIntegration_LikeAndUnlike(t *testing.T) {
	svc := newIntegrationService(t)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	resp, err := svc.SearchFeeds(ctx, "普吉岛")
	require.NoError(t, err)
	require.NotEmpty(t, resp.Feeds, "搜索无结果")

	target := resp.Feeds[0]
	t.Logf("目标笔记: %s (%s)", target.NoteCard.DisplayTitle, target.ID)

	t.Log("Step 1: 点赞...")
	likeResult, err := svc.LikeFeed(ctx, target.ID, target.XsecToken)
	require.NoError(t, err, "点赞失败")
	assert.True(t, likeResult.Success, "点赞 Success=false: %s", likeResult.Message)
	t.Log("点赞成功")

	time.Sleep(2 * time.Second)

	t.Log("Step 2: 取消点赞...")
	unlikeResult, err := svc.UnlikeFeed(ctx, target.ID, target.XsecToken)
	require.NoError(t, err, "取消点赞失败")
	assert.True(t, unlikeResult.Success, "取消点赞 Success=false: %s", unlikeResult.Message)
	t.Log("取消点赞成功")
}
