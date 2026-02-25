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
	"math/rand"
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
			"wait_100ms":           100,
			"wait_200ms":           200,
			"wait_300ms":           300,
			"wait_500ms":           500,
			"wait_800ms":           800,
			"wait_1000ms":          1000,
			"wait_2000ms":          2000,
			"wait_3000ms":          3000,
			"wait_5000ms":          5000,
			"wait_10000ms":         10000,
			"wait_60000ms":         60000,
			"wait_300000ms":        300000,
			"wait_600000ms":        600000,
			"read_time_min_ms":     500,
			"read_time_max_ms":     1200,
			"short_read_min_ms":    600,
			"short_read_max_ms":    1200,
			"scroll_wait_min_ms":   100,
			"scroll_wait_max_ms":   200,
			"post_scroll_min_ms":   300,
			"post_scroll_max_ms":   500,
			"scroll_slow_min_ms":   800,
			"scroll_slow_max_ms":   1200,
			"scroll_normal_min_ms": 400,
			"scroll_normal_max_ms": 600,
			"scroll_fast_min_ms":   100,
			"scroll_fast_max_ms":   200,
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

// humanDelay simulates human-like random delay between operations (1.5s ~ 3.0s)
func humanDelay() {
	ms := 1500 + rand.Intn(1500)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// shortDelay simulates a short human pause (500ms ~ 1200ms)
func shortDelay() {
	ms := 500 + rand.Intn(700)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// testCooldown waits between test cases to avoid detection (5s ~ 8s)
func testCooldown() {
	ms := 5000 + rand.Intn(3000)
	time.Sleep(time.Duration(ms) * time.Millisecond)
}

// TestMain controls the overall test execution order.
// Tests should run in the following order to avoid detection and side-effects:
//  1. Read-only tests first (feed, user, analytics)
//  2. Write tests last (like/unlike, comment, publish/delete)
func TestMain(m *testing.M) {
	// Note: Go test framework runs tests in the order they are defined within
	// each file, and files are processed alphabetically. The naming convention
	// and file layout ensure read tests (feed_test, user_test, analytics_test)
	// run before write tests (write_test, delete_test):
	//   analytics_test.go → delete_test.go → feed_test.go → user_test.go → write_test.go
	// Use -run flags to further control ordering when needed.
	os.Exit(m.Run())
}
