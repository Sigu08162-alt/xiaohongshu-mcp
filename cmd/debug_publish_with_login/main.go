package main

import (
	"bufio"
	"context"
	"flag"
	"log"
	"os"
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
	waitLogin  = flag.Bool("wait-login", true, "是否等待用户手动登录")
)

func main() {
	flag.Parse()

	// 设置日志级别
	logrus.SetLevel(logrus.DebugLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: "2006-01-28 15:04:05",
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

	logrus.Infof("🔍 开始调试发布流程")
	logrus.Infof("  - 无头模式: %v", *headless)
	logrus.Infof("  - 等待登录: %v", *waitLogin)
	logrus.Infof("  - 标题: %s", *title)
	logrus.Infof("  - 内容: %s", *content)
	logrus.Infof("  - 图片数: %d", len(paths))

	// 创建浏览器引擎
	engine := playwright.New(playwright.Config{
		Headless:   *headless,
		CookiePath: "", // 不加载 cookie,让用户重新登录
	})

	// 启动浏览器
	if err := engine.Start(); err != nil {
		log.Fatalf("启动浏览器失败: %v", err)
	}
	defer engine.Close()

	// ��建页面
	page, err := engine.NewPage()
	if err != nil {
		log.Fatalf("创建页面失败: %v", err)
	}
	defer page.Close()

	// 访问发布页面
	publishURL := "https://creator.xiaohongshu.com/publish/publish?source=official&target=image"
	logrus.Infof("📍 打开发布页面: %s", publishURL)
	if err := page.Goto(publishURL); err != nil {
		log.Fatalf("访问页面失败: %v", err)
	}

	// 等待页面稳定
	time.Sleep(3 * time.Second)

	// 检查是否需要登录
	hasUploadInput, _ := page.Has(".upload-input")
	if !hasUploadInput {
		logrus.Warn("⚠️  未检测到上传输入框,可能需要登录")

		if *waitLogin {
			logrus.Info("=================================================================")
			logrus.Info("🔐 请在打开的浏览器窗口中扫码登录")
			logrus.Info("登录完成后,按回车键继续...")
			logrus.Info("=================================================================")

			// 等待用户按回车
			reader := bufio.NewReader(os.Stdin)
			reader.ReadString('\n')

			logrus.Info("✅ 继续执行发布流程...")

			// 重新检查上传输入框
			time.Sleep(2 * time.Second)
			hasUploadInput, _ = page.Has(".upload-input")
			if !hasUploadInput {
				log.Fatal("登录后仍未找到上传输入框,请检查是否已成功登录")
			}
		} else {
			log.Fatal("需要登录,但未启用等待登录模式")
		}
	}

	// 创建发布网关(复用已登录的页面)
	// 注意:这里我们需要直接调用发布逻辑,而不是重新创建新页面
	logrus.Info("========== 开始执行发布 ==========")

	// 准备发布内容
	publishContent := publish.ImageContent{
		Title:      *title,
		Content:    *content,
		ImagePaths: paths,
		Tags:       []string{},
		Location:   "",
		MarkerTags: []string{},
	}

	// 直接使用 gateway 的发布逻辑
	cfg := publishGateway.Config{
		PublishImageURL: publishURL,
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

	// 执行发布
	ctx := context.Background()
	if err := gateway.PublishImage(ctx, publishContent); err != nil {
		log.Fatalf("❌ 发布失败: %v", err)
	}

	logrus.Info("========== 发布成功 ==========")
	logrus.Info("按回车键退出...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}
