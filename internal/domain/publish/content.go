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

func ValidateImageContent(c ImageContent, limits Limits) error {
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
