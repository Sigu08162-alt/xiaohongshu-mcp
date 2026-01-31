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
	noInteractive := flag.Bool("no-interactive", false, "非交互模式,跳过所有等待输入")
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
	if !*noInteractive {
		if finalCookiePath == "" {
			logrus.Info("\n🔐 请在浏览器中登录小红书（如需要）")
			logrus.Info("\n⏸️  按 Enter 继续...")
			fmt.Scanln()
		} else {
			logrus.Info("\n✅ 已加载Cookie文件，无需手动登录")
			logrus.Info("\n⏸️  按 Enter 开始发现页面...")
			fmt.Scanln()
		}
	}

	// 访问首页
	logrus.Infof("\n🌐 访问首页: %s", *homeURL)
	if err := page.Goto(*homeURL); err != nil {
		logrus.Fatalf("访问首页失败: %v", err)
	}

	// 等待页面加载
	logrus.Infof("⏳ 等待 %d 秒让页面加载完成...", *waitTime)
	time.Sleep(time.Duration(*waitTime) * time.Second)

	// 保存页面 HTML 和调试信息
	htmlPath := "debug_homepage.html"
	debugInfo, err := page.Eval(`() => {
		return {
			html: document.documentElement.outerHTML,
			url: window.location.href,
			title: document.title,
			linkCount: document.querySelectorAll('a').length,
			allElements: document.querySelectorAll('*').length,
			bodyText: document.body ? document.body.textContent.substring(0, 200) : 'no body'
		};
	}`)

	if err != nil {
		logrus.Warnf("获取页面信息失败: %v", err)
	} else {
		jsonData, _ := json.Marshal(debugInfo)
		var info struct {
			HTML        string `json:"html"`
			URL         string `json:"url"`
			Title       string `json:"title"`
			LinkCount   int    `json:"linkCount"`
			AllElements int    `json:"allElements"`
			BodyText    string `json:"bodyText"`
		}
		json.Unmarshal(jsonData, &info)

		logrus.Infof("📄 URL: %s", info.URL)
		logrus.Infof("📄 Title: %s", info.Title)
		logrus.Infof("📄 总元素数: %d", info.AllElements)
		logrus.Infof("📄 <a>标签数: %d", info.LinkCount)
		logrus.Infof("📄 页面文本: %s...", info.BodyText)
		logrus.Infof("📄 HTML长度: %d 字节", len(info.HTML))

		if len(info.HTML) > 0 {
			os.WriteFile(htmlPath, []byte(info.HTML), 0644)
			logrus.Infof("📄 已保存页面HTML到: %s", htmlPath)
		} else {
			logrus.Warn("⚠️  HTML为空,可能页面未正确加载")
		}
	}

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
	if !*noInteractive {
		logrus.Info("\n⏸️  按 Enter 关闭浏览器...")
		fmt.Scanln()
	}
}

// discoverLinks 发现页面所有链接
func discoverLinks(page browser.Page) []LinkInfo {
	jsCode := `() => {
		const links = [];
		const visited = new Set();

		// 调试:输出页面信息
		console.log('=== 开始链接发现 ===');
		console.log('URL:', window.location.href);
		console.log('所有 a 标签数量:', document.querySelectorAll('a').length);
		console.log('菜单项数量:', document.querySelectorAll('.menu-item').length);
		console.log('所有可点击元素:', document.querySelectorAll('[onclick], [role="link"], [role="button"]').length);

		// 策略1: 尝试查找菜单项(小红书使用div+路由,不是a标签)
		document.querySelectorAll('.menu-item, [class*="menu"]').forEach((item, idx) => {
			// 尝试从元素或父元素获取路径信息
			const text = item.textContent?.trim() || '';

			// 检查data-*属性
			const dataTo = item.getAttribute('data-to') ||
			              item.getAttribute('data-path') ||
			              item.getAttribute('data-href');

			// 检查onClick事件
			const onClick = item.getAttribute('onclick') || '';

			// 检查Vue/React路由属性
			const to = item.getAttribute('to') || '';

			if (idx < 20) {
				console.log('菜单项 ' + idx + ':', {
					text: text.substring(0, 30),
					dataTo: dataTo,
					onClick: onClick.substring(0, 50),
					to: to,
					className: item.className
				});
			}

			// 如果有文本,尝试提取
			if (text && text.length > 0 && text.length < 100) {
				links.push({
					text: text,
					url: dataTo || to || '', // 可能为空,后续手动补充
					classes: item.className ? item.className.split(' ').filter(c => c) : []
				});
			}
		});

		// 策略2: 查找所有带href的a标签(即使数量为0也尝试)
		document.querySelectorAll('a[href]').forEach((link, idx) => {
			const href = link.href;
			const text = link.textContent?.trim() || '';

			if (idx < 10) {
				console.log('链接 ' + idx + ':', {
					text: text.substring(0, 30),
					href: href.substring(0, 80),
					includes_creator: href.includes('creator.xiaohongshu.com')
				});
			}

			if (href.includes('creator.xiaohongshu.com') && !visited.has(href)) {
				visited.add(href);
				const classes = link.className ? link.className.split(' ').filter(c => c) : [];
				links.push({
					text: text || link.getAttribute('aria-label') || link.title || '',
					url: href,
					classes: classes,
					parent_text: link.parentElement?.textContent?.trim().substring(0, 50) || ''
				});
				console.log('✓ 添加链接:', text.substring(0, 30), '→', href.substring(0, 60));
			}
		});

		// 策略3: 动态从页面获取URL，而不是硬编码
		const pageText = document.body.textContent || '';
		const keywords = ['发布笔记', '笔记管理', '数据看板', '内容分析', '粉丝数据'];

		// 首先尝试从菜单项的实际属性获取URL
		const realUrlMap = {};
		document.querySelectorAll('.menu-item, [class*="menu"], nav a, aside a').forEach(item => {
			const text = item.textContent?.trim() || '';
			const href = item.getAttribute('href') ||
			             item.getAttribute('data-href') ||
			             item.getAttribute('data-to') ||
			             item.querySelector('a')?.href ||
			             '';

			if (text && href && text.length > 0 && text.length < 20) {
				// 如果是相对路径，补全为绝对路径
				let fullUrl = href;
				if (href.startsWith('/')) {
					fullUrl = window.location.origin + href;
				}
				// 标准化URL（移除hash）
				fullUrl = fullUrl.split('#')[0];

				realUrlMap[text] = fullUrl;
				console.log('✓ 从DOM获取URL:', text, '→', fullUrl);
			}
		});

		// fallback URL映射（仅用于没有从DOM获取到的情况）
		const fallbackUrlMap = {
			'发布笔记': '/publish/publish',
			'笔记管理': '/creator/content',
			'数据看板': '/creator/data-board',
			'内容分析': '/creator/content/data',
			'粉丝数据': '/creator/fans'
		};

		keywords.forEach(keyword => {
			if (pageText.includes(keyword)) {
				// 优先使用从DOM获取的真实URL
				let url = realUrlMap[keyword];

				// 如果没有获取到，使用fallback推测
				if (!url && fallbackUrlMap[keyword]) {
					url = window.location.origin + fallbackUrlMap[keyword];
					console.log('⚠ 使用fallback URL:', keyword, '→', url);
				}

				if (url && !visited.has(url)) {
					// 自动补全source=official参数
					if (!url.includes('source=')) {
						const separator = url.includes('?') ? '&' : '?';
						url = url + separator + 'source=official';
						console.log('✓ 补全参数:', url);
					}

					visited.add(url);
					links.push({
						text: keyword,
						url: url,
						classes: realUrlMap[keyword] ? ['from-dom'] : ['inferred'],
						category: realUrlMap[keyword] ? 'dynamic' : 'inferred'
					});
					console.log('✓ 添加链接:', keyword, '→', url);
				}
			}
		});

		console.log('总共发现:', links.length, '个链接');
		console.log('=== 链接发现完成 ===');

		return links;
	}`

	info, err := page.Eval(jsCode)
	if err != nil {
		logrus.Errorf("发现链接失败: %v", err)
		return []LinkInfo{}
	}

	// 解析结果
	jsonData, _ := json.Marshal(info)
	var links []LinkInfo
	json.Unmarshal(jsonData, &links)

	logrus.Infof("📊 发现 %d 个链接", len(links))

	return deduplicateLinks(links)
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
