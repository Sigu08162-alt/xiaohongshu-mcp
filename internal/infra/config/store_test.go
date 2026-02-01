package config

import "testing"

func TestLoadConfig_File(t *testing.T) {
	cfg, err := LoadFromFile("testdata/config.yaml")
	if err != nil || cfg.URLs.Creator.PublishImage == "" || cfg.Limits.MaxTags == 0 {
		t.Fatalf("expected publish_image url and limits")
	}
	if cfg.Timeouts.Navigate == 0 || cfg.Timeouts.ElementWait == 0 {
		t.Fatalf("expected timeouts")
	}
	if cfg.Polling.Publish.TimeoutMs == 0 || cfg.Polling.Publish.IntervalMs == 0 {
		t.Fatalf("expected polling publish config")
	}
}

func TestConfig_LoadRequiresPollingModules(t *testing.T) {
	_, err := LoadFromFile("testdata/config_missing_polling.yaml")
	if err == nil {
		t.Fatalf("expected error when polling config missing")
	}
}

func TestConfig_LoadRequiresVideoDelays(t *testing.T) {
	_, err := LoadFromFile("testdata/config_missing_video_delay.yaml")
	if err == nil {
		t.Fatalf("expected error when video delays missing")
	}
}
