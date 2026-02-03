package main

import (
	"os"

	infraconfig "github.com/vmxmy/xiaohongshu-mcp/internal/infra/config"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/polling"
)

func loadPollingModules() (PollingModules, error) {
	configPath := os.Getenv("XHS_CONFIG_PATH")
	if configPath == "" {
		configPath = "config.yaml"
	}
	cfg, err := infraconfig.LoadFromFile(configPath)
	if err != nil {
		return PollingModules{}, err
	}
	return PollingModules{
		Publish:     toPollingModule(cfg.Polling.Publish),
		Draft:       toPollingModule(cfg.Polling.Draft),
		Video:       toPollingModule(cfg.Polling.Video),
		Interaction: toPollingModule(cfg.Polling.Interaction),
		Analytics:   toPollingModule(cfg.Polling.Analytics),
		Auth:        toPollingModule(cfg.Polling.Auth),
	}, nil
}

func toPollingModule(module infraconfig.PollingModule) polling.Module {
	return polling.Module{
		TimeoutMs:  module.TimeoutMs,
		IntervalMs: module.IntervalMs,
		MaxRetries: module.MaxRetries,
		Delays:     module.Delays,
	}
}
