package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetMyProfile verifies we can fetch our own profile.
func TestGetMyProfile(t *testing.T) {
	shortDelay()
	s := newSuite(t)
	profile, err := s.user.GetMyProfile(ctx(t, 30*time.Second))
	require.NoError(t, err)
	require.NotNil(t, profile)
	t.Logf("profile: nickname=%s", profile.UserBasicInfo.Nickname)
}

// TestGetMyStats verifies we can fetch account stats.
func TestGetMyStats(t *testing.T) {
	shortDelay()
	s := newSuite(t)
	stats, err := s.user.GetMyStats(ctx(t, 60*time.Second))
	require.NoError(t, err)
	assert.NotEmpty(t, stats)
	t.Logf("stats keys: %d", len(stats))
}

// TestGetUserProfile verifies we can fetch another user's profile.
// Uses the author of the first personal feed.
func TestGetUserProfile(t *testing.T) {
	shortDelay()
	s := newSuite(t)

	feeds, err := s.feed.GetMyFeeds(ctx(t, 30*time.Second), "", 3)
	require.NoError(t, err)
	if len(feeds) == 0 {
		t.Skip("no feeds to extract user ID from")
	}

	feed := feeds[0]
	userID := feed.NoteCard.User.UserID
	if userID == "" {
		t.Skip("user ID empty in feed")
	}

	profile, err := s.user.GetUserProfile(ctx(t, 30*time.Second), userID, feed.XsecToken)
	require.NoError(t, err)
	require.NotNil(t, profile)
	t.Logf("user profile: %s", profile.UserBasicInfo.Nickname)
}
