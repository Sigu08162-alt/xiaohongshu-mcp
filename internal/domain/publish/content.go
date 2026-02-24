package publish

import (
	"fmt"
	"strings"
	"time"
)

type ImageContent struct {
	Title        string
	Content      string
	Tags         []string
	ImagePaths   []string
	Location     string
	MarkerTags   []string
	ScheduleTime *time.Time
}

type VideoContent struct {
	Title        string
	Content      string
	Tags         []string
	VideoPath    string
	ScheduleTime *time.Time
}

type Limits struct {
	MaxTags   int
	MinImages int
	MaxImages int
}

// Validate checks that ImageContent satisfies business rules given limits.
func (c ImageContent) Validate(limits Limits) error {
	if len(c.ImagePaths) < limits.MinImages {
		return fmt.Errorf("图片数量不足: %d", len(c.ImagePaths))
	}
	if len(c.ImagePaths) > limits.MaxImages {
		return fmt.Errorf("图片数量过多: %d", len(c.ImagePaths))
	}
	if len(c.Tags) > limits.MaxTags {
		return fmt.Errorf("标签数量过多: %d", len(c.Tags))
	}
	return nil
}

// Validate checks that VideoContent satisfies business rules.
func (c VideoContent) Validate(maxTags int) error {
	if strings.TrimSpace(c.VideoPath) == "" {
		return fmt.Errorf("视频路径不能为空")
	}
	if len(c.Tags) > maxTags {
		return fmt.Errorf("标签数量过多: %d", len(c.Tags))
	}
	return nil
}

// WithProcessedImages returns a copy of ImageContent with replaced image paths.
func (c ImageContent) WithProcessedImages(paths []string) ImageContent {
	c.ImagePaths = paths
	return c
}

// DeduplicateTags removes duplicate tags in-place, preserving order.
func (c *ImageContent) DeduplicateTags() {
	seen := make(map[string]struct{}, len(c.Tags))
	result := c.Tags[:0]
	for _, t := range c.Tags {
		if _, ok := seen[t]; !ok {
			seen[t] = struct{}{}
			result = append(result, t)
		}
	}
	c.Tags = result
}

// ValidateScheduleTime checks that ScheduleTime is between 1 hour and 14 days from now.
func (c ImageContent) ValidateScheduleTime() error {
	return validateScheduleTime(c.ScheduleTime)
}

// ValidateScheduleTime checks that ScheduleTime is between 1 hour and 14 days from now.
func (c VideoContent) ValidateScheduleTime() error {
	return validateScheduleTime(c.ScheduleTime)
}

func validateScheduleTime(t *time.Time) error {
	if t == nil {
		return nil
	}
	now := time.Now()
	min := now.Add(1 * time.Hour)
	max := now.Add(14 * 24 * time.Hour)
	if t.Before(min) {
		return fmt.Errorf("定时发布时间必须至少在1小时后，最早可选: %s", min.Format("2006-01-02 15:04"))
	}
	if t.After(max) {
		return fmt.Errorf("定时发布时间不能超过14天，最晚可选: %s", max.Format("2006-01-02 15:04"))
	}
	return nil
}

func FilterMarkerTags(markers []string) []string {
	if len(markers) == 0 {
		return nil
	}

	filtered := make([]string, 0, len(markers))
	for _, marker := range markers {
		if strings.TrimSpace(marker) == "" {
			continue
		}
		filtered = append(filtered, marker)
	}

	if len(filtered) == 0 {
		return nil
	}
	return filtered
}

// ValidateImageContent is kept for backward compatibility.
func ValidateImageContent(c ImageContent, limits Limits) error {
	return c.Validate(limits)
}
