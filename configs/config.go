package configs

import (
	"os"
	"path/filepath"
)

// ===== Browser =====

var (
	useHeadless = true

	binPath = ""
)

func InitHeadless(h bool) {
	useHeadless = h
}

// IsHeadless 是否无头模式。
func IsHeadless() bool {
	return useHeadless
}

func SetBinPath(b string) {
	binPath = b
}

func GetBinPath() string {
	return binPath
}

// ===== Image =====

const (
	ImagesDir = "xiaohongshu_images"
)

func GetImagesPath() string {
	return filepath.Join(os.TempDir(), ImagesDir)
}

// ===== Username =====

const (
	Username = "xiaohongshu-mcp"
)
