package semantic

import (
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Fingerprint struct {
		Anchors    []Anchor `yaml:"anchors"`
		Metrics    []string `yaml:"metrics"`
		Thresholds struct {
			MinAnchorChanges  int     `yaml:"min_anchor_changes"`
			MinMetricChanges  int     `yaml:"min_metric_changes"`
			MetricChangeRatio float64 `yaml:"metric_change_ratio"`
		} `yaml:"thresholds"`
	} `yaml:"fingerprint"`
	ClickPlan struct {
		Targets  []string `yaml:"targets"`
		Precheck struct {
			RequireVisible   bool `yaml:"require_visible"`
			RequireClickable bool `yaml:"require_clickable"`
		} `yaml:"precheck"`
	} `yaml:"click_plan"`
	Verify struct {
		TimeoutSeconds int `yaml:"timeout_seconds"`
	} `yaml:"verify"`
	Fallbacks struct {
		Retry struct {
			MaxAttempts int `yaml:"max_attempts"`
		} `yaml:"retry"`
	} `yaml:"fallbacks"`
	Outputs struct {
		TracePath   string `yaml:"trace_path"`
		FailurePath string `yaml:"failure_path"`
	} `yaml:"outputs"`
}

type Anchor struct {
	Role       string `yaml:"role"`
	Text       string `yaml:"text"`
	AriaLabel  string `yaml:"aria_label"`
	Selector   string `yaml:"selector"`
	Weight     int    `yaml:"weight"`
	MatchExact bool   `yaml:"match_exact"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
