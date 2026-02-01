package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"net/url"
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
	Trigger     string   `yaml:"trigger_selector,omitempty" json:"trigger_selector,omitempty"`
	Item        string   `yaml:"item_selector,omitempty" json:"item_selector,omitempty"`
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
	outputYAML := flag.String("output", "", "输出YAML文件路径（必填）")
	outputJSON := flag.String("json", "", "可选：同时输出JSON文件")
	headless := flag.Bool("headless", false, "无头模式（默认有头）")
	homeURL := flag.String("home", "https://creator.xiaohongshu.com/new/home?source=official", "起始页面URL")
	systemType := flag.String("system", "creator", "系统类型: creator(创作者) 或 user(普通用户)")
	waitTime := flag.Int("wait", 5, "页面加载等待秒数")
	cookiePath := flag.String("cookies", "", "Cookie文件路径（默认自动查找）")
	noInteractive := flag.Bool("no-interactive", false, "非交互模式,跳过所有等待输入")
	flag.Parse()

	logrus.SetLevel(logrus.InfoLevel)
	logrus.SetFormatter(&logrus.TextFormatter{
		FullTimestamp: true,
		ForceColors:   true,
	})

	if *outputYAML == "" {
		logrus.Fatal("必须通过 --output 指定输出文件")
	}

	// 根据系统类型设置默认起始页
	if *homeURL == "https://creator.xiaohongshu.com/new/home?source=official" && *systemType == "user" {
		*homeURL = "https://www.xiaohongshu.com/explore"
	}

	logrus.Info("=== 小红书页面链接自动发现工具 ===")
	logrus.Infof("系统类型: %s", *systemType)
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
	links := discoverLinks(page, *systemType)

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
func discoverLinks(page browser.Page, systemType string) []LinkInfo {
	// 根据系统类型调用不同的发现策略
	if systemType == "user" {
		return discoverUserSystemLinks(page)
	}
	return discoverCreatorSystemLinks(page)
}

// discoverCreatorSystemLinks 发现创作者系统的链接
func discoverCreatorSystemLinks(page browser.Page) []LinkInfo {
	jsCode := `() => {
		const links = [];
		const triggers = [];

		const roots = [document];
		document.querySelectorAll('*').forEach(el => {
			if (el.shadowRoot) {
				roots.push(el.shadowRoot);
			}
		});

		const queryAll = (selector) => {
			const result = [];
			roots.forEach(root => {
				try {
					root.querySelectorAll(selector).forEach(el => result.push(el));
				} catch (e) {
					// ignore
				}
			});
			return result;
		};

		// 调试:输出页面信息
		console.log('=== 开始链接发现 ===');
		console.log('URL:', window.location.href);
		console.log('所有 a 标签数量:', queryAll('a').length);
		console.log('菜单项数量:', queryAll('.menu-item').length);
		console.log('所有可点击元素:', queryAll('[onclick], [role="link"], [role="button"]').length);

		const cssEscape = (value) => {
			if (!value) return '';
			if (window.CSS && CSS.escape) {
				return CSS.escape(value);
			}
			return value.replace(/[^a-zA-Z0-9_\\-]/g, '\\\\$&');
		};

		const buildSelector = (el) => {
			if (!el || el.nodeType !== 1) return '';
			const parts = [];
			let current = el;
			while (current && current.nodeType === 1 && current !== document.body) {
				const tag = current.tagName.toLowerCase();
				if (current.id) {
					parts.unshift(tag + '#' + cssEscape(current.id));
					break;
				}
				let part = tag;
				const parent = current.parentElement;
				if (parent) {
					const siblings = Array.from(parent.children).filter(c => c.tagName === current.tagName);
					if (siblings.length > 1) {
						const index = siblings.indexOf(current) + 1;
						part += ':nth-of-type(' + index + ')';
					}
				}
				parts.unshift(part);
				current = parent;
			}
			return parts.join(' > ');
		};

		const normalizeUrl = (raw) => {
			if (!raw || typeof raw !== 'string') return '';
			return raw.trim();
		};

		const isVisible = (el) => {
			if (!el) return false;
			const style = window.getComputedStyle(el);
			if (!style) return false;
			if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') return false;
			if (el.offsetParent === null && style.position !== 'fixed') return false;
			return true;
		};

		const getVueLink = (el) => {
			const comp = el.__vueParentComponent;
			if (!comp) return '';
			const props = (comp.props || comp.vnode?.props) || {};
			return props.to || props.href || props.path || '';
		};

		const getReactLink = (el) => {
			const keys = Object.keys(el || {});
			const reactKey = keys.find(k => k.startsWith('__reactProps$'));
			if (!reactKey) return '';
			const props = el[reactKey] || {};
			return props.to || props.href || props.path || '';
		};

		const getAttrLink = (el) => {
			const attrs = Array.from(el.attributes || []);
			for (const attr of attrs) {
				const name = attr.name || '';
				const value = attr.value || '';
				if (!value) continue;
				if (name.includes('href') || name.includes('to') || name.includes('path') || name.includes('url')) {
					return value;
				}
			}
			return '';
		};

		const getDataLink = (el) => {
			const dataset = el.dataset || {};
			for (const key of Object.keys(dataset)) {
				const value = dataset[key];
				if (!value) continue;
				if (key.toLowerCase().includes('url') || key.toLowerCase().includes('href') || key.toLowerCase().includes('path') || key.toLowerCase().includes('to')) {
					return value;
				}
			}
			return '';
		};

		const resolveLink = (el) => {
			return normalizeUrl(
				el.getAttribute('href') ||
				el.getAttribute('data-href') ||
				el.getAttribute('data-to') ||
				el.getAttribute('data-path') ||
				el.getAttribute('to') ||
				getDataLink(el) ||
				getAttrLink(el) ||
				getVueLink(el) ||
				getReactLink(el)
			);
		};

		// 策略1: 尝试查找菜单项(小红书使用div+路由,不是a标签)
		queryAll('.menu-item, [class*="menu"]').forEach((item, idx) => {
			// 尝试从元素或父元素获取路径信息
			const text = item.textContent?.trim() || '';

			const dataTo = item.getAttribute('data-to') ||
			              item.getAttribute('data-path') ||
			              item.getAttribute('data-href') ||
			              '';

			const onClick = item.getAttribute('onclick') || '';
			const to = item.getAttribute('to') || '';
			const resolved = resolveLink(item);

			if (idx < 20) {
				console.log('菜单项 ' + idx + ':', {
					text: text.substring(0, 30),
					dataTo: dataTo,
					onClick: onClick.substring(0, 50),
					to: to,
					resolved: resolved,
					className: item.className
				});
			}

			// 如果有文本,尝试提取（不做筛选）
			if (text && text.length > 0 && text.length < 100) {
				links.push({
					text: text,
					url: resolved,
					classes: item.className ? item.className.split(' ').filter(c => c) : [],
					trigger_selector: buildSelector(item),
					item_selector: buildSelector(item)
				});
			}
		});

		// 策略2: 查找所有带href的a标签(即使数量为0也尝试)
		queryAll('a[href]').forEach((link, idx) => {
			const href = link.href;
			const text = link.textContent?.trim() || '';

			if (idx < 10) {
				console.log('链接 ' + idx + ':', {
					text: text.substring(0, 30),
					href: href.substring(0, 80),
					includes_creator: href.includes('creator.xiaohongshu.com')
				});
			}

			if (href) {
				const classes = link.className ? link.className.split(' ').filter(c => c) : [];
				links.push({
					text: text || link.getAttribute('aria-label') || link.title || '',
					url: href,
					classes: classes,
					parent_text: link.parentElement?.textContent?.trim().substring(0, 50) || '',
					trigger_selector: buildSelector(link),
					item_selector: buildSelector(link)
				});
				console.log('✓ 添加链接:', text.substring(0, 30), '→', href.substring(0, 60));
			}
		});

		// 策略3: 从全局状态中提取URL（若存在），不使用硬编码正则
		try {
			if (window.__INITIAL_STATE__) {
				const queue = [window.__INITIAL_STATE__];
				const seen = new Set();
				const maxNodes = 20000;
				let nodes = 0;

				const tryAddUrl = (value) => {
					if (!value || typeof value !== 'string') return;
					let candidate = value.trim();
					if (!candidate) return;
					if (candidate.startsWith('#')) return;
					if (candidate.startsWith('javascript:')) return;

					let url = '';
					try {
						url = new URL(candidate, window.location.origin).toString();
					} catch (e) {
						return;
					}

					if (!url) return;
					links.push({
						text: '',
						url: url,
						classes: ['from-state'],
						category: 'dynamic',
						trigger_selector: '',
						item_selector: ''
					});
				};

				while (queue.length > 0 && nodes < maxNodes) {
					const current = queue.shift();
					if (!current || typeof current !== 'object') continue;
					if (seen.has(current)) continue;
					seen.add(current);
					nodes += 1;

					if (Array.isArray(current)) {
						for (const item of current) {
							if (typeof item === 'string') {
								tryAddUrl(item);
							} else if (item && typeof item === 'object') {
								queue.push(item);
							}
						}
						continue;
					}

					for (const key of Object.keys(current)) {
						const value = current[key];
						if (typeof value === 'string') {
							tryAddUrl(value);
						} else if (value && typeof value === 'object') {
							queue.push(value);
						}
					}
				}
			}
		} catch (e) {
			console.warn('state parse failed', e);
		}

		// 策略5: 触发器收集（下拉/弹层）
		queryAll('[aria-haspopup], [aria-expanded], [role="button"], [data-popper-placement], [data-popper-reference-hidden]').forEach(el => {
			if (!isVisible(el)) return;
			const text = el.textContent?.trim() || '';
			if (text.length > 20) return;
			const selector = buildSelector(el);
			if (selector) {
				triggers.push(selector);
			}
		});

		console.log('总共发现:', links.length, '个链接');
		console.log('=== 链接发现完成 ===');

		return { links, triggers };
	}`

	info, err := page.Eval(jsCode)
	if err != nil {
		logrus.Errorf("发现链接失败: %v", err)
		return []LinkInfo{}
	}

	// 解析结果
	jsonData, _ := json.Marshal(info)
	var result struct {
		Links    []LinkInfo `json:"links"`
		Triggers []string   `json:"triggers"`
	}
	json.Unmarshal(jsonData, &result)

	links := result.Links
	triggerSeen := map[string]struct{}{}
	for _, selector := range result.Triggers {
		if selector == "" {
			continue
		}
		if _, ok := triggerSeen[selector]; ok {
			continue
		}
		triggerSeen[selector] = struct{}{}
		_ = page.Click(selector)
		time.Sleep(500 * time.Millisecond)

		menuInfo, err := page.Eval(`() => {
			const items = [];
			const isVisible = (el) => {
				if (!el) return false;
				const style = window.getComputedStyle(el);
				if (!style) return false;
				if (style.display === 'none' || style.visibility === 'hidden' || style.opacity === '0') return false;
				if (el.offsetParent === null && style.position !== 'fixed') return false;
				return true;
			};
			const buildSelector = (el) => {
				if (!el || el.nodeType !== 1) return '';
				const parts = [];
				let current = el;
				while (current && current.nodeType === 1 && current !== document.body) {
					const tag = current.tagName.toLowerCase();
					if (current.id) {
						const escaped = window.CSS && CSS.escape ? CSS.escape(current.id) : current.id;
						parts.unshift(tag + '#' + escaped);
						break;
					}
					let part = tag;
					const parent = current.parentElement;
					if (parent) {
						const siblings = Array.from(parent.children).filter(c => c.tagName === current.tagName);
						if (siblings.length > 1) {
							const index = siblings.indexOf(current) + 1;
							part += ':nth-of-type(' + index + ')';
						}
					}
					parts.unshift(part);
					current = parent;
				}
				return parts.join(' > ');
			};
			document.querySelectorAll('[role=\"menuitem\"], [role=\"option\"], [role=\"link\"], a, button, li, div').forEach(item => {
				if (!isVisible(item)) return;
				const text = item.textContent?.trim() || '';
				if (!text || text.length > 20) return;
				const href = item.getAttribute('href') || '';
				items.push({
					text,
					url: href,
					item_selector: buildSelector(item)
				});
			});
			return items;
		}`)
		if err == nil {
			var items []LinkInfo
			if data, err := json.Marshal(menuInfo); err == nil {
				_ = json.Unmarshal(data, &items)
			}
			for _, item := range items {
				item.Trigger = selector
				links = append(links, item)
			}
		}
		_ = page.Press("Escape")
	}

	logrus.Infof("📊 发现 %d 个链接", len(links))

	return links
}

// deduplicateLinks 去重链接

// categorizeLinks 分类整理链接
func categorizeLinks(links []LinkInfo) map[string]LinkInfo {
	result := make(map[string]LinkInfo)

	for _, link := range links {
		// 根据 URL 和文本分类
		key := generateKey(link)
		if existing, ok := result[key]; ok {
			merged := mergeLinkInfo(existing, link)
			merged.Category = detectCategory(merged)
			merged.Description = generateDescription(merged)
			result[key] = merged
			continue
		}

		link.Category = detectCategory(link)
		link.Description = generateDescription(link)
		result[key] = link
	}

	return result
}

func mergeLinkInfo(base, incoming LinkInfo) LinkInfo {
	if base.Text == "" {
		base.Text = incoming.Text
	}
	if base.URL == "" {
		base.URL = incoming.URL
	}
	if base.Description == "" {
		base.Description = incoming.Description
	}
	if base.Category == "" {
		base.Category = incoming.Category
	}
	if base.Trigger == "" {
		base.Trigger = incoming.Trigger
	}
	if base.Item == "" {
		base.Item = incoming.Item
	}
	if len(base.Classes) == 0 {
		base.Classes = incoming.Classes
	} else if len(incoming.Classes) > 0 {
		base.Classes = appendUniqueStrings(base.Classes, incoming.Classes)
	}
	return base
}

func appendUniqueStrings(existing []string, incoming []string) []string {
	seen := make(map[string]struct{}, len(existing))
	for _, item := range existing {
		seen[item] = struct{}{}
	}
	for _, item := range incoming {
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		existing = append(existing, item)
		seen[item] = struct{}{}
	}
	return existing
}

// generateKey 生成链接的唯一键
func generateKey(link LinkInfo) string {
	// 根据 URL 路径生成键
	if link.URL != "" {
		if parsed, err := url.Parse(link.URL); err == nil && parsed.Path != "" {
			path := strings.TrimPrefix(parsed.Path, "/")
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
	return "other"
}

// generateDescription 生成链接描述
func generateDescription(link LinkInfo) string {
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

// discoverUserSystemLinks 发现普通用户系统的链接
func discoverUserSystemLinks(page browser.Page) []LinkInfo {
	jsCode := `() => {
		const links = [];

		console.log('=== 发现普通用户系统链接 ===');
		console.log('URL:', window.location.href);

		// 策略1: 查找导航栏的链接
		document.querySelectorAll('nav a, header a, .nav a, [class*="nav"] a').forEach(link => {
			const href = link.href;
			const text = link.textContent?.trim() || '';

			if (href && text) {
				links.push({
					text: text,
					url: href,
					classes: link.className ? link.className.split(' ').filter(c => c) : [],
					category: 'navigation'
				});
				console.log('✓ 导航链接:', text, '→', href);
			}
		});

		// 策略2: 常见的用户系统页面
		const userPages = [
			{ text: '发现', path: '/explore' },
			{ text: '关注', path: '/followFeed' },
			{ text: '我的主页', path: '/user/profile' },
			{ text: '消息', path: '/chat' },
			{ text: '购物', path: '/lifestyle/shop' },
			{ text: '搜索', path: '/search_result' }
		];

		const pageText = document.body.textContent || '';
		userPages.forEach(({ text, path }) => {
			if (pageText.includes(text)) {
				const url = window.location.origin + path;
				links.push({
					text: text,
					url: url,
					classes: ['inferred'],
					category: 'user-page'
				});
				console.log('✓ 推测用户页面:', text, '→', url);
			}
		});

		console.log('总共发现:', links.length, '个链接');
		return links;
	}`

	info, err := page.Eval(jsCode)
	if err != nil {
		logrus.Errorf("发现链接失败: %v", err)
		return []LinkInfo{}
	}

	jsonData, _ := json.Marshal(info)
	var links []LinkInfo
	json.Unmarshal(jsonData, &links)

	logrus.Infof("📊 发现 %d 个链接", len(links))

	return links
}
