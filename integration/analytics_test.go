package integration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestGetContentAnalytics verifies content analytics data is accessible.
func TestGetContentAnalytics(t *testing.T) {
	shortDelay()
	s := newSuite(t)
	result, err := s.analytics.GetContentAnalytics(ctx(t, 45*time.Second), 10, "likes", "desc")
	require.NoError(t, err)
	require.NotNil(t, result)
	t.Logf("content analytics: %d notes", len(result.Notes))
}

// TestGetFanAnalytics verifies fan analytics data is accessible.
func TestGetFanAnalytics(t *testing.T) {
	s := newSuite(t)
	result, err := s.analytics.GetFanAnalytics(ctx(t, 45*time.Second), "7d")
	require.NoError(t, err)
	require.NotNil(t, result)
	t.Logf("fan analytics: total_fans=%d, new_fans=%d", result.Overview.TotalFans, result.Overview.NewFans)
}
