package integration

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// IntegrationTestLikeUnlike tests like + unlike on the first personal feed.
// Requires explicit: go test ./integration/... -run Integration
func TestIntegrationLikeUnlike(t *testing.T) {
	// Wait before starting to avoid bot-detection patterns
	testCooldown()

	requireCookies(t)
	s := newSuite(t)

	feeds, err := s.feed.GetMyFeeds(ctx(t, 30*time.Second), "", 3)
	require.NoError(t, err)
	require.NotEmpty(t, feeds, "need at least one feed to test like")

	feed := feeds[0]
	t.Logf("testing like on feed: %s", feed.ID)

	err = s.interaction.LikeFeed(ctx(t, 30*time.Second), feed.ID, feed.XsecToken)
	require.NoError(t, err, "like should succeed")

	// Simulate human pause between like and unlike
	humanDelay()

	err = s.interaction.UnlikeFeed(ctx(t, 30*time.Second), feed.ID, feed.XsecToken)
	require.NoError(t, err, "unlike should succeed")

	t.Log("like/unlike cycle completed")
}

// IntegrationTestPostDeleteComment tests posting and deleting a comment.
// Requires explicit: go test ./integration/... -run Integration
func TestIntegrationPostDeleteComment(t *testing.T) {
	// Wait before starting to avoid bot-detection patterns
	testCooldown()

	requireCookies(t)
	s := newSuite(t)

	feeds, err := s.feed.GetMyFeeds(ctx(t, 30*time.Second), "", 3)
	require.NoError(t, err)
	require.NotEmpty(t, feeds, "need at least one feed to test comment")

	feed := feeds[0]
	testComment := fmt.Sprintf("[TEST] integration test comment %d", time.Now().Unix())
	t.Logf("posting comment on feed %s: %s", feed.ID, testComment)

	err = s.interaction.PostComment(ctx(t, 45*time.Second), feed.ID, feed.XsecToken, testComment)
	require.NoError(t, err, "post comment should succeed")

	// Simulate human pause between posting and deleting the comment
	humanDelay()

	// Fetch detail to find the comment ID for deletion
	detail, err := s.feed.GetFeedDetail(ctx(t, 45*time.Second), feed.ID, feed.XsecToken)
	require.NoError(t, err)

	// Find our test comment
	for _, c := range detail.Comments.List {
		if c.Content == testComment {
			err = s.interaction.DeleteComment(ctx(t, 30*time.Second), feed.ID, feed.XsecToken, c.ID, c.UserInfo.UserID)
			require.NoError(t, err, "delete comment should succeed")
			t.Log("comment posted and deleted successfully")
			return
		}
	}
	t.Log("comment posted (cleanup skipped — comment ID not found in detail)")
}
