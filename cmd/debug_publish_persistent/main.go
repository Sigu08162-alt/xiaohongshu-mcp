package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/playwright-community/playwright-go"
	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/internal/domain/publish"
	playwrightEngine "github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser/playwright"
	publishGateway "github.com/xpzouying/xiaohongshu-mcp/internal/infra/xhs/publish"
)

var (
	imagePaths  = flag.String("images", "", "图片路径,逗号分隔")
	title       = flag.String("title", "测试标题", "标题")
	content     = flag.String("content", "测试内容", "内容")
	headless    = flag.Bool("headless", false, "是否使用无头模式")
	userDataDir = flag.String("user-data-dir", "/Users/xumingyang/.config/playwright/xiaohongshu", "浏览器数据目录")
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

	logrus.Infof("🔍 开始调试发布流程(持久化浏览器)")
	logrus.Infof("  - 无头模式: %v", *headless)
	logrus.Infof("  - 数据目录: %s", *userDataDir)
	logrus.Infof("  - 标题: %s", *title)
	logrus.Infof("  - 内容: %s", *content)
	logrus.Infof("  - 图片数: %d", len(paths))

	// 确保数据目录存在
	if err := os.MkdirAll(*userDataDir, 0755); err != nil {
		log.Fatalf("创建数据目录失败: %v", err)
	}

	// 启动 Playwright
	pw, err := playwright.Run()
	if err != nil {
		log.Fatalf("启动 Playwright 失败: %v", err)
	}
	defer pw.Stop()

	// 使用持久化上下文启动浏览器
	logrus.Info("启动持久化浏览器(带DevTools)...")
	ctx, err := pw.Chromium.LaunchPersistentContext(*userDataDir, playwright.BrowserTypeLaunchPersistentContextOptions{
		Headless: playwright.Bool(*headless),
		Devtools: playwright.Bool(true), // 启用 DevTools
		Viewport: &playwright.Size{
			Width:  1920,
			Height: 1080,
		},
		UserAgent: playwright.String("Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"),
		Args: []string{
			"--auto-open-devtools-for-tabs", // 自动打开DevTools
		},
	})
	if err != nil {
		log.Fatalf("启动持久化浏览器失败: %v", err)
	}
	defer ctx.Close()

	logrus.Info("✅ 持久化浏览器启动成功")

	// 创建或获取页面
	var page playwright.Page
	pages := ctx.Pages()
	if len(pages) > 0 {
		page = pages[0]
		logrus.Info("使用现有页面")
	} else {
		page, err = ctx.NewPage()
		if err != nil {
			log.Fatalf("创建页面失败: %v", err)
		}
		logrus.Info("创建新页面")
	}

	// 访问发布页面
	publishURL := "https://creator.xiaohongshu.com/publish/publish?source=official&target=image"
	logrus.Infof("📍 打开发布页面: %s", publishURL)
	if _, err := page.Goto(publishURL); err != nil {
		log.Fatalf("访问页面失败: %v", err)
	}

	// 等待页面稳定
	time.Sleep(3 * time.Second)

	// 获取当前URL
	currentURL := page.URL()
	logrus.Infof("📍 当前URL: %s", currentURL)

	// 检查是否需要登录
	uploadInputLocator := page.Locator(".upload-input")
	count, _ := uploadInputLocator.Count()
	if count == 0 {
		logrus.Warn("⚠️  未检测到上传输入框,可能需要登录")
		logrus.Info("=================================================================")
		logrus.Info("🔐 请在打开的浏览器窗口中扫码登录")
		logrus.Info("登录成功后,页面会自动刷新")
		logrus.Info("看到发布页面后,按回车键继续...")
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
			if _, err := page.Goto(publishURL); err != nil {
				log.Fatalf("重新导航失败: %v", err)
			}
			time.Sleep(3 * time.Second)
		}

		// 再次检查上传输入框
		count, _ = uploadInputLocator.Count()
		if count == 0 {
			// 尝试等待更长时间
			logrus.Info("等待上传输入框出现...")
			for i := 0; i < 10; i++ {
				time.Sleep(2 * time.Second)
				count, _ = uploadInputLocator.Count()
				if count > 0 {
					logrus.Info("✅ 找到上传输入框")
					break
				}
				logrus.Infof("尝试 %d/10...", i+1)
			}

			if count == 0 {
				// 显示页面HTML帮助调试
				html, _ := page.Locator("body").InnerHTML()
				if len(html) > 500 {
					html = html[:500]
				}
				logrus.Errorf("页面HTML片段: %s", html)
				log.Fatal("仍未找到上传输入框,请检查页面状态")
			}
		}
	}

	logrus.Info("✅ 已检测到上传输入框,开始发布流程...")

	// 从这里开始,我们不能直接使用 gateway 因为它会重新创建浏览器
	// 所以我们需要手动执行发布步骤

	// 准备发布内容
	publishContent := publish.ImageContent{
		Title:      *title,
		Content:    *content,
		ImagePaths: paths,
		Tags:       []string{},
		Location:   "",
		MarkerTags: []string{},
	}

	logrus.Info("========== 开始手动发布流程 ==========")
	if err := manualPublish(page, publishContent); err != nil {
		log.Fatalf("❌ 发布失败: %v", err)
	}

	logrus.Info("========== 发布成功 ==========")
	fmt.Println("\n按回车键退出...")
	reader := bufio.NewReader(os.Stdin)
	reader.ReadString('\n')
}

func manualPublish(page playwright.Page, content publish.ImageContent) error {
	// 1. 上传图片
	logrus.Infof("📤 上传 %d 张图片...", len(content.ImagePaths))
	uploadInput := page.Locator(".upload-input")
	if err := uploadInput.SetInputFiles(content.ImagePaths); err != nil {
		return fmt.Errorf("上传图片失败: %w", err)
	}
	time.Sleep(3 * time.Second) // 等待上传完成
	logrus.Info("✅ 图片上传完成")

	// 2. 填写标题
	logrus.Infof("✍️  填写标题: %s", content.Title)
	titleInput := page.Locator("div.d-input input")
	if err := titleInput.Fill(content.Title); err != nil {
		return fmt.Errorf("填写标题失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	logrus.Info("✅ 标题填写完成")

	// 3. 填写内容
	logrus.Infof("✍️  填写内容: %s", content.Content)
	contentEditor := page.Locator("div.ql-editor")
	if err := contentEditor.Fill(content.Content); err != nil {
		return fmt.Errorf("填写内容失败: %w", err)
	}
	time.Sleep(500 * time.Millisecond)
	logrus.Info("✅ 内容填写完成")

	// 4. 等待一会儿让页面渲染
	logrus.Info("⏱️  等待2秒...")
	time.Sleep(2 * time.Second)

	// 5. 点击发布按钮
	logrus.Info("📤 点击发布按钮...")
	submitButton := page.Locator("button.publishBtn")
	if err := submitButton.Click(); err != nil {
		return fmt.Errorf("点击发布按钮失败: %w", err)
	}
	logrus.Info("✅ 发布按钮已点击")

	// 6. 等待发布完成 - 通过URL变化验证
	logrus.Info("⏳ 等待发布完成(检查URL变化)...")
	startTime := time.Now()
	maxWait := 60 * time.Second

	for time.Since(startTime) < maxWait {
		currentURL := page.URL()
		logrus.Debugf("当前URL: %s", currentURL)

		// 检��URL是否包含 published=true
		if strings.Contains(currentURL, "published=true") {
			logrus.Info("✅ 发布成功!URL已更新")
			logrus.Infof("发布完成URL: %s", currentURL)
			return nil
		}

		// 检查是否有错误提示
		errorLocator := page.Locator(".error-message, .d-message--error")
		if count, _ := errorLocator.Count(); count > 0 {
			if errText, err := errorLocator.First().InnerText(); err == nil {
				return fmt.Errorf("发布失败: %s", errText)
			}
		}

		time.Sleep(500 * time.Millisecond)
	}

	// 超时了
	finalURL := page.URL()
	logrus.Warnf("⚠️  发布超时: 60秒内未检测到发布成功")
	logrus.Warnf("最终URL: %s", finalURL)

	// 保存截图
	screenshotPath := filepath.Join("/tmp", fmt.Sprintf("debug_timeout_%d.png", time.Now().Unix()))
	if _, err := page.Screenshot(playwright.PageScreenshotOptions{
		Path: playwright.String(screenshotPath),
	}); err == nil {
		logrus.Infof("已保存截图: %s", screenshotPath)
	}

	return fmt.Errorf("发布超时: 60秒内未检测到发布成功,最终URL: %s", finalURL)
}
