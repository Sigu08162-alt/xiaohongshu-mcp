package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser/playwright"
	"gopkg.in/yaml.v3"
)

// LinkInfo 链接信息
type LinkInfo struct {
	Text        string   `yaml:"text" json:"text"`
	URL         string   `yaml:"url" json:"url"`
	Description string   `yaml:"description,omitempty" json:"description,omitempty"`
	Category    string   `yaml:"category,omitempty" json:"category,omitempty"`
	Classes     []string `yaml:"classes,omitempty" json:"classes,omitempty"`
}

// PageLinks 页面链接集合
type PageLinks struct {
	Version   string              `yaml:"version" json:"version"`
	Generated string              `yaml:"generated" json:"generated"`
	HomePage  string              `yaml:"home_page" json:"home_page"`
	Links     map[string]LinkInfo `yaml:"links" json:"links"`
}

func main() {
	// 命令行参数
	outputYAML := flag.String("output", "discovered_pages.yaml", "输出YAML文件路径")
	outputJSON := flag.String("json", "", "可选：同时输出JSON文件")
	headless := flag.Bool("headless", false, "无头模式（默认有头）")
	homeURL := flag.String("home", "https://creator.xiaohongshu.com/new/home?source=official", "创作者中心首页URL")
	waitTime := flag.Int("wait", 5, "页面加载等待秒数")
	cookiePath := flag.String("cookies", "", "Cookie文件路径（默认自动查找）")
	flag.Parse()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	logrus.Info("=== 小红书页面链接自动发现工具 ===")
	logrus.Infof("首页URL: %s", *homeURL)
	logrus.Infof("输出YAML: %s", *outputYAML)
	logrus.Infof("有头模式: %v", !*headless)
	logrus.Infof("等待时间: %d秒", *waitTime)

	// 查找 cookie 文件
	var finalCookiePath string
	if *cookiePath != "" {
		finalCookiePath = *cookiePath
	} else {
		finalCookiePath = findCookieFile()
	}

	if finalCookiePath != "" {
		logrus.Infof("🍪 Cookie文件: %s", finalCookiePath)
	} else {
		logrus.Warn("⚠️  未找到Cookie文件，需要手动登录")
	}

	// 创建浏览器引擎
	logrus.Info("\n📦 启动浏览器...")
	engine := playwright.New(playwright.Config{
		Headless:          *headless,
		ActionTimeout:     10 * time.Second,
		NavigationTimeout: 60 * time.Second,
		CookiePath:        finalCookiePath,
	})
	defer engine.Close()

	if err := engine.Start(); err != nil {
		logrus.Fatalf("启动浏览器失败: %v", err)
	}

	page, err := engine.NewPage()
	if err != nil {
		logrus.Fatalf("创建页面失败: %v", err)
	}

	// 登录提示
	if finalCookiePath == "" {
		logrus.Info("\n🔐 请在浏览器中登录小红书（如需要）")
		logrus.Info("\n⏸️  按 Enter 继续...")
		fmt.Scanln()
	} else {
		logrus.Info("\n✅ 已加载Cookie文件，无需手动登录")
		logrus.Info("\n⏸️  按 Enter 开始发现页面...")
		fmt.Scanln()
	}

	// 访问首页
	logrus.Infof("\n🌐 访问首页: %s", *homeURL)
	if err := page.Goto(*homeURL); err != nil {
		logrus.Fatalf("访问首页失败: %v", err)
	}

	// 等待页面加载
	logrus.Infof("⏳ 等待 %d 秒让页面加载完成...", *waitTime)
	time.Sleep(time.Duration(*waitTime) * time.Second)

	// 发现所有链接
	logrus.Info("\n🔍 开始发现页面链接...")
	links := discoverLinks(page)

	// 分类整理
	categorizedLinks := categorizeLinks(links)

	// 保存结果
	pageLinks := PageLinks{
		Version:   "1.0.0",
		Generated: time.Now().Format(time.RFC3339),
		HomePage:  *homeURL,
		Links:     categorizedLinks,
	}

	// 保存到 YAML
	logrus.Infof("\n💾 保存到 YAML: %s", *outputYAML)
	if err := saveToYAML(&pageLinks, *outputYAML); err != nil {
		logrus.Fatalf("保存YAML失败: %v", err)
	}

	// 可选：保存到 JSON
	if *outputJSON != "" {
		logrus.Infof("💾 保存到 JSON: %s", *outputJSON)
		if err := saveToJSON(&pageLinks, *outputJSON); err != nil {
			logrus.Errorf("保存JSON失败: %v", err)
		}
	}

	// 显示结果
	displayResults(&pageLinks)

	logrus.Info("\n✅ 发现完成！")
	logrus.Info("\n⏸️  按 Enter 关闭浏览器...")
	fmt.Scanln()
}

// discoverLinks 发现页面所有链接
func discoverLinks(page browser.Page) []LinkInfo {
	jsCode := `() => {
		const links = [];
		const visited = new Set();

		// 查找所有链接
		document.querySelectorAll('a[href]').forEach(link => {
			const href = link.href;
			const text = link.textContent?.trim() || '';

			// 过滤：只保留创作者中心相关链接
			if (href.includes('creator.xiaohongshu.com') && !visited.has(href)) {
				visited.add(href);

				const classes = link.className ? link.className.split(' ').filter(c => c) : [];

				links.push({
					text: text,
					url: href,
					classes: classes,
					parent_text: link.parentElement?.textContent?.trim().substring(0, 50) || ''
				});
			}
		});

		// 查找导航菜单
		const navItems = [];
		document.querySelectorAll('nav a, .nav a, [class*="nav"] a, [class*="menu"] a').forEach(link => {
			if (link.href && link.href.includes('creator.xiaohongshu.com')) {
				navItems.push({
					text: link.textContent?.trim() || '',
					url: link.href,
					category: 'navigation',
					classes: link.className.split(' ').filter(c => c)
				});
			}
		});

		return {
			all_links: links,
			nav_links: navItems
		};
	}`

	info, err := page.Eval(jsCode)
	if err != nil {
		logrus.Errorf("发现链接失败: %v", err)
		return []LinkInfo{}
	}

	// 解析结果
	jsonData, _ := json.Marshal(info)
	var result struct {
		AllLinks []LinkInfo `json:"all_links"`
		NavLinks []LinkInfo `json:"nav_links"`
	}
	json.Unmarshal(jsonData, &result)

	logrus.Infof("📊 发现 %d 个创作者中心链接", len(result.AllLinks))
	logrus.Infof("📊 发现 %d 个导航链接", len(result.NavLinks))

	// 合并并去重
	allLinks := append(result.AllLinks, result.NavLinks...)
	return deduplicateLinks(allLinks)
}

// deduplicateLinks 去重链接
func deduplicateLinks(links []LinkInfo) []LinkInfo {
	seen := make(map[string]bool)
	var unique []LinkInfo

	for _, link := range links {
		if !seen[link.URL] {
			seen[link.URL] = true
			unique = append(unique, link)
		}
	}

	return unique
}

// categorizeLinks 分类整理链接
func categorizeLinks(links []LinkInfo) map[string]LinkInfo {
	result := make(map[string]LinkInfo)

	for _, link := range links {
		// 根据 URL 和文本分类
		key := generateKey(link)
		category := detectCategory(link)

		link.Category = category
		link.Description = generateDescription(link)

		result[key] = link
	}

	return result
}

// generateKey 生成链接的唯一键
func generateKey(link LinkInfo) string {
	// 根据 URL 路径生成键
	url := link.URL

	// 提取路径部分
	if strings.Contains(url, "creator.xiaohongshu.com") {
		parts := strings.Split(url, "creator.xiaohongshu.com")
		if len(parts) > 1 {
			path := strings.Split(parts[1], "?")[0]
			path = strings.TrimPrefix(path, "/")
			path = strings.ReplaceAll(path, "/", "_")

			if path == "" {
				return "home"
			}
			return path
		}
	}

	// 备用：使用文本生成键
	if link.Text != "" {
		key := strings.ToLower(link.Text)
		key = strings.ReplaceAll(key, " ", "_")
		key = strings.ReplaceAll(key, "-", "_")
		return key
	}

	return "unknown"
}

// detectCategory 检测链接类别
func detectCategory(link LinkInfo) string {
	url := strings.ToLower(link.URL)
	text := strings.ToLower(link.Text)

	categories := map[string][]string{
		"publish": {"publish", "发布"},
		"content": {"content", "内容"},
		"data":    {"statistics", "data", "数据", "分析"},
		"fans":    {"fans", "粉丝"},
		"comment": {"comment", "评论"},
		"income":  {"income", "revenue", "收益"},
		"setting": {"setting", "设置"},
		"help":    {"help", "帮助"},
	}

	for category, keywords := range categories {
		for _, keyword := range keywords {
			if strings.Contains(url, keyword) || strings.Contains(text, keyword) {
				return category
			}
		}
	}

	return "other"
}

// generateDescription 生成链接描述
func generateDescription(link LinkInfo) string {
	descriptions := map[string]string{
		"publish":    "内容发布相关页面",
		"content":    "内容管理相关页面",
		"data":       "数据分析相关页面",
		"fans":       "粉丝管理相关页面",
		"comment":    "评论管理相关页面",
		"income":     "收益相关页面",
		"setting":    "设置相关页面",
		"help":       "帮助相关页面",
		"other":      "其他功能页面",
		"navigation": "导航菜单链接",
	}

	if desc, ok := descriptions[link.Category]; ok {
		return desc
	}

	return link.Text
}

// saveToYAML 保存到 YAML 文件
func saveToYAML(data *PageLinks, path string) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	header := `# 小红书创作者中心页面链接
# 自动发现并生成，请勿手动编辑
# 生成时间: ` + data.Generated + `
# 版本: ` + data.Version + `

`
	content := header + string(yamlData)
	return os.WriteFile(path, []byte(content), 0644)
}

// saveToJSON 保存到 JSON 文件
func saveToJSON(data *PageLinks, path string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// displayResults 显示结果
func displayResults(pageLinks *PageLinks) {
	separator := strings.Repeat("=", 60)
	logrus.Info("\n" + separator)
	logrus.Info("📊 发现结果汇总")
	logrus.Info(separator)

	logrus.Infof("📅 生成时间: %s", pageLinks.Generated)
	logrus.Infof("🏠 首页: %s", pageLinks.HomePage)
	logrus.Infof("🔗 发现链接: %d 个", len(pageLinks.Links))

	// 按类别统计
	categoryCount := make(map[string]int)
	for _, link := range pageLinks.Links {
		categoryCount[link.Category]++
	}

	logrus.Info("\n📋 按类别统计:")
	for category, count := range categoryCount {
		logrus.Infof("  - %s: %d 个", category, count)
	}

	// 显示关键链接
	logrus.Info("\n🔑 关键页面:")
	for key, link := range pageLinks.Links {
		if link.Category == "publish" || link.Category == "content" {
			logrus.Infof("  - %s: %s", key, link.URL)
			logrus.Infof("    文本: %s", link.Text)
		}
	}

	logrus.Info("\n" + separator)
}

// findCookieFile 自动查找 cookie 文件
func findCookieFile() string {
	cookiePaths := []string{
		"cookies.json",
		os.Getenv("HOME") + "/.xiaohongshu/cookies.json",
		"./xiaohongshu_cookies.json",
	}

	for _, path := range cookiePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
