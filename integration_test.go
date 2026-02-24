//go:build integration

package main

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vmxmy/xiaohongshu-mcp/configs"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
	"github.com/vmxmy/xiaohongshu-mcp/internal/interfaces/wiring"
)

// newIntegrationService 构建真实 service，复用 main.go 的组装逻辑
func newIntegrationService(t *testing.T) *XiaohongshuService {
	t.Helper()

	cookiePath := "/tmp/cookies.json"
	if p := os.Getenv("COOKIES_PATH"); p != "" {
		cookiePath = p
	}
	if _, err := os.Stat(cookiePath); os.IsNotExist(err) {
		t.Skipf("cookie file not found at %s, run: cp cookies.json /tmp/cookies.json", cookiePath)
	}
	os.Setenv("COOKIES_PATH", cookiePath)

	configs.InitHeadless(true)

	publishUsecase := wiring.InitPublishUsecase(true)
	modules, err := loadPollingModules()
	require.NoError(t, err)

	engineFactory := func() browser.Engine { return newBrowserEngine() }
	feedUsecase := wiring.BuildFeedUsecase(engineFactory, modules.Interaction)
	interactUsecase := wiring.BuildInteractionUsecase(engineFactory, modules.Interaction)
	userUsecase := wiring.BuildUserUsecase(engineFactory, modules.Interaction)
	analyticsUsecase := wiring.BuildAnalyticsUsecase(engineFactory, modules.Analytics)

	svc, err := NewXiaohongshuServiceWithModules(
		publishUsecase, feedUsecase, interactUsecase, userUsecase, analyticsUsecase, modules,
	)
	require.NoError(t, err)
	return svc
}
