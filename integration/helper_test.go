// Package integration contains real-browser integration tests.
// These tests require a valid cookies.json and a running network connection.
//
// Run read-only tests:
//
//	go test ./integration/... -v -timeout 120s
//
// Run write tests (publish/comment/like) explicitly:
//
//	go test ./integration/... -v -run Integration -timeout 180s
package integration

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	appanalytics "github.com/vmxmy/xiaohongshu-mcp/internal/app/analytics"
	appfeed "github.com/vmxmy/xiaohongshu-mcp/internal/app/feed"
	appinteraction "github.com/vmxmy/xiaohongshu-mcp/internal/app/interaction"
	appuser "github.com/vmxmy/xiaohongshu-mcp/internal/app/user"
	"github.com/vmxmy/xiaohongshu-mcp/cookies"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	browserplaywright "github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser/playwright"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
	"github.com/vmxmy/xiaohongshu-mcp/internal/interfaces/wiring"
)

// testSuite holds all usecases backed by a real browser.
type testSuite struct {
	feed        *appfeed.Usecase
	interaction *appinteraction.Usecase
	user        *appuser.Usecase
	analytics   *appanalytics.Usecase
}

// newSuite creates a real browser-backed test suite.
// Skips automatically if cookies.json is missing.
func newSuite(t *testing.T) *testSuite {
	t.Helper()

	cookiePath := cookies.GetCookiesFilePath()
	if _, err := os.Stat(cookiePath); os.IsNotExist(err) {
		t.Skipf("cookies.json not found at %s — skipping integration test", cookiePath)
	}

	cfg := browserplaywright.DefaultConfig()
	cfg.Headless = true
	cfg.CookiePath = cookiePath
	cfg.NavigationTimeout = 30 * time.Second
	cfg.ActionTimeout = 15 * time.Second

	// engineFactory creates a fresh engine per call (pool handles reuse internally)
	engineFactory := func() browser.Engine {
		return browserplaywright.New(cfg)
	}

	noop := polling.Module{
		TimeoutMs:  60000,
		IntervalMs: 1000,
		MaxRetries: 3,
		Delays: map[string]int{
			"wait_100ms":    100,
			"wait_200ms":    200,
			"wait_300ms":    300,
			"wait_500ms":    500,
			"wait_800ms":    800,
			"wait_1000ms":   1000,
			"wait_2000ms":   2000,
			"wait_3000ms":   3000,
			"wait_5000ms":   5000,
			"wait_10000ms":  10000,
			"wait_60000ms":  60000,
			"wait_300000ms": 300000,
			"wait_600000ms": 600000,
		},
	}

	return &testSuite{
		feed:        wiring.BuildFeedUsecase(engineFactory, noop),
		interaction: wiring.BuildInteractionUsecase(engineFactory, noop),
		user:        wiring.BuildUserUsecase(engineFactory, noop),
		analytics:   wiring.BuildAnalyticsUsecase(engineFactory, noop),
	}
}

// ctx returns a context with a per-test timeout.
func ctx(t *testing.T, d time.Duration) context.Context {
	t.Helper()
	c, cancel := context.WithTimeout(context.Background(), d)
	t.Cleanup(cancel)
	return c
}

// requireCookies fails fast if cookies are missing (for write tests).
func requireCookies(t *testing.T) {
	t.Helper()
	_, err := os.Stat(cookies.GetCookiesFilePath())
	require.NoError(t, err, "cookies.json required for this test")
}
