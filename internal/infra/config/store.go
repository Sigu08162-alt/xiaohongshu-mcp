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
	Delays     map[string]int `yaml:"delays"`
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
	if err := validatePollingModule("publish", c.Polling.Publish, []string{
		"page_stable_ms",
		"pre_submit_render_ms",
		"post_content_render_ms",
		"scroll_into_view_wait_ms",
		"click_retry_wait_ms",
		"tag_editor_ready_ms",
		"tag_arrow_step_ms",
		"tag_after_enter_ms",
		"tag_hash_delay_ms",
		"tag_char_delay_ms",
		"tag_after_text_ms",
		"tag_suggestion_click_ms",
		"tag_after_tag_ms",
	}); err != nil {
		return err
	}
	if err := validatePollingModule("draft", c.Polling.Draft, []string{
		"page_stable_ms",
		"post_content_render_ms",
		"scroll_into_view_wait_ms",
		"click_retry_wait_ms",
		"tag_editor_ready_ms",
		"tag_arrow_step_ms",
		"tag_after_enter_ms",
		"tag_hash_delay_ms",
		"tag_char_delay_ms",
		"tag_after_text_ms",
		"tag_suggestion_click_ms",
		"tag_after_tag_ms",
	}); err != nil {
		return err
	}
	if err := validatePollingModule("video", c.Polling.Video, []string{
		"wait_300000ms",
		"wait_3000ms",
		"wait_1000ms",
		"post_content_render_ms",
		"scroll_into_view_wait_ms",
		"draft_save_wait_ms",
	}); err != nil {
		return err
	}
	if err := validatePollingModule("interaction", c.Polling.Interaction, []string{
		"wait_600000ms",
		"wait_60000ms",
		"wait_300000ms",
		"wait_10000ms",
		"wait_800ms",
		"wait_5000ms",
		"wait_3000ms",
		"wait_2000ms",
		"wait_1000ms",
		"wait_500ms",
		"wait_300ms",
		"wait_200ms",
		"wait_100ms",
		"human_delay_min_ms",
		"human_delay_max_ms",
		"reaction_time_min_ms",
		"reaction_time_max_ms",
		"hover_time_min_ms",
		"hover_time_max_ms",
		"read_time_min_ms",
		"read_time_max_ms",
		"short_read_min_ms",
		"short_read_max_ms",
		"scroll_wait_min_ms",
		"scroll_wait_max_ms",
		"post_scroll_min_ms",
		"post_scroll_max_ms",
		"scroll_slow_min_ms",
		"scroll_slow_max_ms",
		"scroll_normal_min_ms",
		"scroll_normal_max_ms",
		"scroll_fast_min_ms",
		"scroll_fast_max_ms",
	}); err != nil {
		return err
	}
	if err := validatePollingModule("analytics", c.Polling.Analytics, []string{
		"wait_300000ms",
		"wait_60000ms",
		"wait_5000ms",
		"wait_2000ms",
		"wait_1000ms",
		"wait_500ms",
	}); err != nil {
		return err
	}
	if err := validatePollingModule("auth", c.Polling.Auth, []string{
		"wait_2000ms",
		"wait_1000ms",
	}); err != nil {
		return err
	}
	return nil
}

func validatePollingModule(name string, module PollingModule, requiredDelays []string) error {
	if module.TimeoutMs <= 0 {
		return fmt.Errorf("polling.%s.timeout_ms missing or invalid", name)
	}
	if module.IntervalMs <= 0 {
		return fmt.Errorf("polling.%s.interval_ms missing or invalid", name)
	}
	if module.MaxRetries <= 0 {
		return fmt.Errorf("polling.%s.max_retries missing or invalid", name)
	}
	for _, key := range requiredDelays {
		if module.Delays == nil {
			return fmt.Errorf("polling.%s.delays missing", name)
		}
		if value, ok := module.Delays[key]; !ok || value <= 0 {
			return fmt.Errorf("polling.%s.delays.%s missing or invalid", name, key)
		}
	}
	return nil
}
