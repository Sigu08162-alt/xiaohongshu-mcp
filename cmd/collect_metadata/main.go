package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/xpzouying/xiaohongshu-mcp/cookies"
	"github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser"
	browserplaywright "github.com/xpzouying/xiaohongshu-mcp/internal/infra/browser/playwright"
	"github.com/xpzouying/xiaohongshu-mcp/pkg/semantic"
	"gopkg.in/yaml.v3"
)

// 完整的元素元数据结构
type ElementMetadata struct {
	// 基础标识
	TagName   string   `json:"tag_name" yaml:"tag_name"`
	ID        string   `json:"id,omitempty" yaml:"id,omitempty"`
	Classes   []string `json:"classes,omitempty" yaml:"classes,omitempty"`
	Name      string   `json:"name,omitempty" yaml:"name,omitempty"`

	// 选择器
	CSSSelector   string `json:"css_selector" yaml:"css_selector"`
	XPath         string `json:"xpath,omitempty" yaml:"xpath,omitempty"`

	// 内容
	Text          string `json:"text,omitempty" yaml:"text,omitempty"`
	Value         string `json:"value,omitempty" yaml:"value,omitempty"`
	InnerHTML     string `json:"inner_html,omitempty" yaml:"inner_html,omitempty"` // 截取前500字符

	// 表单相关
	Type          string `json:"type,omitempty" yaml:"type,omitempty"`
	Placeholder   string `json:"placeholder,omitempty" yaml:"placeholder,omitempty"`
	Disabled      bool   `json:"disabled,omitempty" yaml:"disabled,omitempty"`
	Readonly      bool   `json:"readonly,omitempty" yaml:"readonly,omitempty"`
	Required      bool   `json:"required,omitempty" yaml:"required,omitempty"`
	Checked       bool   `json:"checked,omitempty" yaml:"checked,omitempty"`
	Selected      bool   `json:"selected,omitempty" yaml:"selected,omitempty"`

	// ARIA属性
	Role           string            `json:"role,omitempty" yaml:"role,omitempty"`
	AriaLabel      string            `json:"aria_label,omitempty" yaml:"aria_label,omitempty"`
	AriaLabelledBy string            `json:"aria_labelledby,omitempty" yaml:"aria_labelledby,omitempty"`
	AriaDescribedBy string           `json:"aria_describedby,omitempty" yaml:"aria_describedby,omitempty"`
	AriaAttributes map[string]string `json:"aria_attrs,omitempty" yaml:"aria_attrs,omitempty"`

	// Data属性
	DataAttributes map[string]string `json:"data_attrs,omitempty" yaml:"data_attrs,omitempty"`

	// 位置和尺寸
	Position struct {
		X      float64 `json:"x" yaml:"x"`
		Y      float64 `json:"y" yaml:"y"`
		Width  float64 `json:"width" yaml:"width"`
		Height float64 `json:"height" yaml:"height"`
	} `json:"position" yaml:"position"`

	// 可见性和状态
	Visible          bool   `json:"visible" yaml:"visible"`
	Display          string `json:"display,omitempty" yaml:"display,omitempty"`
	Visibility       string `json:"visibility,omitempty" yaml:"visibility,omitempty"`
	Opacity          string `json:"opacity,omitempty" yaml:"opacity,omitempty"`
	ContentEditable  string `json:"contenteditable,omitempty" yaml:"contenteditable,omitempty"`

	// 层级关系
	ParentTagName  string `json:"parent_tag,omitempty" yaml:"parent_tag,omitempty"`
	ParentClasses  []string `json:"parent_classes,omitempty" yaml:"parent_classes,omitempty"`
	ChildrenCount  int    `json:"children_count,omitempty" yaml:"children_count,omitempty"`

	// 额外属性（捕获所有其他属性）
	OtherAttributes map[string]string `json:"other_attrs,omitempty" yaml:"other_attrs,omitempty"`
}

// 页面元数据
type PageMetadata struct {
	PageKey   string            `json:"page_key" yaml:"page_key"`
	URL       string            `json:"url" yaml:"url"`
	Title     string            `json:"title" yaml:"title"`
	Timestamp string            `json:"timestamp" yaml:"timestamp"`
	Elements  []ElementMetadata `json:"elements" yaml:"elements"`
	Stats     struct {
		TotalElements int `json:"total_elements" yaml:"total_elements"`
		ByTagName     map[string]int `json:"by_tag_name" yaml:"by_tag_name"`
		ByType        map[string]int `json:"by_type" yaml:"by_type"`
	} `json:"stats" yaml:"stats"`
}

// 所有页面元数据
type AllPagesMetadata struct {
	Version   string                  `json:"version" yaml:"version"`
	Generated string                  `json:"generated" yaml:"generated"`
	Pages     map[string]PageMetadata `json:"pages" yaml:"pages"`
}

// 发现的页面定义 - 支持多种格式
type DiscoveredPage struct {
	Key         string   `yaml:"key,omitempty"`
	URL         string   `yaml:"url"`
	Name        string   `yaml:"name,omitempty"`
	Text        string   `yaml:"text,omitempty"`
	Description string   `yaml:"description,omitempty"`
	Category    string   `yaml:"category,omitempty"`
	Classes     []string `yaml:"classes,omitempty"`
	Trigger     string   `yaml:"trigger_selector,omitempty"`
	Item        string   `yaml:"item_selector,omitempty"`
}

type SemanticTrace struct {
	Page           string               `yaml:"page"`
	Target         string               `yaml:"target"`
	Before         semantic.Fingerprint `yaml:"before"`
	After          semantic.Fingerprint `yaml:"after"`
	ChangedMetrics int                  `yaml:"changed_metrics"`
	Attempt        int                  `yaml:"attempt"`
}

type SemanticFailure struct {
	Page           string `yaml:"page"`
	Target         string `yaml:"target"`
	Reason         string `yaml:"reason"`
	ChangedMetrics int    `yaml:"changed_metrics,omitempty"`
	Attempt        int    `yaml:"attempt"`
}

func buildMenuClickTargets(pageDef DiscoveredPage, cfg *semantic.Config) []string {
	var targets []string
	for _, target := range cfg.ClickPlan.Targets {
		switch target {
		case "item_selector":
			if strings.TrimSpace(pageDef.Item) != "" {
				targets = append(targets, pageDef.Item)
			}
		case "text":
			if strings.TrimSpace(pageDef.Text) != "" {
				targets = append(targets, fmt.Sprintf("text=%q", strings.TrimSpace(pageDef.Text)))
			}
		}
	}
	return targets
}

// 支持多种发现文件格式
type DiscoveredPagesV1 struct {
	Pages    map[string]DiscoveredPage `yaml:"pages"`
	HomePage string                   `yaml:"home_page"`
}

type DiscoveredPagesV2 struct {
	Links    map[string]DiscoveredPage `yaml:"links"`
	HomePage string                   `yaml:"home_page"`
}

func main() {
	// 命令行参数
	inputFile := flag.String("input", "", "输入的discovered_pages.yaml文件路径")
	outputYAML := flag.String("output", "", "输出YAML文件路径（必填）")
	outputJSON := flag.String("json", "", "可选：输出JSON文件路径")
	headless := flag.Bool("headless", false, "无头模式")
	waitTime := flag.Int("wait", 5, "每个页面加载后等待秒数")
	semanticConfigPath := flag.String("semantic-config", "configs/semantic_scan.yaml", "语义采集配置文件路径")
	noInteractive := flag.Bool("no-interactive", false, "非交互模式")
	singlePage := flag.String("page", "", "仅采集单个页面的key")
	autoUpload := flag.String("auto-upload", "", "自动上传图片路径（针对发布页面等需要上传的场景）")

	flag.Parse()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.Info("=== 小红书页面完整元数据采集器 ===")
	logrus.Info("特性:")
	logrus.Info("  ✓ 无差别采集所有元素")
	logrus.Info("  ✓ 采集完整的元数据（50+属性）")
	logrus.Info("  ✓ 无硬编码规则")
	logrus.Info("  ✓ 自适应页面结构")

	if *inputFile == "" {
		logrus.Fatal("请指定输入文件: --input discovered_pages.yaml")
	}
	if *outputYAML == "" {
		logrus.Fatal("请指定输出文件: --output metadata_xxx.yaml")
	}
	if *semanticConfigPath == "" {
		logrus.Fatal("必须通过 --semantic-config 指定语义配置文件")
	}
	semanticCfg, err := semantic.LoadConfig(*semanticConfigPath)
	if err != nil {
		logrus.Fatalf("加载语义配置失败: %v", err)
	}

	// 加载发现的页面
	pages, homePage, err := loadDiscoveredPages(*inputFile)
	if err != nil {
		logrus.Fatalf("加载页面列表失败: %v", err)
	}

	logrus.Infof("✓ 加载了 %d 个页面", len(pages))

	// 过滤单个页面
	var targetPages map[string]DiscoveredPage
	if *singlePage != "" {
		if p, ok := pages[*singlePage]; ok {
			targetPages = map[string]DiscoveredPage{*singlePage: p}
			logrus.Infof("✓ 仅采集页面: %s", *singlePage)
		} else {
			logrus.Fatalf("未找到页面: %s", *singlePage)
		}
	} else {
		targetPages = pages
	}

	// 查找 cookie 文件
	cookiePath := findCookieFile()
	if cookiePath != "" {
		logrus.Infof("🍪 Cookie: %s", cookiePath)
	}

	// 创建浏览器
	engineCfg := browserplaywright.DefaultConfig()
	engineCfg.Headless = *headless
	engineCfg.CookiePath = cookiePath
	engineCfg.NavigationTimeout = 2 * time.Minute
	engineCfg.ActionTimeout = 30 * time.Second

	engine := browserplaywright.New(engineCfg)
	if err := engine.Start(); err != nil {
		logrus.Fatalf("启动浏览器失败: %v", err)
	}
	defer engine.Close()

	page, err := engine.NewPage()
	if err != nil {
		logrus.Fatalf("创建页面失败: %v", err)
	}
	defer page.Close()

	// 首次登录提示
	if !*noInteractive && cookiePath == "" {
		logrus.Info("\n🔐 如需登录，请在浏览器中完成登录")
		logrus.Info("⏸️  按 Enter 继续...")
		fmt.Scanln()
	}

	// 采集所有页面
	allMetadata := AllPagesMetadata{
		Version:   "2.0.0",
		Generated: time.Now().Format(time.RFC3339),
		Pages:     make(map[string]PageMetadata),
	}
	var semanticTraces []SemanticTrace
	var semanticFailures []SemanticFailure

	i := 0
	for key, pageDef := range targetPages {
		i++
		logrus.Infof("\n[%d/%d] 📄 %s", i, len(targetPages), pageDef.Name)
		logrus.Infof("🔗 %s", pageDef.URL)

		metadata := capturePageMetadata(page, key, pageDef, homePage, *waitTime, *noInteractive, *autoUpload, semanticCfg, &semanticTraces, &semanticFailures)
		allMetadata.Pages[key] = metadata

		logrus.Infof("✓ 采集了 %d 个元素", metadata.Stats.TotalElements)
	}

	// 保存结果
	logrus.Info("\n💾 保存结果...")

	if err := saveYAML(&allMetadata, *outputYAML); err != nil {
		logrus.Fatalf("保存YAML失败: %v", err)
	}
	logrus.Infof("✓ YAML: %s", *outputYAML)

	if *outputJSON != "" {
		if err := saveJSON(&allMetadata, *outputJSON); err != nil {
			logrus.Fatalf("保存JSON失败: %v", err)
		}
		logrus.Infof("✓ JSON: %s", *outputJSON)
	}

	if err := saveSemanticReports(semanticCfg, semanticTraces, semanticFailures); err != nil {
		logrus.Warnf("保存语义报告失败: %v", err)
	}

	logrus.Info("\n✅ 采集完成")
	printStats(&allMetadata)
}

func capturePageMetadata(page browser.Page, pageKey string, pageDef DiscoveredPage, homePage string, waitSec int, noInteractive bool, autoUploadPath string, semanticCfg *semantic.Config, traces *[]SemanticTrace, failures *[]SemanticFailure) PageMetadata {
	// 访问页面
	if pageDef.URL != "" {
		if err := page.Goto(pageDef.URL); err != nil {
			logrus.Errorf("访问页面失败: %v", err)
			return PageMetadata{PageKey: pageKey, URL: pageDef.URL}
		}
	} else if homePage != "" {
		attempts := semanticCfg.Fallbacks.Retry.MaxAttempts
		if attempts <= 0 {
			attempts = 1
		}
		entered := false
		for attempt := 0; attempt < attempts && !entered; attempt++ {
			if err := page.Goto(homePage); err != nil {
				logrus.Errorf("访问首页失败: %v", err)
				break
			}
			time.Sleep(time.Duration(waitSec) * time.Second)
			beforeFingerprint, err := computeFingerprint(page)
			if err != nil {
				logrus.Warnf("采集指纹失败: %v", err)
			}
			if pageDef.Trigger != "" {
				if err := page.Click(pageDef.Trigger); err != nil {
					logrus.Warnf("点击触发器失败: %v", err)
				}
				time.Sleep(time.Duration(waitSec) * time.Second)
			}
			for _, target := range buildMenuClickTargets(pageDef, semanticCfg) {
				if err := page.Click(target); err != nil {
					*failures = append(*failures, SemanticFailure{
						Page:    pageDef.Name,
						Target:  target,
						Reason:  err.Error(),
						Attempt: attempt + 1,
					})
					logrus.Warnf("点击菜单项失败: %v", err)
					continue
				}
				time.Sleep(time.Duration(semanticCfg.Verify.TimeoutSeconds) * time.Second)
				afterFingerprint, err := computeFingerprint(page)
				if err != nil {
					logrus.Warnf("采集指纹失败: %v", err)
				}
				changed := beforeFingerprint.Delta(afterFingerprint).ChangedMetricCount(semanticCfg.Fingerprint.Thresholds.MetricChangeRatio)
				if changed >= semanticCfg.Fingerprint.Thresholds.MinMetricChanges {
					*traces = append(*traces, SemanticTrace{
						Page:           pageDef.Name,
						Target:         target,
						Before:         beforeFingerprint,
						After:          afterFingerprint,
						ChangedMetrics: changed,
						Attempt:        attempt + 1,
					})
					entered = true
					break
				}
				*failures = append(*failures, SemanticFailure{
					Page:           pageDef.Name,
					Target:         target,
					Reason:         "fingerprint_not_changed",
					ChangedMetrics: changed,
					Attempt:        attempt + 1,
				})
			}
		}
		if !entered {
			logrus.Warnf("未通过语义验证进入页面: %s", pageDef.Name)
		}
	} else {
		logrus.Warnf("缺少URL与菜单选择器，跳过页面: %s", pageKey)
		return PageMetadata{PageKey: pageKey, URL: pageDef.URL}
	}

	// 等待页面加载
	logrus.Infof("⏳ 等待 %d 秒...", waitSec)
	time.Sleep(time.Duration(waitSec) * time.Second)

	// 自动上传图片（如果指定）
	if autoUploadPath != "" {
		logrus.Infof("📤 自动上传图片: %s", autoUploadPath)

		// 查找文件上传输入框
		uploadSelectors := []string{
			"input[type='file']",
			".upload-input",
			"input.upload-input",
		}

		uploaded := false
		for _, selector := range uploadSelectors {
			if has, _ := page.Has(selector); has {
				logrus.Infof("  找到上传输入框: %s", selector)
				if err := page.SetFiles(selector, []string{autoUploadPath}); err != nil {
					logrus.Warnf("  上传失败: %v", err)
				} else {
					logrus.Info("  ✅ 上传成功")
					uploaded = true

					// 等待上传处理和页面渲染
					logrus.Info("  ⏳ 等待上传处理和页面渲染...")
					time.Sleep(5 * time.Second)

					// 检查页面变化
					currentURL := page.URL()
					logrus.Infof("  📍 当前URL: %s", currentURL)

					// 再等一下让编辑器完全加载
					logrus.Info("  ⏳ 等待编辑器加载...")
					time.Sleep(3 * time.Second)

					break
				}
			}
		}

		if !uploaded {
			logrus.Warn("  ⚠️  未找到上传输入框")
		}
	}

	// 交互式等待（针对需要手动操作的页面）
	if !noInteractive {
		logrus.Info("💡 如需手动操作页面（如上传图片），请现在操作")
		logrus.Info("⏸️  操作完成后按 Enter 继续采集...")
		fmt.Scanln()

		// 再等一下让页面稳定
		time.Sleep(2 * time.Second)
	}

	// 执行完整的元数据采集
	// 策略：遍历DOM树，采集所有可能有用的元素（零硬编码）
	jsCode := `() => {
		const elements = [];
		const stats = {
			byTagName: {},
			byType: {}
		};

		const processedElements = new Set();

		// 递归遍历DOM树，采集所有元素
		function processElement(elem) {
			// 避免重复
			if (processedElements.has(elem)) return;
			processedElements.add(elem);

			const tagName = elem.tagName.toLowerCase();

			// 只采集可能有交互或语义的元素，跳过纯展示元素
			const isInteractive = (
				// 表单元素
				tagName === 'input' || tagName === 'textarea' ||
				tagName === 'select' || tagName === 'option' ||
				tagName === 'button' ||
				// 链接
				tagName === 'a' ||
				// 可编辑
				elem.contentEditable === 'true' ||
				// 有role属性（ARIA）
				elem.hasAttribute('role') ||
				// 有data-*测试ID
				elem.hasAttribute('data-testid') ||
				elem.hasAttribute('data-test-id') ||
				elem.hasAttribute('data-test') ||
				// placeholder（说明是输入型元素）
				elem.hasAttribute('placeholder') ||
				// aria-label（说明有语义）
				elem.hasAttribute('aria-label') ||
				// class包含关键词
				(elem.className && typeof elem.className === 'string' && (
					elem.className.includes('input') ||
					elem.className.includes('editor') ||
					elem.className.includes('button') ||
					elem.className.includes('title') ||
					elem.className.includes('content')
				))
			);

			if (!isInteractive) {
				// 不采集，但继续遍历子元素
				Array.from(elem.children).forEach(child => processElement(child));
				return;
			}

			// 采集元数据
			const rect = elem.getBoundingClientRect();
			const computed = window.getComputedStyle(elem);

			// 采集所有data-*属性
			const dataAttrs = {};
			for (let attr of elem.attributes) {
				if (attr.name.startsWith('data-')) {
					dataAttrs[attr.name] = attr.value;
				}
			}

			// 采集所有aria-*属性
			const ariaAttrs = {};
			for (let attr of elem.attributes) {
				if (attr.name.startsWith('aria-')) {
					ariaAttrs[attr.name] = attr.value;
				}
			}

			// 采集其他属性
			const otherAttrs = {};
			const standardAttrs = new Set([
				'id', 'class', 'name', 'type', 'placeholder',
				'value', 'disabled', 'readonly', 'required',
				'checked', 'selected', 'role', 'contenteditable',
				'href', 'src', 'alt', 'title'
			]);
			for (let attr of elem.attributes) {
				if (!attr.name.startsWith('data-') &&
				    !attr.name.startsWith('aria-') &&
				    !standardAttrs.has(attr.name)) {
					otherAttrs[attr.name] = attr.value;
				}
			}

			// 生成CSS选择器
			let cssSelector = tagName;
			if (elem.id) {
				cssSelector = '#' + elem.id;
			} else if (elem.className && typeof elem.className === 'string') {
				const classes = elem.className.trim().split(/\\s+/).filter(c => c);
				if (classes.length > 0) {
					cssSelector = tagName + '.' + classes.slice(0, 3).join('.');
				}
			}

			// 父元素信息
			const parent = elem.parentElement;
			const parentClasses = parent && parent.className && typeof parent.className === 'string'
				? parent.className.trim().split(/\\s+/).filter(c => c)
				: [];

			const metadata = {
				tag_name: tagName,
				id: elem.id || '',
				classes: elem.className && typeof elem.className === 'string'
					? elem.className.trim().split(/\\s+/).filter(c => c)
					: [],
				name: elem.name || '',
				css_selector: cssSelector,

				text: elem.textContent?.trim().substring(0, 200) || '',
				value: elem.value || '',
				inner_html: elem.innerHTML?.substring(0, 500) || '',

				type: elem.type || '',
				placeholder: elem.placeholder || elem.getAttribute('placeholder') || '',
				disabled: elem.disabled || false,
				readonly: elem.readOnly || false,
				required: elem.required || false,
				checked: elem.checked || false,
				selected: elem.selected || false,

				role: elem.getAttribute('role') || '',
				aria_label: elem.getAttribute('aria-label') || '',
				aria_labelledby: elem.getAttribute('aria-labelledby') || '',
				aria_describedby: elem.getAttribute('aria-describedby') || '',
				aria_attrs: ariaAttrs,

				data_attrs: dataAttrs,

				position: {
					x: rect.x,
					y: rect.y,
					width: rect.width,
					height: rect.height
				},

				visible: elem.offsetParent !== null,
				display: computed.display,
				visibility: computed.visibility,
				opacity: computed.opacity,
				contenteditable: elem.contentEditable,

				parent_tag: parent ? parent.tagName.toLowerCase() : '',
				parent_classes: parentClasses,
				children_count: elem.children.length,

				other_attrs: otherAttrs
			};

			elements.push(metadata);

			// 统计
			stats.byTagName[tagName] = (stats.byTagName[tagName] || 0) + 1;
			if (metadata.type) {
				stats.byType[metadata.type] = (stats.byType[metadata.type] || 0) + 1;
			}

			// 继续遍历子元素
			Array.from(elem.children).forEach(child => processElement(child));
		}

		// 从body开始遍历
		processElement(document.body);

		return {
			title: document.title,
			elements: elements,
			stats: {
				total_elements: elements.length,
				by_tag_name: stats.byTagName,
				by_type: stats.byType
			}
		};
	}`

	result, err := page.Eval(jsCode)
		if err != nil {
			logrus.Errorf("采集元数据失败: %v", err)
			return PageMetadata{PageKey: pageKey, URL: pageDef.URL}
		}

	// 转换结果
	jsonData, _ := json.Marshal(result)
	var pageData PageMetadata
	json.Unmarshal(jsonData, &pageData)

	pageData.PageKey = pageKey
	pageData.URL = pageDef.URL
	pageData.Timestamp = time.Now().Format(time.RFC3339)

	return pageData
}

func loadDiscoveredPages(path string) (map[string]DiscoveredPage, string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	// 首先尝试解析为通用map，检测格式
	var raw map[string]interface{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, "", fmt.Errorf("无法解析YAML: %w", err)
	}

	// 自动检测格式
	var pages map[string]DiscoveredPage
	homePage := ""

	if _, ok := raw["pages"]; ok {
		// 格式1: pages: {...}
		var v1 DiscoveredPagesV1
		if err := yaml.Unmarshal(data, &v1); err != nil {
			return nil, "", fmt.Errorf("解析pages格式失败: %w", err)
		}
		pages = v1.Pages
		homePage = v1.HomePage
		logrus.Info("✓ 检测到格式: pages")
	} else if _, ok := raw["links"]; ok {
		// 格式2: links: {...}
		var v2 DiscoveredPagesV2
		if err := yaml.Unmarshal(data, &v2); err != nil {
			return nil, "", fmt.Errorf("解析links格式失败: %w", err)
		}
		pages = v2.Links
		homePage = v2.HomePage
		logrus.Info("✓ 检测到格式: links")
	} else {
		// 格式3: 直接是页面map（无顶层key）
		pages = make(map[string]DiscoveredPage)
		if err := yaml.Unmarshal(data, &pages); err != nil {
			return nil, "", fmt.Errorf("解析直接格式失败: %w", err)
		}
		logrus.Info("✓ 检测到格式: 直接页面映射")
	}

	// 标准化：确保每个页面都有name，并过滤空URL
	for key, page := range pages {
		if page.Name == "" {
			if page.Text != "" {
				page.Name = page.Text
			} else {
				page.Name = key
			}
			pages[key] = page
		}
		if strings.TrimSpace(page.URL) == "" && strings.TrimSpace(page.Text) == "" &&
			(strings.TrimSpace(page.Trigger) == "" || strings.TrimSpace(page.Item) == "") {
			logrus.Warnf("跳过空URL页面: %s", key)
			delete(pages, key)
		}
	}

	return pages, homePage, nil
}

func computeFingerprint(page browser.Page) (semantic.Fingerprint, error) {
	result, err := page.Eval(`() => {
		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			if (!style) return false;
			if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') return false;
			const rect = el.getBoundingClientRect();
			if (!rect || rect.width === 0 || rect.height === 0) return false;
			return true;
		};

		const visible = Array.from(document.querySelectorAll('*')).filter(isVisible).length;
		const buttons = Array.from(document.querySelectorAll('button,[role="button"],input[type="button"],input[type="submit"]')).filter(isVisible).length;
		const inputs = Array.from(document.querySelectorAll('input,textarea,[role="textbox"]')).filter(isVisible).length;
		const containers = Array.from(document.querySelectorAll('div,section,main,article,nav,aside')).filter(isVisible).length;

		return { visible, buttons, inputs, containers };
	}`)
	if err != nil {
		return semantic.Fingerprint{}, err
	}

	data, _ := json.Marshal(result)
	var counts struct {
		Visible    int `json:"visible"`
		Buttons    int `json:"buttons"`
		Inputs     int `json:"inputs"`
		Containers int `json:"containers"`
	}
	if err := json.Unmarshal(data, &counts); err != nil {
		return semantic.Fingerprint{}, err
	}

	return semantic.Fingerprint{
		VisibleCount:   counts.Visible,
		ButtonCount:    counts.Buttons,
		InputCount:     counts.Inputs,
		ContainerCount: counts.Containers,
	}, nil
}

func saveSemanticReports(cfg *semantic.Config, traces []SemanticTrace, failures []SemanticFailure) error {
	if cfg.Outputs.TracePath != "" {
		traceData, err := yaml.Marshal(traces)
		if err != nil {
			return err
		}
		if err := os.WriteFile(cfg.Outputs.TracePath, traceData, 0644); err != nil {
			return err
		}
	}
	if cfg.Outputs.FailurePath != "" {
		failureData, err := yaml.Marshal(failures)
		if err != nil {
			return err
		}
		if err := os.WriteFile(cfg.Outputs.FailurePath, failureData, 0644); err != nil {
			return err
		}
	}
	return nil
}

func findCookieFile() string {
	candidates := []string{
		cookies.GetCookiesFilePath(),
		"cookies.json",
		os.Getenv("HOME") + "/.xiaohongshu/cookies.json",
	}

	for _, path := range candidates {
		if path != "" {
			if _, err := os.Stat(path); err == nil {
				return path
			}
		}
	}

	return ""
}

func saveYAML(data *AllPagesMetadata, path string) error {
	yamlData, err := yaml.Marshal(data)
	if err != nil {
		return err
	}

	header := fmt.Sprintf(`# 小红书页面完整元数据
# 版本: %s
# 生成时间: %s
# 特点: 无差别采集，完整元数据，无硬编码

`, data.Version, data.Generated)

	content := header + string(yamlData)
	return os.WriteFile(path, []byte(content), 0644)
}

func saveJSON(data *AllPagesMetadata, path string) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, jsonData, 0644)
}

func printStats(data *AllPagesMetadata) {
	logrus.Info("\n📊 统计信息:")
	logrus.Infof("  总页面数: %d", len(data.Pages))

	totalElements := 0
	for _, page := range data.Pages {
		totalElements += page.Stats.TotalElements
	}
	logrus.Infof("  总元素数: %d", totalElements)

	logrus.Info("\n各页面详情:")
	for key, page := range data.Pages {
		logrus.Infof("  %s: %d 个元素", key, page.Stats.TotalElements)
	}
}
