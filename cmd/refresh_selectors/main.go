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

// ElementInfo 元素详细信息
type ElementInfo struct {
	Text        string   `yaml:"text" json:"text"`
	Selector    string   `yaml:"selector" json:"selector"`
	Classes     []string `yaml:"classes,omitempty" json:"classes,omitempty"`
	ID          string   `yaml:"id,omitempty" json:"id,omitempty"`
	TagName     string   `yaml:"tag_name,omitempty" json:"tag_name,omitempty"`
	Placeholder string   `yaml:"placeholder,omitempty" json:"placeholder,omitempty"`
	Visible     bool     `yaml:"visible" json:"visible"`
	Disabled    bool     `yaml:"disabled,omitempty" json:"disabled,omitempty"`
	Type        string   `yaml:"type,omitempty" json:"type,omitempty"`
}

// PageSnapshot 单个页面的快照
type PageSnapshot struct {
	PageName    string                   `yaml:"page_name" json:"page_name"`
	URL         string                   `yaml:"url" json:"url"`
	Timestamp   string                   `yaml:"timestamp" json:"timestamp"`
	Buttons     []ElementInfo            `yaml:"buttons" json:"buttons"`
	Inputs      []ElementInfo            `yaml:"inputs" json:"inputs"`
	Containers  []ElementInfo            `yaml:"containers" json:"containers"`
	Links       []ElementInfo            `yaml:"links" json:"links"`
	AllElements []map[string]interface{} `yaml:"all_elements,omitempty" json:"all_elements,omitempty"`
}

// AllPagesSnapshot 所有页面的快照
type AllPagesSnapshot struct {
	Version   string                  `yaml:"version" json:"version"`
	Generated string                  `yaml:"generated" json:"generated"`
	Pages     map[string]PageSnapshot `yaml:"pages" json:"pages"`
}

// PageDefinition 待采集的页面定义
type PageDefinition struct {
	Name string
	URL  string
	Desc string
}

// 默认页面列表（硬编码）
var defaultPages = []PageDefinition{
	{
		Name: "publish_image",
		URL:  "https://creator.xiaohongshu.com/publish/publish?source=official&target=image",
		Desc: "图文发布页面",
	},
	{
		Name: "publish_video",
		URL:  "https://creator.xiaohongshu.com/publish/publish?source=official&target=video",
		Desc: "视频发布页面",
	},
	{
		Name: "creator_home",
		URL:  "https://creator.xiaohongshu.com/new/home?source=official",
		Desc: "创作者中心首页",
	},
}

func main() {
	// 命令行参数
	outputYAML := flag.String("output", "selectors_all_pages.yaml", "输出YAML文件路径")
	outputJSON := flag.String("json", "", "可选：同时输出JSON文件")
	headless := flag.Bool("headless", false, "无头模式（默认有头）")
	singlePage := flag.String("page", "", "仅采集单个页面 (publish_image|publish_video|creator_home)")
	waitTime := flag.Int("wait", 3, "每个页面加载后等待秒数")
	cookiePath := flag.String("cookies", "", "Cookie文件路径（默认自动查找）")
	pagesFile := flag.String("pages", "", "从discovered_pages.yaml加载页面列表")
	noInteractive := flag.Bool("no-interactive", false, "非交互模式,跳过所有等待输入")
	flag.Parse()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	logrus.Info("=== 小红书页面组件全量采集工具 ===")
	logrus.Infof("输出YAML: %s", *outputYAML)
	if *outputJSON != "" {
		logrus.Infof("输出JSON: %s", *outputJSON)
	}
	logrus.Infof("有头模式: %v", !*headless)
	logrus.Infof("等待时间: %d秒/页面", *waitTime)

	// 查找 cookie 文件
	var finalCookiePath string
	if *cookiePath != "" {
		finalCookiePath = *cookiePath
	} else {
		// 自动查找 cookie 文件
		finalCookiePath = findCookieFile()
	}

	if finalCookiePath != "" {
		logrus.Infof("🍪 Cookie文件: %s", finalCookiePath)
	} else {
		logrus.Warn("⚠️  未找到Cookie文件，需要手动登录")
	}

	// 加载页面列表
	var targetPages []PageDefinition
	if *pagesFile != "" {
		logrus.Infof("📄 从文件加载页面列表: %s", *pagesFile)
		var err error
		targetPages, err = loadPagesFromFile(*pagesFile)
		if err != nil {
			logrus.Warnf("加载页面列表失败: %v，使用默认列表", err)
			targetPages = defaultPages
		} else {
			logrus.Infof("✅ 成功加载 %d 个页面", len(targetPages))
		}
	} else {
		targetPages = defaultPages
		logrus.Info("📋 使用默认页面列表")
	}

	// 过滤要采集的页面
	var pagesToCapture []PageDefinition
	if *singlePage != "" {
		found := false
		for _, p := range targetPages {
			if p.Name == *singlePage {
				pagesToCapture = []PageDefinition{p}
				found = true
				break
			}
		}
		if !found {
			logrus.Fatalf("未找到页面: %s，可用页面: publish_image, publish_video, creator_home, content_list", *singlePage)
		}
	} else {
		pagesToCapture = targetPages
	}

	logrus.Infof("\n📋 将采集 %d 个页面:", len(pagesToCapture))
	for _, p := range pagesToCapture {
		logrus.Infof("  - %s: %s", p.Name, p.Desc)
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

	// 首次登录提示
	if !*noInteractive {
		if finalCookiePath == "" {
			logrus.Info("\n🔐 请在浏览器中登录小红书（如需要）")
			logrus.Info("   登录后将保持会话，后续页面无需重复登录")
			logrus.Info("\n⏸️  按 Enter 继续...")
			fmt.Scanln()
		} else {
			logrus.Info("\n✅ 已加载Cookie文件，无需手动登录")
			logrus.Info("   如果Cookie已过期，请运行: ./login.sh")
			logrus.Info("\n⏸️  按 Enter 开始采集...")
			fmt.Scanln()
		}
	}

	// 采集所有页面
	allSnapshots := AllPagesSnapshot{
		Version:   "1.0.0",
		Generated: time.Now().Format(time.RFC3339),
		Pages:     make(map[string]PageSnapshot),
	}

	for i, pageDef := range pagesToCapture {
		logrus.Infof("\n[%d/%d] 📄 采集页面: %s (%s)", i+1, len(pagesToCapture), pageDef.Desc, pageDef.Name)
		logrus.Infof("🌐 URL: %s", pageDef.URL)

		// 导航到页面
		if err := page.Goto(pageDef.URL); err != nil {
			logrus.Errorf("❌ 导航失败: %v，跳过此页面", err)
			continue
		}

		// 等待页面稳定
		logrus.Infof("⏳ 等待 %d 秒让页面加载完成...", *waitTime)
		time.Sleep(time.Duration(*waitTime) * time.Second)

		// 特殊处理：如果是发布页面，需要先上传图片才能看到输入框
		if strings.Contains(pageDef.URL, "/publish/publish") {
			logrus.Info("📸 检测到发布页面，尝试触发图片上传以加载输入框...")

			// 创建一个1x1像素的测试图片
			testImagePath := createTestImage()

			// 尝试上传图片
			uploadErr := page.SetFiles("input[type=\"file\"]", []string{testImagePath})
			if uploadErr != nil {
				logrus.Warnf("⚠️  图片上传失败: %v，继续采集当前状态", uploadErr)
			} else {
				logrus.Info("✅ 已上传测试图片，等待输入框加载...")
				time.Sleep(8 * time.Second) // 增加等待时间到8秒

				// 调试：检查是否有输入框出现
				checkJS := `() => {
					return {
						inputs: document.querySelectorAll('input').length,
						textareas: document.querySelectorAll('textarea').length,
						contenteditable: document.querySelectorAll('[contenteditable="true"]').length,
						roleTextbox: document.querySelectorAll('[role="textbox"]').length
					};
				}`
				if debugInfo, err := page.Eval(checkJS); err == nil {
					logrus.Infof("🔍 调试: %+v", debugInfo)
				}
			}

			// 清理测试图片
			os.Remove(testImagePath)
		}

		// 采集组件信息
		snapshot := capturePageComponents(page, pageDef.Name)
		allSnapshots.Pages[pageDef.Name] = *snapshot

		logrus.Infof("✅ 采集完成: 按钮=%d, 输入框=%d, 容器=%d, 链接=%d",
			len(snapshot.Buttons), len(snapshot.Inputs), len(snapshot.Containers), len(snapshot.Links))
	}

	// 保存到 YAML 文件
	logrus.Infof("\n💾 保存到 YAML: %s", *outputYAML)
	if err := saveToYAML(&allSnapshots, *outputYAML); err != nil {
		logrus.Fatalf("保存YAML失败: %v", err)
	}

	// 可选：保存到 JSON 文件
	if *outputJSON != "" {
		logrus.Infof("💾 保存到 JSON: %s", *outputJSON)
		if err := saveToJSON(&allSnapshots, *outputJSON); err != nil {
			logrus.Errorf("保存JSON失败: %v", err)
		}
	}

	// 显示汇总
	displaySummary(&allSnapshots)

	logrus.Info("\n✅ 采集完成！")
	if !*noInteractive {
		logrus.Info("\n⏸️  按 Enter 关闭浏览器...")
		fmt.Scanln()
	}
}

// capturePageComponents 采集单个页面的组件信息
func capturePageComponents(page browser.Page, pageName string) *PageSnapshot {
	jsCode := `() => {
		const result = {
			url: window.location.href,
			timestamp: new Date().toISOString(),
			buttons: [],
			inputs: [],
			containers: [],
			links: [],
			all_elements: []
		};

		// 专门查找输入相关的元素
		try {
			console.log('=== 搜索输入相关元素 ===');

			// 策略1: 查找所有可能的输入元素
			const inputCandidates = [];

			// 标准输入
			document.querySelectorAll('input, textarea').forEach(elem => {
				inputCandidates.push({
					type: 'standard',
					tag: elem.tagName.toLowerCase(),
					classes: Array.from(elem.classList).slice(0, 5),
					attrs: {
						type: elem.type,
						placeholder: elem.placeholder,
						name: elem.name
					}
				});
			});

			// contenteditable元素
			document.querySelectorAll('[contenteditable="true"]').forEach(elem => {
				inputCandidates.push({
					type: 'contenteditable',
					tag: elem.tagName.toLowerCase(),
					classes: Array.from(elem.classList).slice(0, 5),
					text: elem.textContent.substring(0, 50)
				});
			});

			// role=textbox
			document.querySelectorAll('[role="textbox"]').forEach(elem => {
				inputCandidates.push({
					type: 'role-textbox',
					tag: elem.tagName.toLowerCase(),
					classes: Array.from(elem.classList).slice(0, 5),
					text: elem.textContent.substring(0, 50)
				});
			});

			// 包含input/editor/title关键词的class
			document.querySelectorAll('[class*="input"], [class*="editor"], [class*="title"]').forEach(elem => {
				if (elem.tagName.toLowerCase() !== 'input' && elem.tagName.toLowerCase() !== 'textarea') {
					inputCandidates.push({
						type: 'class-keyword',
						tag: elem.tagName.toLowerCase(),
						classes: Array.from(elem.classList).slice(0, 5),
						text: elem.textContent.substring(0, 50),
						visible: elem.offsetParent !== null
					});
				}
			});

			console.log('找到候选输入元素:', inputCandidates.length);
			result.all_elements = inputCandidates;

		} catch (e) {
			console.error('Error:', e);
		}

		// 1. 采集所有按钮
		document.querySelectorAll('button').forEach((btn) => {
			const text = btn.textContent?.trim() || '';
			const classes = btn.className ? btn.className.split(' ').filter(c => c) : [];

			result.buttons.push({
				text: text,
				selector: classes[0] ? 'button.' + classes[0] : 'button',
				classes: classes,
				id: btn.id || '',
				tag_name: 'button',
				visible: btn.offsetParent !== null,
				disabled: btn.disabled,
				type: btn.type || ''
			});
		});

		// 2. 采集所有输入框 - 标准元素
		document.querySelectorAll('input, textarea, [contenteditable="true"]').forEach((input) => {
			const classes = input.className ? input.className.split(' ').filter(c => c) : [];
			const tagName = input.tagName.toLowerCase();

			result.inputs.push({
				text: input.value || input.textContent?.trim() || '',
				selector: input.id ? '#' + input.id : (classes[0] ? '.' + classes[0] : tagName),
				classes: classes,
				id: input.id || '',
				tag_name: tagName,
				placeholder: input.placeholder || input.getAttribute('data-placeholder') || '',
				visible: input.offsetParent !== null,
				type: input.type || ''
			});
		});

		// 3. 采集所有容器
		document.querySelectorAll('div[class], section[class], main[class]').forEach((container) => {
			const classes = container.className ? container.className.split(' ').filter(c => c) : [];
			if (classes.length > 0) {
				result.containers.push({
					text: '',
					selector: '.' + classes[0],
					classes: classes,
					id: container.id || '',
					tag_name: container.tagName.toLowerCase(),
					visible: container.offsetParent !== null
				});
			}
		});

		// 4. 采集所有链接
		document.querySelectorAll('a[href]').forEach((link) => {
			const text = link.textContent?.trim() || '';
			const classes = link.className ? link.className.split(' ').filter(c => c) : [];

			if (text) {
				result.links.push({
					text: text,
					selector: classes[0] ? 'a.' + classes[0] : 'a',
					classes: classes,
					id: link.id || '',
					tag_name: 'a',
					visible: link.offsetParent !== null
				});
			}
		});

		return result;
	}`

	info, err := page.Eval(jsCode)
	if err != nil {
		logrus.Errorf("采集组件信息失败: %v", err)
		return &PageSnapshot{PageName: pageName}
	}

	// 转换为结构体
	jsonData, _ := json.Marshal(info)
	var snapshot PageSnapshot
	json.Unmarshal(jsonData, &snapshot)
	snapshot.PageName = pageName

	return &snapshot
}

// saveToYAML 保存到 YAML 文件
func saveToYAML(data *AllPagesSnapshot, path string) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	// 添加文件头注释
	header := `# 小红书页面组件全量快照
# 自动生成，请勿手动编辑
# 生成时间: ` + data.Generated + `
# 版本: ` + data.Version + `

`
	content := header + string(yamlData)
	return os.WriteFile(path, []byte(content), 0644)
}

// saveToJSON 保存到 JSON 文件
func saveToJSON(data *AllPagesSnapshot, path string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

// displaySummary 显示汇总信息
func displaySummary(snapshot *AllPagesSnapshot) {
	separator := strings.Repeat("=", 60)
	logrus.Info("\n" + separator)
	logrus.Info("📊 采集汇总")
	logrus.Info(separator)

	logrus.Infof("📅 生成时间: %s", snapshot.Generated)
	logrus.Infof("📄 页面数量: %d", len(snapshot.Pages))

	for name, page := range snapshot.Pages {
		logrus.Infof("\n🔹 %s:", name)
		logrus.Infof("   URL: %s", page.URL)
		logrus.Infof("   按钮: %d | 输入框: %d | 容器: %d | 链接: %d",
			len(page.Buttons), len(page.Inputs), len(page.Containers), len(page.Links))

		// 显示关键按钮
		publishBtn := findElement(page.Buttons, "发布")
		if publishBtn != nil {
			logrus.Infof("   ✓ 发布按钮: %s (classes: %v)", publishBtn.Selector, publishBtn.Classes)
		}

		draftBtn := findElement(page.Buttons, "暂存")
		if draftBtn != nil {
			logrus.Infof("   ✓ 暂存按钮: %s (classes: %v)", draftBtn.Selector, draftBtn.Classes)
		}
	}

	logrus.Info("\n" + separator)
}

// findElement 查找元素
func findElement(elements []ElementInfo, keyword string) *ElementInfo {
	for _, elem := range elements {
		if strings.Contains(elem.Text, keyword) {
			return &elem
		}
	}
	return nil
}

// findCookieFile 自动查找 cookie 文件
func findCookieFile() string {
	// 按优先级查找
	cookiePaths := []string{
		"cookies.json", // 当前目录
		os.Getenv("HOME") + "/.xiaohongshu/cookies.json", // 用户目录
		"./xiaohongshu_cookies.json",                     // 备用名称
	}

	for _, path := range cookiePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// DiscoveredPages discovered_pages.yaml 结构
type DiscoveredPages struct {
	Links map[string]struct {
		Text        string `yaml:"text"`
		URL         string `yaml:"url"`
		Description string `yaml:"description"`
		Category    string `yaml:"category"`
	} `yaml:"links"`
}

// loadPagesFromFile 从 discovered_pages.yaml 加载页面列表
func loadPagesFromFile(path string) ([]PageDefinition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var discovered DiscoveredPages
	if err := yaml.Unmarshal(data, &discovered); err != nil {
		return nil, err
	}

	var pages []PageDefinition
	for name, link := range discovered.Links {
		// 过滤掉不需要的页面（如帮助、设置等）
		if shouldSkip(link.Category) {
			continue
		}

		pages = append(pages, PageDefinition{
			Name: name,
			URL:  link.URL,
			Desc: link.Description,
		})
	}

	return pages, nil
}

// shouldSkip 判断是否跳过某个类别
func shouldSkip(category string) bool {
	skipCategories := []string{
		"help",
		"setting",
		"other",
	}

	for _, skip := range skipCategories {
		if category == skip {
			return true
		}
	}

	return false
}

// createTestImage 创建一个1x1像素的PNG测试图片
func createTestImage() string {
	// 最小的1x1透明PNG (67字节)
	pngData := []byte{
		0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A,
		0x00, 0x00, 0x00, 0x0D, 0x49, 0x48, 0x44, 0x52,
		0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01,
		0x08, 0x06, 0x00, 0x00, 0x00, 0x1F, 0x15, 0xC4,
		0x89, 0x00, 0x00, 0x00, 0x0A, 0x49, 0x44, 0x41,
		0x54, 0x78, 0x9C, 0x63, 0x00, 0x01, 0x00, 0x00,
		0x05, 0x00, 0x01, 0x0D, 0x0A, 0x2D, 0xB4, 0x00,
		0x00, 0x00, 0x00, 0x49, 0x45, 0x4E, 0x44, 0xAE,
		0x42, 0x60, 0x82,
	}

	tmpfile := "/tmp/test_upload_" + fmt.Sprintf("%d", time.Now().Unix()) + ".png"
	os.WriteFile(tmpfile, pngData, 0644)
	return tmpfile
}
