package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
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

	// 创建页面
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

	// 获取当前URL
	currentURL := page.URL()
	logrus.Infof("📍 当前URL: %s", currentURL)

	// 检查是否需要登录
	hasUploadInput, _ := page.Has(".upload-input")
	if !hasUploadInput {
		logrus.Warn("⚠️  未检测到上传输入框,可能需要登录")
		logrus.Info("=================================================================")
		logrus.Info("🔐 请在打开的浏览器窗口中扫码登录")
		logrus.Info("登录成功后,请在浏览器中手动导航到发布页面")
		logrus.Info("然后按回车键继续...")
		logrus.Info("=================================================================")

		// 等待用户按回车
		reader := bufio.NewReader(os.Stdin)
		reader.ReadString('\n')

		logrus.Info("✅ 继续检查...")

		// 等待页面加载
		time.Sleep(2 * time.Second)

		// 重新获取URL
		currentURL = page.URL()
		logrus.Infof("📍 当前URL: %s", currentURL)

		// 检查是否在发布页面
		if !strings.Contains(currentURL, "creator.xiaohongshu.com/publish") {
			logrus.Warnf("当前不在发布页面,尝试重新导航...")
			if err := page.Goto(publishURL); err != nil {
				log.Fatalf("重新导航失败: %v", err)
			}
			time.Sleep(3 * time.Second)
		}

		// 再次检查上传输入框
		hasUploadInput, _ = page.Has(".upload-input")
		if !hasUploadInput {
			// 尝试等待更长时间
			logrus.Info("等待上传输入框出现...")
			for i := 0; i < 10; i++ {
				time.Sleep(2 * time.Second)
				hasUploadInput, _ = page.Has(".upload-input")
				if hasUploadInput {
					logrus.Info("✅ 找到上传输入框")
					break
				}
				logrus.Infof("尝试 %d/10...", i+1)
			}

			if !hasUploadInput {
				// 显示页面HTML帮助调试
				html, _ := page.HTML("body")
				if len(html) > 500 {
					html = html[:500]
				}
				logrus.Errorf("页面HTML片段: %s", html)
				log.Fatal("仍未找到上传输入框,请检查页面状态")
			}
		}
	}

	logrus.Info("✅ 已检测到上传输入框,开始发布流程...")

	// 创建发布网关
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
	fmt.Println("\n按回车键退出...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}
