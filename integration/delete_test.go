package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestIntegrationDeleteFeed tests DeleteFeed on the most recent personal feed.
// Strategy: fetch my feeds, pick the LAST one (oldest), delete it.
// WARNING: This is a destructive operation. Only run explicitly.
//
// Run with:
//
//	go test ./integration/... -run TestIntegrationDeleteFeed -v -timeout 120s
func TestIntegrationDeleteFeed(t *testing.T) {
	requireCookies(t)
	s := newSuite(t)

	// 1. Get my feeds
	feeds, err := s.feed.GetMyFeeds(ctx(t, 30*time.Second), "", 20)
	require.NoError(t, err, "GetMyFeeds should succeed")
	require.NotEmpty(t, feeds, "need at least one feed to test delete")

	// 2. Pick the last feed (oldest) to minimize impact
	target := feeds[len(feeds)-1]
	t.Logf("target feed to delete: id=%s title=%s", target.ID, target.NoteCard.DisplayTitle)

	// 3. Confirm it exists via GetMyFeeds (sanity check)
	t.Logf("proceeding with delete on feed_id=%s xsec_token=%s", target.ID, target.XsecToken)

	// 4. Delete
	err = s.feed.DeleteFeed(ctx(t, 60*time.Second), target.ID, target.XsecToken)
	if err != nil {
		t.Logf("DeleteFeed error: %v", err)
		t.Logf("selector fallback debug: check slog output above for 'selector attempt failed' messages")
	}
	require.NoError(t, err, "DeleteFeed should succeed")

	t.Logf("✅ feed %s deleted successfully", target.ID)

	// 5. Verify: re-fetch and confirm feed is gone
	time.Sleep(2 * time.Second)
	feeds2, err := s.feed.GetMyFeeds(ctx(t, 30*time.Second), "", 20)
	require.NoError(t, err, "re-fetch feeds should succeed")

	for _, f := range feeds2 {
		if f.ID == target.ID {
			t.Errorf("feed %s still exists after delete — delete may have failed silently", target.ID)
			return
		}
	}
	t.Logf("✅ verified: feed %s no longer in feed list", target.ID)
}
