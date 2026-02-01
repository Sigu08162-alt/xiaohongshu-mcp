package playwright

import (
	"os"
	"testing"
)

func TestEngineConfig_Defaults(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.Headless {
		t.Fatalf("expected headless true")
	}
}

func TestEngineConfig_ViewportFromEnv(t *testing.T) {
	os.Setenv("XHS_VIEWPORT_WIDTH", "1366")
	os.Setenv("XHS_VIEWPORT_HEIGHT", "900")
	t.Cleanup(func() {
		os.Unsetenv("XHS_VIEWPORT_WIDTH")
		os.Unsetenv("XHS_VIEWPORT_HEIGHT")
	})
	cfg := DefaultConfig()
	if cfg.ViewportWidth != 1366 || cfg.ViewportHeight != 900 {
		t.Fatalf("expected viewport 1366x900, got %dx%d", cfg.ViewportWidth, cfg.ViewportHeight)
	}
}
