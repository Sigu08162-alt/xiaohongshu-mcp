package xiaohongshu

import (
	"log/slog"
	"math/rand"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

func clickEmptyPosition(page browser.Page) {
	x := 380 + rand.Intn(100)
	y := 20 + rand.Intn(60)
	mouse := page.Mouse()
	mouse.MoveTo(float64(x), float64(y))
	mouse.Click(browser.MouseButtonLeft)
}

func uploadImages(page browser.Page, imagesPaths []string) error {
	pp := page.WithTimeout(30 * time.Second)

	// 验证文件路径有效性
	validPaths := make([]string, 0, len(imagesPaths))
	for _, path := range imagesPaths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			slog.Warn("图片文件不存在:", "arg1", path)
			continue
		}
		validPaths = append(validPaths, path)

		slog.Info("获取有效图片：", "arg1", path)
	}

	// 等待上传输入框出现
	uploadInput, err := pp.Element(`input[type="file"]`)
	if err != nil {
		return errors.Wrap(err, "找不到上传输入框")
	}

	// 上传多个文件
	if err := uploadInput.SetFiles(validPaths); err != nil {
		return errors.Wrap(err, "设置文件失败")
	}

	// 等待并验证上传完成
	return waitForUploadComplete(pp, len(validPaths))
}

// waitForUploadComplete 等待并验证上传完成
func waitForUploadComplete(page browser.Page, expectedCount int) error {
	maxWaitTime := 60 * time.Second
	checkInterval := 500 * time.Millisecond
	start := time.Now()

	slog.Info("开始等待图片上传完成", "expected_count", expectedCount)

	for time.Since(start) < maxWaitTime {
		// 使用具体的pr类名检查已上传的图片
		uploadedImages, err := page.Elements(".img-preview-area .pr")

		slog.Info("uploadedImages", "uploadedImages", uploadedImages)

		if err == nil {
			currentCount := len(uploadedImages)
			slog.Info("检测到已上传图片", "current_count", currentCount, "expected_count", expectedCount)
			if currentCount >= expectedCount {
				slog.Info("所有图片上传完成", "count", currentCount)
				return nil
			}
		} else {
			slog.Debug("未找到已上传图片元素")
		}

		time.Sleep(checkInterval)
	}

	return errors.New("上传超时，请检查网络连接和图片大小")
}
