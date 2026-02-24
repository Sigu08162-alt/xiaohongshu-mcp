package publish_test

import (
	"testing"
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
)

var defaultLimits = publish.Limits{MinImages: 1, MaxImages: 9, MaxTags: 30}

func TestImageContent_Validate_OK(t *testing.T) {
	c := publish.ImageContent{
		Title:      "测试标题",
		ImagePaths: []string{"/tmp/a.jpg"},
		Tags:       []string{"旅行"},
	}
	if err := c.Validate(defaultLimits); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestImageContent_Validate_TooFewImages(t *testing.T) {
	c := publish.ImageContent{Title: "t", ImagePaths: []string{}}
	if err := c.Validate(defaultLimits); err == nil {
		t.Error("expected error for 0 images")
	}
}

func TestImageContent_Validate_TooManyImages(t *testing.T) {
	paths := make([]string, 10)
	for i := range paths {
		paths[i] = "/tmp/img.jpg"
	}
	c := publish.ImageContent{Title: "t", ImagePaths: paths}
	if err := c.Validate(defaultLimits); err == nil {
		t.Error("expected error for 10 images")
	}
}

func TestImageContent_Validate_TooManyTags(t *testing.T) {
	tags := make([]string, 31)
	for i := range tags {
		tags[i] = "tag"
	}
	c := publish.ImageContent{Title: "t", ImagePaths: []string{"/tmp/a.jpg"}, Tags: tags}
	if err := c.Validate(defaultLimits); err == nil {
		t.Error("expected error for 31 tags")
	}
}

func TestImageContent_DeduplicateTags(t *testing.T) {
	c := publish.ImageContent{Tags: []string{"a", "b", "a", "c", "b"}}
	c.DeduplicateTags()
	if len(c.Tags) != 3 {
		t.Errorf("expected 3 unique tags, got %d: %v", len(c.Tags), c.Tags)
	}
}

func TestVideoContent_Validate_EmptyPath(t *testing.T) {
	c := publish.VideoContent{Title: "t", VideoPath: ""}
	if err := c.Validate(30); err == nil {
		t.Error("expected error for empty video path")
	}
}

func TestVideoContent_Validate_OK(t *testing.T) {
	c := publish.VideoContent{Title: "t", VideoPath: "/tmp/video.mp4"}
	if err := c.Validate(30); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidateScheduleTime_TooSoon(t *testing.T) {
	soon := time.Now().Add(30 * time.Minute)
	c := publish.ImageContent{
		ImagePaths:   []string{"/tmp/a.jpg"},
		ScheduleTime: &soon,
	}
	if err := c.ValidateScheduleTime(); err == nil {
		t.Error("expected error for schedule time < 1 hour")
	}
}

func TestValidateScheduleTime_TooLate(t *testing.T) {
	late := time.Now().Add(15 * 24 * time.Hour)
	c := publish.ImageContent{
		ImagePaths:   []string{"/tmp/a.jpg"},
		ScheduleTime: &late,
	}
	if err := c.ValidateScheduleTime(); err == nil {
		t.Error("expected error for schedule time > 14 days")
	}
}

func TestValidateScheduleTime_OK(t *testing.T) {
	valid := time.Now().Add(2 * time.Hour)
	c := publish.ImageContent{
		ImagePaths:   []string{"/tmp/a.jpg"},
		ScheduleTime: &valid,
	}
	if err := c.ValidateScheduleTime(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestFilterMarkerTags_RemovesEmpty(t *testing.T) {
	result := publish.FilterMarkerTags([]string{"tag1", "", "  ", "tag2"})
	if len(result) != 2 {
		t.Errorf("expected 2 tags, got %d: %v", len(result), result)
	}
}

func TestFilterMarkerTags_NilInput(t *testing.T) {
	result := publish.FilterMarkerTags(nil)
	if result != nil {
		t.Errorf("expected nil, got %v", result)
	}
}
