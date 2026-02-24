package publish

import (
	"fmt"
	"strings"
	"time"

	"github.com/vmxmy/xiaohongshu-mcp/internal/infra/browser"
)

func resolveUploadSelectors(selectors map[string]string) UploadStateSelectors {
	return UploadStateSelectors{
		UploadingMask:  getSelectorOrDefault(selectors, "uploading_mask", ".mask.uploading"),
		UploadingClass: getSelectorOrDefault(selectors, "uploading_class", "[class*='uploading']"),
		UploadPreview:  getSelectorOrDefault(selectors, "upload_preview", "img.preview"),
		UploadingToast: getSelectorOrDefault(selectors, "uploading_toast", ".creator-publish-toast"),
	}
}

func getSelectorOrDefault(selectors map[string]string, key, defaultValue string) string {
	if selectors == nil {
		return defaultValue
	}
	if value, ok := selectors[key]; ok && strings.TrimSpace(value) != "" {
		return value
	}
	return defaultValue
}

func waitForUploadComplete(page browser.Page, selectors UploadStateSelectors, expectedCount int, maxWait, interval time.Duration) error {
	deadline := time.Now().Add(maxWait)
	uploadingSelectors := append(splitSelectors(selectors.UploadingMask), splitSelectors(selectors.UploadingClass)...)
	previewSelectors := splitSelectors(selectors.UploadPreview)

	for {
		isUploading := false
		for _, sel := range uploadingSelectors {
			if visible, _ := page.IsVisible(sel); visible {
				isUploading = true
				break
			}
		}

		countOk := true
		if expectedCount > 0 && len(previewSelectors) > 0 {
			countOk = false
			if v, err := page.Eval(`(selectors) => {
				let maxCount = 0;
				for (const sel of selectors) {
					const count = document.querySelectorAll(sel).length;
					if (count > maxCount) maxCount = count;
				}
				return maxCount;
			}`, previewSelectors); err == nil {
				switch n := v.(type) {
				case int:
					countOk = n >= expectedCount
				case int64:
					countOk = int(n) >= expectedCount
				case float64:
					countOk = int(n) >= expectedCount
				case float32:
					countOk = int(n) >= expectedCount
				}
			}
		}

		if !isUploading && countOk {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("图片上传中，请稍后")
		}
		time.Sleep(interval)
	}
}
