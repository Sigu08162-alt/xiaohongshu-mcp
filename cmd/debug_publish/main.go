package main

import (
	"context"
	"flag"
	"log"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/internal/domain/publish"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser/playwright"
	publishGateway "github.com/xpzouying/xiaohongshu-mcp/internal/infra/xhs/publish"
)

var (
	imagePaths = flag.String("images", "", "图片路径,逗号分隔")
	title      = flag.String("title", "测试标题", "标题")
	content    = flag.String("content", "测试内容", "内容")
	headless   = flag.Bool("headless", false, "是否使用无头模式")
)

func main() {
	flag.Parse()

	// 设置日志级别
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-02 15:04:05",
	})

	if *imagePaths == "" {
		log.Fatal("请提供图片路径: -images path1,path2")
	}

	// 解析图片路径
	paths := []string{}
	for _, p := range strings.Split(*imagePaths, ",") {
		p = strings.TrimSpace(p)
		if p != "" {
			paths = append(paths, p)
		}
	}

	if len(paths) == 0 {
		log.Fatal("没有有效的图片路径")
	}

	logrus.Infof("开始调试发布流程")
	logrus.Infof("  - 无头模式: %v", *headless)
	logrus.Infof("  - 标题: %s", *title)
	logrus.Infof("  - 内容: %s", *content)
	logrus.Infof("  - 图片数: %d", len(paths))

	// 创建浏览器引擎
	engine := playwright.New(playwright.Config{
		Headless:   *headless,
		CookiePath: "/Users/xumingyang/.config/playwright/xiaohongshu/cookies.json",
	})

	// 创建发布网关
	cfg := publishGateway.Config{
		PublishImageURL: "https://creator.xiaohongshu.com/publish/publish?source=official&target=image",
		PublishVideoURL: "https://creator.xiaohongshu.com/publish/publish?source=official&target=video",
		Selectors: map[string]string{
			"upload_input": ".upload-input",
			"title_input":  "div.d-input input",
			"content":      "div.ql-editor",
			"submit":       "button.publishBtn",
			"save_draft":   "button.saveDraftBtn",
		},
	}

	gateway, err := publishGateway.NewGateway(cfg, engine)
	if err != nil {
		log.Fatalf("创建发布网关失败: %v", err)
	}

	// 准备发布内容
	publishContent := publish.ImageContent{
		Title:      *title,
		Content:    *content,
		ImagePaths: paths,
		Tags:       []string{},
		Location:   "",
		MarkerTags: []string{},
	}

	// 执行发布
	ctx := context.Background()
	logrus.Info("========== 开始执行发布 ==========")
	if err := gateway.PublishImage(ctx, publishContent); err != nil {
		log.Fatalf("❌ 发布失败: %v", err)
	}

	logrus.Info("========== 发布成功 ==========")
	time.Sleep(5 * time.Second)
}
