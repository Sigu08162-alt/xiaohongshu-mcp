package publish

import (
	"context"

	"github.com/xpzouying/xiaohongshu-mcp/internal/app/ports"
	"github.com/xpzouying/xiaohongshu-mcp/internal/domain/publish"
	"github.com/xpzouying/xiaohongshu-mcp/pkg/downloader"
)

// ImageProcessorInterface 图片处理接口，用于依赖注入和测试
type ImageProcessorInterface interface {
	ProcessImages(images []string) ([]string, error)
}

type Usecase struct {
	Gateway        ports.PublishGateway
	Limits         publish.Limits
	ImageProcessor ImageProcessorInterface // 图片处理器
}

func (u Usecase) PublishImage(ctx context.Context, content publish.ImageContent) error {
	// 1. 处理图片（URL下载 + 路径验证）
	processedPaths, err := u.ImageProcessor.ProcessImages(content.ImagePaths)
	if err != nil {
		return err
	}
	content.ImagePaths = processedPaths

	// 2. 验证
	if err := publish.ValidateImageContent(content, u.Limits); err != nil {
		return err
	}

	// 3. 委托 Gateway 执行
	return u.Gateway.PublishImage(ctx, content)
}

func (u Usecase) SaveImageDraft(ctx context.Context, content publish.ImageContent) error {
	// 1. 处理图片（URL下载 + 路径验证）- 与 PublishImage 保持一致
	processedPaths, err := u.ImageProcessor.ProcessImages(content.ImagePaths)
	if err != nil {
		return err
	}
	content.ImagePaths = processedPaths

	// 2. 验证
	if err := publish.ValidateImageContent(content, u.Limits); err != nil {
		return err
	}

	// 3. 委托 Gateway 执行
	return u.Gateway.SaveImageDraft(ctx, content)
}

func (u Usecase) SaveVideoDraft(ctx context.Context, content publish.VideoContent) error {
	return u.Gateway.SaveVideoDraft(ctx, content)
}

// 确保 downloader.ImageProcessor 实现了 ImageProcessorInterface
var _ ImageProcessorInterface = (*downloader.ImageProcessor)(nil)
