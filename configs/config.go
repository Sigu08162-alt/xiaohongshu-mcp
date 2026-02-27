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

// ===== Version =====

// Version 是当前服务版本号，用于对比本地和生产环境是否对齐。
// 每次发布时手动更新。
const Version = "2.1.0"

// ===== Username =====

// DefaultUsername 仅作为最终兜底，正常情况下应从页面读取真实用户名。
const DefaultUsername = "xiaohongshu-mcp"
