package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	URLs struct {
		Creator struct {
			PublishImage string `yaml:"publish_image"`
			PublishVideo string `yaml:"publish_video"`
		} `yaml:"creator"`
	} `yaml:"urls"`
	Limits struct {
		MaxTags   int `yaml:"max_tags"`
		MinImages int `yaml:"min_images"`
		MaxImages int `yaml:"max_images"`
	} `yaml:"limits"`
	Timeouts struct {
		Navigate    int `yaml:"navigate"`
		ElementWait int `yaml:"element_wait"`
		ImageUpload int `yaml:"image_upload"`
	} `yaml:"timeouts"`
	Polling struct {
		Publish     PollingModule `yaml:"publish"`
		Draft       PollingModule `yaml:"draft"`
		Video       PollingModule `yaml:"video"`
		Interaction PollingModule `yaml:"interaction"`
		Analytics   PollingModule `yaml:"analytics"`
		Auth        PollingModule `yaml:"auth"`
	} `yaml:"polling"`
}

type PollingModule struct {
	TimeoutMs  int `yaml:"timeout_ms"`
	IntervalMs int `yaml:"interval_ms"`
	MaxRetries int `yaml:"max_retries"`
}

func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Config) validate() error {
	if err := validatePollingModule("publish", c.Polling.Publish); err != nil {
		return err
	}
	if err := validatePollingModule("draft", c.Polling.Draft); err != nil {
		return err
	}
	if err := validatePollingModule("video", c.Polling.Video); err != nil {
		return err
	}
	if err := validatePollingModule("interaction", c.Polling.Interaction); err != nil {
		return err
	}
	if err := validatePollingModule("analytics", c.Polling.Analytics); err != nil {
		return err
	}
	if err := validatePollingModule("auth", c.Polling.Auth); err != nil {
		return err
	}
	return nil
}

func validatePollingModule(name string, module PollingModule) error {
	if module.TimeoutMs <= 0 {
		return fmt.Errorf("polling.%s.timeout_ms missing or invalid", name)
	}
	if module.IntervalMs <= 0 {
		return fmt.Errorf("polling.%s.interval_ms missing or invalid", name)
	}
	if module.MaxRetries <= 0 {
		return fmt.Errorf("polling.%s.max_retries missing or invalid", name)
	}
	return nil
}
