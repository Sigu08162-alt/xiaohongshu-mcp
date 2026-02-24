package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vmxmy/xiaohongshu-mcp/xiaohongshu"
)

// TestListFeeds verifies we can fetch the home feed.
func TestListFeeds(t *testing.T) {
	s := newSuite(t)
	feeds, err := s.feed.ListFeeds(ctx(t, 30*time.Second))
	require.NoError(t, err)
	assert.NotEmpty(t, feeds, "home feed should return at least one item")
	t.Logf("got %d feeds, first id: %s", len(feeds), feeds[0].ID)
}

// TestGetMyFeeds verifies we can fetch our own published notes.
func TestGetMyFeeds(t *testing.T) {
	s := newSuite(t)
	feeds, err := s.feed.GetMyFeeds(ctx(t, 30*time.Second), "", 10)
	require.NoError(t, err)
	t.Logf("got %d personal feeds", len(feeds))
}

// TestSearchFeeds verifies keyword search returns results.
func TestSearchFeeds(t *testing.T) {
	s := newSuite(t)
	feeds, err := s.feed.SearchFeeds(ctx(t, 45*time.Second), "普吉岛", xiaohongshu.FilterOption{})
	require.NoError(t, err)
	assert.NotEmpty(t, feeds, "search should return results for '普吉岛'")
	t.Logf("search returned %d results", len(feeds))
}

// TestGetFeedDetail verifies we can fetch a note's detail and comments.
// Uses the first feed from GetMyFeeds as the target.
func TestGetFeedDetail(t *testing.T) {
	s := newSuite(t)
	feeds, err := s.feed.GetMyFeeds(ctx(t, 30*time.Second), "", 5)
	require.NoError(t, err)
	if len(feeds) == 0 {
		t.Skip("no personal feeds found — skipping detail test")
	}

	feed := feeds[0]
	detail, err := s.feed.GetFeedDetail(ctx(t, 45*time.Second), feed.ID, feed.XsecToken)
	require.NoError(t, err)
	require.NotNil(t, detail)
	t.Logf("feed detail: title=%s", detail.Note.Title)
}
