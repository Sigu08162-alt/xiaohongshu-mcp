package publish_test

import (
	"context"
	"errors"
	"testing"

	appublish "github.com/vmxmy/xiaohongshu-mcp/internal/app/publish"
	"github.com/vmxmy/xiaohongshu-mcp/internal/app/testkit"
	"github.com/vmxmy/xiaohongshu-mcp/internal/domain/publish"
)

// fakeImageProcessor satisfies appublish.ImageProcessorInterface
type fakeImageProcessor struct {
	Paths []string
	Err   error
}

func (f *fakeImageProcessor) ProcessImages(images []string) ([]string, error) {
	if f.Err != nil {
		return nil, f.Err
	}
	if f.Paths != nil {
		return f.Paths, nil
	}
	return images, nil
}

var testLimits = publish.Limits{MinImages: 1, MaxImages: 9, MaxTags: 30}

func TestPublishImage_OK(t *testing.T) {
	gw := &testkit.FakePublishGateway{}
	uc := appublish.Usecase{
		Gateway:        gw,
		Limits:         testLimits,
		ImageProcessor: &fakeImageProcessor{},
	}

	err := uc.PublishImage(context.Background(), publish.ImageContent{
		Title:      "测试",
		ImagePaths: []string{"/tmp/a.jpg"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gw.ImageCalls != 1 {
		t.Errorf("expected 1 publish call, got %d", gw.ImageCalls)
	}
}

func TestPublishImage_ValidationFails(t *testing.T) {
	gw := &testkit.FakePublishGateway{}
	uc := appublish.Usecase{
		Gateway:        gw,
		Limits:         testLimits,
		ImageProcessor: &fakeImageProcessor{},
	}

	// 0 images → validation error
	err := uc.PublishImage(context.Background(), publish.ImageContent{
		Title:      "测试",
		ImagePaths: []string{},
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if gw.ImageCalls != 0 {
		t.Errorf("gateway should not be called on validation failure")
	}
}

func TestPublishImage_ImageProcessorError(t *testing.T) {
	gw := &testkit.FakePublishGateway{}
	uc := appublish.Usecase{
		Gateway:        gw,
		Limits:         testLimits,
		ImageProcessor: &fakeImageProcessor{Err: errors.New("download failed")},
	}

	err := uc.PublishImage(context.Background(), publish.ImageContent{
		Title:      "测试",
		ImagePaths: []string{"https://example.com/img.jpg"},
	})
	if err == nil {
		t.Fatal("expected error from image processor")
	}
	if gw.ImageCalls != 0 {
		t.Errorf("gateway should not be called when processor fails")
	}
}

func TestPublishImage_GatewayError(t *testing.T) {
	gw := &testkit.FakePublishGateway{Err: errors.New("browser error")}
	uc := appublish.Usecase{
		Gateway:        gw,
		Limits:         testLimits,
		ImageProcessor: &fakeImageProcessor{},
	}

	err := uc.PublishImage(context.Background(), publish.ImageContent{
		Title:      "测试",
		ImagePaths: []string{"/tmp/a.jpg"},
	})
	if err == nil {
		t.Fatal("expected gateway error")
	}
}
